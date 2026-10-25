package file

import (
	"os"
	"path/filepath"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// Create Write  ...
//
//	@param ctx
func Create(ctx *gin.Context) {
	file := ctx.DefaultQuery("file", "")

	if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	bytes, err := ctx.GetRawData()
	if err == nil && len(bytes) > 0 {
		if err := os.WriteFile(file, bytes, os.ModePerm); err != nil {
			api.Failure(ctx, -2, "write file error:"+err.Error())
			return
		}
	}
	api.Success(ctx, map[string]interface{}{
		"dest": file,
	})
}
