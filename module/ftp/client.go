package ftp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jlaffaye/ftp"
)

const (
	defaultTimeout = 60 * time.Second
)

// FTPClient
type FTPClient struct {
	serverConn *ftp.ServerConn
	prefix     string
}

// DefaultFTPClient
//
//	@param server
//	@param user
//	@param password
//	@return *FTPClient
//	@return error
func DefaultFTPClient(server, user, password string) (*FTPClient, error) {
	conn, err := ftp.Dial(server, ftp.DialWithTimeout(defaultTimeout))
	if err != nil {
		return nil, fmt.Errorf("connect failed: %v", err)
	}
	err = conn.Login(user, password)
	if err != nil {
		return nil, fmt.Errorf("login failed: %v", err)
	}
	return &FTPClient{
		serverConn: conn,
	}, nil
}

// NewFTPClient
//
//	@param server
//	@param user
//	@param password
//	@param opts
//	@return *FTPClient
//	@return error
func NewFTPClient(server, user, password string, opts ...ftp.DialOption) (*FTPClient, error) {
	conn, err := ftp.Dial(server, opts...)
	if err != nil {
		return nil, fmt.Errorf("connect failed: %v", err)
	}
	err = conn.Login(user, password)
	if err != nil {
		return nil, fmt.Errorf("login failed: %v", err)
	}
	return &FTPClient{
		serverConn: conn,
	}, nil
}

func (c *FTPClient) SetPrefix(prefix string) {
	c.prefix = prefix
}

// Upload
//
//	@param localFile
//	@param remoteFile
//	@return error
func (c *FTPClient) Upload(localFile, remoteFile string) error {
	f, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("open file failed: %v", err)
	}
	defer f.Close()
	if c.prefix != "" {
		remoteFile = filepath.Join(c.prefix, remoteFile)
	}
	err = c.serverConn.Stor(remoteFile, f)
	if err != nil {
		return fmt.Errorf("upload file failed: %v", err)
	}
	return nil
}

// UploadReader 从 reader 上传文件到远程路径
//
//	@param reader
//	@param remoteFile
//	@return error
func (c *FTPClient) UploadReader(reader io.Reader, remoteFile string) error {
	if c.prefix != "" {
		remoteFile = filepath.Join(c.prefix, remoteFile)
	}
	if err := c.serverConn.Stor(remoteFile, reader); err != nil {
		return fmt.Errorf("upload file failed: %v", err)
	}
	return nil
}

// Delete
//
//	@param remoteFile
//	@return error
func (c *FTPClient) Delete(remoteFile string) error {
	if c.prefix != "" {
		remoteFile = filepath.Join(c.prefix, remoteFile)
	}
	return c.serverConn.Delete(remoteFile)
}

// Rename
//
//	@param from
//	@param to
//	@return error
func (c *FTPClient) Rename(from, to string) error {
	if c.prefix != "" {
		from = filepath.Join(c.prefix, from)
		to = filepath.Join(c.prefix, to)
	}
	return c.serverConn.Rename(from, to)
}

// MakeDir
//
//	@param path
//	@return error
func (c *FTPClient) MakeDir(path string) error {
	if c.prefix != "" {
		path = filepath.Join(c.prefix, path)
	}
	return c.serverConn.MakeDir(path)
}

// List
//
//	@param path
//	@return []*ftp.Entry
//	@return error
func (c *FTPClient) List(path string) ([]*ftp.Entry, error) {
	if c.prefix != "" {
		path = filepath.Join(c.prefix, path)
	}
	entries, err := c.serverConn.List(path)
	if err != nil {
		return nil, fmt.Errorf("list directory failed: %v", err)
	}
	return entries, nil
}

func (c *FTPClient) GetByte(path string) ([]byte, error) {
	if c.prefix != "" {
		path = filepath.Join(c.prefix, path)
	}
	resp, err := c.serverConn.Retr(path)

	if err != nil {
		return nil, fmt.Errorf("get file failed: %v", err)
	}

	defer resp.Close()

	return io.ReadAll(resp)

}

func (c *FTPClient) download(path string) (*ftp.Response, error) {
	if c.prefix != "" {
		path = filepath.Join(c.prefix, path)
	}
	resp, err := c.serverConn.Retr(path)

	if err != nil {
		return nil, fmt.Errorf("get file failed: %v", err)
	}
	return resp, nil
}

func (c *FTPClient) SetTransferTypeBinary() error {
	return c.serverConn.Type(ftp.TransferTypeBinary)
}

// DownloadTo
//
//	@param path
//	@param localFile
//	@return error
func (c *FTPClient) DownloadTo(path, localFile string) error {
	resp, err := c.download(path)
	if err != nil {
		return fmt.Errorf("download file failed: %v", err)
	}
	defer resp.Close()

	bytes, err := io.ReadAll(resp)
	if err != nil {
		return fmt.Errorf("read response failed: %v", err)
	}
	return os.WriteFile(localFile, bytes, 0644)
}

// Get
//
//	@param path
//	@return *ftp.Entry
//	@return error
func (c *FTPClient) Get(path string) (*ftp.Entry, error) {
	if c.prefix != "" {
		path = filepath.Join(c.prefix, path)
	}
	return c.serverConn.GetEntry(path)
}

// Close
//
//	@return error
func (c *FTPClient) Close() error {
	return c.serverConn.Quit()
}

func (c *FTPClient) GetServerConn() *ftp.ServerConn {
	return c.serverConn
}
