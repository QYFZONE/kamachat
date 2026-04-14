package dao

import (
	"kama_chat_server/internal/model"
	"time"

	"gorm.io/gorm"
)

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

// 保存更新后的group信息
func (d *groupInfoDao) SaveGroup(group *model.GroupInfo) error {
	return GormDB.Save(group).Error
}

// SoftDeleteGroupByGroupId 软删除指定群聊
// groupId: 群聊 id
// deletedTime: 删除时间
func (d *groupInfoDao) SoftDeleteGroupByGroupId(groupId string, deletedTime time.Time) error {
	deletedAt := gorm.DeletedAt{
		Time:  deletedTime,
		Valid: true,
	}

	return GormDB.Model(&model.GroupInfo{}).
		Where("uuid = ?", groupId).
		Updates(map[string]interface{}{
			"deleted_at": deletedAt,
			"updated_at": deletedTime,
		}).Error
}
