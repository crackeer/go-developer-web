package container

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	cryptoModule "github.com/crackeer/go-developer-web/module/crypto"

	redisV8 "github.com/go-redis/redis/v8"
	mysqlDSN "github.com/go-sql-driver/mysql"
	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	clientLock sync.Mutex
	gormMap    = map[string]*gorm.DB{}
	redisMap   = map[string]*redisV8.Client{}
)

// ConfigSecret 读取资源密码的加密密钥（环境变量 CONFIG_SECRET）。
func ConfigSecret() string {
	if v := strings.TrimSpace(os.Getenv("CONFIG_SECRET")); v != "" {
		return v
	}
	return "devops-console-default-secret"
}

// ClearClientCache 配置更新 / 删除后清理该资源已缓存的连接，下次使用时按新配置重建。
func ClearClientCache(resourceID string) {
	clientLock.Lock()
	defer clientLock.Unlock()
	if db, ok := gormMap[resourceID]; ok {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		delete(gormMap, resourceID)
	}
	if client, ok := redisMap[resourceID]; ok {
		client.Close()
		delete(redisMap, resourceID)
	}
}

func GetRedisClientByID(resourceID string) *redisV8.Client {
	clientLock.Lock()
	defer clientLock.Unlock()
	name := resourceID
	if client, ok := redisMap[name]; ok {
		return client
	}
	value, _ := strconv.ParseInt(resourceID, 10, 64)
	cfg, err := QueryRedisConfig(value)
	if err != nil {
		return nil
	}
	// 入库的密码为密文，连接前需解密
	password, err := cryptoModule.AESGCMDecrypt(ConfigSecret(), cfg.Password)
	if err != nil {
		return nil
	}
	// new redis client
	client := redisV8.NewClient(&redisV8.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username: cfg.Username,
		Password: password,
		DB:       cfg.DB,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil
	}
	redisMap[name] = client
	return client
}

// GetMySQLDBByID 按配置 ID 获取（并缓存）MySQL 连接，连接失败时返回具体原因。
func GetMySQLDBByID(resourceID string) (*gorm.DB, error) {
	clientLock.Lock()
	defer clientLock.Unlock()
	name := resourceID
	if db, ok := gormMap[name]; ok {
		return db, nil
	}
	value, _ := strconv.ParseInt(resourceID, 10, 64)
	cfg, err := QueryMySQLConfig(value)
	if err != nil {
		return nil, fmt.Errorf("mysql config %s not found", resourceID)
	}
	// 入库的密码为密文，连接前需解密
	password, err := cryptoModule.AESGCMDecrypt(ConfigSecret(), cfg.Password)
	if err != nil {
		return nil, fmt.Errorf("decrypt password error: %s", err.Error())
	}
	// 使用配置中的默认数据库
	dsnConfig := mysqlDSN.NewConfig()
	dsnConfig.User = cfg.Username
	dsnConfig.Passwd = password
	dsnConfig.Net = "tcp"
	dsnConfig.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	dsnConfig.DBName = cfg.Database
	dsnConfig.ParseTime = true
	dsnConfig.Params = map[string]string{"charset": "utf8mb4"}
	dsn := dsnConfig.FormatDSN()
	db, err := gorm.Open(mysqlDriver.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect mysql error: %s", err.Error())
	}
	gormMap[name] = db
	return db, nil
}
