package mysql

import (
	"encoding/json"

	"github.com/crackeer/go-developer-web/api"

	"github.com/gin-gonic/gin"
)

// Create Query
//
//	@param ctx
func Create(ctx *gin.Context) {

	table := ctx.Param("table")

	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	data := api.AllPostParams(ctx)
	if err := gormDB.Table(table).Create(&data).Error; err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, data)
}

// BatchCreate
//
//	@param ctx
func BatchCreate(ctx *gin.Context) {
	data, err := ctx.GetRawData()
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	list := []map[string]interface{}{}
	if err := json.Unmarshal(data, &list); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	table := ctx.Param("table")

	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	if err := gormDB.Table(table).CreateInBatches(list, 100).Error; err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, data)
}
