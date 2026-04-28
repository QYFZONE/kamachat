package dao

import "kama_chat_server/internal/model"

type messageDao struct{}

var Message = new(messageDao)

func (d *messageDao) CreateMessage(message *model.Message) error {
	return GormDB.Create(message).Error
}

func (d *messageDao) UpdateMessageStatus(messageUUID string, status int8) error {
	return GormDB.Model(&model.Message{}).
		Where("uuid = ?", messageUUID).
		Update("status", status).Error
}

// GetMessageListByUserIds 根据双方用户id获取聊天记录
func (d *messageDao) GetMessageListByUserIds(userOneId, userTwoId string) ([]model.Message, error) {
	var messageList []model.Message

	err := GormDB.
		Where("(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)",
			userOneId, userTwoId, userTwoId, userOneId).
		Order("created_at ASC").
		Find(&messageList).Error

	return messageList, err
}

// GetGroupMessageListByGroupId 根据群聊id获取群聊消息列表
func (d *messageDao) GetGroupMessageListByGroupId(groupId string) ([]model.Message, error) {
	var messageList []model.Message

	err := GormDB.
		Where("receive_id = ?", groupId).
		Order("created_at ASC").
		Find(&messageList).Error

	return messageList, err
}
