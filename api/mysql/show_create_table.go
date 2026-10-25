package mysql

import (
	"github.com/crackeer/go-developer-web/api"

	"github.com/gin-gonic/gin"
)

// ShowCreateTable   ...
//
//	@param ctx
func ShowCreateTable(ctx *gin.Context) {
	table := ctx.Param("table")
	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	data := map[string]interface{}{}
	if err := gormDB.Raw("show create table `" + table + "`").Scan(&data).Error; err != nil {
		api.Failure(ctx, -2, err.Error())
		return
	}
	var sql string
	if value, ok := data["Create Table"]; ok {
		if stringValue, ok := value.(string); ok {
			sql = stringValue
		}
	}

	api.Success(ctx, map[string]interface{}{
		"sql": sql,
	})
}
