package k3s

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/describe"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

// describeGKMapping 维护 resource 名称（含常见别名）到 GroupKind 的映射。
// 目前仅开放 node；后续如需支持 deployment/service/pod 等，在此补充对应条目即可。
var describeGKMapping = map[string]schema.GroupKind{
	"nodes": {Group: "", Kind: "Node"},
	"node":  {Group: "", Kind: "Node"},
}

// clusterScopedKinds 标记忽略 namespace 的 cluster-scoped 资源。
var clusterScopedKinds = map[schema.GroupKind]bool{
	{Group: "", Kind: "Node"}: true,
}

// Describe 生成与 `kubectl describe` 等价的可读文本（基于 client-go，不依赖宿主机 kubectl）
// GET /api/k3s/describe/:resource?name=xxx[&namespace=default]
func Describe(ctx *gin.Context) {
	resource := strings.ToLower(strings.TrimSpace(ctx.Param("resource")))
	name := strings.TrimSpace(ctx.Query("name"))
	namespace := strings.TrimSpace(ctx.DefaultQuery("namespace", "default"))
	if len(resource) < 1 {
		api.Failure(ctx, -1, "missing path param: resource")
		return
	}
	if len(name) < 1 {
		api.Failure(ctx, -1, "missing query param: name")
		return
	}

	gk, ok := describeGKMapping[resource]
	if !ok {
		api.Failure(ctx, -1, "unsupported resource for describe: "+resource)
		return
	}

	cfg, err := getRestConfig()
	if err != nil {
		api.Failure(ctx, -1, "init k8s rest config failed: "+err.Error())
		return
	}

	// cluster-scoped 资源（如 node）忽略 namespace，避免描述器按错误 namespace 查找。
	if clusterScopedKinds[gk] {
		namespace = ""
	}

	d, ok := describe.DescriberFor(gk, cfg)
	if !ok {
		api.Failure(ctx, -1, "no describer for resource: "+resource)
		return
	}

	out, err := d.Describe(namespace, name, describe.DescriberSettings{ShowEvents: true})
	if err != nil {
		api.Failure(ctx, -1, "describe failed: "+err.Error())
		return
	}

	api.Success(ctx, map[string]string{"output": out})
}
