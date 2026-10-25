package mysql

import (
	"github.com/crackeer/go-developer-web/api"

	"github.com/gin-gonic/gin"
)

// ShowIndex  ...
//
//	@param ctx
func ShowIndex(ctx *gin.Context) {
	table := ctx.Param("table")
	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	indexes := []map[string]interface{}{}
	if err := gormDB.Raw("show index from `" + table + "`").Find(&indexes).Error; err != nil {
		api.Failure(ctx, -1, "show index error")
		return
	}
	api.Success(ctx, indexes)
}
