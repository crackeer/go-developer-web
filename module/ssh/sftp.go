package ssh

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// defaultSFTPSessionTimeout OpenSFTP 未指定超时时的默认会话总时长上限
const defaultSFTPSessionTimeout = 60 * time.Second

// SFTPSession 一次性的 SFTP 会话，用于目录列表 / 重命名 / 删除 / 新建目录等短操作。
// Close 会依次关闭 SFTP 会话和 SSH 连接，可重复调用。
type SFTPSession struct {
	Client *sftp.Client

	ssh       *ssh.Client
	timer     *time.Timer
	closeOnce sync.Once
}

// OpenSFTP 连接远程服务器并建立 SFTP 会话
//
//	@param timeout 整个会话（含握手和所有操作）的最长时间，<= 0 时使用默认值；
//	               超时后会强制关闭 SSH 连接，让阻塞中的 SFTP 调用立刻返回错误。
//
// 调用方必须调用返回值的 Close。
func OpenSFTP(host, port, user, password string, timeout time.Duration) (*SFTPSession, error) {
	if host == "" {
		return nil, fmt.Errorf("ssh host is empty")
	}
	if port == "" {
		port = "22"
	}
	if user == "" {
		user = "root"
	}
	if timeout <= 0 {
		timeout = defaultSFTPSessionTimeout
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	client, err := dialSSH(addr, newSSHConfig(user, password), nil)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s error: %w", addr, err)
	}

	s := &SFTPSession{ssh: client}
	s.timer = time.AfterFunc(timeout, func() { _ = client.Close() })

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("sftp new client error: %w", err)
	}
	s.Client = sftpClient
	return s, nil
}

// Close 关闭 SFTP 会话和 SSH 连接
func (s *SFTPSession) Close() {
	s.closeOnce.Do(func() {
		s.timer.Stop()
		if s.Client != nil {
			_ = s.Client.Close()
		}
		_ = s.ssh.Close()
	})
}
