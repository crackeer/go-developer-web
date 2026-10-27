package mysql

import (
	"github.com/crackeer/go-developer-web/api"

	"github.com/gin-gonic/gin"
)

// Query
//
//	@param ctx
func Update(ctx *gin.Context) {
	updateData := api.AllPostParams(ctx)
	table := ctx.Param("table")
	// 路由组的 :id 为配置 ID，记录主键使用 :row_id
	id := ctx.Param("row_id")

	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	affected := gormDB.Table(table).Where(map[string]interface{}{
		"id": id,
	}).Updates(updateData).RowsAffected

	api.Success(ctx, map[string]interface{}{
		"affected": affected,
	})
}
