// Package fs 本地文件管理（config_type=fs），所有路径都限定在资源配置的根目录内。
package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/crackeer/go-developer-web/api"
	"github.com/crackeer/go-developer-web/container"
	"github.com/gin-gonic/gin"
)

// errOutOfRoot 路径越出根目录。
var errOutOfRoot = errors.New("路径超出根目录范围")

// getRootDir 按 path 中的 :id 取本地文件资源的根目录（已解析符号链接）。
func getRootDir(ctx *gin.Context) (string, error) {
	id := ctx.Param("id")
	_, config, err := container.QueryFSResource(id)
	if err != nil {
		return "", fmt.Errorf("本地文件配置不存在：%s", id)
	}
	if config.RootDir == "" {
		return "", fmt.Errorf("本地文件配置 %s 未设置根目录", id)
	}
	root, err := filepath.EvalSymlinks(config.RootDir)
	if err != nil {
		return "", fmt.Errorf("根目录不可访问：%s", err.Error())
	}
	return root, nil
}

// withinRoot 判断 target 是否等于 root 或位于 root 之下。
func withinRoot(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// resolvePath 把相对根目录的路径（如 /a/b.txt）转换为磁盘绝对路径。
// 先做词法清理防止 ../ 越界，再对已存在的最近祖先解析符号链接，防止通过软链跳出根目录。
func resolvePath(root, rel string) (string, error) {
	target := filepath.Join(root, filepath.Clean("/"+rel))
	if !withinRoot(root, target) {
		return "", errOutOfRoot
	}
	existing := target
	for {
		if _, err := os.Lstat(existing); err == nil {
			break
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			break
		}
		existing = parent
	}
	real, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", err
	}
	if !withinRoot(root, real) {
		return "", errOutOfRoot
	}
	return target, nil
}

// relPath 磁盘绝对路径转为相对根目录的路径（以 / 开头），用于返回给前端。
func relPath(root, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == "." {
		return "/"
	}
	return "/" + filepath.ToSlash(rel)
}

// validName 校验新文件 / 文件夹名：不能为空、不能包含路径分隔符或为 . / ..。
func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("非法名称：%q", name)
	}
	return name, nil
}

// pathRequest 只携带一个路径的请求体。
type pathRequest struct {
	Path string `json:"path"`
}

// nameRequest 携带路径和名称的请求体（重命名 / 新建文件夹）。
type nameRequest struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// resolveFromQuery 读取根目录并解析 query 中的 path。
func resolveFromQuery(ctx *gin.Context) (string, string, bool) {
	return resolve(ctx, ctx.DefaultQuery("path", "/"))
}

// resolve 读取根目录并解析给定相对路径，失败时直接写出错误响应。
func resolve(ctx *gin.Context, rel string) (string, string, bool) {
	root, err := getRootDir(ctx)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return "", "", false
	}
	target, err := resolvePath(root, rel)
	if err != nil {
		api.Failure(ctx, -1, err.Error())
		return "", "", false
	}
	return root, target, true
}
