package ftp

import (
	"strings"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// DeleteRequest
type DeleteRequest struct {
	Path string `json:"path"`
}

// Delete
//
//	@Summary		delete file
//	@Description	delete file, path is relative to prefix
//	@Tags			ftp
//	@Accept			json
func Delete(c *gin.Context) {
	request := &DeleteRequest{}
	if err := c.ShouldBindJSON(request); err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	target := strings.TrimSpace(request.Path)
	if target == "" || target == "." || target == "/" {
		api.Failure(c, -1, "path is required")
		return
	}
	if err := checkPath(target); err != nil {
		api.Failure(c, -1, err.Error())
		return
	}

	client, err := GetFtpClient(c)
	if err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	defer client.Close()

	if err := client.Delete(target); err != nil {
		api.Failure(c, -2, err.Error())
		return
	}
	api.Success(c, map[string]string{"path": target})
}
