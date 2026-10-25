package k3s

import (
	"context"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// Delete 删除 K3s/K8s 集群资源
// DELETE /api/k3s/delete/:resource?name=xxx[&namespace=default]
func Delete(ctx *gin.Context) {
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

	var delErr error
	switch resource {
	// core/v1 (namespace-scoped)
	case "pods", "pod":
		delErr = cs.CoreV1().Pods(namespace).Delete(c, name, metav1.DeleteOptions{})
	case "services", "service", "svc":
		delErr = cs.CoreV1().Services(namespace).Delete(c, name, metav1.DeleteOptions{})
	case "configmaps", "configmap", "cm":
		delErr = cs.CoreV1().ConfigMaps(namespace).Delete(c, name, metav1.DeleteOptions{})
	case "serviceaccounts", "serviceaccount", "sa":
		delErr = cs.CoreV1().ServiceAccounts(namespace).Delete(c, name, metav1.DeleteOptions{})

	// apps/v1 (namespace-scoped)
	case "deployments", "deployment", "deploy":
		delErr = cs.AppsV1().Deployments(namespace).Delete(c, name, metav1.DeleteOptions{})
	case "daemonsets", "daemonset", "ds":
		delErr = cs.AppsV1().DaemonSets(namespace).Delete(c, name, metav1.DeleteOptions{})
	case "statefulsets", "statefulset", "sts":
		delErr = cs.AppsV1().StatefulSets(namespace).Delete(c, name, metav1.DeleteOptions{})

	// batch/v1 (namespace-scoped)
	case "cronjobs", "cronjob", "cj":
		delErr = cs.BatchV1().CronJobs(namespace).Delete(c, name, metav1.DeleteOptions{})

	// traefik CRD (namespace-scoped)
	case "ingressroutes", "ingressroute", "ir",
		"ingressroutetcps", "ingressroutetcp", "irtcp":
		cfg, err := getRestConfig()
		if err != nil {
			api.Failure(ctx, -1, "init k8s rest config failed: "+err.Error())
			return
		}
		dc, err := dynamic.NewForConfig(cfg)
		if err != nil {
			api.Failure(ctx, -1, "init k8s dynamic client failed: "+err.Error())
			return
		}
		gvrs := getIngressRouteGVRs()
		if !(resource == "ingressroutes" || resource == "ingressroute" || resource == "ir") {
			gvrs = getIngressRouteTCPGVRs()
		}
		var lastErr error
		for _, gvr := range gvrs {
			err := dc.Resource(gvr).Namespace(namespace).Delete(c, name, metav1.DeleteOptions{})
			if err == nil {
				lastErr = nil
				break
			}
			lastErr = err
		}
		delErr = lastErr
	default:
		api.Failure(ctx, -1, "unsupported resource: "+resource)
		return
	}

	if delErr != nil {
		if apierrors.IsNotFound(delErr) {
			api.Failure(ctx, -1, "not found: "+delErr.Error())
			return
		}
		api.Failure(ctx, -1, "delete resource failed: "+delErr.Error())
		return
	}

	api.Success(ctx, map[string]interface{}{
		"resource":  resource,
		"name":      name,
		"namespace": namespace,
		"deleted":   true,
	})
}
