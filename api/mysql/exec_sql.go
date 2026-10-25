package mysql

import (
	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/module/crypto"

	"github.com/gin-gonic/gin"
)

// ExecSQLRequest
type ExecSQLRequest struct {
	SQL string `json:"sql"`
}

// ExecSQL
//
//	@param ctx
func ExecSQL(ctx *gin.Context) {
	request := &ExecSQLRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	sql, err := crypto.Base64Decode(request.SQL)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	if err := gormDB.Exec(sql).Error; err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, nil)
}
