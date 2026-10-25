package k3s

import (
	"io"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

const defaultLogTailLines = int64(100)

// Log 获取 Pod 容器日志
// GET /api/k3s/log/:pod_name?namespace=<ns>[&tail=<n>][&container=<name>]
// tail 默认 100，传入时需为非负整数；tail=0 表示不限制行数（全量日志）。
func Log(ctx *gin.Context) {
	podName := strings.TrimSpace(ctx.Param("pod_name"))
	namespace := strings.TrimSpace(ctx.DefaultQuery("namespace", "default"))
	container := strings.TrimSpace(ctx.Query("container"))
	tailStr := strings.TrimSpace(ctx.Query("tail"))

	if podName == "" {
		api.Failure(ctx, -1, "missing path param: pod_name")
		return
	}
	if namespace == "" {
		namespace = "default"
	}

	tailLines := defaultLogTailLines
	if tailStr != "" {
		n, err := strconv.ParseInt(tailStr, 10, 64)
		// tail=0 表示不限制行数（全量日志）
		if err != nil || n < 0 {
			api.Failure(ctx, -1, "invalid query param tail: must be non-negative integer")
			return
		}
		if n == 0 {
			// tail=0：不限制行数，稍后保持 opts.TailLines 为 nil
			tailLines = 0
		} else {
			tailLines = n
		}
	}

	cs, err := getClientset()
	if err != nil {
		api.Failure(ctx, -1, "init k8s client failed: "+err.Error())
		return
	}

	opts := &corev1.PodLogOptions{
		Container: container,
	}
	// 注意：tail=0 时 opts.TailLines 会保持 nil，从而返回全量日志
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}

	req := cs.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx.Request.Context())
	if err != nil {
		api.Failure(ctx, -1, "get logs stream failed: "+err.Error())
		return
	}
	defer stream.Close()

	body, err := io.ReadAll(stream)
	if err != nil {
		api.Failure(ctx, -1, "read logs failed: "+err.Error())
		return
	}

	api.Success(ctx, map[string]interface{}{
		"log": string(body),
	})
}
