package dao

import "kama_chat_server/internal/model"

type groupInfoDao struct{}

var Group = new(groupInfoDao)

// 向表中插入一条新信息
func (d *groupInfoDao) CreateNewGroup(newGroup *model.GroupInfo) error {
	return GormDB.Create(newGroup).Error
}

// 使用用户ownerId 获取我创建的群聊
func (d *groupInfoDao) GetGroupInfoByOwnerId(ownerId string) ([]model.GroupInfo, error) {
	var groupInfos []model.GroupInfo
	err := GormDB.Where("ownerId = ?", ownerId).Find(&groupInfos).Error
	return groupInfos, err
}

// 使用groupId 获取群聊
func (d *groupInfoDao) GetGroupInfoByGroupId(groupId string) (*model.GroupInfo, error) {
	var groupInfo model.GroupInfo
	err := GormDB.Where("gruopId = ?", groupId).First(&groupInfo).Error
	return &groupInfo, err
}

func (d *groupInfoDao) SaveGroup(group *model.GroupInfo) error {
	return GormDB.Save(group).Error
}
