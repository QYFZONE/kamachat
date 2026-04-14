package dao

import (
	"kama_chat_server/internal/model"
	"kama_chat_server/pkg/enum/contact/contact_status_enum"
	"time"

	"gorm.io/gorm"
)

type contactInfoDao struct{}

var Contact = new(contactInfoDao)

// 向表中插入一条新信息
func (d *contactInfoDao) CreateNewContact(newGroup *model.UserContact) error {
	return GormDB.Create(newGroup).Error
}

// QuitGroupByUserIdAndGroupId 将用户和群聊的联系人关系标记为退群并软删除
// userId: 当前用户 id
// groupId: 当前群聊 id
// deletedTime: 软删除时间
func (d *contactInfoDao) QuitGroupByUserIdAndGroupId(userId, groupId string, deletedTime time.Time) error {
	deletedAt := gorm.DeletedAt{
		Time:  deletedTime,
		Valid: true,
	}

	return GormDB.Model(&model.UserContact{}).
		Where("user_id = ? AND contact_id = ?", userId, groupId).
		Updates(map[string]interface{}{
			"deleted_at": deletedAt,
			"status":     contact_status_enum.QUIT_GROUP,
		}).Error
}

// SoftDeleteGroupContactsByGroupId 软删除群聊对应的所有联系人关系
// groupId: 群聊 id
// deletedTime: 删除时间
func (d *contactInfoDao) SoftDeleteGroupContactsByGroupId(groupId string, deletedTime time.Time) error {
	deletedAt := gorm.DeletedAt{
		Time:  deletedTime,
		Valid: true,
	}

	return GormDB.Model(&model.UserContact{}).
		Where("contact_id = ?", groupId).
		Update("deleted_at", deletedAt).Error
}
