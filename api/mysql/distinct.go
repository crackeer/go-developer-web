package mysql

import (
	"fmt"

	"github.com/crackeer/go-developer-web/api"
	mysqlModule "github.com/crackeer/go-developer-web/module/mysql"

	"github.com/gin-gonic/gin"
)

// Distinct
//
//	@param ctx
func Distinct(ctx *gin.Context) {
	request := &QueryRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	table := ctx.Param("table")
	column := ctx.Param("column")

	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	if gormDB == nil {
		api.Failure(ctx, -1, "db nil")
		return
	}
	builder := gormDB.Table(table)
	if len(request.Where) > 0 {
		sql, params := mysqlModule.BuildQueryRaws(request.Where)
		builder = builder.Where(sql, params...)
	}
	list := []string{}
	if err := builder.Select(fmt.Sprintf("distinct %s as %s", column, column)).Pluck(column, &list).Error; err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, list)
}
