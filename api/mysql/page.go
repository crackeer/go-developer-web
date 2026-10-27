package mysql

import (
	"fmt"

	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/container"
	"github.com/gin-gonic/gin"
)

// PageData MySQL 查询页（/mysql/:id）的页面数据：配置名称与默认库，不含账号密码。
func PageData(ctx *gin.Context) (*api.PageData, error) {
	id := ctx.Param("id")
	resource, config, err := container.QueryMySQLResource(id)
	if err != nil {
		return nil, fmt.Errorf("MySQL 配置不存在：%s", id)
	}
	return &api.PageData{
		Title:       fmt.Sprintf("MySQL - %s", resource.Name),
		Description: fmt.Sprintf("%s:%d / %s", config.Host, config.Port, config.Database),
		Breadcrumbs: []api.Breadcrumb{
			{Name: fmt.Sprintf("MySQL- %s - %s", resource.Name, config.Database)},
		},
		Extension: map[string]interface{}{
			"config_id":   resource.ID,
			"config_name": resource.Name,
			"database":    config.Database,
		},
	}, nil
}
