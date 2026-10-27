package resource

import (
	"errors"
	"strings"

	"github.com/crackeer/go-developer-web/container"
	cryptoModule "github.com/crackeer/go-developer-web/module/crypto"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Create
//
//	@Summary		新建配置
//	@Description	创建一条资源连接配置；password 加密后入库
//	@Tags			config
func Create(ctx *gin.Context) {
	conn, ok := getDB(ctx)
	if !ok {
		return
	}
	request := &SaveRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	record, config, invalid := buildResource(request)
	if invalid != "" {
		api.Failure(ctx, -1, invalid)
		return
	}
	if request.Password != "" {
		encrypted, err := cryptoModule.AESGCMEncrypt(ConfigSecret(), request.Password)
		if err != nil {
			api.Failure(ctx, -1, "encrypt password error: "+err.Error())
			return
		}
		config.Password = encrypted
	}
	if err := record.EncodeConfig(config); err != nil {
		api.Failure(ctx, -1, "encode config error: "+err.Error())
		return
	}
	if err := container.CreateResource(conn, record); err != nil {
		api.Failure(ctx, -1, friendlyDbError(err))
		return
	}
	api.Success(ctx, Mask(*record))
}

// Update
//
//	@Summary		更新配置
//	@Description	按 ID 更新配置；密码留空表示保留原密码
//	@Tags			config
func Update(ctx *gin.Context) {
	conn, ok := getDB(ctx)
	if !ok {
		return
	}
	id := ctx.Param("id")
	request := &SaveRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	old, err := container.GetResource(conn, id)
	if err != nil {
		api.Failure(ctx, -1, "config not found")
		return
	}
	// 类型不允许修改（config JSON 的结构随类型变化）。
	if request.Type != "" && request.Type != old.ConfigType {
		api.Failure(ctx, -1, "type can not be changed")
		return
	}
	request.Type = old.ConfigType
	record, config, invalid := buildResource(request)
	if invalid != "" {
		api.Failure(ctx, -1, invalid)
		return
	}
	if request.Password != "" {
		encrypted, err := cryptoModule.AESGCMEncrypt(ConfigSecret(), request.Password)
		if err != nil {
			api.Failure(ctx, -1, "encrypt password error: "+err.Error())
			return
		}
		config.Password = encrypted
	} else {
		oldConfig, err := old.DecodeConfig()
		if err != nil {
			api.Failure(ctx, -1, "decode config error: "+err.Error())
			return
		}
		config.Password = oldConfig.Password
	}
	record.ID = old.ID
	if err := record.EncodeConfig(config); err != nil {
		api.Failure(ctx, -1, "encode config error: "+err.Error())
		return
	}
	if err := container.UpdateResource(conn, record); err != nil {
		api.Failure(ctx, -1, friendlyDbError(err))
		return
	}
	container.ClearClientCache(id)
	updated, err := container.GetResource(conn, id)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, Mask(*updated))
}

// Delete
//
//	@Summary		删除配置
//	@Description	按 ID 删除配置
//	@Tags			config
func Delete(ctx *gin.Context) {
	conn, ok := getDB(ctx)
	if !ok {
		return
	}
	deleted, err := container.DeleteResource(conn, ctx.Param("id"))
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	if !deleted {
		api.Failure(ctx, -1, "config not found")
		return
	}
	container.ClearClientCache(ctx.Param("id"))
	api.Success(ctx, nil)
}

// friendlyDbError 将唯一键冲突等数据库错误转为可读提示。
func friendlyDbError(err error) string {
	message := err.Error()
	if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(message, "UNIQUE constraint failed") {
		return "同名配置已存在"
	}
	return message
}
