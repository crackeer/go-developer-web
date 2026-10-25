package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type StsClient struct {
	Client *sts.Client
	Region string
	Host   string
}

type TempToken struct {
	TmpSecretID  string `json:"tmpSecretId"`
	TmpSecretKey string `json:"tmpSecretKey"`
	SessionToken string `json:"sessionToken"`
	TTLBinding   int    `json:"ttl binding:"`
	Prefix       string `json:"prefix"`
	Expire       int64  `json:"expire"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	Host         string `json:"host"`
}

func (s *StsClient) GetTempToken(bucketName, prefix string) (*TempToken, error) {
	// 调用 AssumeRole
	arn := fmt.Sprintf("arn:aws:s3:::%s/%s", bucketName, prefix)
	durationSeconds := int32(3600) // 最大不超过 MinIO 的 max-session-duration（默认 3600 秒）
	result, err := s.Client.AssumeRole(context.TODO(), &sts.AssumeRoleInput{
		RoleArn:         aws.String(arn), // MinIO 中可任意填写，会被忽略
		RoleSessionName: aws.String("web-upload-session"),
		DurationSeconds: &durationSeconds,
	})
	if err != nil {
		return nil, err
	}

	token := &TempToken{
		TmpSecretID:  *result.Credentials.AccessKeyId,
		TmpSecretKey: *result.Credentials.SecretAccessKey,
		SessionToken: *result.Credentials.SessionToken,
		Prefix:       prefix,
		Expire:       result.Credentials.Expiration.Unix(),
		Bucket:       bucketName,
		Host:         s.Host,
		Region:       s.Region,
	}

	return token, nil
}

func NewSTSClient(accessKey, secretKey string, region string, endpoint string) (*StsClient, error) {
	// 创建静态凭证提供者
	staticCreds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	// 加载配置（自定义 endpoint）
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(staticCreds),
		config.WithBaseEndpoint(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config error: %w", err)
	}

	// 创建 STS 客户端
	stsClient := sts.NewFromConfig(cfg)
	return &StsClient{
		Client: stsClient,
		Region: region,
		Host:   endpoint,
	}, nil
}
