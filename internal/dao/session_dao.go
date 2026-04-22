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

// GetSessionBySendIdAndReceiveId 根据发送方和接收方获取会话
func (d *sessionDao) GetSessionBySendIdAndReceiveId(sendId, receiveId string) (*model.Session, error) {
	var session model.Session
	err := GormDB.Where("send_id = ? AND receive_id = ?", sendId, receiveId).First(&session).Error
	return &session, err
}

// CreateSession 创建会话
func (d *sessionDao) CreateSession(session *model.Session) error {
	return GormDB.Create(session).Error
}

func (d *sessionDao) GetSessionListBySendId(sendId string) ([]model.Session, error) {
	var sessions []model.Session
	err := GormDB.
		Where("send_id = ?", sendId).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (d *sessionDao) GetSessionByUuid(uuid string) (*model.Session, error) {
	var session model.Session
	err := GormDB.Where("uuid = ?", uuid).First(&session).Error
	return &session, err
}

// SoftDeleteSessionByUuid 根据会话id软删除会话
func (d *sessionDao) SoftDeleteSessionByUuid(sessionId string, deletedTime time.Time) error {
	deletedAt := gorm.DeletedAt{
		Time:  deletedTime,
		Valid: true,
	}

	return GormDB.Model(&model.Session{}).
		Where("uuid = ?", sessionId).
		Update("deleted_at", deletedAt).Error
}
