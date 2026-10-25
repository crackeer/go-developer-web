package k3s

import (
	"context"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// RolloutRestart 实现类似 `kubectl rollout restart` 的能力（通过修改 PodTemplate 注解触发滚动重启）
// POST /api/k3s/rollout/restart/:resource?name=xxx[&namespace=default]
func RolloutRestart(ctx *gin.Context) {
	resource := strings.ToLower(strings.TrimSpace(ctx.Param("resource")))
	name := strings.TrimSpace(ctx.Query("name"))
	namespace := strings.TrimSpace(ctx.DefaultQuery("namespace", "default"))
	if len(name) < 1 {
		api.Failure(ctx, -1, "missing query param: name")
		return
	}
	if len(resource) < 1 {
		api.Failure(ctx, -1, "missing path param: resource")
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

	restartedAt := time.Now().Format(time.RFC3339)
	const restartedAtKey = "kubectl.kubernetes.io/restartedAt"

	switch resource {
	case "deployments", "deployment", "deploy":
		obj, getErr := cs.AppsV1().Deployments(namespace).Get(c, name, metav1.GetOptions{})
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				api.Failure(ctx, -1, "not found: "+getErr.Error())
				return
			}
			api.Failure(ctx, -1, "get resource failed: "+getErr.Error())
			return
		}
		if obj.Spec.Template.Annotations == nil {
			obj.Spec.Template.Annotations = map[string]string{}
		}
		obj.Spec.Template.Annotations[restartedAtKey] = restartedAt
		_, updateErr := cs.AppsV1().Deployments(namespace).Update(c, obj, metav1.UpdateOptions{})
		if updateErr != nil {
			api.Failure(ctx, -1, "restart failed: "+updateErr.Error())
			return
		}
	case "daemonsets", "daemonset", "ds":
		obj, getErr := cs.AppsV1().DaemonSets(namespace).Get(c, name, metav1.GetOptions{})
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				api.Failure(ctx, -1, "not found: "+getErr.Error())
				return
			}
			api.Failure(ctx, -1, "get resource failed: "+getErr.Error())
			return
		}
		if obj.Spec.Template.Annotations == nil {
			obj.Spec.Template.Annotations = map[string]string{}
		}
		obj.Spec.Template.Annotations[restartedAtKey] = restartedAt
		_, updateErr := cs.AppsV1().DaemonSets(namespace).Update(c, obj, metav1.UpdateOptions{})
		if updateErr != nil {
			api.Failure(ctx, -1, "restart failed: "+updateErr.Error())
			return
		}
	case "statefulsets", "statefulset", "sts":
		obj, getErr := cs.AppsV1().StatefulSets(namespace).Get(c, name, metav1.GetOptions{})
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				api.Failure(ctx, -1, "not found: "+getErr.Error())
				return
			}
			api.Failure(ctx, -1, "get resource failed: "+getErr.Error())
			return
		}
		if obj.Spec.Template.Annotations == nil {
			obj.Spec.Template.Annotations = map[string]string{}
		}
		obj.Spec.Template.Annotations[restartedAtKey] = restartedAt
		_, updateErr := cs.AppsV1().StatefulSets(namespace).Update(c, obj, metav1.UpdateOptions{})
		if updateErr != nil {
			api.Failure(ctx, -1, "restart failed: "+updateErr.Error())
			return
		}
	default:
		api.Failure(ctx, -1, "unsupported resource: "+resource)
		return
	}

	api.Success(ctx, map[string]interface{}{
		"resource":    resource,
		"name":        name,
		"namespace":   namespace,
		"action":      "restarted",
		"restartedAt": restartedAt,
	})
}
