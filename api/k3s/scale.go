package k3s

import (
	"context"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// Scale 实现类似 `kubectl scale` 的能力，更新 deployment / statefulset 的副本数
// POST /api/k3s/scale/:resource?name=xxx&replicas=3[&namespace=default]
func Scale(ctx *gin.Context) {
	resource := strings.ToLower(strings.TrimSpace(ctx.Param("resource")))
	name := strings.TrimSpace(ctx.Query("name"))
	namespace := strings.TrimSpace(ctx.DefaultQuery("namespace", "default"))
	replicasStr := strings.TrimSpace(ctx.Query("replicas"))

	if len(name) < 1 {
		api.Failure(ctx, -1, "missing query param: name")
		return
	}
	if len(resource) < 1 {
		api.Failure(ctx, -1, "missing path param: resource")
		return
	}
	if len(replicasStr) < 1 {
		api.Failure(ctx, -1, "missing query param: replicas")
		return
	}

	replicas, err := strconv.ParseInt(replicasStr, 10, 32)
	if err != nil || replicas < 0 {
		api.Failure(ctx, -1, "invalid replicas value: "+replicasStr)
		return
	}
	replicasInt32 := int32(replicas)

	cs, err := getClientset()
	if err != nil {
		api.Failure(ctx, -1, "init k8s client failed: "+err.Error())
		return
	}

	c := context.Background()
	if ctx.Request != nil && ctx.Request.Context() != nil {
		c = ctx.Request.Context()
	}

	var previousReplicas int32

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
		if obj.Spec.Replicas != nil {
			previousReplicas = *obj.Spec.Replicas
		}
		obj.Spec.Replicas = &replicasInt32
		_, updateErr := cs.AppsV1().Deployments(namespace).Update(c, obj, metav1.UpdateOptions{})
		if updateErr != nil {
			api.Failure(ctx, -1, "scale failed: "+updateErr.Error())
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
		if obj.Spec.Replicas != nil {
			previousReplicas = *obj.Spec.Replicas
		}
		obj.Spec.Replicas = &replicasInt32
		_, updateErr := cs.AppsV1().StatefulSets(namespace).Update(c, obj, metav1.UpdateOptions{})
		if updateErr != nil {
			api.Failure(ctx, -1, "scale failed: "+updateErr.Error())
			return
		}

	default:
		api.Failure(ctx, -1, "unsupported resource for scale: "+resource+", only deployment and statefulset are supported")
		return
	}

	api.Success(ctx, map[string]interface{}{
		"resource":         resource,
		"name":             name,
		"namespace":        namespace,
		"action":           "scaled",
		"previousReplicas": previousReplicas,
		"currentReplicas":  replicasInt32,
	})
}
