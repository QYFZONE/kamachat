package gorm

import (
	"kama_chat_server/internal/dao"
	"kama_chat_server/internal/dto/respond"
	myredis "kama_chat_server/internal/service/redis"
	"kama_chat_server/pkg/constants"
	"kama_chat_server/pkg/zlog"
	"time"
)

type admitService struct{}

var AdmitService = new(admitService)

// GetUserInfoList 获取用户列表除了ownerId之外 - 管理员
// 管理员少，而且如果用户更改了，那么管理员会一直频繁删除redis，更新redis，比较麻烦，所以管理员暂时不使用redis缓存
func (a *admitService) GetUserInfoList(ownerId string) (string, []respond.GetUserListRespond, int) {
	users, err := dao.User.GetUsersExcept(ownerId)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	rsp := make([]respond.GetUserListRespond, 0, len(users))
	for _, user := range users {
		rsp = append(rsp, respond.GetUserListRespond{
			Uuid:      user.Uuid,
			Telephone: user.Telephone,
			Nickname:  user.Nickname,
			Status:    user.Status,
			IsAdmin:   user.IsAdmin,
			IsDeleted: user.DeletedAt.Valid,
		})
	}

	return "获取用户列表成功", rsp, 0
}

func (a *admitService) AbleUsers(uuidlist []string) (string, int) {
	if len(uuidlist) == 0 {
		return "提交列表为空", -2
	}
	if err := dao.User.AbleUsersByUserIdList(uuidlist); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, uuid := range uuidlist {
		a.deleteUserCache(uuid)
	}
	return "启用用户成功", 0
}

func (a *admitService) DisableUsers(uuidlist []string) (string, int) {
	if len(uuidlist) == 0 {
		return "提交列表为空", -2
	}

	if err := dao.User.DisableUsersByUserIdList(uuidlist); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, uuid := range uuidlist {
		a.deleteUserCache(uuid)
	}
	return "禁用用户成功", 0
}

func (a *admitService) GetGroupInfoList() (string, []respond.LoadAllGroupRespond, int) {
	groupInfoList, err := dao.Group.GetAllGroupInfo()
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	var resp []respond.LoadAllGroupRespond

	for _, groupInfo := range groupInfoList {

		resp = append(resp, respond.LoadAllGroupRespond{
			GroupId:   groupInfo.Uuid,
			GroupName: groupInfo.Name,
			Avatar:    groupInfo.Avatar,
			OwnerId:   groupInfo.OwnerId,
			MemberCnt: groupInfo.MemberCnt,
			Status:    groupInfo.Status,
			IsDeleted: groupInfo.DeletedAt.Valid,
		})
	}

	if len(resp) == 0 {
		return "目前没有群聊", nil, 0
	}

	return "获取成功", resp, 0
}

func (a *admitService) AbleGroups(uuidlist []string) (string, int) {
	if len(uuidlist) == 0 {
		return "提交列表为空", -2
	}
	if err := dao.Group.AbleGroupsByGroupIdList(uuidlist, time.Now()); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, uuid := range uuidlist {
		a.deleteGroupCache(uuid)
	}
	return "启用群聊成功", 0
}

func (a *admitService) DisableGroups(uuidlist []string) (string, int) {
	if len(uuidlist) == 0 {
		return "提交列表为空", -2
	}
	if err := dao.Group.DisableGroupsByGroupIdList(uuidlist, time.Now()); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, uuid := range uuidlist {
		a.deleteGroupCache(uuid)
	}
	return "禁用群聊成功", 0
}

func (a *admitService) deleteUserCache(uuid string) {
	if err := myredis.DelKey("user_info_" + uuid); err != nil {
		zlog.Error(err.Error())
	}
	if err := myredis.DelKey("user:info:" + uuid); err != nil {
		zlog.Error(err.Error())
	}
}

func (a *admitService) deleteGroupCache(uuid string) {
	if err := myredis.DelKey("group:info:" + uuid); err != nil {
		zlog.Error(err.Error())
	}
	if err := myredis.DelKey("group:member_list:" + uuid); err != nil {
		zlog.Error(err.Error())
	}
}
