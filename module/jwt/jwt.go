package jwt

import (
	"fmt"
	"os"
	"strconv"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Claims 自定义 JWT claims
type Claims struct {
	Username string `json:"username"`
	jwtv5.RegisteredClaims
}

// DefaultExpireHours 默认过期时间（小时）
const DefaultExpireHours = 24

// getSecret 从环境变量 JWT_SECRET 获取密钥，未设置则使用默认值
func getSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) == 0 {
		secret = "devops-console-default-jwt-secret-change-me"
	}
	return []byte(secret)
}

// getExpireDuration 从环境变量 JWT_EXPIRE_HOURS 获取过期时长，默认 24 小时
func getExpireDuration() time.Duration {
	hours := DefaultExpireHours
	if val := os.Getenv("JWT_EXPIRE_HOURS"); len(val) > 0 {
		if h, err := strconv.Atoi(val); err == nil && h > 0 {
			hours = h
		}
	}
	return time.Duration(hours) * time.Hour
}

// GenerateToken 根据用户名生成 JWT token
func GenerateToken(username string) (string, error) {
	now := time.Now()
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:    "devops-console",
			Subject:   username,
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(getExpireDuration())),
		},
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString(getSecret())
}

// ParseToken 解析并验证 JWT token，返回 claims
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwtv5.ParseWithClaims(tokenString, &Claims{}, func(t *jwtv5.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return getSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// GetExpireSeconds 返回 token 过期秒数（用于设置 cookie maxAge）
func GetExpireSeconds() int {
	return int(getExpireDuration().Seconds())
}
