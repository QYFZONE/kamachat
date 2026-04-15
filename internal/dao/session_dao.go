package dao

import (
	"kama_chat_server/internal/model"
	"time"

	"gorm.io/gorm"
)

type sessionDao struct{}

var Session = new(sessionDao)

// SoftDeleteGroupSession 软删除用户与群聊之间的会话记录
// userId: 当前用户 id
// groupId: 当前群聊 id
// deletedTime: 软删除时间
func (d *sessionDao) SoftDeleteGroupSession(userId, groupId string, deletedTime time.Time) error {
	deletedAt := gorm.DeletedAt{
		Time:  deletedTime,
		Valid: true,
	}

	return GormDB.Model(&model.Session{}).
		Where("send_id = ? AND receive_id = ?", userId, groupId).
		Update("deleted_at", deletedAt).Error
}

// SoftDeleteGroupSessionsByGroupId 软删除群聊对应的所有会话
// groupId: 群聊 id
// deletedTime: 删除时间
func (d *sessionDao) SoftDeleteGroupSessionsByGroupId(groupId string, deletedTime time.Time) error {
	deletedAt := gorm.DeletedAt{
		Time:  deletedTime,
		Valid: true,
	}

	return GormDB.Model(&model.Session{}).
		Where("receive_id = ?", groupId).
		Update("deleted_at", deletedAt).Error
}

// UpdateGroupSessionsByGroupId 更新群聊对应的所有会话中的群名和头像
// groupId: 群聊id
// groupName: 新群名
// avatar: 新群头像
func (d *sessionDao) UpdateGroupSessionsByGroupId(groupId, groupName, avatar string) error {
	return GormDB.Model(&model.Session{}).
		Where("receive_id = ?", groupId).
		Updates(map[string]interface{}{
			"receive_name": groupName,
			"avatar":       avatar,
		}).Error
}

// SoftDeleteUserSession 软删除用户之间的会话
func (d *sessionDao) SoftDeleteUserSession(sendId, receiveId string, deletedTime time.Time) error {
	deletedAt := gorm.DeletedAt{
		Time:  deletedTime,
		Valid: true,
	}

	return GormDB.Model(&model.Session{}).
		Where("send_id = ? AND receive_id = ?", sendId, receiveId).
		Update("deleted_at", deletedAt).Error
}
