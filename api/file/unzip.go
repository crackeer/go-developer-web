package file

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/module/fsutil"

	"github.com/gin-gonic/gin"
)

// Unzip  ...
//
//	@param ctx
func Unzip(ctx *gin.Context) {
	file := ctx.DefaultQuery("file", "")
	targetDir := ctx.DefaultQuery("target_dir", "")
	async := ctx.DefaultQuery("async", "1") == "1"
	if len(file) < 1 {
		api.Failure(ctx, -1, "file is required")
		return
	}
	fileDir, name := filepath.Split(file)
	name = strings.TrimSuffix(name, ".zip")
	dest := filepath.Join(fileDir, fmt.Sprintf("%s-%d", name, time.Now().Unix()))
	if len(targetDir) > 0 {
		dest = targetDir
	}
	if async {
		go func() {
			fsutil.Unzip(file, dest)
		}()
	}
	if err := fsutil.Unzip(file, dest); err != nil {
		api.Failure(ctx, -1, fmt.Sprintf("unzip error:%s", err.Error()))
		return
	}
	api.Success(ctx, map[string]interface{}{
		"dest": dest,
	})
}
