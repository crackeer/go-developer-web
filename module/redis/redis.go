package redis

import (
	"os"
	"time"

	redisV8 "github.com/go-redis/redis/v8"
)

// GetRedisClient
//
//	@param addr
//	@param username
//	@param password
//	@param db
//	@return *redisV8.Client
func GetRedisClient(addr string, username, password string, db int) *redisV8.Client {
	return redisV8.NewClient(&redisV8.Options{
		DB:           db,
		Addr:         addr,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Username:     username,
		Password:     password,
	})
}

// GetDefaultBusinessRedisClient
//
//	@return *redisV8.Client
func GetDefaultBusinessRedisClient(addr, username, password string) *redisV8.Client {
	if len(addr) < 1 {
		addr = os.Getenv("REDIS_ADDR")
	}

	if len(username) < 1 {
		username = os.Getenv("REDIS_USERNAME")
	}

	if len(password) < 1 {
		password = os.Getenv("REDIS_PASSWORD")
	}

	return GetRedisClient(addr, username, password, 0)
}
