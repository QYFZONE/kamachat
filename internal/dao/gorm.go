package dao

import (
	"fmt"

	"kama_chat_server/internal/config"
	"kama_chat_server/internal/model"
	"kama_chat_server/pkg/zlog"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var GormDB *gorm.DB

func init() {
	conf := config.GetConfig()

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		conf.MysqlConfig.User,
		conf.MysqlConfig.Password,
		conf.MysqlConfig.Host,
		conf.MysqlConfig.Port,
		conf.MysqlConfig.DatabaseName,
	)

	var err error
	GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		zlog.Fatal("mysql connect failed: " + err.Error())
	}

	err = GormDB.AutoMigrate(
		&model.UserInfo{},
		&model.GroupInfo{},
		&model.UserContact{},
		&model.Session{},
		&model.ContactApply{},
		&model.Message{},
	)
	if err != nil {
		zlog.Fatal("mysql auto migrate failed: " + err.Error())
	}
}
