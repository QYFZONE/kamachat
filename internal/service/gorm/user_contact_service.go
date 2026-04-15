package gorm

import (
	"encoding/json"
	"errors"
	"fmt"
	"kama_chat_server/internal/dao"
	"kama_chat_server/internal/dto/request"
	"kama_chat_server/internal/dto/respond"
	"kama_chat_server/internal/model"
	myredis "kama_chat_server/internal/service/redis"
	"kama_chat_server/pkg/constants"
	"kama_chat_server/pkg/enum/contact/contact_status_enum"
	"kama_chat_server/pkg/enum/contact/contact_type_enum"
	"kama_chat_server/pkg/enum/contact_apply/contact_apply_status_enum"
	"kama_chat_server/pkg/enum/group_info/group_status_enum"
	"kama_chat_server/pkg/enum/user_info/user_status_enum"
	"kama_chat_server/pkg/util/random"
	"kama_chat_server/pkg/zlog"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type userContactService struct{}

var UserContactService = new(userContactService)

// GetUserList 获取用户联系人列表
// 关于用户被禁用的问题，这里查到的是所有联系人，如果被禁用或被拉黑会以弹窗的形式提醒，无法打开会话框；如果被删除，是搜索不到该联系人的。

func (u *userContactService) GetUserList(ownerId string) (string, []respond.MyUserListRespond, int) {
	cacheKey := "user:contact_list" + ownerId
	// 先查缓存
	cacheValue, err := myredis.GetKeyNilIsErr(cacheKey)
	if err == nil {
		var rsp []respond.MyUserListRespond
		if err := json.Unmarshal([]byte(cacheValue), &rsp); err != nil {
			zlog.Error("unmarshal user list cache failed: " + err.Error())
		} else {
			return "获取用户列表成功", rsp, 0
		}
	} else if !errors.Is(err, redis.Nil) {
		zlog.Error("get user list cache failed: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 查联系人关系
	contactList, err := dao.Contact.GetUserContactListByOwnerId(ownerId)
	if err != nil {
		zlog.Error("get user contact list failed: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	if len(contactList) == 0 {
		message := "目前不存在联系人"
		zlog.Info(message)
		return message, nil, 0
	}

	// 提取联系人用户id
	contactIds := make([]string, 0, len(contactList))
	for _, contact := range contactList {
		contactIds = append(contactIds, contact.ContactId)
	}
	// 批量查用户信息
	userList, err := dao.User.GetUserInfoListByUuids(contactIds)
	if err != nil {
		zlog.Error("get user info list failed: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	userListRsp := make([]respond.MyUserListRespond, 0, len(contactList))

	for _, user := range userList {
		userListRsp = append(userListRsp, respond.MyUserListRespond{
			UserId:   user.Uuid,
			UserName: user.Nickname,
			Avatar:   user.Avatar,
		})
	}

	// 回填缓存
	jsonData, err := json.Marshal(userListRsp)
	if err != nil {
		zlog.Error("marshal user list failed: " + err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			zlog.Error("set user list cache failed: " + err.Error())
		}
	}

	return "获取用户列表成功", userListRsp, 0
}

// LoadMyJoinedGroup 获取我加入的群聊
func (u *userContactService) LoadMyJoinedGroup(ownerId string) (string, []respond.LoadMyJoinedGroupRespond, int) {
	cacheKey := "user:group_joined_list:" + ownerId

	// 先查缓存
	cacheValue, err := myredis.GetKeyNilIsErr(cacheKey)
	if err == nil {
		var rsp []respond.LoadMyJoinedGroupRespond
		if err := json.Unmarshal([]byte(cacheValue), &rsp); err != nil {
			zlog.Error("unmarshal joined group list cache failed: " + err.Error())
			// 缓存损坏，继续查数据库
		} else {
			return "获取加入群成功", rsp, 0
		}
	} else if !errors.Is(err, redis.Nil) {
		zlog.Error("get joined group list cache failed: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 查当前用户加入的群关系
	// 查当前用户加入的群关系
	contactList, err := dao.Contact.GetJoinedGroupContactListByOwnerId(ownerId)
	if err != nil {
		zlog.Error("get joined group contact list failed: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	if len(contactList) == 0 {
		return "目前不存在加入的群聊", nil, 0
	}
	// 提取 groupId
	groupIds := make([]string, 0, len(contactList))
	for _, contact := range contactList {
		if contact.ContactId != "" {
			groupIds = append(groupIds, contact.ContactId)
		}
	}

	// 批量查群信息
	groupList, err := dao.Group.GetGroupInfoListByGroupIds(groupIds)
	if err != nil {
		zlog.Error("get group info list failed: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 按联系人顺序组装返回，并过滤掉“自己创建的群”
	groupListRsp := make([]respond.LoadMyJoinedGroupRespond, 0, len(contactList))
	for _, group := range groupList {
		// 只保留“加入的群”，不包含“我创建的群”
		if group.OwnerId == ownerId {
			continue
		}

		groupListRsp = append(groupListRsp, respond.LoadMyJoinedGroupRespond{
			GroupId:   group.Uuid,
			GroupName: group.Name,
			Avatar:    group.Avatar,
		})
	}
	// 回填缓存
	jsonData, err := json.Marshal(groupListRsp)
	if err != nil {
		zlog.Error("marshal joined group list failed: " + err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			zlog.Error("set joined group list cache failed: " + err.Error())
		}
	}

	return "获取加入群成功", groupListRsp, 0
}

// GetContactInfo 获取联系人信息
// 调用前提：该联系人未被删除，且用户仍在群中或联系人关系仍有效
func (u *userContactService) GetContactInfo(contactId string) (string, respond.GetContactInfoRespond, int) {
	if contactId == "" {
		return "参数错误", respond.GetContactInfoRespond{}, -2
	}

	// 群聊联系人
	if contactId[0] == 'G' {
		cacheKey := "group:info:" + contactId

		// 先查缓存
		cacheValue, err := myredis.GetKeyNilIsErr(cacheKey)
		if err == nil {
			var rep respond.GetContactInfoRespond
			if err := json.Unmarshal([]byte(cacheValue), &rep); err != nil {
				zlog.Error("unmarshal group contact info cache failed: " + err.Error())
			} else {
				return "获取联系人信息成功", rep, 0
			}
		} else if !errors.Is(err, redis.Nil) {
			zlog.Error("get group contact info cache failed: " + err.Error())
			return constants.SYSTEM_ERROR, respond.GetContactInfoRespond{}, -1
		}

		// 查数据库
		group, err := dao.Group.GetGroupInfoByGroupId(contactId)
		if err != nil {
			zlog.Error("get group info failed: " + err.Error())
			return constants.SYSTEM_ERROR, respond.GetContactInfoRespond{}, -1
		}

		if group.Status == group_status_enum.DISABLE {
			zlog.Error("该群聊处于禁用状态")
			return "该群聊处于禁用状态", respond.GetContactInfoRespond{}, -2
		}

		rep := respond.GetContactInfoRespond{
			ContactId:        group.Uuid,
			ContactName:      group.Name,
			ContactAvatar:    group.Avatar,
			ContactNotice:    group.Notice,
			ContactAddMode:   group.AddMode,
			ContactMembers:   group.Members,
			ContactMemberCnt: group.MemberCnt,
			ContactOwnerId:   group.OwnerId,
		}

		// 回填缓存
		jsonData, err := json.Marshal(rep)
		if err != nil {
			zlog.Error("marshal group contact info failed: " + err.Error())
		} else {
			if err := myredis.SetKeyEx(cacheKey, string(jsonData), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Error("set group contact info cache failed: " + err.Error())
			}
		}

		return "获取联系人信息成功", rep, 0
	}

	// 用户联系人
	cacheKey := "user:info:" + contactId

	// 先查缓存
	cacheValue, err := myredis.GetKeyNilIsErr(cacheKey)
	if err == nil {
		var rep respond.GetContactInfoRespond
		if err := json.Unmarshal([]byte(cacheValue), &rep); err != nil {
			zlog.Error("unmarshal user contact info cache failed: " + err.Error())
		} else {
			return "获取联系人信息成功", rep, 0
		}
	} else if !errors.Is(err, redis.Nil) {
		zlog.Error("get user contact info cache failed: " + err.Error())
		return constants.SYSTEM_ERROR, respond.GetContactInfoRespond{}, -1
	}

	// 查数据库
	user, err := dao.User.GetUserInfoByUuid(contactId)
	if err != nil {
		zlog.Error("get user info failed: " + err.Error())
		return constants.SYSTEM_ERROR, respond.GetContactInfoRespond{}, -1
	}

	if user.Status == user_status_enum.DISABLE {
		zlog.Info("该用户处于禁用状态")
		return "该用户处于禁用状态", respond.GetContactInfoRespond{}, -2
	}

	rep := respond.GetContactInfoRespond{
		ContactId:        user.Uuid,
		ContactName:      user.Nickname,
		ContactAvatar:    user.Avatar,
		ContactBirthday:  user.Birthday,
		ContactEmail:     user.Email,
		ContactPhone:     user.Telephone,
		ContactGender:    user.Gender,
		ContactSignature: user.Signature,
	}

	// 回填缓存
	jsonData, err := json.Marshal(rep)
	if err != nil {
		zlog.Error("marshal user contact info failed: " + err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			zlog.Error("set user contact info cache failed: " + err.Error())
		}
	}

	return "获取联系人信息成功", rep, 0
}

// DeleteContact 删除联系人（只包含用户）
func (u *userContactService) DeleteContact(ownerId, contactId string) (string, int) {
	if ownerId == "" || contactId == "" {
		return "参数错误", -2
	}

	now := time.Now()
	// 更新自己这边的联系人关系为已删除
	if err := dao.Contact.DeleteUserContact(ownerId, contactId, now, contact_status_enum.DELETE); err != nil {
		zlog.Error("delete self contact failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 更新对方这边的联系人关系为被删除
	if err := dao.Contact.DeleteUserContact(contactId, ownerId, now, contact_status_enum.BE_DELETE); err != nil {
		zlog.Error("delete peer contact failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 删除双方会话
	if err := dao.Session.SoftDeleteUserSession(ownerId, contactId, now); err != nil {
		zlog.Error("delete self session failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if err := dao.Session.SoftDeleteUserSession(contactId, ownerId, now); err != nil {
		zlog.Error("delete peer session failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 删除双方申请记录
	if err := dao.ContactApply.SoftDeleteUserApply(ownerId, contactId, now); err != nil {
		zlog.Error("delete self apply failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if err := dao.ContactApply.SoftDeleteUserApply(contactId, ownerId, now); err != nil {
		zlog.Error("delete peer apply failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 删除双方联系人列表缓存
	if err := myredis.DelKey("user:contact_list:" + ownerId); err != nil {
		zlog.Error("delete owner contact list cache failed: " + err.Error())
	}
	if err := myredis.DelKey("user:contact_list:" + contactId); err != nil {
		zlog.Error("delete contact contact list cache failed: " + err.Error())
	}

	return "删除联系人成功", 0
}

// ApplyContact 申请添加联系人
func (u *userContactService) ApplyContact(req request.ApplyContactRequest) (string, int) {
	if req.OwnerId == "" || req.ContactId == "" {
		return "参数错误", -2
	}
	// 申请用户
	if req.ContactId[0] == 'U' {
		if req.OwnerId == req.ContactId {
			return "不能添加自己", -2
		}

		user, err := dao.User.GetUserInfoByUuid(req.OwnerId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				zlog.Error("用户不存在")
				return "用户不存在", -2
			}
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}

		if user.Status == user_status_enum.DISABLE {
			zlog.Info("用户已被禁用")
			return "用户已被禁用", -2
		}
		// 查询历史申请记录
		contactApply, err := dao.ContactApply.GetContactApplyByUserIdAndContactId(req.OwnerId, req.ContactId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 没有申请记录，创建新申请
				newApply := &model.ContactApply{
					Uuid:        fmt.Sprintf("A%s", random.GetNowAndLenRandomString(11)),
					UserId:      req.OwnerId,
					ContactId:   req.ContactId,
					ContactType: contact_type_enum.USER,
					Status:      contact_apply_status_enum.PENDING,
					Message:     req.Message,
					LastApplyAt: time.Now(),
				}
				if err := dao.ContactApply.CreateContactApply(newApply); err != nil {
					zlog.Error(err.Error())
					return constants.SYSTEM_ERROR, -1
				}
				return "申请成功", 0
			}
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}

		// 已被对方拉黑，不能再次申请
		if contactApply.Status == contact_apply_status_enum.BLACK {
			return "对方已将你拉黑", -2
		}
		// 更新原申请记录为最新申请
		contactApply.LastApplyAt = time.Now()
		contactApply.Status = contact_apply_status_enum.PENDING
		contactApply.Message = req.Message

		if err := dao.ContactApply.SaveContactApply(contactApply); err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		return "申请成功", 0
	}

	// 申请群聊
	if req.ContactId[0] == 'G' {
		// 查询目标群聊是否存在
		group, err := dao.Group.GetGroupInfoByGroupId(req.ContactId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				zlog.Error("群聊不存在")
				return "群聊不存在", -2
			}
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		// 群聊不能处于禁用状态
		if group.Status == group_status_enum.DISABLE {
			zlog.Info("群聊已被禁用")
			return "群聊已被禁用", -2
		}
		// 不能申请加入自己创建的群聊
		if group.OwnerId == req.OwnerId {
			return "不能申请加入自己创建的群聊", -2
		}
		// 查询历史申请记录
		contactApply, err := dao.ContactApply.GetContactApplyByUserIdAndContactId(req.OwnerId, req.ContactId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 没有申请记录，创建新申请
				newApply := &model.ContactApply{
					Uuid:        fmt.Sprintf("A%s", random.GetNowAndLenRandomString(11)),
					UserId:      req.OwnerId,
					ContactId:   req.ContactId,
					ContactType: contact_type_enum.GROUP,
					Status:      contact_apply_status_enum.PENDING,
					Message:     req.Message,
					LastApplyAt: time.Now(),
				}
				if err := dao.ContactApply.CreateContactApply(newApply); err != nil {
					zlog.Error(err.Error())
					return constants.SYSTEM_ERROR, -1
				}
				return "申请成功", 0
			}
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}

		// 已被禁止申请该群聊
		if contactApply.Status == contact_apply_status_enum.BLACK {
			return "你已被禁止申请该群聊", -2
		}

		// 更新原申请记录为最新申请
		contactApply.LastApplyAt = time.Now()
		contactApply.Status = contact_apply_status_enum.PENDING
		contactApply.Message = req.Message

		if err := dao.ContactApply.SaveContactApply(contactApply); err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		return "申请成功", 0
	}
	return "用户/群聊不存在", -2
}
