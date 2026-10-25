package mysql

import (
	"github.com/crackeer/go-developer-web/api"

	"github.com/gin-gonic/gin"
)

// Table ...
//
//	@param ctx
func Table(ctx *gin.Context) {
	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	list := []map[string]interface{}{}
	if err := gormDB.Raw("show tables").Scan(&list).Error; err != nil {
		api.Failure(ctx, -2, err.Error())
		return
	}
	tables := []string{}
	for _, value := range list {
		for _, value := range value {
			tables = append(tables, value.(string))
		}
	}

	api.Success(ctx, tables)
}
