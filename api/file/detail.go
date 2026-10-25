package file

import (
	"os"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

func Detail(ctx *gin.Context) {
	file := ctx.DefaultQuery("file", "")
	if len(file) < 1 {
		api.Failure(ctx, -1, "file is required")
		return
	}
	stat, err := os.Stat(file)
	if err != nil {
		api.Failure(ctx, -1, "stat file error:"+err.Error())
		return
	}
	api.Success(ctx, map[string]interface{}{
		"name":          stat.Name(),
		"size":          stat.Size(),
		"last_modified": stat.ModTime().Format("2006-01-02 15:04:05"),
	})
}
