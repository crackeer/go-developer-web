package file

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// UploadFile ...
//
//	@param ctx
func UploadFile(ctx *gin.Context) {

	fileInfo, err := ctx.FormFile("file")
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	uploadPath := ctx.DefaultQuery("dir", "")

	if len(uploadPath) < 1 {
		uploadPath = fmt.Sprintf("/tmp/tmp_upload_%d", time.Now().Unix())
		if err := os.MkdirAll(uploadPath, 0777); err != nil {
			api.Failure(ctx, -1, "mkdir tmp path error:"+err.Error())
			return
		}
	}
	localFile := filepath.Join(uploadPath, fileInfo.Filename)
	if _, err := os.Stat(localFile); err == nil {
		prefix := time.Now().Format("2006-01-02-15-04")
		localFile = filepath.Join(uploadPath, prefix+"_"+fileInfo.Filename)
		for i := 2; ; i++ {
			if _, err := os.Stat(localFile); os.IsNotExist(err) {
				break
			}
			localFile = filepath.Join(uploadPath, fmt.Sprintf("%s_%d_%s", prefix, i, fileInfo.Filename))
		}
	}

	err = ctx.SaveUploadedFile(fileInfo, localFile)
	if err != nil {
		api.Failure(ctx, -1, "save upload file error:"+err.Error())
		return
	}
	api.Success(ctx, map[string]interface{}{
		"dest": localFile,
	})
}
