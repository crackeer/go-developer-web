package s3client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

func Stat(ctx *gin.Context) {
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
	output, err := client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, *output)
}
