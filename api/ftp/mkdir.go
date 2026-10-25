package ftp

import (
	"strings"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// MkdirRequest
type MkdirRequest struct {
	Path string `json:"path"`
}

// Mkdir
//
//	@Summary		make directory
//	@Description	create a new directory, path is relative to prefix
//	@Tags			ftp
//	@Accept			json
func Mkdir(c *gin.Context) {
	request := &MkdirRequest{}
	if err := c.ShouldBindJSON(request); err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	path := strings.TrimSpace(request.Path)
	if path == "" {
		api.Failure(c, -1, "path is required")
		return
	}
	if err := checkPath(path); err != nil {
		api.Failure(c, -1, err.Error())
		return
	}

	client, err := GetFtpClient(c)
	if err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	defer client.Close()

	if err := client.MakeDir(path); err != nil {
		api.Failure(c, -2, err.Error())
		return
	}
	api.Success(c, map[string]string{"path": path})
}
