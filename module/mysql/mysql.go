package mysql

import (
	"encoding/json"
	"fmt"
	"sort"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GetMySQLTables
//
//	@param gormDB
//	@return []string
func GetMySQLTables(gormDB *gorm.DB) []string {
	tables := []string{}
	list := []map[string]interface{}{}
	if err := gormDB.Raw("show tables").Scan(&list).Error; err != nil {
		return tables
	}

	for _, value := range list {
		for _, value := range value {
			tables = append(tables, value.(string))
		}
	}

	return tables
}

// TableDesc
type TableDesc struct {
	Default interface{} `json:"Default"`
	Extra   string      `json:"Extra"`
	Field   string      `json:"Field"`
	Key     string      `json:"Key"`
	Null    string      `json:"Null"`
	Type    string      `json:"Type"`
}

// GetMySQLTableFields
//
//	@param gormDB
//	@param table
//	@return []TableDesc
func GetMySQLTableFields(gormDB *gorm.DB, table string) []TableDesc {
	list := []TableDesc{}
	fileds := []map[string]interface{}{}
	if err := gormDB.Raw("desc `" + table + "`").Find(&fileds).Error; err != nil {
		return list
	}
	bytes, _ := json.Marshal(fileds)
	if err := json.Unmarshal(bytes, &list); err != nil {
		fmt.Printf("json unmarshal error: %v", err)
		return list
	}
	return list
}

// TableIndex 表示一个索引的定义（按列顺序聚合）。
type TableIndex struct {
	Name    string   // 索引名
	Unique  bool     // 是否唯一索引
	Primary bool     // 是否主键
	Columns []string // 组成索引的列（按 Seq_in_index 顺序）
}

// GetMySQLTableIndexes 读取表已存在的索引，按索引名聚合。
//
//	@param gormDB
//	@param table
//	@return map[string]TableIndex 以索引名为 key
func GetMySQLTableIndexes(gormDB *gorm.DB, table string) map[string]TableIndex {
	indexes := map[string]TableIndex{}
	rows := []map[string]interface{}{}
	if err := gormDB.Raw("show index from `" + table + "`").Find(&rows).Error; err != nil {
		return indexes
	}

	// 先按 Seq_in_index 排序，保证组合索引列顺序正确。
	toString := func(v interface{}) string {
		if v == nil {
			return ""
		}
		if b, ok := v.([]byte); ok {
			return string(b)
		}
		return fmt.Sprintf("%v", v)
	}
	toInt := func(v interface{}) int {
		var n int
		fmt.Sscanf(toString(v), "%d", &n)
		return n
	}

	sort.SliceStable(rows, func(i, j int) bool {
		return toInt(rows[i]["Seq_in_index"]) < toInt(rows[j]["Seq_in_index"])
	})

	for _, row := range rows {
		name := toString(row["Key_name"])
		if name == "" {
			continue
		}
		idx := indexes[name]
		idx.Name = name
		idx.Primary = name == "PRIMARY"
		// Non_unique = 0 表示唯一索引。
		idx.Unique = toInt(row["Non_unique"]) == 0
		idx.Columns = append(idx.Columns, toString(row["Column_name"]))
		indexes[name] = idx
	}

	return indexes
}

// GetCreateTableSQL
//
//	@param gormDB
//	@param table
//	@return string
func GetCreateTableSQL(gormDB *gorm.DB, table string) string {
	data := map[string]interface{}{}
	if err := gormDB.Raw("show create table `" + table + "`").Scan(&data).Error; err != nil {
		return ""
	}
	if value, ok := data["Create Table"]; ok {
		if stringValue, ok := value.(string); ok {
			return stringValue
		}
	}

	return ""
}

// 注意：go-sql-driver/mysql 的超时参数名是 readTimeout / writeTimeout（驼峰）。
// 写成 read_timeout / write_timeout 会被当作 MySQL 会话变量下发，服务端报
// "Error 1193: Unknown system variable 'read_timeout'"。
var gormDSN string = "%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&timeout=10s&readTimeout=30s&writeTimeout=30s&loc="
var gormDSN1 string = "%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=true&timeout=10s&readTimeout=30s&writeTimeout=30s&loc="

// GetGormDBFrom
//
//	@param user
//	@param password
//	@param host
//	@param port
//	@param database
//	@return *gorm.DB
//	@return error
func GetGormDBFrom(user, password, host, port, database string) (*gorm.DB, error) {
	dsn := fmt.Sprintf(gormDSN, user, password, host, port, database)
	if len(database) < 1 {
		dsn = fmt.Sprintf(gormDSN1, user, password, host, port)
	}
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

// UpdateDataByID ...
//
//	@param db
//	@param table
//	@param id
//	@param field
//	@param value
//	@return error
func UpdateDataByID(db *gorm.DB, table string, id interface{}, data map[string]interface{}) error {
	return db.Table(table).Where(map[string]interface{}{"id": id}).Updates(data).Error
}

// DeleteDataByID
//
//	@param db
//	@param table
//	@param id
//	@return error
func DeleteDataByID(db *gorm.DB, table string, id interface{}) error {
	sql := fmt.Sprintf("delete from %s where id = ?", table)
	return db.Exec(sql, id).Error
}

// QueryList
//
//	@param db
//	@param table
//	@param where
//	@return []map[string]interface{}
//	@return error
func QueryList(db *gorm.DB, table string, where map[string]interface{}) ([]map[string]interface{}, error) {
	list := []map[string]interface{}{}
	if err := db.Table(table).Where(where).Find(&list).Error; err != nil {
		return list, err
	}
	return list, nil
}

// CreateData
//
//	@param db
//	@param table
//	@param data
//	@return error
func CreateData(db *gorm.DB, table string, data map[string]interface{}) error {
	return db.Table(table).Create(&data).Error
}
