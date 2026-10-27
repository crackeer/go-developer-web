package container

import (
	"fmt"

	env "github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload"
	"gorm.io/gorm"
)

var appConfig *AppConfig

var sqliteDB *gorm.DB

type AppConfig struct {
	Port          int64  `json:"port" env:"PORT" envDefault:"8080"`
	LoginPassword string `json:"loginPassword" env:"LOGIN_PASSWORD"`
	SQLitePath    string `json:"sqlitePath" env:"SQLITE_PATH" envDefault:"./devops-console.db"`
	Env           string `json:"env" env:"ENV"`
}

// IsDevelop 是否为开发环境（ENV=develop），开发环境下页面模板与静态资源直接读取本地文件。
func (c *AppConfig) IsDevelop() bool {
	return c.Env == "develop"
}

func InitAppConfig() error {
	appConfig = &AppConfig{}
	var err error
	if err := env.Parse(appConfig); err != nil {
		return err
	}
	sqliteDB, err = OpenDB(appConfig.SQLitePath)
	if err != nil {
		return fmt.Errorf("open sqlite %s error: %s", appConfig.SQLitePath, err.Error())
	}

	return nil
}

func GetAppConfig() *AppConfig {
	return appConfig
}

func GetSQLiteDB() *gorm.DB {
	return sqliteDB
}
