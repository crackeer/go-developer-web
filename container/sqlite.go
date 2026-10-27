package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func OpenDB(path string) (*gorm.DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite dir %s error: %s", dir, err.Error())
		}
	}
	conn, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s error: %s", path, err.Error())
	}
	if err := conn.AutoMigrate(&Resource{}); err != nil {
		return nil, fmt.Errorf("sqlite auto migrate error: %s", err.Error())
	}
	return conn, nil
}

// ConnectionConfig 资源的连接参数，按 config_type 以 JSON 存入 Resource.Config。
// 各类型只序列化自身需要的字段：
//
//	mysql: {"host":"10.0.0.1","port":3306,"username":"root","password":"<密文>","database":"db_name"}
//	ftp:   {"host":"10.0.0.1","port":21,"username":"ftp","password":"<密文>","prefix":"/data/ftp"}
//	ssh:   {"host":"10.0.0.1","port":22,"username":"root","password":"<密文>"}
//	s3:    {"host":"http://minio:9000","username":"AK","password":"<密文>","region":"us-east-1","bucket":"my-bucket"}
//	redis: {"host":"10.0.0.1","port":6379,"username":"","password":"<密文>","db":0}
type ConnectionConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"` // 密文
	Database string `json:"database,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Region   string `json:"region,omitempty"`
	Bucket   string `json:"bucket,omitempty"`
	DB       int    `json:"db,omitempty"`
}

// SetExtra 按类型扩展字段 key 写入连接参数。
func (c *ConnectionConfig) SetExtra(key, value string) {
	switch key {
	case "database":
		c.Database = value
	case "prefix":
		c.Prefix = value
	case "region":
		c.Region = value
	case "bucket":
		c.Bucket = value
	case "db":
		if n, err := strconv.Atoi(value); err == nil && n >= 0 {
			c.DB = n
		}
	}
}

// Extra 返回类型扩展字段，用于接口返回。
func (c *ConnectionConfig) Extra() map[string]string {
	extra := map[string]string{}
	if c.Database != "" {
		extra["database"] = c.Database
	}
	if c.Prefix != "" {
		extra["prefix"] = c.Prefix
	}
	if c.Region != "" {
		extra["region"] = c.Region
	}
	if c.Bucket != "" {
		extra["bucket"] = c.Bucket
	}
	if c.DB > 0 {
		extra["db"] = strconv.Itoa(c.DB)
	}
	return extra
}

// Resource 外部资源（MySQL / FTP / SSH / S3 / Redis 等）的统一配置，存储在 SQLite。
// 连接参数按 config_type 以自定义 JSON 存放在 Config 中，见 ConnectionConfig。
type Resource struct {
	ID              uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name            string    `gorm:"column:name;type:varchar(128);not null;uniqueIndex:uk_config_type_name" json:"name"`
	ConfigType      string    `gorm:"column:config_type;type:varchar(32);not null;uniqueIndex:uk_config_type_name" json:"config_type"`
	Config          string    `gorm:"column:config;type:text;not null;default:'{}'" json:"config"`
	CreateTimestamp time.Time `gorm:"column:create_timestamp;autoCreateTime" json:"create_timestamp"`
	UpdateTimestamp time.Time `gorm:"column:update_timestamp;autoUpdateTime" json:"update_timestamp"`
}

// TableName GORM 表名。
func (Resource) TableName() string {
	return "resource"
}

// DecodeConfig 将 Config JSON 解析为连接参数。
func (r *Resource) DecodeConfig() (*ConnectionConfig, error) {
	config := &ConnectionConfig{}
	if r.Config == "" {
		return config, nil
	}
	if err := json.Unmarshal([]byte(r.Config), config); err != nil {
		return nil, err
	}
	return config, nil
}

// EncodeConfig 将连接参数序列化后写入 Config。
func (r *Resource) EncodeConfig(config *ConnectionConfig) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	r.Config = string(raw)
	return nil
}

// ListResources 按类型列出资源，configType 为空时返回全部。
func ListResources(db *gorm.DB, configType string) ([]Resource, error) {
	query := db.Model(&Resource{})
	if configType != "" {
		query = query.Where("config_type = ?", configType)
	}
	rows := []Resource{}
	if err := query.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetResource 按 ID 查询单条资源。
func GetResource(db *gorm.DB, id string) (*Resource, error) {
	var resource Resource
	if err := db.Where("id = ?", id).First(&resource).Error; err != nil {
		return nil, err
	}
	return &resource, nil
}

// CreateResource 新增资源，主键与时间戳回填到 record。
func CreateResource(db *gorm.DB, record *Resource) error {
	return db.Create(record).Error
}

// UpdateResource 按 ID 更新资源的名称与配置（config_type 不允许修改）。
func UpdateResource(db *gorm.DB, record *Resource) error {
	return db.Model(&Resource{}).Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"name":   record.Name,
			"config": record.Config,
		}).Error
}

// DeleteResource 按 ID 删除资源，返回是否删除了记录。
func DeleteResource(db *gorm.DB, id string) (bool, error) {
	result := db.Where("id = ?", id).Delete(&Resource{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// MySQLConfig MySQL 连接配置（config_type=mysql 时的 Config JSON）。
type MySQLConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"` // 密文
	Database string `json:"database,omitempty"`
}

// FTPConfig FTP 连接配置（config_type=ftp 时的 Config JSON）。
type FTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"` // 密文
	Prefix   string `json:"prefix,omitempty"`
}

// SSHConfig SSH 连接配置（config_type=ssh 时的 Config JSON）。
type SSHConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"` // 密文
}

// S3Config S3 连接配置（config_type=s3 时的 Config JSON）。
type S3Config struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region,omitempty"`
	Bucket    string `json:"bucket,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
}

// RedisConfig Redis 连接配置（config_type=redis 时的 Config JSON）。
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"` // 密文
	DB       int    `json:"db,omitempty"`
}

// decodeConfig 将 resource.config 的 JSON 解析为指定类型的连接配置。
func decodeConfig[T any](resource *Resource) (*T, error) {
	config := new(T)
	if resource.Config == "" {
		return config, nil
	}
	if err := json.Unmarshal([]byte(resource.Config), config); err != nil {
		return nil, err
	}
	return config, nil
}

// queryResourceByID 按类型与 ID 查询单条资源。
func queryResourceByID(configType string, id int64) (*Resource, error) {
	var resource Resource
	if err := sqliteDB.Where("config_type = ? AND id = ?", configType, id).First(&resource).Error; err != nil {
		return nil, err
	}
	return &resource, nil
}

// QueryMySQLConfig 按 ID 查询 MySQL 连接配置。
func QueryMySQLConfig(id int64) (*MySQLConfig, error) {
	resource, err := queryResourceByID("mysql", id)
	if err != nil {
		return nil, err
	}
	return decodeConfig[MySQLConfig](resource)
}

// QueryMySQLResource 按 ID 查询 MySQL 资源及其连接配置；不存在时返回 gorm.ErrRecordNotFound。
func QueryMySQLResource(idStr string) (*Resource, *MySQLConfig, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, nil, gorm.ErrRecordNotFound
	}
	resource, err := queryResourceByID("mysql", id)
	if err != nil {
		return nil, nil, err
	}
	config, err := decodeConfig[MySQLConfig](resource)
	if err != nil {
		return nil, nil, err
	}
	return resource, config, nil
}

// QueryRedisConfig 按 ID 查询 Redis 连接配置。
func QueryRedisConfig(id int64) (*RedisConfig, error) {
	resource, err := queryResourceByID("redis", id)
	if err != nil {
		return nil, err
	}
	return decodeConfig[RedisConfig](resource)
}

// QueryFTPConfig 按 ID 查询 FTP 连接配置。
func QueryFTPConfig(idStr string) (*FTPConfig, error) {
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resource, err := queryResourceByID("ftp", id)
	if err != nil {
		return nil, err
	}
	return decodeConfig[FTPConfig](resource)
}

// QuerySSHConfig 按 ID 查询 SSH 连接配置。
func QuerySSHConfig(idStr string) (*SSHConfig, error) {
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resource, err := queryResourceByID("ssh", id)
	if err != nil {
		return nil, err
	}
	return decodeConfig[SSHConfig](resource)
}

// QueryS3Config 按 ID 查询 S3 连接配置。
func QueryS3Config(idStr string) (*S3Config, error) {
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resource, err := queryResourceByID("s3", id)
	if err != nil {
		return nil, err
	}
	return decodeConfig[S3Config](resource)
}
