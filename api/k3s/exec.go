package k3s

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

type execBodyJSON struct {
	Command []string `json:"command"`
}

// Exec 在 Pod 内执行命令（类似 kubectl exec）
// POST /api/k3s/exec/:pod_name?namespace=<ns>[&container=<name>]
// Body:
//   - JSON: {"command":["/bin/sh","-c","echo hello"]}
//   - 纯文本: 作为 shell 一行命令执行（/bin/sh -c "<body>"）
func Exec(ctx *gin.Context) {
	podName := strings.TrimSpace(ctx.Param("pod_name"))
	namespace := strings.TrimSpace(ctx.DefaultQuery("namespace", "default"))
	container := strings.TrimSpace(ctx.Query("container"))

	if podName == "" {
		api.Failure(ctx, -1, "missing path param: pod_name")
		return
	}
	if namespace == "" {
		namespace = "default"
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		api.Failure(ctx, -1, "read body failed: "+err.Error())
		return
	}

	var command []string
	var j execBodyJSON
	if json.Unmarshal(body, &j) == nil && len(j.Command) > 0 {
		command = j.Command
	} else {
		cmdline := strings.TrimSpace(string(body))
		if cmdline == "" {
			api.Failure(ctx, -1, "empty command body")
			return
		}
		command = []string{"/bin/sh", "-c", cmdline}
	}

	cfg, err := getRestConfig()
	if err != nil {
		api.Failure(ctx, -1, "rest config failed: "+err.Error())
		return
	}
	cs, err := getClientset()
	if err != nil {
		api.Failure(ctx, -1, "init k8s client failed: "+err.Error())
		return
	}

	req := cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		api.Failure(ctx, -1, "new executor failed: "+err.Error())
		return
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamWithContext(ctx.Request.Context(), remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
		Tty:    false,
	})

	out := map[string]interface{}{
		"stdout": stdout.String(),
		"stderr": stderr.String(),
	}
	if err != nil {
		out["error"] = err.Error()
	}
	api.Success(ctx, out)
}
