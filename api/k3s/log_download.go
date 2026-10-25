package k3s

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

const maxDownloadLogPods = 100

// Pod 名称符合 DNS-1123 子域名规则，同时可避免 zip 条目名出现路径穿越
var podNameRegexp = regexp.MustCompile(`^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$`)

// LogDownload 批量下载多个 Pod 的日志，打包成一个 zip 返回
// GET /api/k3s/log/download?namespace=<ns>&pods=<p1,p2,...>[&tail=<n>][&container=<name>]
// pods 支持逗号分隔，也支持重复传参（pods=a&pods=b）；最多 50 个。
// tail 默认 0（全量日志），传入时需为非负整数。
// container 可选，对所有 Pod 生效。
// 成功时直接以流的方式输出 zip，失败（参数错误等）时返回 JSON。
// 单个 Pod 获取失败不会中断整体，zip 中会生成 <pod>.error.txt 记录原因。
// 注意：该路由不要挂 api.DoResponseJSON()，否则会在 zip 之后追加 JSON 破坏文件。
func LogDownload(ctx *gin.Context) {
	namespace := strings.TrimSpace(ctx.DefaultQuery("namespace", "default"))
	container := strings.TrimSpace(ctx.Query("container"))
	tailStr := strings.TrimSpace(ctx.Query("tail"))
	if namespace == "" {
		namespace = "default"
	}

	pods := parsePods(ctx.QueryArray("pods"))
	if len(pods) == 0 {
		failJSON(ctx, "missing query param: pods")
		return
	}
	if len(pods) > maxDownloadLogPods {
		failJSON(ctx, fmt.Sprintf("too many pods: max %d", maxDownloadLogPods))
		return
	}
	for _, p := range pods {
		if !podNameRegexp.MatchString(p) {
			failJSON(ctx, "invalid pod name: "+p)
			return
		}
	}

	var tailLines int64
	if tailStr != "" {
		n, err := strconv.ParseInt(tailStr, 10, 64)
		if err != nil || n < 0 {
			failJSON(ctx, "invalid query param tail: must be non-negative integer")
			return
		}
		tailLines = n
	}

	cs, err := getClientset()
	if err != nil {
		failJSON(ctx, "init k8s client failed: "+err.Error())
		return
	}

	fileName := fmt.Sprintf("pod-logs_%s_%s.zip",
		sanitizeFileName(namespace), time.Now().Format("20060102-150405"))
	ctx.Header("Content-Type", "application/zip")
	ctx.Header("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", fileName, url.PathEscape(fileName)))
	ctx.Status(http.StatusOK)

	zw := zip.NewWriter(ctx.Writer)
	defer zw.Close()

	now := time.Now()
	for _, pod := range pods {
		// 客户端断开后不再继续
		if ctx.Request.Context().Err() != nil {
			return
		}

		opts := &corev1.PodLogOptions{Container: container}
		if tailLines > 0 {
			opts.TailLines = &tailLines
		}

		stream, err := cs.CoreV1().Pods(namespace).GetLogs(pod, opts).Stream(ctx.Request.Context())
		if err != nil {
			writeZipText(zw, pod+".error.txt", now, "get logs stream failed: "+err.Error())
			continue
		}

		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:     pod + ".log",
			Method:   zip.Deflate,
			Modified: now,
		})
		if err != nil {
			stream.Close()
			return
		}
		_, copyErr := io.Copy(w, stream)
		stream.Close()
		if copyErr != nil {
			// 已开始写入，无法再回退；追加错误说明后继续处理下一个 Pod
			writeZipText(zw, pod+".error.txt", now, "read logs failed: "+copyErr.Error())
		}
	}
}

// parsePods 支持重复传参与逗号分隔，去空白并去重，保持传入顺序
func parsePods(raw []string) []string {
	seen := make(map[string]struct{})
	var pods []string
	for _, item := range raw {
		for _, p := range strings.Split(item, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			pods = append(pods, p)
		}
	}
	return pods
}

func writeZipText(zw *zip.Writer, name string, modified time.Time, text string) {
	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: modified,
	})
	if err != nil {
		return
	}
	_, _ = io.WriteString(w, text)
}

var unsafeFileNameRegexp = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

func sanitizeFileName(s string) string {
	return unsafeFileNameRegexp.ReplaceAllString(s, "_")
}

// failJSON 在尚未输出 zip 时，按项目统一格式返回 JSON 错误
func failJSON(ctx *gin.Context, message string) {
	api.Failure(ctx, -1, message)
}
