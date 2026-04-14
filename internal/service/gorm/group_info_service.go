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
	"kama_chat_server/pkg/enum/group_info/group_status_enum"
	"kama_chat_server/pkg/util/random"
	"kama_chat_server/pkg/zlog"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type groupInfoService struct {
}

var GroupInfoService = groupInfoService{}

// CreateGroup 创建群聊
func (g *groupInfoService) CreateGroup(groupReq request.CreateGroupRequest) (string, int) {
	if groupReq.OwnerId == "" || groupReq.Name == "" {
		return "参数错误", -2
	}

	now := time.Now()

	group := model.GroupInfo{
		Uuid:      fmt.Sprintf("G%s", random.GetNowAndLenRandomString(11)),
		Name:      groupReq.Name,
		Notice:    groupReq.Notice,
		OwnerId:   groupReq.OwnerId,
		MemberCnt: 1,
		AddMode:   groupReq.AddMode,
		Avatar:    groupReq.Avatar,
		Status:    group_status_enum.NORMAL,
		CreatedAt: now,
		UpdatedAt: now,
	}

	members := []string{groupReq.OwnerId}
	membersJSON, err := json.Marshal(members)
	if err != nil {
		zlog.Error("marshal group members failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	group.Members = membersJSON

	if err := dao.Group.CreateNewGroup(&group); err != nil {
		zlog.Error("create group failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	contact := model.UserContact{
		UserId:      groupReq.OwnerId,
		ContactId:   group.Uuid,
		ContactType: contact_type_enum.GROUP,
		Status:      contact_status_enum.NORMAL,
		CreatedAt:   now,
		UpdateAt:    now,
	}

	if err := myredis.DelKey("user:group_created_list:" + groupReq.OwnerId); err != nil {
		zlog.Error("delete my group list cache failed: " + err.Error())
	}

	if err := dao.Contact.CreateNewContact(&contact); err != nil {
		zlog.Error("create owner contact failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	if err := myredis.DelKey("contact:mygroup_list_" + groupReq.OwnerId); err != nil {
		zlog.Error("delete my group list cache failed: " + err.Error())
	}

	return "创建成功", 0
}

// LoadMyGroup 获取我创建的群聊
func (g *groupInfoService) LoadMyGroup(ownerId string) (string, []respond.LoadMyGroupRespond, int) {
	cacheKey := "user:group_created_list:" + ownerId

	//先查缓存
	repString, err := myredis.GetKey(cacheKey)
	if err == nil {
		//在redis命中 解析
		var groupListRsp []respond.LoadMyGroupRespond
		if err := json.Unmarshal([]byte(repString), &groupListRsp); err != nil {
			zlog.Error("缓存数据解析失败: " + err.Error())
			// 解析失败，继续查数据库（视为缓存未命中）
		} else {
			return "获取成功", groupListRsp, 0
		}
	} else if !errors.Is(err, redis.Nil) {
		// 非 Nil 错误（连接失败等）
		zlog.Error("Redis 查询异常: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 3. 缓存未命中，查数据库
	groupList, err := dao.Group.GetGroupInfoByOwnerId(ownerId)
	if err != nil {
		zlog.Error("数据库查询失败: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	// 5. 构造响应

	groupListRsp := make([]respond.LoadMyGroupRespond, 0, len(groupList))
	for _, group := range groupList {
		groupListRsp = append(groupListRsp, respond.LoadMyGroupRespond{
			GroupId:   group.Uuid,
			GroupName: group.Name,
			Avatar:    group.Avatar,
		})
	}
	// 存入缓存
	jsonData, err := json.Marshal(groupListRsp)
	if err != nil {
		zlog.Error("JSON 序列化失败: " + err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), 10*time.Minute); err != nil {
			zlog.Error("Redis 回填失败: " + err.Error())
			// 不影响主流程，仅记录日志
		}
	}

	return "获取用户信息成功", groupListRsp, 0
}

// CheckGroupAddMode 检查群聊加群方式
func (g *groupInfoService) CheckGroupAddMode(groupId string) (string, int8, int) {
	cacheKey := "group:add_mode:" + groupId
	//先查缓存
	rspString, err := myredis.GetKeyNilIsErr(cacheKey)
	if err == nil {
		//在redis命中
		if mode, err := strconv.ParseInt(rspString, 10, 8); err != nil {
			zlog.Error("群加群方式缓存解析失败: " + err.Error())
			// 解析失败，继续查数据库
		} else {
			return "获取成功", int8(mode), 0
		}
	} else if !errors.Is(err, redis.Nil) {
		// Redis 真异常
		zlog.Error("Redis 查询异常: " + err.Error())
		return constants.SYSTEM_ERROR, 0, -1
	}

	// 缓存未命中，查数据库
	groupInfo, err := dao.Group.GetGroupInfoByGroupId(groupId)
	if err != nil {
		zlog.Error("数据库查询群信息失败: " + err.Error())
		return constants.SYSTEM_ERROR, 0, -1
	}

	// 回填缓存
	if err := myredis.SetKeyEx(cacheKey, strconv.FormatInt(int64(groupInfo.AddMode), 10), 10*time.Minute); err != nil {
		zlog.Error("群加群方式缓存回填失败: " + err.Error())
		// 不影响主流程
	}

	return "获取成功", groupInfo.AddMode, 0
}

// EnterGroupDirectly 直接进群
// contactId 是群聊Id
func (g *groupInfoService) EnterGroupDirectly(userId, contactId string) (string, int) {
	group, err := dao.Group.GetGroupInfoByGroupId(contactId)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 已在群中，直接返回
	for _, member := range members {
		if member == userId {
			return "用户已在群中", -2
		}
	}

	members = append(members, userId)

	data, err := json.Marshal(members)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	group.Members = data
	group.MemberCnt++

	if err := dao.Group.SaveGroup(group); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	now := time.Now()
	newContact := model.UserContact{
		UserId:      userId,
		ContactId:   contactId,
		ContactType: contact_type_enum.GROUP,
		Status:      contact_status_enum.NORMAL,
		CreatedAt:   now,
		UpdateAt:    now,
	}

	if err := dao.Contact.CreateNewContact(&newContact); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 删缓存
	if err := myredis.DelKey("group:info:" + contactId); err != nil {
		zlog.Error(err.Error())
	}
	if err := myredis.DelKey("group:member_list:" + contactId); err != nil {
		zlog.Error(err.Error())
	}

	if err := myredis.DelKey("user:group_joined_list:" + userId); err != nil {
		zlog.Error(err.Error())
	}
	return "进群成功", 0
}

// LeaveGroup 退群
func (g *groupInfoService) LeaveGroup(userId, groupId string) (string, int) {
	// 查询群信息
	group, err := dao.Group.GetGroupInfoByGroupId(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 群主不能直接退群
	if group.OwnerId == userId {
		return "群主不能直接退群", -2
	}

	// 解析群成员列表
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 从成员列表中移除当前用户
	found := false
	for i, member := range members {
		if member == userId {
			found = true
			members = append(members[:i], members[i+1:]...)
			break
		}
	}
	if !found {
		return "用户不在群中", -2
	}

	// 更新群成员信息
	data, err := json.Marshal(members)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	group.Members = data
	group.MemberCnt--
	if err := dao.Group.SaveGroup(group); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	now := time.Now()

	// 软删除群会话
	if err := dao.Session.SoftDeleteGroupSession(userId, groupId, now); err != nil {
		zlog.Error("soft delete group session failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 更新群联系人关系为已退群
	if err := dao.Contact.QuitGroupByUserIdAndGroupId(userId, groupId, now); err != nil {
		zlog.Error("quit group contact failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 删除入群申请记录
	if err := dao.ContactApply.SoftDeleteGroupApplyByUserIdAndGroupId(userId, groupId, now); err != nil {
		zlog.Error("soft delete group apply failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 删除相关缓存
	if err := myredis.DelKey("group:info:" + groupId); err != nil {
		zlog.Error(err.Error())
	}
	if err := myredis.DelKey("group:member_list:" + groupId); err != nil {
		zlog.Error(err.Error())
	}
	//if err := myredis.DelKey("user:group_joined_list:" + userId); err != nil {
	//	zlog.Error(err.Error())
	//}
	//if err := myredis.DelKey("group:session_list:" + userId); err != nil {
	//	zlog.Error(err.Error())
	//}

	return "退群成功", 0
}

// DismissGroup 解散群聊
func (g *groupInfoService) DismissGroup(ownerId, groupId string) (string, int) {
	group, err := dao.Group.GetGroupInfoByGroupId(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if group.OwnerId == ownerId {
		return "无权限解散群聊", -2
	}

	now := time.Now()
	// 软删除群信息
	if err := dao.Group.SoftDeleteGroupByGroupId(groupId, now); err != nil {
		zlog.Error("soft delete group failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 软删除群相关会话
	if err := dao.Session.SoftDeleteGroupSessionsByGroupId(groupId, now); err != nil {
		zlog.Error("soft delete group sessions failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 软删除群相关联系人关系
	if err := dao.Contact.SoftDeleteGroupContactsByGroupId(groupId, now); err != nil {
		zlog.Error("soft delete group contacts failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 软删除群相关申请记录
	if err := dao.ContactApply.SoftDeleteGroupAppliesByGroupId(groupId, now); err != nil {
		zlog.Error("soft delete group applies failed: " + err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 删除相关缓存
	// 我创建的群列表缓存
	if err := myredis.DelKey("user:group_created_list:" + ownerId); err != nil {
		zlog.Error("delete user group created list cache failed: " + err.Error())
	}

	// 群信息缓存
	if err := myredis.DelKey("group:info:" + groupId); err != nil {
		zlog.Error("delete group info cache failed: " + err.Error())
	}

	// 群成员列表缓存
	if err := myredis.DelKey("group:member_list:" + groupId); err != nil {
		zlog.Error("delete group member list cache failed: " + err.Error())
	}

	// 所有用户的“我加入的群列表”都可能受影响
	// 当前先按前缀删，后续如果有成员列表可以精确删除对应用户缓存
	if err := myredis.DelKeysWithPrefix("user:group_joined_list:"); err != nil {
		zlog.Error("delete user group joined list cache failed: " + err.Error())
	}
	return "解散群聊成功", 0
}

// GetGroupInfo查询群聊信息
func (g *groupInfoService) GetGroupInfo(groupId string) (string, *respond.GetGroupInfoRespond, int) {
	repString, err := myredis.GetKeyNilIsErr("group:info:" + groupId)
	if err == nil {
		var rep respond.GetGroupInfoRespond
		err = json.Unmarshal([]byte(repString), &rep)
		if err != nil {
			zlog.Error(err.Error())
		} else {
			return "获取成功", &rep, 0
		}
	} else if !errors.Is(err, redis.Nil) {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	group, err := dao.Group.GetGroupInfoByGroupId(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	rep := &respond.GetGroupInfoRespond{
		Uuid:      group.Uuid,
		Name:      group.Name,
		Notice:    group.Notice,
		Avatar:    group.Avatar,
		MemberCnt: group.MemberCnt,
		OwnerId:   group.OwnerId,
		AddMode:   group.AddMode,
		Status:    group.Status,
		IsDeleted: group.DeletedAt.Valid,
	}
	jsonData, err := json.Marshal(rep)
	if err != nil {
		zlog.Error("marshal group info failed: " + err.Error())
	} else {
		if err := myredis.SetKeyEx("group:info:"+groupId, string(jsonData), 10*time.Minute); err != nil {
			zlog.Error("set group info cache failed: " + err.Error())
		}
	}
	return "获取成功", rep, 0
}

// GetGroupMemberList 获取群聊成员列表
func (g *groupInfoService) GetGroupMemberList(groupId string) (string, []respond.GetGroupMemberListRespond, int) {
	cacheKey := "group:member_list:" + groupId

	// 先查缓存
	repString, err := myredis.GetKeyNilIsErr(cacheKey)
	if err == nil {
		var repMemberList []respond.GetGroupMemberListRespond
		if unmarshalErr := json.Unmarshal([]byte(repString), &repMemberList); unmarshalErr != nil {
			zlog.Error("unmarshal group member list cache failed: " + unmarshalErr.Error())
			// 缓存解析失败，继续查数据库
		} else {
			return "获取群聊成员列表成功", repMemberList, 0
		}
	} else if !errors.Is(err, redis.Nil) {
		zlog.Error("get group member list cache failed: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 查群信息
	group, err := dao.Group.GetGroupInfoByGroupId(groupId)
	if err != nil {
		zlog.Error("get group info failed: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 解析成员 id 列表
	var memberIds []string
	if err := json.Unmarshal(group.Members, &memberIds); err != nil {
		zlog.Error("unmarshal group members failed: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 查询成员信息并组装返回
	repMemberList := make([]respond.GetGroupMemberListRespond, 0, len(memberIds))
	for _, memberId := range memberIds {
		userInfo, err := dao.User.GetUserInfoByUuid(memberId)
		if err != nil {
			zlog.Error("get user info failed: " + err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}

		repMemberList = append(repMemberList, respond.GetGroupMemberListRespond{
			UserId:   userInfo.Uuid,
			Nickname: userInfo.Nickname,
			Avatar:   userInfo.Avatar,
		})
	}

	// 回填缓存
	jsonData, err := json.Marshal(repMemberList)
	if err != nil {
		zlog.Error("marshal group member list failed: " + err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), 10*time.Minute); err != nil {
			zlog.Error("set group member list cache failed: " + err.Error())
		}
	}

	return "获取群聊成员列表成功", repMemberList, 0
}
