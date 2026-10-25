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
	Port          int64  `json:"port" env:"PORT" default:"8080"`
	LoginPassword string `json:"loginPassword" env:"LOGIN_PASSWORD"`
	SQLitePath    string `json:"sqlitePath" env:"SQLITE_PATH" default:"./devops-console.db"`
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
