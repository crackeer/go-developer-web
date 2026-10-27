package mysql

import (
	"github.com/crackeer/go-developer-web/container"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetDB(ctx *gin.Context) (*gorm.DB, error) {
	id := ctx.Param("id")
	return container.GetMySQLDBByID(id)
}
