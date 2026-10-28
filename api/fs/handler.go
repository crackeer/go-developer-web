package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/module/fsutil"
	"github.com/gin-gonic/gin"
)

// FileItem 目录列表项，path 为相对根目录的路径。
type FileItem struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	IsDir  bool   `json:"is_dir"`
	Size   int64  `json:"size"`
	Modify string `json:"modify"`
}

// List 列出目录内容，目录在前、按名称排序。GET /api/fs/:id/list?path=/a
func List(ctx *gin.Context) {
	root, dir, ok := resolveFromQuery(ctx)
	if !ok {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	list := make([]FileItem, 0, len(entries))
	for _, entry := range entries {
		item := FileItem{
			Name:  entry.Name(),
			Path:  relPath(root, filepath.Join(dir, entry.Name())),
			IsDir: entry.IsDir(),
		}
		if info, err := entry.Info(); err == nil {
			item.Size = info.Size()
			item.Modify = info.ModTime().Format("2006-01-02 15:04:05")
		}
		list = append(list, item)
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].IsDir != list[j].IsDir {
			return list[i].IsDir
		}
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
	api.Success(ctx, map[string]interface{}{
		"path": relPath(root, dir),
		"list": list,
	})
}

// Download 下载文件。GET /api/fs/:id/download?path=/a.txt
func Download(ctx *gin.Context) {
	_, file, ok := resolveFromQuery(ctx)
	if !ok {
		return
	}
	stat, err := os.Stat(file)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	if stat.IsDir() {
		api.Failure(ctx, -1, "不能直接下载文件夹，请先 zip 压缩")
		return
	}
	ctx.FileAttachment(file, filepath.Base(file))
}

// Upload 上传文件到目录，同名文件自动加时间前缀避免覆盖。POST /api/fs/:id/upload?path=/a（multipart: file）
func Upload(ctx *gin.Context) {
	root, dir, ok := resolveFromQuery(ctx)
	if !ok {
		return
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	name, err := validName(filepath.Base(fileHeader.Filename))
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
		api.Failure(ctx, -1, "上传目录不存在")
		return
	}
	dest := filepath.Join(dir, name)
	if _, err := os.Stat(dest); err == nil {
		prefix := time.Now().Format("2006-01-02-15-04-05")
		dest = filepath.Join(dir, prefix+"_"+name)
		for i := 2; ; i++ {
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				break
			}
			dest = filepath.Join(dir, fmt.Sprintf("%s_%d_%s", prefix, i, name))
		}
	}
	if err := ctx.SaveUploadedFile(fileHeader, dest); err != nil {
		api.Failure(ctx, -1, "save upload file error: "+err.Error())
		return
	}
	api.Success(ctx, map[string]interface{}{"dest": relPath(root, dest)})
}

// Delete 删除文件或文件夹（递归），不允许删除根目录。POST /api/fs/:id/delete {"path":"/a"}
func Delete(ctx *gin.Context) {
	request := &pathRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	root, target, ok := resolve(ctx, request.Path)
	if !ok {
		return
	}
	if target == root {
		api.Failure(ctx, -1, "不能删除根目录")
		return
	}
	if _, err := os.Lstat(target); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	if err := os.RemoveAll(target); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, nil)
}

// Rename 在同一目录内重命名，目标已存在时报错。POST /api/fs/:id/rename {"path":"/a.txt","name":"b.txt"}
func Rename(ctx *gin.Context) {
	request := &nameRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	name, err := validName(request.Name)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	root, source, ok := resolve(ctx, request.Path)
	if !ok {
		return
	}
	if source == root {
		api.Failure(ctx, -1, "不能重命名根目录")
		return
	}
	dest := filepath.Join(filepath.Dir(source), name)
	if _, err := os.Lstat(dest); err == nil {
		api.Failure(ctx, -1, "目标已存在："+name)
		return
	}
	if err := os.Rename(source, dest); err != nil {
		api.Failure(ctx, -1, "rename error: "+err.Error())
		return
	}
	api.Success(ctx, map[string]interface{}{"dest": relPath(root, dest)})
}

// Mkdir 在目录下新建文件夹。POST /api/fs/:id/mkdir {"path":"/a","name":"sub"}
func Mkdir(ctx *gin.Context) {
	request := &nameRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	name, err := validName(request.Name)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	root, dir, ok := resolve(ctx, request.Path)
	if !ok {
		return
	}
	dest := filepath.Join(dir, name)
	if _, err := os.Lstat(dest); err == nil {
		api.Failure(ctx, -1, "已存在："+name)
		return
	}
	if err := os.Mkdir(dest, 0o755); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	api.Success(ctx, map[string]interface{}{"dest": relPath(root, dest)})
}

// Zip 后台压缩文件夹为同级的 <name>-<时间戳>.zip。POST /api/fs/:id/zip {"path":"/a"}
func Zip(ctx *gin.Context) {
	request := &pathRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	root, source, ok := resolve(ctx, request.Path)
	if !ok {
		return
	}
	if source == root {
		api.Failure(ctx, -1, "不能压缩根目录")
		return
	}
	if _, err := os.Stat(source); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	dest := fmt.Sprintf("%s-%d.zip", source, time.Now().Unix())
	// 大目录压缩耗时较长，后台执行，前端稍后刷新查看
	go func() {
		if err := fsutil.Zip(source, dest); err != nil {
			fmt.Printf("fs zip %s error: %s\n", source, err.Error())
		}
	}()
	api.Success(ctx, map[string]interface{}{"dest": relPath(root, dest)})
}

// Unzip 后台解压 zip 到同级的 <name>-<时间戳> 目录。POST /api/fs/:id/unzip {"path":"/a.zip"}
func Unzip(ctx *gin.Context) {
	request := &pathRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	root, source, ok := resolve(ctx, request.Path)
	if !ok {
		return
	}
	if !strings.HasSuffix(strings.ToLower(source), ".zip") {
		api.Failure(ctx, -1, "只支持 .zip 文件")
		return
	}
	if _, err := os.Stat(source); err != nil {
		api.Failure(ctx, -1, err.Error())
		return
	}
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	dest := filepath.Join(filepath.Dir(source), fmt.Sprintf("%s-%d", base, time.Now().Unix()))
	go func() {
		if err := fsutil.Unzip(source, dest); err != nil {
			fmt.Printf("fs unzip %s error: %s\n", source, err.Error())
		}
	}()
	api.Success(ctx, map[string]interface{}{"dest": relPath(root, dest)})
}
