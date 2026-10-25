package ftp

import (
	"errors"
	"strings"

	"github.com/crackeer/go-developer-web/container"
	ftpModule "github.com/crackeer/go-developer-web/module/ftp"

	"github.com/gin-gonic/gin"
)

// checkPath 校验相对 prefix 的路径，禁止通过 ".." 逃出 prefix 目录
func checkPath(path string) error {
	for _, seg := range strings.Split(strings.ReplaceAll(path, "\\", "/"), "/") {
		if seg == ".." {
			return errors.New("path must not contain '..'")
		}
	}
	return nil
}

// GetFtpClient 从 vrapi.strategy 表(customize_deploy/ftp_config)获取 FTP 配置并创建客户端
//
//	@param ctx
//	@return *ftpModule.FTPClient
//	@return error
func GetFtpClient(ctx *gin.Context) (*ftpModule.FTPClient, error) {
	configID := ctx.Param("id")
	cfg, err := container.QueryFTPConfig(configID)
	if err != nil {
		return nil, err
	}
	client, err := ftpModule.DefaultFTPClient(cfg.Host, cfg.Username, cfg.Password)
	if err != nil {
		return nil, err
	}
	client.SetPrefix(cfg.Prefix)
	return client, nil
}
