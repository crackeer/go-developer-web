package resource

import (
	"fmt"
	"os"
	"strings"

	"github.com/crackeer/go-developer-web/container"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 支持的配置类型。
const (
	TypeMySQL = "mysql"
	TypeFTP   = "ftp"
	TypeSSH   = "ssh"
	TypeS3    = "s3"
	TypeRedis = "redis"
)

// testTimeout 连通性测试的超时时间。
const testTimeoutSeconds = 5

// ConfigSecret 读取密码加密密钥。
func ConfigSecret() string {
	if v := strings.TrimSpace(os.Getenv("CONFIG_SECRET")); v != "" {
		return v
	}
	return "devops-console-default-secret"
}

// FieldDef 类型扩展字段（extra JSON 内）的表单定义。
type FieldDef struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	Required    bool   `json:"required"`
}

// TypeMeta 某一配置类型的展示与表单元数据，驱动前端动态渲染。
type TypeMeta struct {
	Type            string     `json:"type"`
	Label           string     `json:"label"`
	HostLabel       string     `json:"host_label"`
	HostPlaceholder string     `json:"host_placeholder"`
	UserLabel       string     `json:"user_label"`
	PasswordLabel   string     `json:"password_label"`
	RequireHost     bool       `json:"require_host"`
	DefaultPort     int        `json:"default_port"`
	ExtraFields     []FieldDef `json:"extra_fields"`
}

// TypeMetas 全部类型的元数据。
func TypeMetas() []TypeMeta {
	return []TypeMeta{
		{
			Type:            TypeMySQL,
			Label:           "MySQL",
			HostLabel:       "主机",
			HostPlaceholder: "10.0.0.1",
			UserLabel:       "用户名",
			PasswordLabel:   "密码",
			RequireHost:     true,
			DefaultPort:     3306,
			ExtraFields: []FieldDef{
				{Key: "database", Label: "数据库", Placeholder: "连接的库名（可选）"},
			},
		},
		{
			Type:            TypeFTP,
			Label:           "FTP",
			HostLabel:       "主机",
			HostPlaceholder: "10.0.0.1",
			UserLabel:       "用户名",
			PasswordLabel:   "密码",
			RequireHost:     true,
			DefaultPort:     21,
			ExtraFields: []FieldDef{
				{Key: "prefix", Label: "根目录", Placeholder: "文件管理起始目录（可选）"},
			},
		},
		{
			Type:            TypeSSH,
			Label:           "SSH",
			HostLabel:       "主机",
			HostPlaceholder: "10.0.0.1",
			UserLabel:       "用户名",
			PasswordLabel:   "密码",
			RequireHost:     true,
			DefaultPort:     22,
		},
		{
			Type:            TypeS3,
			Label:           "S3",
			HostLabel:       "Endpoint",
			HostPlaceholder: "http://minio.example.com:9000（AWS 官方可留空）",
			UserLabel:       "AccessKey",
			PasswordLabel:   "SecretKey",
			RequireHost:     false,
			DefaultPort:     0,
			ExtraFields: []FieldDef{
				{Key: "region", Label: "Region", Placeholder: "us-east-1", Required: true},
				{Key: "bucket", Label: "Bucket", Placeholder: "默认 Bucket（可选）"},
			},
		},
		{
			Type:            TypeRedis,
			Label:           "Redis",
			HostLabel:       "主机",
			HostPlaceholder: "10.0.0.1",
			UserLabel:       "用户名",
			PasswordLabel:   "密码",
			RequireHost:     true,
			DefaultPort:     6379,
			ExtraFields: []FieldDef{
				{Key: "db", Label: "DB", Placeholder: "数据库编号（可选，默认 0）"},
			},
		},
	}
}

// TypeMetaMap 按类型取元数据。
var typeMetaMap = func() map[string]TypeMeta {
	m := map[string]TypeMeta{}
	for _, meta := range TypeMetas() {
		m[meta.Type] = meta
	}
	return m
}()

// GetTypeMeta 校验类型并返回元数据。
func GetTypeMeta(configType string) (TypeMeta, bool) {
	meta, ok := typeMetaMap[configType]
	return meta, ok
}

// SaveRequest 创建 / 更新配置的请求体。
type SaveRequest struct {
	Type     string            `json:"type" binding:"required"`
	Name     string            `json:"name"`
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Extra    map[string]string `json:"extra"`
	Enabled  *int              `json:"enabled"`
}

// MaskedConfig 列表 / 详情的返回结构：密码不回传，仅提示是否已配置。
type MaskedConfig struct {
	ID          uint              `json:"id"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	Username    string            `json:"username"`
	HasPassword bool              `json:"has_password"`
	Extra       map[string]string `json:"extra"`
	Enabled     int               `json:"enabled"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// Mask 转为脱敏结构（密码只返回是否已配置）。
func Mask(r container.Resource) MaskedConfig {
	config, err := r.DecodeConfig()
	if err != nil {
		config = &container.ConnectionConfig{}
	}
	return MaskedConfig{
		ID:          r.ID,
		Type:        r.ConfigType,
		Name:        r.Name,
		Host:        config.Host,
		Port:        config.Port,
		Username:    config.Username,
		HasPassword: config.Password != "",
		Extra:       config.Extra(),
		Enabled:     config.Enabled,
		CreatedAt:   r.CreateTimestamp.Format("2006-01-02 15:04:05"),
		UpdatedAt:   r.UpdateTimestamp.Format("2006-01-02 15:04:05"),
	}
}

// normalizeExtra 只保留该类型元数据中声明过的扩展字段。
func normalizeExtra(meta TypeMeta, extra map[string]string) map[string]string {
	out := map[string]string{}
	for _, field := range meta.ExtraFields {
		if v, ok := extra[field.Key]; ok {
			out[field.Key] = strings.TrimSpace(v)
		}
	}
	return out
}

// getDB 取 SQLite 连接。
func getDB(ctx *gin.Context) (*gorm.DB, bool) {
	return container.GetSQLiteDB(), true
}

// buildResource 校验请求体，返回待入库的资源（config 尚未编码）与连接参数。
func buildResource(req *SaveRequest) (*container.Resource, *container.ConnectionConfig, string) {
	meta, ok := GetTypeMeta(req.Type)
	if !ok {
		return nil, nil, fmt.Sprintf("unsupported type: %s", req.Type)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, nil, "name is required"
	}
	host := strings.TrimSpace(req.Host)
	if meta.RequireHost && host == "" {
		return nil, nil, fmt.Sprintf("%s is required", strings.ToLower(meta.HostLabel))
	}
	port := req.Port
	if port <= 0 {
		port = meta.DefaultPort
	}
	enabled := 1
	if req.Enabled != nil {
		enabled = *req.Enabled
		if enabled != 0 {
			enabled = 1
		}
	}
	config := &container.ConnectionConfig{
		Host:     host,
		Port:     port,
		Username: strings.TrimSpace(req.Username),
		Password: req.Password,
		Enabled:  enabled,
	}
	for key, value := range normalizeExtra(meta, req.Extra) {
		config.SetExtra(key, value)
	}
	return &container.Resource{Name: name, ConfigType: req.Type}, config, ""
}
