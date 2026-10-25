package ftp

import (
	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// List
//
//	@Summary		list files
//	@Description	list files
//	@Tags			ftp
//	@Accept			json
func List(c *gin.Context) {
	client, err := GetFtpClient(c)
	if err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	defer client.Close()
	path := c.DefaultQuery("path", ".")
	entries, err := client.List(path)
	if err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	api.Success(c, entries)

}
