package mysql

import (
	"github.com/crackeer/go-developer-web/api"
	mysqlModule "github.com/crackeer/go-developer-web/module/mysql"

	"github.com/gin-gonic/gin"
)

// Query
//
//	@param ctx
func Count(ctx *gin.Context) {
	request := &QueryRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	table := ctx.Param("table")

	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	var total int64
	builder := gormDB.Table(table)
	if len(request.Where) > 0 {
		sql, params := mysqlModule.BuildQueryRaws(request.Where)
		builder = builder.Where(sql, params...)
	}

	builder.Count(&total)
	api.Success(ctx, map[string]interface{}{
		"total": total,
	})
}
