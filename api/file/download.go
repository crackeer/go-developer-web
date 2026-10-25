package file

import (
	"path/filepath"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// DownloadFile
//
//	@param ctx
func DownloadFile(ctx *gin.Context) {
	filePath := ctx.DefaultQuery("file", "")
	if len(filePath) < 1 {
		api.Failure(ctx, -1, "file path nil")
		return
	}
	_, name := filepath.Split(filePath)
	ctx.Abort()
	ctx.FileAttachment(filePath, name)
}
