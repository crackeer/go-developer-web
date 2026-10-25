package resource

import (
	"github.com/crackeer/go-developer-web/api"

	"github.com/gin-gonic/gin"
)

// Types
//
//	@Summary		类型元数据
//	@Description	返回支持的配置类型及表单字段定义，供前端动态渲染
//	@Tags			config
func Types(ctx *gin.Context) {
	api.Success(ctx, TypeMetas())
}
