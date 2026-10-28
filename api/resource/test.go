package resource

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/crackeer/go-developer-web/container"
	cryptoModule "github.com/crackeer/go-developer-web/module/crypto"
	"github.com/go-redis/redis/v8"

	"github.com/crackeer/go-developer-web/api"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/jlaffaye/ftp"
	"golang.org/x/crypto/ssh"
)

// Test
//
//	@Summary		连通性测试
//	@Description	按 ID 测试配置的连通性（MySQL ping / FTP 登录 / SSH 登录 / S3 ListBuckets / Redis PING）
//	@Tags			config
func Test(ctx *gin.Context) {
	conn, ok := getDB(ctx)
	if !ok {
		return
	}
	record, err := container.GetResource(conn, ctx.Param("id"))
	if err != nil {
		api.Failure(ctx, -1, "config not found")
		return
	}
	config, err := record.DecodeConfig()
	if err != nil {
		api.Failure(ctx, -1, "decode config error: "+err.Error())
		return
	}
	password, err := cryptoModule.AESGCMDecrypt(ConfigSecret(), config.Password)
	if err != nil {
		api.Failure(ctx, -1, "decrypt password error: "+err.Error())
		return
	}

	begin := time.Now()
	testErr := testByType(record.ConfigType, config, password)
	cost := time.Since(begin).Milliseconds()

	if testErr != nil {
		api.Success(ctx, map[string]interface{}{
			"id":        record.ID,
			"ok":        false,
			"message":   testErr.Error(),
			"cost_ms":   cost,
			"tested_at": time.Now().Format("2006-01-02 15:04:05"),
		})
		return
	}
	api.Success(ctx, map[string]interface{}{
		"id":        record.ID,
		"ok":        true,
		"message":   "连接成功",
		"cost_ms":   cost,
		"tested_at": time.Now().Format("2006-01-02 15:04:05"),
	})
}

// testByType 按类型执行连通性测试，统一 5 秒超时。
func testByType(configType string, config *container.ConnectionConfig, password string) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), testTimeoutSeconds*time.Second)
	defer cancel()

	switch configType {
	case TypeMySQL:
		return testMySQL(timeoutCtx, config, password)
	case TypeFTP:
		return testFTP(config, password)
	case TypeSSH:
		return testSSH(config, password)
	case TypeS3:
		return testS3(timeoutCtx, config, password)
	case TypeRedis:
		return testRedis(timeoutCtx, config, password)
	case TypeFS:
		return testFS(config)
	default:
		return fmt.Errorf("unsupported type: %s", configType)
	}
}

// testMySQL 打开连接并 Ping。
func testMySQL(ctx context.Context, config *container.ConnectionConfig, password string) error {
	// 必须用 NewConfig 取驱动默认值；直接写结构体字面量时 AllowNativePasswords 为 false，
	// 会导致 mysql_native_password 账户报 "this user requires mysql native password authentication"。
	dsnConfig := mysql.NewConfig()
	dsnConfig.User = config.Username
	dsnConfig.Passwd = password
	dsnConfig.Net = "tcp"
	dsnConfig.Addr = fmt.Sprintf("%s:%d", config.Host, config.Port)
	dsnConfig.DBName = config.Database
	dsnConfig.Params = map[string]string{"charset": "utf8mb4"}
	db, err := sql.Open("mysql", dsnConfig.FormatDSN())
	if err != nil {
		return err
	}
	defer db.Close()
	return db.PingContext(ctx)
}

// testFTP 拨号并登录。
func testFTP(config *container.ConnectionConfig, password string) error {
	client, err := ftp.Dial(fmt.Sprintf("%s:%d", config.Host, config.Port),
		ftp.DialWithTimeout(testTimeoutSeconds*time.Second))
	if err != nil {
		return err
	}
	defer client.Quit()
	return client.Login(config.Username, password)
}

// testSSH 建立会话验证账密。
func testSSH(config *container.ConnectionConfig, password string) error {
	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         testTimeoutSeconds * time.Second,
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), sshConfig)
	if err != nil {
		return err
	}
	return client.Close()
}

// testS3 ListBuckets 验证 AK/SK 与 Endpoint 可用。
func testS3(ctx context.Context, config *container.ConnectionConfig, password string) error {
	region := config.Region
	if region == "" {
		region = "us-east-1"
	}
	cfg := aws.NewConfig()
	cfg.Region = region
	if config.Host != "" {
		cfg.BaseEndpoint = aws.String(config.Host)
	}
	cfg.Credentials = credentials.NewStaticCredentialsProvider(config.Username, password, "")
	cfg.HTTPClient = &http.Client{
		Timeout: testTimeoutSeconds * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 内网自签证书场景
		},
	}
	client := s3.NewFromConfig(*cfg)
	_, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	return err
}

// testRedis 建立连接并 PING。
// testFS 检查根目录存在且可读取。
func testFS(config *container.ConnectionConfig) error {
	stat, err := os.Stat(config.Host)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s 不是目录", config.Host)
	}
	_, err = os.ReadDir(config.Host)
	return err
}

// testRedis 建立连接并 PING。
func testRedis(ctx context.Context, config *container.ConnectionConfig, password string) error {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Username: config.Username, // Redis 6+ ACL 用户名，无 ACL 时留空
		Password: password,
		DB:       config.DB,
	})
	defer client.Close()
	_, err := client.Ping(ctx).Result()
	return err
}
