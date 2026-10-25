package file

import (
	"os"
	"path/filepath"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

func GetDirSize(ctx *gin.Context) {
	dirPath := ctx.DefaultQuery("dir", "")
	if len(dirPath) < 1 {
		api.Failure(ctx, -1, "dir path nil")
		return
	}

	size, err := calcDirSize(dirPath)
	if err != nil {
		api.Failure(ctx, -1, "calc dir size error:"+err.Error())
		return
	}
	api.Success(ctx, map[string]interface{}{
		"dir":  dirPath,
		"size": size,
	})
}

func calcDirSize(dirPath string) (int64, error) {
	var totalSize int64

	// 递归遍历文件夹
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // 遇到错误直接返回
		}
		// 只统计文件，跳过文件夹本身
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}
