package file

import (
	"fmt"
	"strings"
	"time"

	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/module/fsutil"

	"github.com/gin-gonic/gin"
)

// ZipDir  ...
//
//	@param ctx
func ZipDir(ctx *gin.Context) {
	dir := ctx.DefaultQuery("dir", "")
	async := ctx.DefaultQuery("async", "1") == "1"
	targetZip := ctx.DefaultQuery("target_zip", "")
	if len(dir) < 1 || dir == "/" {
		api.Failure(ctx, -1, "dir is required")
		return
	}
	dir = strings.TrimSuffix(dir, "/")
	dest := fmt.Sprintf("%s-%d.zip", dir, time.Now().Unix())
	if len(targetZip) > 0 {
		dest = targetZip
	}
	if async {
		go func() {
			fsutil.Unzip(dir, dest)
		}()
	}
	if err := fsutil.Zip(dir, dest); err != nil {
		api.Failure(ctx, -1, fmt.Sprintf("zip error:%s", err.Error()))
		return
	}
	api.Success(ctx, map[string]interface{}{
		"dest": dest,
	})
}
