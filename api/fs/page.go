package fs

import (
	"fmt"

	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/container"
	"github.com/gin-gonic/gin"
)

// PageData 本地文件管理页（/fs/:id）的页面数据。
func PageData(ctx *gin.Context) (*api.PageData, error) {
	id := ctx.Param("id")
	resource, config, err := container.QueryFSResource(id)
	if err != nil {
		return nil, fmt.Errorf("本地文件配置不存在：%s", id)
	}
	return &api.PageData{
		Title:       fmt.Sprintf("本地文件 - %s", resource.Name),
		Description: config.RootDir,
		Breadcrumbs: []api.Breadcrumb{
			{Name: fmt.Sprintf("本地文件 - %s", resource.Name)},
		},
		Extension: map[string]interface{}{
			"config_id":   resource.ID,
			"config_name": resource.Name,
			"root_dir":    config.RootDir,
		},
	}, nil
}
