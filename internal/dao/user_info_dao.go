package dao

import (
	"gorm.io/gorm"
	"kama_chat_server/internal/model"
)

var GormDB *gorm.DB

type userInfoDao struct{}

var User = new(userInfoDao)

func (d *userInfoDao) GetUserInfoByTelephone(telephone string) (*model.UserInfo, error) {
	var user model.UserInfo
	err := GormDB.Where("telephone = ?", telephone).First(&user).Error
	return &user, err
}
