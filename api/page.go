package api

import "github.com/gin-gonic/gin"

// Breadcrumb 面包屑的一项；URL 为空表示当前页（不可点击）。
type Breadcrumb struct {
	Name string
	URL  string
}

// PageData 页面渲染数据。
type PageData struct {
	Title           string                 // <title>
	Description     string                 // <meta name="description">
	HideBreadcrumbs bool                   // 是否隐藏面包屑
	Breadcrumbs     []Breadcrumb           // 页头面包屑，首页自动补在最前
	Extension       map[string]interface{} // 注入前端 window.PageData
}

// DataFunc 按请求生成页面数据；返回 error 时渲染错误页。
type DataFunc func(ctx *gin.Context) (*PageData, error)
