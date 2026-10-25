package s3client

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// ListObject
//
//	@param ctx
func ListObject(ctx *gin.Context) {
	client, err := GetS3Client(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	bucket := ctx.DefaultQuery("bucket", "")
	prefix := ctx.DefaultQuery("prefix", "")

	if len(bucket) == 0 {
		api.Failure(ctx, -1, "bucket is empty")
		return
	}

	if len(prefix) > 0 {
		if !strings.HasSuffix(prefix, "/") {
			prefix = prefix + "/"
		}
	}

	fmt.Println("bucket: ", bucket)
	fmt.Println("prefix: ", prefix)

	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Delimiter: aws.String("/"),
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
	})

	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	list := []map[string]interface{}{}
	for _, object := range output.CommonPrefixes {
		tmp := map[string]interface{}{
			"key":           *object.Prefix,
			"name":          strings.TrimPrefix(*object.Prefix, prefix),
			"type":          "directory",
			"last_modified": "",
		}
		list = append(list, tmp)
	}

	for _, object := range output.Contents {
		lastModify := object.LastModified.Format("2006-01-02 15:04:05")
		tmp := map[string]interface{}{
			"key":           *object.Key,
			"name":          strings.TrimPrefix(*object.Key, prefix),
			"size":          *object.Size,
			"last_modified": lastModify,
			"type":          "file",
		}
		list = append(list, tmp)
	}

	api.Success(ctx, list)

}
