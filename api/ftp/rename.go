package ftp

import (
	"strings"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// RenameRequest
type RenameRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Rename
//
//	@Summary		rename file
//	@Description	rename or move file, from/to are relative to prefix
//	@Tags			ftp
//	@Accept			json
func Rename(c *gin.Context) {
	request := &RenameRequest{}
	if err := c.ShouldBindJSON(request); err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	from := strings.TrimSpace(request.From)
	to := strings.TrimSpace(request.To)
	if from == "" || to == "" {
		api.Failure(c, -1, "from and to are required")
		return
	}
	if err := checkPath(from); err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	if err := checkPath(to); err != nil {
		api.Failure(c, -1, err.Error())
		return
	}

	client, err := GetFtpClient(c)
	if err != nil {
		api.Failure(c, -1, err.Error())
		return
	}
	defer client.Close()

	if err := client.Rename(from, to); err != nil {
		api.Failure(c, -2, err.Error())
		return
	}
	api.Success(c, map[string]string{"from": from, "to": to})
}
