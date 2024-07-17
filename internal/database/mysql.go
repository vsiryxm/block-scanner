package database

import (
	"fmt"
	"sync"

	"block-scanner/config"
	"block-scanner/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

func GetDB() *gorm.DB {
	return db
}

func InitDB(cfg *config.Config) error {
	var err error
	once.Do(func() {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.MySQL.User,
			cfg.MySQL.Password,
			cfg.MySQL.Host,
			cfg.MySQL.Port,
			cfg.MySQL.Database,
		)

		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			return
		}

		// 自动迁移模型
		db.AutoMigrate(&models.Block{}, &models.Transaction{})

	})
	return err
}

func CloseDB() {
	if db != nil {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
}
