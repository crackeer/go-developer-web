package mysql

import (
	"github.com/crackeer/go-developer-web/api"
	mysqlModule "github.com/crackeer/go-developer-web/module/mysql"

	"github.com/gin-gonic/gin"
)

const DefaultPageSize = 10

// QueryRequest
type QueryRequest struct {
	Where    map[string]interface{} `json:"where"`
	Page     int                    `json:"page"`
	OrderBy  string                 `json:"order_by"`
	PageSize int                    `json:"page_size"`
}

// Query
//
//	@param ctx
func Query(ctx *gin.Context) {
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
	if request.Page < 1 {
		request.Page = 1
	}
	if request.PageSize < 1 {
		request.PageSize = DefaultPageSize
	}
	list := []map[string]interface{}{}
	builder := gormDB.Table(table)
	if len(request.Where) > 0 {
		sql, params := mysqlModule.BuildQueryRaws(request.Where)
		builder = builder.Where(sql, params...)
	}

	fields := mysqlModule.GetMySQLTableFields(gormDB, table)
	orderBy := ""
	for _, field := range fields {
		if field.Extra == "auto_increment" {
			orderBy = field.Field
		}
	}
	builder = builder.Offset((request.Page - 1) * request.PageSize).Limit(request.PageSize)
	if len(request.OrderBy) > 0 {
		builder = builder.Order(request.OrderBy)
	} else if len(orderBy) > 0 {
		builder = builder.Order(orderBy + " desc")
	}

	if err := builder.Find(&list).Error; err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, list)
}
