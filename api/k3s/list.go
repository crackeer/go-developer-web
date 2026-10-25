package k3s

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// List 获取 K3s/K8s 集群某类资源的列表
// GET /api/k3s/list/:resource?namespace=xxx
func List(ctx *gin.Context) {
	resource := strings.ToLower(strings.TrimSpace(ctx.Param("resource")))
	namespace := strings.TrimSpace(ctx.DefaultQuery("namespace", "default"))
	if len(resource) < 1 {
		api.Failure(ctx, -1, "missing path param: resource")
		return
	}

	cs, err := getClientset()
	if err != nil {
		api.Failure(ctx, -1, "init k8s client failed: "+err.Error())
		return
	}

	var obj interface{}
	var listErr error
	c := context.Background()
	if ctx.Request != nil && ctx.Request.Context() != nil {
		c = ctx.Request.Context()
	}

	switch resource {
	// core/v1 (namespace-scoped)
	case "pods", "pod":
		obj, listErr = cs.CoreV1().Pods(namespace).List(c, metav1.ListOptions{})
	case "services", "service", "svc":
		obj, listErr = cs.CoreV1().Services(namespace).List(c, metav1.ListOptions{})
	case "configmaps", "configmap", "cm":
		obj, listErr = cs.CoreV1().ConfigMaps(namespace).List(c, metav1.ListOptions{})
	case "secrets", "secret":
		obj, listErr = cs.CoreV1().Secrets(namespace).List(c, metav1.ListOptions{})
	case "serviceaccounts", "serviceaccount", "sa":
		obj, listErr = cs.CoreV1().ServiceAccounts(namespace).List(c, metav1.ListOptions{})

	// core/v1 (cluster-scoped)
	case "namespaces", "namespace", "ns":
		obj, listErr = cs.CoreV1().Namespaces().List(c, metav1.ListOptions{})
	case "nodes", "node":
		obj, listErr = cs.CoreV1().Nodes().List(c, metav1.ListOptions{})

	// apps/v1 (namespace-scoped)
	case "deployments", "deployment", "deploy":
		obj, listErr = cs.AppsV1().Deployments(namespace).List(c, metav1.ListOptions{})
	case "daemonsets", "daemonset", "ds":
		obj, listErr = cs.AppsV1().DaemonSets(namespace).List(c, metav1.ListOptions{})
	case "statefulsets", "statefulset", "sts":
		obj, listErr = cs.AppsV1().StatefulSets(namespace).List(c, metav1.ListOptions{})
	case "replicasets", "replicaset", "rs":
		obj, listErr = cs.AppsV1().ReplicaSets(namespace).List(c, metav1.ListOptions{})
	case "cronjobs", "cronjob", "cj":
		obj, listErr = cs.BatchV1().CronJobs(namespace).List(c, metav1.ListOptions{})
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
		var merged *unstructured.UnstructuredList
		var lastErr error
		for _, gvr := range gvrs {
			u, err := dc.Resource(gvr).Namespace(namespace).List(c, metav1.ListOptions{})
			if err != nil {
				lastErr = err
				continue
			}
			lastErr = nil
			if merged == nil {
				merged = u
				continue
			}
			// 合并多套 CRD 的结果，避免第一个返回空 items 导致漏数据
			if u != nil && len(u.Items) > 0 {
				merged.Items = append(merged.Items, u.Items...)
			}
		}
		if merged == nil {
			listErr = lastErr
		} else {
			obj = merged
			listErr = nil
		}
	default:
		api.Failure(ctx, -1, "unsupported resource: "+resource)
		return
	}

	if listErr != nil {
		api.Failure(ctx, -1, "list resource failed: "+listErr.Error())
		return
	}

	api.Success(ctx, obj)
}
