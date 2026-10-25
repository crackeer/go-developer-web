package mysql

import (
	"fmt"

	"github.com/crackeer/go-developer-web/api"

	"github.com/gin-gonic/gin"
)

// Query
//
//	@param ctx
func DeleteByID(ctx *gin.Context) {
	table := ctx.Param("table")
	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	id := ctx.Param("id")

	if err := gormDB.Exec(fmt.Sprintf("delete from %s where id = %s", table, id)).Error; err != nil {
		api.Failure(ctx, -1, "delete error")
		return
	}

	api.Success(ctx, nil)
}
