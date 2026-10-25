package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/crackeer/go-developer-web/api/file"
	ftpAPI "github.com/crackeer/go-developer-web/api/ftp"
	k3sHandler "github.com/crackeer/go-developer-web/api/k3s"
	mysqlHandler "github.com/crackeer/go-developer-web/api/mysql"
	redisHandler "github.com/crackeer/go-developer-web/api/redis"
	"github.com/crackeer/go-developer-web/api/resource"
	"github.com/crackeer/go-developer-web/api/s3client"
	"github.com/crackeer/go-developer-web/container"

	_ "embed"

	"github.com/gin-gonic/gin"

	rollRender "github.com/unrolled/render"
)

//go:embed pages
var tmplPage embed.FS
var (
	//go:embed assets
	assets embed.FS

	renderClient *rollRender.Render
)

func newRenderClient(layout string) *rollRender.Render {
	object := rollRender.New(rollRender.Options{
		Layout:     layout,
		Directory:  "pages",
		Extensions: []string{".tmpl"},
		Delims: rollRender.Delims{
			Left:  "{[{",
			Right: "}]}",
		},
		IsDevelopment:               true,
		RenderPartialsWithoutPrefix: true,
		FileSystem:                  &rollRender.EmbedFileSystem{FS: tmplPage},
	})
	return object
}

func main() {
	if err := container.InitAppConfig(); err != nil {
		panic(fmt.Sprintf("init app config error: %s", err.Error()))
	}
	appConfig := container.GetAppConfig()
	renderClient = newRenderClient("layout")
	router := gin.Default()

	assetsSub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(fmt.Sprintf("assets sub fs error: %s", err.Error()))
	}
	router.StaticFS("/assets", http.FS(assetsSub))
	router.Use(gin.Recovery(), gin.Logger(), gin.BasicAuth(gin.Accounts{
		"admin": appConfig.LoginPassword,
	}))

	// 文件管理
	fileGroup := router.Group("/api/file")
	fileGroup.GET("list", file.ListFile)
	fileGroup.GET("download", file.DownloadFile)
	fileGroup.POST("upload", file.UploadFile)
	fileGroup.GET("zip", file.ZipDir)
	fileGroup.GET("unzip", file.Unzip)
	fileGroup.GET("delete", file.DeleteFile)
	fileGroup.GET("mkdirall", file.MkdirAll)
	fileGroup.GET("download_static", file.DownloadStatic)
	fileGroup.GET("rename", file.Rename)
	fileGroup.POST("create", file.Create)
	fileGroup.GET("detail", file.Detail)
	fileGroup.GET("dir_size", file.GetDirSize)

	// MySQL
	mysqlGroup := router.Group("/api/mysql/:id")
	mysqlGroup.POST("query/:table", mysqlHandler.Query)
	mysqlGroup.POST("count/:table", mysqlHandler.Count)
	mysqlGroup.POST("group_by_count/:table", mysqlHandler.GroupByCount)
	mysqlGroup.POST("update/:table/:id", mysqlHandler.Update)
	mysqlGroup.POST("create/:table", mysqlHandler.Create)
	mysqlGroup.POST("batch_create/:table", mysqlHandler.BatchCreate)
	mysqlGroup.GET("desc/:table", mysqlHandler.Desc)
	mysqlGroup.GET("show_create_table/:table", mysqlHandler.ShowCreateTable)
	mysqlGroup.GET("show_index", mysqlHandler.ShowIndex)
	mysqlGroup.POST("distinct/:column", mysqlHandler.Distinct)
	mysqlGroup.POST("delete/:id", mysqlHandler.DeleteByID)
	mysqlGroup.GET("table", mysqlHandler.Table)
	mysqlGroup.POST("exec_sql", mysqlHandler.ExecSQL)
	mysqlGroup.GET("global_variables", mysqlHandler.GlobalVariables)

	// k3s / k8s
	k3sRouter := router.Group("/api/k3s/:id")
	k3sRouter.GET("/describe/:resource", k3sHandler.Describe)
	k3sRouter.GET("/list/:resource", k3sHandler.List)
	k3sRouter.DELETE("/delete/:resource", k3sHandler.Delete)
	k3sRouter.POST("/apply", k3sHandler.Apply)
	k3sRouter.POST("/exec/:pod_name", k3sHandler.Exec)
	k3sRouter.GET("/log/download", k3sHandler.LogDownload)
	k3sRouter.GET("/log/:pod_name", k3sHandler.Log)
	k3sRouter.POST("/rollout/restart/:resource", k3sHandler.RolloutRestart)
	k3sRouter.POST("/scale/:resource", k3sHandler.Scale)
	k3sRouter.POST("/patch/:resource", k3sHandler.Patch)
	k3sRouter.GET("/node/list", k3sHandler.Node)
	k3sRouter.POST("/node/label/add", k3sHandler.NodeLabelAdd)
	k3sRouter.POST("/node/label/remove", k3sHandler.NodeLabelRemove)

	// redis
	redisRouter := router.Group("/api/redis/:id")
	redisRouter.GET("/scan", redisHandler.Scan)
	redisRouter.GET("/get", redisHandler.Get)
	redisRouter.GET("/random", redisHandler.Random)
	redisRouter.DELETE("/delete", redisHandler.Delete)

	// ssh
	// sshRouter := router.Group("/api/ssh/:id")
	// sshRouter.POST("/exec", sshAPI.Exec)
	// sshRouter.POST("/test", sshAPI.Test)
	// sshRouter.POST("upload", sshAPI.Upload)
	// sshRouter.GET("/download", sshAPI.Download)
	// sshRouter.GET("/file/nodes", sshAPI.FileNodes)
	// sshRouter.GET("/file/list", sshAPI.FileList)
	// sshRouter.GET("/file/download", sshAPI.FileDownload)
	// sshRouter.POST("/file/rename", sshAPI.FileRename)
	// sshRouter.POST("/file/delete", sshAPI.FileDelete)
	// sshRouter.POST("/file/mkdir", sshAPI.FileMkdir)
	// sshRouter.POST("/file/upload", sshAPI.FileUpload)

	// ftp
	ftpRouter := router.Group("/api/ftp/:id")
	ftpRouter.GET("/list", ftpAPI.List)
	ftpRouter.GET("/download", ftpAPI.Download)
	ftpRouter.POST("/upload", ftpAPI.Upload)
	ftpRouter.POST("/delete", ftpAPI.Delete)
	ftpRouter.POST("/rename", ftpAPI.Rename)
	ftpRouter.POST("/mkdir", ftpAPI.Mkdir)

	// s3
	s3Router := router.Group("/api/s3/:id")
	s3Router.GET("bucket/list", s3client.ListBucket)
	s3Router.GET("object/list", s3client.ListObject)
	s3Router.GET("object/download", s3client.Download)
	s3Router.POST("object/upload", s3client.Upload)
	s3Router.GET("object/peek_zip", s3client.PeekZip)
	s3Router.GET("object/stat", s3client.Stat)
	s3Router.GET("token", s3client.Token)

	// 资源配置管理（MySQL / FTP / SSH / S3 / Redis，SQLite 存储）
	configGroup := router.Group("/api/config")
	configGroup.GET("types", resource.Types)
	configGroup.GET("list", resource.List)
	configGroup.POST("create", resource.Create)
	configGroup.POST("update/:id", resource.Update)
	configGroup.POST("delete/:id", resource.Delete)
	configGroup.POST("test/:id", resource.Test)

	router.GET("/mysql/:id", createPageHandler("service/mysql"))
	router.GET("/ftp/:id", createPageHandler("service/ftp"))
	router.GET("/ssh/:id", createPageHandler("service/ssh"))
	router.GET("/s3/:id", createPageHandler("service/s3"))
	router.GET("/redis/:id", createPageHandler("service/redis"))
	router.NoRoute(createPageHandler("index"))
	router.Run(fmt.Sprintf(":%d", appConfig.Port))
}

func createPageHandler(page string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		renderClient.HTML(ctx.Writer, http.StatusOK, page, nil)
	}
}
