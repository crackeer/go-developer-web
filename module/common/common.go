package common

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
)

// GetStringValFromMap
//
//	@param container
//	@param key
//	@return string
func GetStringValFromMap(container map[string]interface{}, key string) string {
	val, ok := container[key]

	if !ok {
		return ""
	}

	if v, ok := val.(json.Number); ok {
		val, _ := v.Int64()
		return fmt.Sprintf("%d", val)
	}

	if v, ok := val.(int64); ok {
		return fmt.Sprintf("%d", v)
	}

	if v, ok := val.(int); ok {
		return fmt.Sprintf("%d", v)
	}

	if v, ok := val.(float64); ok {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}

	if v, ok := val.(float32); ok {
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	}

	if v, ok := val.(string); ok {
		return v
	}

	bytes, _ := json.Marshal(val)
	return string(bytes)
}

func ToString(src interface{}) string {
	return asString(src)
}

func asString(src interface{}) string {
	switch v := src.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	rv := reflect.ValueOf(src)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	case reflect.Float32:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 32)
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	}
	return fmt.Sprintf("%v", src)
}

// GetEnv 读取环境变量
func GetEnv(key string) string {
	return os.Getenv(key)
}

// GetHost 返回服务所在主机地址，默认 127.0.0.1
func GetHost() string {
	if value := GetEnv("GF_HOST"); len(value) > 0 {
		return value
	}
	return "127.0.0.1"
}
