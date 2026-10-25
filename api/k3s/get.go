package k3s

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

var (
	clientOnce sync.Once
	clientset  *kubernetes.Clientset
	clientErr  error
)

func getClientset() (*kubernetes.Clientset, error) {
	clientOnce.Do(func() {
		cfg, err := getRestConfig()
		if err != nil {
			clientErr = err
			return
		}
		cs, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			clientErr = err
			return
		}
		clientset = cs
	})
	return clientset, clientErr
}

func getRestConfig() (*rest.Config, error) {
	kubeconfigPath := strings.TrimSpace(os.Getenv("KUBE_CONFIG"))
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}

func getIngressRouteGVRs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		{Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutes"},
		{Group: "traefik.containo.us", Version: "v1alpha1", Resource: "ingressroutes"},
	}
}

func getIngressRouteTCPGVRs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		{Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutetcps"},
		{Group: "traefik.containo.us", Version: "v1alpha1", Resource: "ingressroutetcps"},
	}
}

// Get 获取 K3s/K8s 集群资源详情
// GET /api/k3s/get/:resource?name=xxx[&namespace=default]
func Get(ctx *gin.Context) {
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

	var obj interface{}
	var getErr error
	c := context.Background()
	if ctx.Request != nil && ctx.Request.Context() != nil {
		c = ctx.Request.Context()
	}

	switch resource {
	// core/v1 (namespace-scoped)
	case "pods", "pod":
		obj, getErr = cs.CoreV1().Pods(namespace).Get(c, name, metav1.GetOptions{})
	case "services", "service", "svc":
		obj, getErr = cs.CoreV1().Services(namespace).Get(c, name, metav1.GetOptions{})
	case "configmaps", "configmap", "cm":
		obj, getErr = cs.CoreV1().ConfigMaps(namespace).Get(c, name, metav1.GetOptions{})
	case "secrets", "secret":
		obj, getErr = cs.CoreV1().Secrets(namespace).Get(c, name, metav1.GetOptions{})
	case "serviceaccounts", "serviceaccount", "sa":
		obj, getErr = cs.CoreV1().ServiceAccounts(namespace).Get(c, name, metav1.GetOptions{})
	case "namespaces", "namespace", "ns":
		// cluster-scoped
		obj, getErr = cs.CoreV1().Namespaces().Get(c, name, metav1.GetOptions{})

	// core/v1 (cluster-scoped)
	case "nodes", "node":
		obj, getErr = cs.CoreV1().Nodes().Get(c, name, metav1.GetOptions{})

	// apps/v1 (namespace-scoped)
	case "deployments", "deployment", "deploy":
		obj, getErr = cs.AppsV1().Deployments(namespace).Get(c, name, metav1.GetOptions{})
	case "daemonsets", "daemonset", "ds":
		obj, getErr = cs.AppsV1().DaemonSets(namespace).Get(c, name, metav1.GetOptions{})
	case "statefulsets", "statefulset", "sts":
		obj, getErr = cs.AppsV1().StatefulSets(namespace).Get(c, name, metav1.GetOptions{})
	case "replicasets", "replicaset", "rs":
		obj, getErr = cs.AppsV1().ReplicaSets(namespace).Get(c, name, metav1.GetOptions{})
	case "cronjobs", "cronjob", "cj":
		obj, getErr = cs.BatchV1().CronJobs(namespace).Get(c, name, metav1.GetOptions{})
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
			u, err := dc.Resource(gvr).Namespace(namespace).Get(c, name, metav1.GetOptions{})
			if err == nil {
				obj = u
				lastErr = nil
				break
			}
			lastErr = err
		}
		getErr = lastErr
	default:
		api.Failure(ctx, -1, "unsupported resource: "+resource)
		return
	}

	if getErr != nil {
		if apierrors.IsNotFound(getErr) {
			api.Failure(ctx, -1, "not found: "+getErr.Error())
			return
		}
		api.Failure(ctx, -1, "get resource failed: "+getErr.Error())
		return
	}

	format := strings.ToLower(strings.TrimSpace(ctx.Query("format")))
	if format == "yaml" {
		// 通过 clientset 拿到的强类型对象 TypeMeta(apiVersion/kind) 为空，这是 client-go 的已知行为。
		// 若直接 Marshal，YAML 顶层会缺少 kind，导致保存(apply)时解码报 "Object 'Kind' is missing"。
		// 这里用 scheme 回填 GVK（unstructured 对象自带 GVK，不走这里）。
		if runtimeObj, ok := obj.(runtime.Object); ok {
			if gvks, _, gerr := scheme.Scheme.ObjectKinds(runtimeObj); gerr == nil && len(gvks) > 0 {
				runtimeObj.GetObjectKind().SetGroupVersionKind(gvks[0])
			}
		}
		// 去掉 managedFields：该字段由 kube-apiserver 自动维护，仅用于 server-side-apply 记账，
		// 不属于期望状态，展示和编辑时都应隐藏，避免误导用户。
		if accessor, aerr := meta.Accessor(obj); aerr == nil {
			accessor.SetManagedFields(nil)
			// 剥掉 kubectl 记账类 annotation：这些由 kubectl 自动写入，不属于期望状态，
			// 展示/编辑时去掉以减少噪音；删除后不影响运行，必要时 kubectl 会重新写回。
			if anns := accessor.GetAnnotations(); len(anns) > 0 {
				delete(anns, "kubectl.kubernetes.io/last-applied-configuration")
				if len(anns) == 0 {
					accessor.SetAnnotations(nil)
				} else {
					accessor.SetAnnotations(anns)
				}
			}
		}
		b, err := yaml.Marshal(obj)
		if err != nil {
			api.Failure(ctx, -1, "marshal yaml failed: "+err.Error())
			return
		}
		ctx.Data(http.StatusOK, "application/yaml; charset=utf-8", b)
		ctx.Abort()
		return
	}

	api.Success(ctx, obj)
}
