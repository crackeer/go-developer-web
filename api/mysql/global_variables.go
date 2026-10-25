package mysql

import (
	"github.com/crackeer/go-developer-web/api"

	"github.com/gin-gonic/gin"
)

// GlobalVariables 查询 MySQL 全局变量
//
// 路由: GET /api/mysql/global_variables
// 全局变量属于 MySQL 服务端级别, 与具体库无关, 故自动选取一条可用连接执行。
// 可选 query 参数 like 用于模糊过滤变量名, 例如 ?like=max%25
//
//	@param ctx
func GlobalVariables(ctx *gin.Context) {
	gormDB, err := GetDB(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}

	variables := []map[string]interface{}{}
	like := ctx.Query("like")
	if len(like) > 0 {
		err = gormDB.Raw("SHOW GLOBAL VARIABLES LIKE ?", like).Find(&variables).Error
	} else {
		err = gormDB.Raw("SHOW GLOBAL VARIABLES").Find(&variables).Error
	}
	if err != nil {
		api.Failure(ctx, -1, "show global variables error")
		return
	}
	api.Success(ctx, variables)
}
