package ftp

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// Upload
//
//	@Summary		upload file
//	@Description	multipart/form-data: file(上传文件), path(目标目录，相对 prefix，默认 "."), name(可选，目标文件名，默认使用上传文件名)
//	@Tags			ftp
//	@Accept			mpfd
func Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		api.Failure(c, -1, fmt.Sprintf("get upload file error: %s", err.Error()))
		return
	}

	dir := strings.TrimSpace(c.DefaultPostForm("path", "."))
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		name = fileHeader.Filename
	}
	name = filepath.Base(name)
	if name == "." || name == "/" || name == "" {
		api.Failure(c, -1, "invalid file name")
		return
	}
	if err := checkPath(dir); err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	remoteFile := path.Join(dir, name)

	file, err := fileHeader.Open()
	if err != nil {
		api.Failure(c, -1, fmt.Sprintf("open upload file error: %s", err.Error()))
		return
	}
	defer file.Close()

	client, err := GetFtpClient(c)
	if err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	defer client.Close()

	if err := client.UploadReader(file, remoteFile); err != nil {
		api.Failure(c, -2, err.Error())
		return
	}
	api.Success(c, map[string]string{"path": remoteFile})
}
