package file

import (
	"os"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// DeleteFile   ...
//
//	@param ctx
func MkdirAll(ctx *gin.Context) {
	file := ctx.DefaultQuery("dir", "")
	if len(file) < 1 {
		api.Failure(ctx, -1, "dur is required")
		return
	}
	if err := os.MkdirAll(file, os.ModePerm); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, nil)
}
