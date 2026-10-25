package s3client

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

func PeekZip(ctx *gin.Context) {
	bucket := ctx.Query("bucket")
	inZipFile := ctx.Query("name")
	if len(bucket) == 0 {
		api.Failure(ctx, -1, "bucket is empty")
		return
	}
	if len(inZipFile) == 0 {
		api.Failure(ctx, -1, "name is empty")
		return
	}
	client, err := GetS3Client(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	key := ctx.Query("key")

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
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		api.Failure(ctx, -4, err.Error())
		return
	}
	for _, file := range zipReader.File {

		if file.FileInfo().IsDir() {
			continue
		}
		if file.Name == inZipFile {
			fmt.Println(file.Name, file.FileInfo().IsDir(), inZipFile)
			rc, err := file.Open()
			if err != nil {
				api.Failure(ctx, -5, err.Error())
				return
			}
			defer rc.Close()
			body, err := io.ReadAll(rc)
			if err != nil {
				api.Failure(ctx, -6, err.Error())
				return
			}
			api.Success(ctx, map[string]interface{}{
				"content": string(body),
			})
			return
		}
	}
	api.Failure(ctx, -7, "file not found")
}
