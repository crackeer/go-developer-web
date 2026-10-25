package s3client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/module/fsutil"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

// Upload
//
//	@param ctx
func Upload(ctx *gin.Context) {
	bucket := ctx.Query("bucket")
	if len(bucket) == 0 {
		api.Failure(ctx, -1, "bucket is empty")
		return
	}
	client, err := GetS3Client(ctx)
	if err != nil {
		api.Failure(ctx, -2, err.Error())
		return
	}
	key := ctx.Query("key")
	if len(key) > 0 {
		uploadObject(ctx, client, bucket, key)
		return
	}

	prefix := ctx.Query("prefix")
	if len(prefix) > 0 {
		uploadDirectory(ctx, client, bucket, prefix)
		return
	}

	api.Failure(ctx, -1, "key or prefix is empty")

}

func uploadObject(ctx *gin.Context, client *s3.Client, bucket string, key string) {
	fileInfo, err := ctx.FormFile("file")
	if err != nil {
		api.Failure(ctx, -3, err.Error())
		return
	}

	file, err := fileInfo.Open()
	if err != nil {
		api.Failure(ctx, -4, err.Error())
		return
	}
	defer file.Close()
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		api.Failure(ctx, -5, err.Error())
		return
	}
	api.Success(ctx, map[string]interface{}{
		"bucket": bucket,
		"key":    key,
	})
}

func uploadDirectory(ctx *gin.Context, client *s3.Client, bucket string, prefix string) {
	fileInfo, err := ctx.FormFile("file")
	if err != nil {
		api.Failure(ctx, -3, err.Error())
		return
	}
	file, err := fileInfo.Open()
	if err != nil {
		api.Failure(ctx, -4, err.Error())
		return
	}
	defer file.Close()
	unzipDir := fmt.Sprintf("/tmp/s3/unzip_%d", time.Now().Unix())
	if err := os.MkdirAll(unzipDir, 0777); err != nil {
		api.Failure(ctx, -5, "mkdir tmp path error:"+err.Error())
		return
	}
	defer os.RemoveAll(unzipDir)
	if err := fsutil.UnzipFrom(file, unzipDir); err != nil {
		api.Failure(ctx, -6, "unzip error:"+err.Error())
		return
	}
	if strings.HasSuffix(prefix, "/") {
		prefix = strings.TrimSuffix(prefix, "/")
	}
	fileMapping, _ := fsutil.GetFileMapping(unzipDir)
	uploadKeys := []string{}
	for key, value := range fileMapping {
		tmpFile, err := os.Open(value)
		if err != nil {
			api.Failure(ctx, -7, fmt.Sprintf("read file %s error:%s", value, err.Error()))
			return
		}
		_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(fmt.Sprintf("%s/%s", prefix, key)),
			Body:   tmpFile,
		})
		uploadKeys = append(uploadKeys, fmt.Sprintf("%s/%s", prefix, key))
		if err != nil {
			api.Failure(ctx, -7, err.Error())
			return
		}
	}
	api.Success(ctx, map[string]interface{}{
		"files":      fileMapping,
		"uploadKeys": uploadKeys,
	})
}
