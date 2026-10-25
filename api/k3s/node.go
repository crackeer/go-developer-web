package k3s

import (
	"context"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crackeer/go-developer-web/api"
	"github.com/gin-gonic/gin"
)

type NodeSummary struct {
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Roles      string            `json:"roles"`
	Version    string            `json:"version"`
	InternalIP string            `json:"internal_ip"`
	PodCIDR    string            `json:"pod_cidr"`
	CreateTime string            `json:"create_time"`
	Labels     map[string]string `json:"labels"`
}

// Node 获取 K3s/K8s Node 摘要列表
// GET /api/k3s/node/list
func Node(ctx *gin.Context) {
	cs, err := getClientset()
	if err != nil {
		api.Failure(ctx, -1, "init k8s client failed: "+err.Error())
		return
	}

	c := context.Background()
	if ctx.Request != nil && ctx.Request.Context() != nil {
		c = ctx.Request.Context()
	}

	list, err := cs.CoreV1().Nodes().List(c, metav1.ListOptions{})
	if err != nil {
		api.Failure(ctx, -1, "list nodes failed: "+err.Error())
		return
	}

	out := make([]NodeSummary, 0, len(list.Items))
	for i := 0; i < len(list.Items); i++ {
		n := list.Items[i]
		out = append(out, NodeSummary{
			Name:       n.Name,
			Status:     getNodeReadyStatus(n),
			Roles:      getNodeRoles(n),
			Version:    n.Status.NodeInfo.KubeletVersion,
			InternalIP: getNodeInternalIP(n),
			PodCIDR:    n.Spec.PodCIDR,
			CreateTime: n.CreationTimestamp.Format("2006-01-02 15:04:05"),
			Labels:     n.Labels,
		})
	}

	api.Success(ctx, out)
}

func getNodeReadyStatus(n corev1.Node) string {
	for i := 0; i < len(n.Status.Conditions); i++ {
		c := n.Status.Conditions[i]
		if c.Type == corev1.NodeReady {
			if c.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func getNodeRoles(n corev1.Node) string {
	if len(n.Labels) < 1 {
		return "worker"
	}
	roles := make([]string, 0, 3)
	for k := range n.Labels {
		if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			r := strings.TrimPrefix(k, "node-role.kubernetes.io/")
			if r == "" {
				r = "worker"
			}
			roles = append(roles, r)
		}
	}
	if len(roles) < 1 {
		return "worker"
	}
	sort.Strings(roles)
	return strings.Join(roles, ",")
}

func getNodeInternalIP(n corev1.Node) string {
	for i := 0; i < len(n.Status.Addresses); i++ {
		a := n.Status.Addresses[i]
		if a.Type == corev1.NodeInternalIP {
			return a.Address
		}
	}
	return ""
}

// NodeLabelAdd 给 Node 添加标签
// POST /api/k3s/node/label/add
func NodeLabelAdd(ctx *gin.Context) {
	nodeName := strings.TrimSpace(ctx.Query("name"))
	if nodeName == "" {
		api.Failure(ctx, -1, "node name is required")
		return
	}

	var req struct {
		Labels map[string]string `json:"labels"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.Failure(ctx, -1, "invalid request body: "+err.Error())
		return
	}

	if len(req.Labels) == 0 {
		api.Failure(ctx, -1, "labels is required")
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

	node, err := cs.CoreV1().Nodes().Get(c, nodeName, metav1.GetOptions{})
	if err != nil {
		api.Failure(ctx, -1, "get node failed: "+err.Error())
		return
	}

	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	for k, v := range req.Labels {
		node.Labels[k] = v
	}

	_, err = cs.CoreV1().Nodes().Update(c, node, metav1.UpdateOptions{})
	if err != nil {
		api.Failure(ctx, -1, "update node failed: "+err.Error())
		return
	}

	api.Success(ctx, map[string]string{"node": nodeName})
}

// NodeLabelRemove 从 Node 删除标签
// POST /api/k3s/node/label/remove
func NodeLabelRemove(ctx *gin.Context) {
	nodeName := strings.TrimSpace(ctx.Query("name"))
	if nodeName == "" {
		api.Failure(ctx, -1, "node name is required")
		return
	}

	var req struct {
		LabelKeys []string `json:"labelKeys"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.Failure(ctx, -1, "invalid request body: "+err.Error())
		return
	}

	if len(req.LabelKeys) == 0 {
		api.Failure(ctx, -1, "label keys is required")
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

	node, err := cs.CoreV1().Nodes().Get(c, nodeName, metav1.GetOptions{})
	if err != nil {
		api.Failure(ctx, -1, "get node failed: "+err.Error())
		return
	}

	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	for _, k := range req.LabelKeys {
		delete(node.Labels, k)
	}

	_, err = cs.CoreV1().Nodes().Update(c, node, metav1.UpdateOptions{})
	if err != nil {
		api.Failure(ctx, -1, "update node failed: "+err.Error())
		return
	}

	api.Success(ctx, map[string]string{"node": nodeName})
}
