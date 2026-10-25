package s3

import (
	"crypto/tls"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GetS3Client(accessKey, secretKey string, region, endpoint string) (*s3.Client, error) {
	cfg := aws.NewConfig()
	cfg.Region = region
	cfg.BaseEndpoint = aws.String(endpoint)
	cfg.Credentials = credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
	cfg.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return s3.NewFromConfig(*cfg), nil
}
