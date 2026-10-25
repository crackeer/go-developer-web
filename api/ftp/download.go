package ftp

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// List
//
//	@Summary		list files
//	@Description	list files
//	@Tags			ftp
//	@Accept			json
func Download(c *gin.Context) {
	client, err := GetFtpClient(c)
	if err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	path := c.DefaultQuery("path", ".")
	defer client.Close()

	_, name := filepath.Split(path)

	// GetByte 会自动拼接配置中的 prefix
	bytes, err := client.GetByte(path)
	if err != nil {
		api.Failure(c, -2, err.Error())
		return
	}
	http.ServeContent(c.Writer, c.Request, name, time.Now(), strings.NewReader(string(bytes)))

}
