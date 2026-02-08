package db

import (
	"fmt"
	"note/config"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitMySQL 初始化 MySQL 连接
func InitMySQL(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// 配置 GORM 日志，根据你的 Env 决定级别
	var gormLogger logger.Interface
	if cfg.AppEnv == "dev" {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	// 设置连接池 (生产环境必备)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(10)           // 空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最长存活时间

	zap.L().Info("MySQL connected successfully")
	return db
}
