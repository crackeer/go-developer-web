package container

import (
	"context"
	"fmt"
	"strconv"

	redisV8 "github.com/go-redis/redis/v8"
	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	gormMap  map[string]*gorm.DB
	redisMap map[string]*redisV8.Client
)

func GetRedisClientByID(resourceID string) *redisV8.Client {
	name := resourceID
	if client, ok := redisMap[name]; ok {
		return client
	}
	value, _ := strconv.ParseInt(resourceID, 10, 64)
	cfg, err := QueryRedisConfig(value)
	if err != nil {
		return nil
	}
	// new redis client
	client := redisV8.NewClient(&redisV8.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil
	}
	redisMap[name] = client
	return client
}

func GetMySQLDBByID(resourceID string) *gorm.DB {
	name := resourceID
	if db, ok := gormMap[name]; ok {
		return db
	}
	value, _ := strconv.ParseInt(resourceID, 10, 64)
	cfg, err := QueryMySQLConfig(value)
	if err != nil {
		return nil
	}
	// new mysql db
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	db, err := gorm.Open(mysqlDriver.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil
	}
	gormMap[name] = db
	return db
}
