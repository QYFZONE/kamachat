package dao

import (
	"gorm.io/gorm"
	"kama_chat_server/internal/model"
	"time"
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
