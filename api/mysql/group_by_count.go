package mysql

import (
	"github.com/crackeer/go-developer-web/api"
	mysqlModule "github.com/crackeer/go-developer-web/module/mysql"

	"github.com/gin-gonic/gin"
)

// GroupByCountRequest
type GroupByCountRequest struct {
	Where   map[string]interface{} `json:"where"`
	GroupBy string                 `json:"group_by"`
}

// GroupByCount 按指定字段分组统计数量，默认按 source 分组
//
//	@param ctx
func GroupByCount(ctx *gin.Context) {
	request := &GroupByCountRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	table := ctx.Param("table")
	groupBy := request.GroupBy
	if len(groupBy) == 0 {
		groupBy = "source"
	}

	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	// 校验分组字段是否存在，避免 SQL 注入
	fields := mysqlModule.GetMySQLTableFields(gormDB, table)
	fieldExists := false
	for _, field := range fields {
		if field.Field == groupBy {
			fieldExists = true
			break
		}
	}
	if !fieldExists {
		api.Failure(ctx, -1, "group_by field not found: "+groupBy)
		return
	}

	builder := gormDB.Table(table)
	if len(request.Where) > 0 {
		sql, params := mysqlModule.BuildQueryRaws(request.Where)
		builder = builder.Where(sql, params...)
	}

	list := []map[string]interface{}{}
	if err := builder.Select(groupBy + ", count(*) as count").
		Group(groupBy).
		Order("count desc").
		Find(&list).Error; err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, list)
}
