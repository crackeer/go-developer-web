package s3client

import (
	"github.com/crackeer/go-developer-web/api"

	"github.com/gin-gonic/gin"
)

func Token(ctx *gin.Context) {
	bucketName := ctx.DefaultQuery("bucket", "")
	if len(bucketName) < 1 {
		api.Failure(ctx, -1, "param error")
		return
	}
	prefix := ctx.DefaultQuery("prefix", "")
	if len(prefix) < 1 {
		api.Failure(ctx, -1, "prefix is empty")
		return
	}

	stsClient, err := GetSTSClient(ctx)
	if err != nil {
		api.Failure(ctx, -2, err.Error())
		return
	}

	token, err := stsClient.GetTempToken(bucketName, prefix)
	if err != nil {
		api.Failure(ctx, -3, err.Error())
		return
	}

	api.Success(ctx, token)
}
