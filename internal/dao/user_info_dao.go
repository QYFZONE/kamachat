package dao

import "kama_chat_server/internal/model"

type userInfoDao struct{}

var User = new(userInfoDao)

// 用Telephone字段查询 用户信息
func (d *userInfoDao) GetUserInfoByTelephone(telephone string) (*model.UserInfo, error) {
	var user model.UserInfo
	// gorm默认排除软删除，所以翻译过来的select语句是:
	// SELECT * FROM `user_info` WHERE telephone = '18089596095' AND `user_info`.`deleted_at` IS NULL ORDER BY `user_info`.`id` LIMIT 1
	err := GormDB.Where("telephone = ?", telephone).First(&user).Error
	return &user, err
}

// 用Uuid字段查询 用户信息
func (d *userInfoDao) GetUserInfoByUuid(uuid string) (*model.UserInfo, error) {
	var user model.UserInfo

	err := GormDB.Where("uuid = ?", uuid).First(&user).Error
	return &user, err
}

// GetUsersExcept 查询除指定uuid外的所有用户
func (d *userInfoDao) GetUsersExcept(uuid string) ([]model.UserInfo, error) {
	var users []model.UserInfo
	err := GormDB.Unscoped().Where("uuid != ?", uuid).Find(&users).Error
	return users, err
}

func (d *userInfoDao) SaveUser(user *model.UserInfo) error {
	return GormDB.Save(user).Error
}

// 向表中插入一条新信息
func (d *userInfoDao) CreateNewUser(newUser *model.UserInfo) error {
	return GormDB.Create(newUser).Error
}
