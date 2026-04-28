package chat

import (
	"encoding/json"

	"kama_chat_server/internal/dao"
	"kama_chat_server/internal/model"
)

// store.go 是 chat 包和 dao 包之间的一层很薄的适配。
// 这样 chat 逻辑只依赖自己需要的最小能力，而不是直接依赖所有 dao 细节。

type messageStore interface {
	SaveMessage(message *model.Message) error
	UpdateMessageStatus(messageUUID string, status int8) error
	GetGroupMembers(groupID string) ([]string, error)
}

type daoMessageStore struct{}

var chatStore messageStore = daoMessageStore{}

// SaveMessage 把消息持久化到 message 表。
func (daoMessageStore) SaveMessage(message *model.Message) error {
	return dao.Message.CreateMessage(message)
}

// UpdateMessageStatus 更新消息发送状态。
func (daoMessageStore) UpdateMessageStatus(messageUUID string, status int8) error {
	return dao.Message.UpdateMessageStatus(messageUUID, status)
}

// GetGroupMembers 先查群，再把群成员 JSON 解析成 []string。
// 这是群发消息时成员列表的来源。
func (daoMessageStore) GetGroupMembers(groupID string) ([]string, error) {
	group, err := dao.Group.GetGroupInfoByGroupId(groupID)
	if err != nil {
		return nil, err
	}

	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		return nil, err
	}

	return members, nil
}
