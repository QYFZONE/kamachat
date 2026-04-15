package dao

import (
	"kama_chat_server/internal/model"
	"kama_chat_server/pkg/enum/contact/contact_status_enum"
	"kama_chat_server/pkg/enum/contact/contact_type_enum"
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

// GetUserContactListByOwnerId 获取指定用户的好友联系人关系列表
// 仅返回联系人类型为用户、且未被删除的关系
func (d *contactInfoDao) GetUserContactListByOwnerId(ownerId string) ([]model.UserContact, error) {
	var contactList []model.UserContact

	err := GormDB.
		Order("created_at DESC").
		Where("user_id = ? AND contact_type = ? AND status != ?", ownerId, contact_type_enum.USER, 4).
		Find(&contactList).Error

	return contactList, err
}

// GetJoinedGroupContactListByOwnerId 获取指定用户加入的群聊关系列表
// 过滤掉已退群、已被踢出等无效关系
func (d *contactInfoDao) GetJoinedGroupContactListByOwnerId(ownerId string) ([]model.UserContact, error) {
	var contactList []model.UserContact

	err := GormDB.
		Order("created_at DESC").
		Where("user_id = ? AND contact_type = ? AND status != ? AND status != ?", ownerId, contact_type_enum.GROUP, 6, 7).
		Find(&contactList).Error

	return contactList, err
}

// DeleteUserContact 软删除用户联系人关系，并更新状态
func (d *contactInfoDao) DeleteUserContact(userId, contactId string, deletedTime time.Time, status int8) error {
	deletedAt := gorm.DeletedAt{
		Time:  deletedTime,
		Valid: true,
	}

	return GormDB.Model(&model.UserContact{}).
		Where("user_id = ? AND contact_id = ?", userId, contactId).
		Updates(map[string]interface{}{
			"deleted_at": deletedAt,
			"status":     status,
		}).Error
}

// GetContactApplyByUserIdAndContactId 获取申请记录
func (d *contactApplyDao) GetContactApplyByUserIdAndContactId(userId, contactId string) (*model.ContactApply, error) {
	var contactApply model.ContactApply
	err := GormDB.Where("user_id = ? AND contact_id = ?", userId, contactId).First(&contactApply).Error
	return &contactApply, err
}

// CreateContactApply 创建申请记录
func (d *contactApplyDao) CreateContactApply(contactApply *model.ContactApply) error {
	return GormDB.Create(contactApply).Error
}

// SaveContactApply 保存申请记录
func (d *contactApplyDao) SaveContactApply(contactApply *model.ContactApply) error {
	return GormDB.Save(contactApply).Error
}
