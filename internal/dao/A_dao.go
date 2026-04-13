package dao

import "gorm.io/gorm"

// 只在这里定义一次，整个 dao 包共用
var GormDB *gorm.DB

// SetDB 供 main.go 初始化时调用
func SetDB(db *gorm.DB) {
	GormDB = db
}
