package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// sshDialTimeout SSH 建立 TCP 连接的超时时间
// 不设置时 ssh.Dial 在目标主机无响应的情况下会一直等待操作系统级别的 TCP 超时，
// 导致 HTTP 请求长时间挂起。
const sshDialTimeout = 10 * time.Second

// 下面两个超时是包级变量而不是常量，仅为了让单元测试可以缩短到秒级；生产代码不要修改。
var (
	// sshHandshakeTimeout SSH 握手（banner / 密钥交换 / 认证）的最长时间。
	// ssh.ClientConfig.Timeout 只限制 TCP 拨号，ssh.Dial 随后调用 NewClientConn 时没有任何 deadline，
	// 对端接受 TCP 但不发送 SSH banner 时会一直挂住，所以这里自己设置 conn deadline。
	sshHandshakeTimeout = 15 * time.Second
	// sftpIdleTimeout SFTP 上传连接允许的"无任何数据收发"的最长时间（进度续期，而不是总时长上限）。
	// tar 文件可能有几个 GB，总时长无法预估；只要还有数据在流动就会不断续期。
	sftpIdleTimeout = 60 * time.Second
)

// newSSHConfig 构造 SSH 客户端配置（每次调用都返回新的配置，不缓存、不共享）
func newSSHConfig(user, password string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshDialTimeout,
	}
}

// dialSSH 与 ssh.Dial 等价，但握手阶段有整体 deadline：
//  1. net.DialTimeout 建立 TCP 连接（config.Timeout）；
//  2. 对 conn 设置 deadline = now + sshHandshakeTimeout 后再 ssh.NewClientConn；
//  3. 握手成功后清除 deadline（长时间运行的命令不受影响），失败则关闭 conn。
//
// wrap 不为 nil 时，在握手前用它包装底层 net.Conn（SFTP 上传用于统计收发进度）。
func dialSSH(addr string, config *ssh.ClientConfig, wrap func(net.Conn) net.Conn) (*ssh.Client, error) {
	handshakeTimeout := sshHandshakeTimeout

	conn, err := net.DialTimeout("tcp", addr, config.Timeout)
	if err != nil {
		return nil, err
	}
	if wrap != nil {
		conn = wrap(conn)
	}

	if err := conn.SetDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		_ = conn.Close()
		if isTimeoutErr(err) {
			return nil, fmt.Errorf("ssh handshake timeout after %s: %w", handshakeTimeout, err)
		}
		return nil, err
	}
	// 握手完成，清除 deadline。mux 读 goroutine 此时已经阻塞在 Read 上，清除 deadline 对挂起中的 Read 同样生效。
	if err := conn.SetDeadline(time.Time{}); err != nil {
		_ = c.Close()
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func isTimeoutErr(err error) bool {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// progressConn 包装 net.Conn，记录最近一次成功收发数据的时间，用于判断 SFTP 传输是否卡死。
// 只用于 SFTP 上传连接；执行 `k3s ctr image import` 这类可能静默很久的命令时不使用。
type progressConn struct {
	net.Conn
	last atomic.Int64 // 最近一次进度的 UnixNano
}

func newProgressConn(c net.Conn) *progressConn {
	p := &progressConn{Conn: c}
	p.touch()
	return p
}

func (p *progressConn) touch() { p.last.Store(time.Now().UnixNano()) }

// idleFor 距离最近一次成功收发数据过去了多久
func (p *progressConn) idleFor() time.Duration {
	return time.Duration(time.Now().UnixNano() - p.last.Load())
}

func (p *progressConn) Read(b []byte) (int, error) {
	n, err := p.Conn.Read(b)
	if n > 0 {
		p.touch()
	}
	return n, err
}

func (p *progressConn) Write(b []byte) (int, error) {
	n, err := p.Conn.Write(b)
	if n > 0 {
		p.touch()
	}
	return n, err
}

// watchIdle 启动看门狗：连接连续 idle 时间没有任何成功的读写时调用 onStall（只调用一次）。
// 返回的 stop 用于结束看门狗，可重复调用。
func watchIdle(p *progressConn, idle time.Duration, onStall func()) (stop func()) {
	done := make(chan struct{})
	go func() {
		timer := time.NewTimer(idle)
		defer timer.Stop()
		for {
			select {
			case <-done:
				return
			case <-timer.C:
			}
			remain := idle - p.idleFor()
			if remain <= 0 {
				onStall()
				return
			}
			timer.Reset(remain)
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(done) }) }
}

// SSHExecResult SSH 命令执行结果
type SSHExecResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

// SSHExecCommand 通过 SSH 登录远程服务器并执行命令，返回 stdout 和 stderr
//
//	@param host     目标主机 IP 或域名
//	@param port     SSH 端口，通常为 "22"
//	@param user     SSH 登录用户名
//	@param password SSH 登录密码
//	@param command  要执行的命令
//	@return *SSHExecResult
//	@return error
func SSHExecCommand(host, port, user, password, command string) (*SSHExecResult, error) {
	return SSHExecCommandWithTimeout(host, port, user, password, command, 0)
}

// SSHExecCommandWithTimeout 与 SSHExecCommand 相同，但限制命令的最长执行时间
//
//	@param timeout 命令最长执行时间，<= 0 表示不限时
//
// 超时后会关闭 SSH 连接以终止等待，并返回超时错误；
// 每次调用都会新建 SSH 连接，函数返回前所有连接/会话都会被关闭。
// 拨号 + 握手 + 打开会话各自有独立的超时（见 sshDialTimeout / sshHandshakeTimeout），
// timeout 只作用于命令执行本身；命令执行期间不设置空闲 deadline（导入镜像可能长时间没有输出）。
func SSHExecCommandWithTimeout(host, port, user, password, command string, timeout time.Duration) (*SSHExecResult, error) {
	if host == "" {
		return nil, fmt.Errorf("ssh host is empty")
	}
	if port == "" {
		port = "22"
	}
	if user == "" {
		user = "root"
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	client, err := dialSSH(addr, newSSHConfig(user, password), nil)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s error: %w", addr, err)
	}
	defer client.Close()

	// NewSession 需要等对端确认 channel 打开；握手后才卡死的节点也不能无限等待
	openTimeout := sshHandshakeTimeout
	openTimer := time.AfterFunc(openTimeout, func() { _ = client.Close() })
	session, err := client.NewSession()
	if !openTimer.Stop() {
		if session != nil {
			_ = session.Close()
		}
		return nil, fmt.Errorf("ssh new session timeout after %s", openTimeout)
	}
	if err != nil {
		return nil, fmt.Errorf("ssh new session error: %w", err)
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// 在 goroutine 中执行，便于超时后主动关闭连接；channel 带缓冲，避免 goroutine 泄漏
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	var timeoutCh <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		timeoutCh = timer.C
	}

	var runErr error
	select {
	case runErr = <-done:
	case <-timeoutCh:
		// 关闭连接让 session.Run 返回，再读取 buffer，避免与 goroutine 并发读写
		_ = client.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("ssh run command `%s` timeout after %s", command, timeout)
		}
		return &SSHExecResult{
			Stdout: stdoutBuf.String(),
			Stderr: stderrBuf.String(),
		}, fmt.Errorf("ssh run command `%s` timeout after %s", command, timeout)
	}

	if runErr != nil {
		return &SSHExecResult{
			Stdout: stdoutBuf.String(),
			Stderr: stderrBuf.String(),
		}, fmt.Errorf("ssh run command `%s` error: %w", command, runErr)
	}

	return &SSHExecResult{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}, nil
}

// SSHUploadFile 通过 SFTP 将本地文件（io.Reader）上传到远程服务器
//
//	@param host       目标主机 IP 或域名
//	@param port       SSH 端口，通常为 "22"
//	@param user       SSH 登录用户名
//	@param password   SSH 登录密码
//	@param reader     要上传的文件内容
//	@param remotePath 远程服务器上的目标文件路径
//	@return error
//
// 每次调用都会新建 SSH/SFTP 连接，函数返回前全部关闭；
// 上传中途失败时会尽力删除远端已写入的半截文件。
//
// 超时语义：握手有整体 deadline（sshHandshakeTimeout）；握手之后是"进度续期"的空闲超时——
// 连续 sftpIdleTimeout 内连接上没有任何成功的数据收发（包括 reader 卡住不再产出数据、
// 对端不再读写）就判定为传输卡死，返回包含 "sftp transfer stalled" 的错误，而不是总时长上限。
// reader 卡死时无法被强行打断：此时本函数返回，但读 reader 的 goroutine 要等 reader.Read 返回才会退出，
// 因此 reader 应是本地文件/内存这类最终一定会返回的来源。
func SSHUploadFile(host, port, user, password string, reader io.Reader, remotePath string) error {
	if host == "" {
		return fmt.Errorf("ssh host is empty")
	}
	if port == "" {
		port = "22"
	}
	if user == "" {
		user = "root"
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	var pc *progressConn
	client, err := dialSSH(addr, newSSHConfig(user, password), func(c net.Conn) net.Conn {
		pc = newProgressConn(c)
		return pc
	})
	if err != nil {
		return fmt.Errorf("ssh dial %s error: %w", addr, err)
	}
	defer client.Close()

	// 看门狗：空闲超过 idle 就关闭 SSH 连接，让所有阻塞在连接上的 SFTP 调用（NewClient/Create/写入/Remove/Close）立刻返回
	idle := sftpIdleTimeout
	stalled := make(chan struct{})
	stopWatch := watchIdle(pc, idle, func() {
		close(stalled)
		_ = client.Close()
	})
	defer stopWatch()

	// fail 在传输卡死时统一换成明确的 "sftp transfer stalled" 错误，否则保持原有错误文案
	fail := func(msg string, err error) error {
		select {
		case <-stalled:
			if err == nil {
				return fmt.Errorf("sftp transfer stalled: no data transferred for %s (%s)", idle, msg)
			}
			return fmt.Errorf("sftp transfer stalled: no data transferred for %s (%s): %w", idle, msg, err)
		default:
			return fmt.Errorf("%s: %w", msg, err)
		}
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fail("sftp new client error", err)
	}
	defer sftpClient.Close()

	dstFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fail(fmt.Sprintf("sftp create remote file %s error", remotePath), err)
	}

	// 拷贝放进 goroutine：reader 卡死时看门狗触发后本函数仍能返回。
	// 注意此后不能再调用 dstFile 的方法——sftp.File.ReadFrom 持有 f.mu，Close 会一直阻塞。
	copyDone := make(chan error, 1)
	go func() {
		_, err := io.CopyBuffer(dstFile, reader, make([]byte, 32*1024))
		copyDone <- err
	}()

	select {
	case err := <-copyDone:
		if err != nil {
			// 上传中途失败：关闭并删除远端半截文件，避免 /tmp 被残留文件逐渐占满
			// （连接已半死时这两步也会被看门狗限时）
			_ = dstFile.Close()
			_ = sftpClient.Remove(remotePath)
			return fail(fmt.Sprintf("sftp upload to %s error", remotePath), err)
		}
	case <-stalled:
		return fail(fmt.Sprintf("sftp upload to %s error", remotePath), nil)
	}

	// SFTP 的 Close 失败意味着数据可能没有完整落盘，必须当作上传失败处理
	if err := dstFile.Close(); err != nil {
		_ = sftpClient.Remove(remotePath)
		return fail(fmt.Sprintf("sftp close remote file %s error", remotePath), err)
	}

	return nil
}

// SSHRemoteFile 通过 SFTP 打开的远程文件，Close 会依次关闭文件、SFTP 会话和 SSH 连接
type SSHRemoteFile struct {
	// File 远程文件，可直接 Read
	File *sftp.File
	// Size 文件大小（字节）
	Size int64

	sftpClient *sftp.Client
	client     *ssh.Client
	stopWatch  func()
	closeOnce  sync.Once
	// stalled 在看门狗判定传输卡死并关闭连接后被关闭
	stalled chan struct{}
	idle    time.Duration
}

// Read 读取远程文件内容；传输卡死时返回包含 "sftp transfer stalled" 的错误
func (f *SSHRemoteFile) Read(p []byte) (int, error) {
	n, err := f.File.Read(p)
	if err != nil && err != io.EOF {
		select {
		case <-f.stalled:
			return n, fmt.Errorf("sftp transfer stalled: no data transferred for %s: %w", f.idle, err)
		default:
		}
	}
	return n, err
}

// Close 关闭文件、SFTP 会话和 SSH 连接，可重复调用
func (f *SSHRemoteFile) Close() error {
	f.closeOnce.Do(func() {
		f.stopWatch()
		_ = f.File.Close()
		_ = f.sftpClient.Close()
		_ = f.client.Close()
	})
	return nil
}

// SSHOpenFile 通过 SFTP 打开远程服务器上的文件用于流式下载
//
//	@param host       目标主机 IP 或域名
//	@param port       SSH 端口，为空时默认 "22"
//	@param user       SSH 登录用户名，为空时默认 "root"
//	@param password   SSH 登录密码
//	@param remotePath 远程服务器上的文件路径
//
// 调用方必须调用返回值的 Close。超时语义与 SSHUploadFile 一致：握手有整体 deadline，
// 之后连续 sftpIdleTimeout 没有任何数据收发就关闭连接，让阻塞的读立刻返回错误。
func SSHOpenFile(host, port, user, password, remotePath string) (*SSHRemoteFile, error) {
	if host == "" {
		return nil, fmt.Errorf("ssh host is empty")
	}
	if remotePath == "" {
		return nil, fmt.Errorf("remote path is empty")
	}
	if port == "" {
		port = "22"
	}
	if user == "" {
		user = "root"
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	var pc *progressConn
	client, err := dialSSH(addr, newSSHConfig(user, password), func(c net.Conn) net.Conn {
		pc = newProgressConn(c)
		return pc
	})
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s error: %w", addr, err)
	}

	idle := sftpIdleTimeout
	stalled := make(chan struct{})
	stopWatch := watchIdle(pc, idle, func() {
		close(stalled)
		_ = client.Close()
	})

	cleanup := func() {
		stopWatch()
		_ = client.Close()
	}
	wrapErr := func(msg string, err error) error {
		select {
		case <-stalled:
			return fmt.Errorf("sftp transfer stalled: no data transferred for %s (%s): %w", idle, msg, err)
		default:
			return fmt.Errorf("%s: %w", msg, err)
		}
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		cleanup()
		return nil, wrapErr("sftp new client error", err)
	}

	file, err := sftpClient.Open(remotePath)
	if err != nil {
		_ = sftpClient.Close()
		cleanup()
		return nil, wrapErr(fmt.Sprintf("sftp open remote file %s error", remotePath), err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		_ = sftpClient.Close()
		cleanup()
		return nil, wrapErr(fmt.Sprintf("sftp stat remote file %s error", remotePath), err)
	}
	if info.IsDir() {
		_ = file.Close()
		_ = sftpClient.Close()
		cleanup()
		return nil, fmt.Errorf("remote path %s is a directory", remotePath)
	}

	return &SSHRemoteFile{
		File:       file,
		Size:       info.Size(),
		sftpClient: sftpClient,
		client:     client,
		stopWatch:  stopWatch,
		stalled:    stalled,
		idle:       idle,
	}, nil
}
