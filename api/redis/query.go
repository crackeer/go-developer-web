package redis

import (
	"context"
	"strings"
	"time"

	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/container"

	"github.com/gin-gonic/gin"
	redisV8 "github.com/go-redis/redis/v8"
)

// scanLimit 单次扫描返回的最大 key 数量上限
const scanLimit = 1000

// valuePreviewLimit 单个值展示时的最大字符数，防止超大 value 撑爆响应
const valuePreviewLimit = 20000

// keyItem 单个 key 的查询结果
type keyItem struct {
	Key       string      `json:"key"`
	Type      string      `json:"type"`
	TTL       int64       `json:"ttl"` // 单位秒：-1 永不过期，-2 不存在
	Value     interface{} `json:"value"`
	Truncated bool        `json:"truncated,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// getRedisClient 读取 business redis 配置并建立连接
func getRedisClient(ctx *gin.Context) *redisV8.Client {
	id := ctx.Param("id")
	return container.GetRedisClientByID(id)
}

// reqContext 基于请求派生 context，默认 15s 超时
func reqContext(ctx *gin.Context) (context.Context, context.CancelFunc) {
	parent := context.Background()
	if ctx.Request != nil && ctx.Request.Context() != nil {
		parent = ctx.Request.Context()
	}
	return context.WithTimeout(parent, 15*time.Second)
}

// readValue 按 key 的类型读取其值，并做长度截断保护
func readValue(c context.Context, client *redisV8.Client, key, keyType string) (interface{}, bool, error) {
	switch keyType {
	case "string":
		v, err := client.Get(c, key).Result()
		if err != nil {
			return nil, false, err
		}
		if len(v) > valuePreviewLimit {
			return v[:valuePreviewLimit], true, nil
		}
		return v, false, nil
	case "list":
		// 最多取前 scanLimit 个元素
		v, err := client.LRange(c, key, 0, int64(scanLimit-1)).Result()
		if err != nil {
			return nil, false, err
		}
		total, _ := client.LLen(c, key).Result()
		return v, total > int64(len(v)), nil
	case "set":
		v, err := client.SMembers(c, key).Result()
		if err != nil {
			return nil, false, err
		}
		if len(v) > scanLimit {
			return v[:scanLimit], true, nil
		}
		return v, false, nil
	case "zset":
		v, err := client.ZRangeWithScores(c, key, 0, int64(scanLimit-1)).Result()
		if err != nil {
			return nil, false, err
		}
		list := make([]map[string]interface{}, 0, len(v))
		for _, z := range v {
			list = append(list, map[string]interface{}{"member": z.Member, "score": z.Score})
		}
		total, _ := client.ZCard(c, key).Result()
		return list, total > int64(len(list)), nil
	case "hash":
		v, err := client.HGetAll(c, key).Result()
		if err != nil {
			return nil, false, err
		}
		return v, false, nil
	case "none":
		return nil, false, nil
	default:
		// stream 等其它类型不展开，返回类型提示
		return "(不支持预览的类型: " + keyType + ")", false, nil
	}
}

// describeKey 查询单个 key 的类型、TTL 和值
func describeKey(c context.Context, client *redisV8.Client, key string) keyItem {
	item := keyItem{Key: key}
	keyType, err := client.Type(c, key).Result()
	if err != nil {
		item.Error = err.Error()
		return item
	}
	item.Type = keyType
	if keyType == "none" {
		item.TTL = -2
		return item
	}
	if ttl, terr := client.TTL(c, key).Result(); terr == nil {
		item.TTL = int64(ttl.Seconds())
	}
	value, truncated, verr := readValue(c, client, key, keyType)
	if verr != nil {
		item.Error = verr.Error()
		return item
	}
	item.Value = value
	item.Truncated = truncated
	return item
}

// Scan 按 prefix 扫描最多 scanLimit 条 key，并返回每条 key 的类型与值
// GET /api/redis/scan?prefix=xxx
func Scan(ctx *gin.Context) {
	prefix := strings.TrimSpace(ctx.Query("prefix"))
	if prefix == "" {
		api.Failure(ctx, -1, "missing query param: prefix")
		return
	}

	client := getRedisClient(ctx)
	if client == nil {
		api.Failure(ctx, -1, "init redis client failed")
		return
	}
	defer client.Close()

	c, cancel := reqContext(ctx)
	defer cancel()

	match := prefix + "*"
	keys := make([]string, 0, scanLimit)
	var cursor uint64
	// 用 SCAN 游标遍历，累计到 scanLimit 即停止，避免 KEYS 阻塞
	for {
		batch, next, scanErr := client.Scan(c, cursor, match, 200).Result()
		if scanErr != nil {
			api.Failure(ctx, -1, "scan failed: "+scanErr.Error())
			return
		}
		for _, k := range batch {
			keys = append(keys, k)
			if len(keys) >= scanLimit {
				break
			}
		}
		cursor = next
		if cursor == 0 || len(keys) >= scanLimit {
			break
		}
	}

	items := make([]keyItem, 0, len(keys))
	for _, k := range keys {
		items = append(items, describeKey(c, client, k))
	}

	api.Success(ctx, map[string]interface{}{
		"prefix":  prefix,
		"match":   match,
		"count":   len(items),
		"limit":   scanLimit,
		"reached": len(items) >= scanLimit,
		"keys":    items,
	})
}

// Get 查询单个 key 的类型与值
// GET /api/redis/get?key=xxx
func Get(ctx *gin.Context) {
	key := strings.TrimSpace(ctx.Query("key"))
	if key == "" {
		api.Failure(ctx, -1, "missing query param: key")
		return
	}

	client := getRedisClient(ctx)
	if client == nil {
		api.Failure(ctx, -1, "init redis client failed")
		return
	}
	defer client.Close()

	c, cancel := reqContext(ctx)
	defer cancel()

	item := describeKey(c, client, key)
	api.Success(ctx, item)
}

// randomKeyLimit 随机取样的 key 数量
const randomKeyLimit = 100

// Random 随机返回最多 randomKeyLimit 个 key，并查询其类型与值
// GET /api/redis/random
func Random(ctx *gin.Context) {
	client := getRedisClient(ctx)
	if client == nil {
		api.Failure(ctx, -1, "init redis client failed")
		return
	}
	defer client.Close()

	c, cancel := reqContext(ctx)
	defer cancel()

	// 用 RANDOMKEY 多次取样并去重；库较小或空时提前停止，避免无谓重试
	seen := make(map[string]struct{}, randomKeyLimit)
	keys := make([]string, 0, randomKeyLimit)
	// 允许的最大尝试次数，避免库中唯一 key 远少于 100 时死循环
	maxAttempts := randomKeyLimit * 20
	emptyStreak := 0
	for attempt := 0; attempt < maxAttempts && len(keys) < randomKeyLimit; attempt++ {
		k, err := client.RandomKey(c).Result()
		if err != nil {
			// redis.Nil 表示库为空，直接结束
			if err == redisV8.Nil {
				break
			}
			api.Failure(ctx, -1, "randomkey failed: "+err.Error())
			return
		}
		if _, ok := seen[k]; ok {
			emptyStreak++
			// 连续多次只拿到重复 key，说明库内唯一 key 已基本取尽，提前结束
			if emptyStreak > randomKeyLimit {
				break
			}
			continue
		}
		emptyStreak = 0
		seen[k] = struct{}{}
		keys = append(keys, k)
	}

	items := make([]keyItem, 0, len(keys))
	for _, k := range keys {
		items = append(items, describeKey(c, client, k))
	}

	api.Success(ctx, map[string]interface{}{
		"count": len(items),
		"limit": randomKeyLimit,
		"keys":  items,
	})
}

// Delete 删除指定 key
// DELETE /api/redis/delete?key=xxx
func Delete(ctx *gin.Context) {
	key := strings.TrimSpace(ctx.Query("key"))
	if key == "" {
		api.Failure(ctx, -1, "missing query param: key")
		return
	}

	client := getRedisClient(ctx)
	if client == nil {
		api.Failure(ctx, -1, "init redis client failed")
		return
	}
	defer client.Close()

	c, cancel := reqContext(ctx)
	defer cancel()

	deleted, err := client.Del(c, key).Result()
	if err != nil {
		api.Failure(ctx, -1, "delete failed: "+err.Error())
		return
	}

	api.Success(ctx, map[string]interface{}{
		"key":     key,
		"deleted": deleted, // 1 表示删除成功，0 表示 key 不存在
	})
}
