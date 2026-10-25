package s3client

import (
	"context"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

func ListBucket(ctx *gin.Context) {
	client, err := GetS3Client(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	// Get the first page of results for ListObjectsV2 for a bucket
	output, err := client.ListBuckets(context.TODO(), nil)

	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	list := []string{}
	for _, object := range output.Buckets {
		list = append(list, *object.Name)
	}
	api.Success(ctx, list)

}
