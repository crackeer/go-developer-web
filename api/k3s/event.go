package k3s

import (
	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

func Event(ctx *gin.Context) {
	api.Success(ctx, map[string]interface{}{
		"message": "Hello, World!",
	})
}
