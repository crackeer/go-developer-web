package file

import (
	"os"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// DeleteFile   ...
//
//	@param ctx
func DeleteFile(ctx *gin.Context) {
	file := ctx.DefaultQuery("file", "")
	if len(file) < 1 {
		api.Failure(ctx, -1, "file is required")
		return
	}
	if err := os.RemoveAll(file); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, nil)
}
