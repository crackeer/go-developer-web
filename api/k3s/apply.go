package k3s

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

type ApplyResult struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Action    string `json:"action"` // created / updated / error
	Error     string `json:"error,omitempty"`
}

func getDynamicClientAndMapper() (dynamic.Interface, meta.RESTMapper, error) {
	cfg, err := getRestConfig()
	if err != nil {
		return nil, nil, err
	}
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return nil, nil, err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	return dc, mapper, nil
}

// Apply 实现类似 `kubectl apply -f` 的能力
// POST /api/k3s/apply[?namespace=xxx]
// Body: YAML（可多文档）
func Apply(ctx *gin.Context) {
	body, err := ctx.GetRawData()
	if err != nil {
		api.Failure(ctx, -1, "read body failed: "+err.Error())
		return
	}
	if len(bytes.TrimSpace(body)) == 0 {
		api.Failure(ctx, -1, "empty body")
		return
	}

	nsOverride := strings.TrimSpace(ctx.Query("namespace"))

	dc, mapper, err := getDynamicClientAndMapper()
	if err != nil {
		api.Failure(ctx, -1, "init k8s dynamic client failed: "+err.Error())
		return
	}

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(body), 4096)
	results := []ApplyResult{}
	c := context.Background()
	if ctx.Request != nil && ctx.Request.Context() != nil {
		c = ctx.Request.Context()
	}

	for {
		u := &unstructured.Unstructured{}
		if err := decoder.Decode(u); err != nil {
			if err == io.EOF {
				break
			}
			// 解码过程中出现非 EOF 错误，整体失败
			api.Failure(ctx, -1, "parse yaml error: "+err.Error())
			return
		}
		// 可能解出空文档
		if u.Object == nil || len(u.Object) == 0 {
			continue
		}

		gvk := u.GroupVersionKind()
		kind := gvk.Kind
		name := u.GetName()
		namespace := u.GetNamespace()

		// 通过 RESTMapper 获取 GVR 及是否 namespaced
		mapping, err := mapper.RESTMapping(schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}, gvk.Version)
		if err != nil {
			results = append(results, ApplyResult{
				Kind:      kind,
				Name:      name,
				Namespace: namespace,
				Action:    "error",
				Error:     "RESTMapping error: " + err.Error(),
			})
			continue
		}

		// 命名空间覆盖逻辑（仅对 namespace-scoped 资源生效）
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace && nsOverride != "" {
			namespace = nsOverride
			u.SetNamespace(nsOverride)
		}

		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if namespace == "" {
				namespace = "default"
				u.SetNamespace(namespace)
			}
			dr = dc.Resource(mapping.Resource).Namespace(namespace)
		} else {
			dr = dc.Resource(mapping.Resource)
		}

		// 先 Get 判断是否存在
		current, getErr := dr.Get(c, name, metav1.GetOptions{})
		if getErr != nil {
			if errors.IsNotFound(getErr) {
				// Create
				_, createErr := dr.Create(c, u, metav1.CreateOptions{})
				if createErr != nil {
					results = append(results, ApplyResult{
						Kind:      kind,
						Name:      name,
						Namespace: namespace,
						Action:    "error",
						Error:     "create failed: " + createErr.Error(),
					})
				} else {
					results = append(results, ApplyResult{
						Kind:      kind,
						Name:      name,
						Namespace: namespace,
						Action:    "created",
					})
				}
				continue
			}
			results = append(results, ApplyResult{
				Kind:      kind,
				Name:      name,
				Namespace: namespace,
				Action:    "error",
				Error:     "get failed: " + getErr.Error(),
			})
			continue
		}

		// Update: 保留 resourceVersion
		u.SetResourceVersion(current.GetResourceVersion())
		_, updateErr := dr.Update(c, u, metav1.UpdateOptions{})
		if updateErr != nil {
			results = append(results, ApplyResult{
				Kind:      kind,
				Name:      name,
				Namespace: namespace,
				Action:    "error",
				Error:     "update failed: " + updateErr.Error(),
			})
			continue
		}

		results = append(results, ApplyResult{
			Kind:      kind,
			Name:      name,
			Namespace: namespace,
			Action:    "updated",
		})
	}

	if len(results) == 0 {
		api.Failure(ctx, -1, "no valid resource found in yaml")
		return
	}

	api.Success(ctx, results)
}
