package file

import (
	"os"
	"path/filepath"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// Rename  ...
//
//	@param ctx
func Rename(ctx *gin.Context) {
	file := ctx.DefaultQuery("file", "")
	newName := ctx.DefaultQuery("name", "")
	if len(file) < 1 || len(newName) < 1 {
		api.Failure(ctx, -1, "file is required")
		return
	}
	dest := filepath.Join(filepath.Dir(file), newName)
	err := os.Rename(file, dest)
	if err != nil {
		api.Failure(ctx, -2, "rename error:"+err.Error())
		return
	}
	api.Success(ctx, map[string]interface{}{"dest": dest})
}
