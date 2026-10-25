package mysql

import (
	"github.com/crackeer/go-developer-web/api"

	"github.com/gin-gonic/gin"
)

// Desc  ...
//
//	@param ctx
func Desc(ctx *gin.Context) {
	table := ctx.Param("table")
	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	fields := []map[string]interface{}{}
	if err := gormDB.Raw("desc `" + table + "`").Find(&fields).Error; err != nil {
		api.Failure(ctx, -1, "desc error")
		return
	}
	api.Success(ctx, fields)
}
