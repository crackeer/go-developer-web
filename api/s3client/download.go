package s3client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/module/fsutil"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

// Download
//
//	@param ctx
func Download(ctx *gin.Context) {
	bucket := ctx.Query("bucket")
	if len(bucket) == 0 {
		api.Failure(ctx, -1, "bucket is empty")
		return
	}
	client, err := GetS3Client(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	key := ctx.Query("key")
	if len(key) > 0 {
		downloadObject(ctx, client, bucket, key)
		return
	}

	prefix := ctx.DefaultQuery("prefix", "")
	downloadPrefix(ctx, client, bucket, prefix)
}

func downloadObject(ctx *gin.Context, client *s3.Client, bucket string, key string) {
	output, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	defer output.Body.Close()
	body, err := io.ReadAll(output.Body)
	if err != nil {
		api.Failure(ctx, -2, err.Error())
		return
	}
	name := strings.ReplaceAll(key, "/", "-")
	http.ServeContent(ctx.Writer, ctx.Request, name, *output.LastModified, bytes.NewReader(body))
}

func downloadPrefix(ctx *gin.Context, client *s3.Client, bucket string, prefix string) {
	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(5000),
	})
	if len(prefix) < 1 {
		api.Failure(ctx, -1, "prefix is empty")
		return
	}

	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	tmpDownloadDir := fmt.Sprintf("/tmp/s3/download_%s", fsutil.GetNowTimeStamp())
	if err := os.MkdirAll(tmpDownloadDir, os.ModePerm); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	// 1. download all object
	var downloadObjectTo = func(prefix, key, workDir string) error {
		output, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return err
		}
		defer output.Body.Close()
		body, err := io.ReadAll(output.Body)
		if err != nil {
			return err
		}
		filePath := filepath.Join(workDir, strings.TrimPrefix(key, prefix))
		dir, _ := filepath.Split(filePath)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
		return os.WriteFile(filePath, body, os.ModePerm)
	}
	for _, object := range output.Contents {
		if err := downloadObjectTo(prefix, *object.Key, tmpDownloadDir); err != nil {
			api.Failure(ctx, -1, err.Error())
			return
		}
	}

	// 2. zip all object to single file
	zipName := fmt.Sprintf("/tmp/s3/download_%s.zip", fsutil.GetNowTimeStamp())
	if err := fsutil.Zip(tmpDownloadDir, zipName); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	ctx.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(zipName)))

	http.ServeFile(ctx.Writer, ctx.Request, zipName)
}
