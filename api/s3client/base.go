package s3client

import (
	"github.com/crackeer/go-developer-web/container"
	s3Module "github.com/crackeer/go-developer-web/module/s3"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

// GetS3Downloader
//
//	@param ctx
//	@return *s3.Client
//	@return error
func GetS3Client(ctx *gin.Context) (*s3.Client, error) {
	id := ctx.Param("id")
	config, err := container.QueryS3Config(id)
	if err != nil {
		return nil, err
	}
	return s3Module.GetS3Client(config.AccessKey, config.SecretKey, config.Region, config.Endpoint)
}

func GetSTSClient(ctx *gin.Context) (*s3Module.StsClient, error) {
	id := ctx.Param("id")
	config, err := container.QueryS3Config(id)
	if err != nil {
		return nil, err
	}
	return s3Module.NewSTSClient(config.AccessKey, config.SecretKey, config.Region, config.Endpoint)
}
