package main

import (
	"fmt"
	"kama_chat_server/internal/config"
	"kama_chat_server/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1. 加载配置
	cfg := config.GetConfig()

	// 2. 构建数据库连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MysqlConfig.User,
		cfg.MysqlConfig.Password,
		cfg.MysqlConfig.Host,
		cfg.MysqlConfig.Port,
		cfg.MysqlConfig.DatabaseName,
	)

	// 3. 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}

	// 4. 自动建表（核心！）
	err = db.AutoMigrate(
		&model.UserInfo{},
		&model.UserContact{},
		&model.ContactApply{},
		&model.GroupInfo{},
		&model.Session{},
		&model.Message{},
	)
	if err != nil {
		panic("建表失败: " + err.Error())
	}

	fmt.Println("✅ 数据库表创建成功！")
}
