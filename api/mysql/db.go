package mysql

import (
	"errors"

	"github.com/crackeer/go-developer-web/container"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetDB(ctx *gin.Context) (*gorm.DB, error) {
	id := ctx.Param("id")
	db := container.GetMySQLDBByID(id)
	if db == nil {
		return nil, errors.New("db not found")
	}
	return db, nil
}
