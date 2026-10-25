package k3s

import (
	"context"
	"encoding/json"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// Patch 实现类似 `kubectl patch` 的能力，对 deployment / statefulset / daemonset 提交 patch
// POST /api/k3s/patch/:resource?name=xxx[&namespace=default][&type=strategic|merge|json]
// body: 原始 patch JSON
//   - type=strategic（默认）：strategic merge patch，列表（如 containers）按 name 合并，值为 null 表示删除该字段
//   - type=merge：JSON merge patch (RFC 7386)，数组会被整体替换，慎用于 containers
//   - type=json：JSON patch (RFC 6902)，body 为操作数组
func Patch(ctx *gin.Context) {
	resource := strings.ToLower(strings.TrimSpace(ctx.Param("resource")))
	name := strings.TrimSpace(ctx.Query("name"))
	namespace := strings.TrimSpace(ctx.DefaultQuery("namespace", "default"))
	patchTypeStr := strings.ToLower(strings.TrimSpace(ctx.DefaultQuery("type", "strategic")))

	if len(name) < 1 {
		api.Failure(ctx, -1, "missing query param: name")
		return
	}
	if len(resource) < 1 {
		api.Failure(ctx, -1, "missing path param: resource")
		return
	}

	var patchType types.PatchType
	switch patchTypeStr {
	case "strategic", "strategic-merge":
		patchType = types.StrategicMergePatchType
	case "merge":
		patchType = types.MergePatchType
	case "json":
		patchType = types.JSONPatchType
	default:
		api.Failure(ctx, -1, "unsupported patch type: "+patchTypeStr+", only strategic, merge and json are supported")
		return
	}

	body, err := ctx.GetRawData()
	if err != nil {
		api.Failure(ctx, -1, "read request body failed: "+err.Error())
		return
	}
	if len(strings.TrimSpace(string(body))) < 1 {
		api.Failure(ctx, -1, "empty patch body")
		return
	}
	if !json.Valid(body) {
		api.Failure(ctx, -1, "patch body is not valid JSON")
		return
	}

	cs, err := getClientset()
	if err != nil {
		api.Failure(ctx, -1, "init k8s client failed: "+err.Error())
		return
	}

	c := context.Background()
	if ctx.Request != nil && ctx.Request.Context() != nil {
		c = ctx.Request.Context()
	}

	var patchErr error
	var resourceVersion string

	switch resource {
	case "deployments", "deployment", "deploy":
		obj, e := cs.AppsV1().Deployments(namespace).Patch(c, name, patchType, body, metav1.PatchOptions{})
		patchErr = e
		if e == nil && obj != nil {
			resourceVersion = obj.ResourceVersion
		}
	case "statefulsets", "statefulset", "sts":
		obj, e := cs.AppsV1().StatefulSets(namespace).Patch(c, name, patchType, body, metav1.PatchOptions{})
		patchErr = e
		if e == nil && obj != nil {
			resourceVersion = obj.ResourceVersion
		}
	case "daemonsets", "daemonset", "ds":
		obj, e := cs.AppsV1().DaemonSets(namespace).Patch(c, name, patchType, body, metav1.PatchOptions{})
		patchErr = e
		if e == nil && obj != nil {
			resourceVersion = obj.ResourceVersion
		}
	default:
		api.Failure(ctx, -1, "unsupported resource for patch: "+resource+", only deployment, statefulset and daemonset are supported")
		return
	}

	if patchErr != nil {
		if apierrors.IsNotFound(patchErr) {
			api.Failure(ctx, -1, "not found: "+patchErr.Error())
			return
		}
		api.Failure(ctx, -1, "patch failed: "+patchErr.Error())
		return
	}

	api.Success(ctx, map[string]interface{}{
		"resource":        resource,
		"name":            name,
		"namespace":       namespace,
		"action":          "patched",
		"patchType":       patchTypeStr,
		"resourceVersion": resourceVersion,
	})
}
