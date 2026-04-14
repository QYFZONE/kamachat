package dao

import (
	"kama_chat_server/internal/model"
	"time"

	"gorm.io/gorm"
)

type contactApplyDao struct{}

var ContactApply = new(contactApplyDao)

// QuitGroupByUserIdAndGroupId 将用户和群聊的联系人关系标记为退群并软删除
// userId: 当前用户 id
// groupId: 当前群聊 id
// deletedTime: 软删除时间
func (d *contactApplyDao) SoftDeleteGroupApplyByUserIdAndGroupId(userId, groupId string, deletedTime time.Time) error {
	deletedAt := gorm.DeletedAt{
		Time:  deletedTime,
		Valid: true,
	}

	return GormDB.Model(&model.ContactApply{}).
		Where("contact_id = ? AND user_id = ?", groupId, userId).
		Update("deleted_at", deletedAt).Error
}

// SoftDeleteGroupAppliesByGroupId 软删除群聊对应的所有申请记录
// groupId: 群聊 id
// deletedTime: 删除时间
func (d *contactApplyDao) SoftDeleteGroupAppliesByGroupId(groupId string, deletedTime time.Time) error {
	deletedAt := gorm.DeletedAt{
		Time:  deletedTime,
		Valid: true,
	}

	return GormDB.Model(&model.ContactApply{}).
		Where("contact_id = ?", groupId).
		Update("deleted_at", deletedAt).Error
}
