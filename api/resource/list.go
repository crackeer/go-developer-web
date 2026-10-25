package resource

import (
	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/container"

	"github.com/gin-gonic/gin"
)

// List
//
//	@Summary		配置列表
//	@Description	按类型列出资源连接配置（密码脱敏）；type 为空时返回全部
//	@Tags			config
//	@Accept			json
func List(ctx *gin.Context) {
	conn, ok := getDB(ctx)
	if !ok {
		return
	}
	configType := ctx.Query("type")
	if configType != "" {
		if _, valid := GetTypeMeta(configType); !valid {
			api.Failure(ctx, -1, "unsupported type: "+configType)
			return
		}
	}
	rows, err := container.ListResources(conn, configType)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	list := make([]MaskedConfig, 0, len(rows))
	for _, row := range rows {
		list = append(list, Mask(row))
	}
	api.Success(ctx, list)
}
