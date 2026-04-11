package dao

import (
	"kama_chat_server/internal/model"

	"gorm.io/gorm"
)

var GormDB *gorm.DB

type userInfoDao struct{}

var User = new(userInfoDao)

func (d *userInfoDao) GetUserInfoByTelephone(telephone string) (*model.UserInfo, error) {
	var user model.UserInfo
	// gorm默认排除软删除，所以翻译过来的select语句是:
	// SELECT * FROM `user_info` WHERE telephone = '18089596095' AND `user_info`.`deleted_at` IS NULL ORDER BY `user_info`.`id` LIMIT 1
	err := GormDB.Where("telephone = ?", telephone).First(&user).Error
	return &user, err
}

func (d *userInfoDao) CreatNewUser(newUser *model.UserInfo) error {
	return GormDB.Create(newUser).Error
}
