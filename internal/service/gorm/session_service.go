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
	"kama_chat_server/pkg/enum/group_info/group_status_enum"
	"kama_chat_server/pkg/enum/user_info/user_status_enum"
	"kama_chat_server/pkg/util/random"
	"kama_chat_server/pkg/zlog"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type sessionService struct {
}

var SessionService = new(sessionService)

// CreateSession 创建会话
func (s *sessionService) CreateSession(req request.CreateSessionRequest) (string, string, int) {
	if req.SendId == "" || req.ReceiveId == "" {
		return "参数错误", "", -2
	}

	// 校验发送方用户是否存在
	_, err := dao.User.GetUserInfoByUuid(req.SendId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "发送方用户不存在", "", -2
		}
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, "", -1
	}
	now := time.Now()

	session := &model.Session{
		Uuid:      fmt.Sprintf("S%s", random.GetNowAndLenRandomString(11)),
		SendId:    req.SendId,
		ReceiveId: req.ReceiveId,
		CreatedAt: now,
	}

	// 单聊会话
	if req.ReceiveId[0] == 'U' {
		receiveUser, err := dao.User.GetUserInfoByUuid(req.ReceiveId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "接收方用户不存在", "", -2
			}
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, "", -1
		}

		if receiveUser.Status == user_status_enum.DISABLE {
			zlog.Error("该用户被禁用了")
			return "该用户被禁用了", "", -2
		}

		session.ReceiveName = receiveUser.Nickname
		session.Avatar = receiveUser.Avatar
	} else {
		// 群聊会话
		receiveGroup, err := dao.Group.GetGroupInfoByGroupId(req.ReceiveId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "群聊不存在", "", -2
			}
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, "", -1
		}

		if receiveGroup.Status == group_status_enum.DISABLE {
			zlog.Error("该群聊被禁用了")
			return "该群聊被禁用了", "", -2
		}

		session.ReceiveName = receiveGroup.Name
		session.Avatar = receiveGroup.Avatar
	}

	if err := dao.Session.CreateSession(session); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, "", -1
	}

	// 回填单条会话缓存
	cacheKey := "session:pair:" + req.SendId + ":" + req.ReceiveId
	jsonData, err := json.Marshal(session)
	if err != nil {
		zlog.Error(err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			zlog.Error(err.Error())
		}
	}

	// 删除会话列表缓存
	if req.ReceiveId[0] == 'G' {
		if err := myredis.DelKey("user:group_session_list:" + req.SendId); err != nil {
			zlog.Error(err.Error())
		}
	} else {
		if err := myredis.DelKey("user:session_list:" + req.SendId); err != nil {
			zlog.Error(err.Error())
		}
	}

	return "会话创建成功", session.Uuid, 0
}

// OpenSession 打开会话
func (s *sessionService) OpenSession(req request.OpenSessionRequest) (string, string, int) {
	cacheKey := "session:pair:" + req.SendId + ":" + req.ReceiveId

	// 先查缓存
	// 先查缓存
	rspString, err := myredis.GetKeyNilIsErr(cacheKey)
	if err == nil {
		var session model.Session
		if err := json.Unmarshal([]byte(rspString), &session); err == nil {
			return "打开会话成功", session.Uuid, 0
		} else {
			zlog.Error(err.Error())
		}
	} else if !errors.Is(err, redis.Nil) {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, "", -1
	}

	session, err := dao.Session.GetSessionBySendIdAndReceiveId(req.SendId, req.ReceiveId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zlog.Info("会话没有找到，将新建会话")
			createReq := request.CreateSessionRequest{
				SendId:    req.SendId,
				ReceiveId: req.ReceiveId,
			}
			return s.CreateSession(createReq)
		}
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, "", -1
	}
	return "打开会话成功", session.Uuid, 0
}

// GetUserSessionList 获取用户会话列表
func (s *sessionService) GetUserSessionList(ownerId string) (string, []respond.UserSessionListRespond, int) {
	cacheKey := "user:session_list:" + ownerId

	//先查缓存
	repString, err := myredis.GetKeyNilIsErr(cacheKey)
	if err == nil {
		var rsp []respond.UserSessionListRespond
		if err := json.Unmarshal([]byte(repString), &rsp); err != nil {
			zlog.Error(err.Error())
		} else {
			return "获取成功", rsp, 0
		}
	} else if !errors.Is(err, redis.Nil) {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 缓存未命中，查数据库
	sessionList, err := dao.Session.GetSessionListBySendId(ownerId)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 只保留单聊会话
	sessionListRsp := make([]respond.UserSessionListRespond, 0, len(sessionList))
	for i := 0; i < len(sessionList); i++ {
		if sessionList[i].ReceiveId != "" && sessionList[i].ReceiveId[0] == 'U' {
			sessionListRsp = append(sessionListRsp, respond.UserSessionListRespond{
				SessionId: sessionList[i].Uuid,
				Avatar:    sessionList[i].Avatar,
				UserId:    sessionList[i].ReceiveId,
				Username:  sessionList[i].ReceiveName,
			})
		}
	}
	if len(sessionListRsp) == 0 {
		zlog.Info("未创建用户会话")
		return "未创建用户会话", nil, 0
	}

	// 回填缓存
	jsonData, err := json.Marshal(sessionListRsp)
	if err != nil {
		zlog.Error(err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			zlog.Error(err.Error())
		}
	}

	return "获取成功", sessionListRsp, 0
}

func (s *sessionService) GetGroupSessionList(ownerId string) (string, []respond.GroupSessionListRespond, int) {
	cacheKey := "user:group_session_list:" + ownerId
	// 先查缓存

	repString, err := myredis.GetKeyNilIsErr(cacheKey)
	if err == nil {
		var rsp []respond.GroupSessionListRespond
		if err := json.Unmarshal([]byte(repString), &rsp); err != nil {
			zlog.Error(err.Error())
		} else {
			return "获取成功", rsp, 0
		}
	} else if !errors.Is(err, redis.Nil) {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 查数据库
	sessionList, err := dao.Session.GetSessionListBySendId(ownerId)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	sessionListRsp := make([]respond.GroupSessionListRespond, 0, len(sessionList))
	for i := 0; i < len(sessionList); i++ {
		if sessionList[i].ReceiveId != "" && sessionList[i].ReceiveId[0] == 'G' {
			sessionListRsp = append(sessionListRsp, respond.GroupSessionListRespond{
				SessionId: sessionList[i].Uuid,
				Avatar:    sessionList[i].Avatar,
				GroupId:   sessionList[i].ReceiveId,
				GroupName: sessionList[i].ReceiveName,
			})
		}
	}
	if len(sessionListRsp) == 0 {
		zlog.Info("未创建群聊会话")
		return "未创建群聊会话", nil, 0
	}

	jsonData, err := json.Marshal(sessionListRsp)
	if err != nil {
		zlog.Error(err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			zlog.Error(err.Error())
		}
	}

	return "获取成功", sessionListRsp, 0
}

// DeleteSession 删除会话
func (s *sessionService) DeleteSession(ownerId, sessionId string) (string, int) {
	if ownerId == "" || sessionId == "" {
		return "参数错误", -2
	}
	// 查询会话
	session, err := dao.Session.GetSessionByUuid(sessionId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "会话不存在", -2
		}
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	now := time.Now()

	// 软删除会话
	if err := dao.Session.SoftDeleteSessionByUuid(sessionId, now); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 删除单条会话缓存
	if err := myredis.DelKey("session:pair:" + session.SendId + ":" + session.ReceiveId); err != nil {
		zlog.Error(err.Error())
	}

	// 删除会话列表缓存
	if session.ReceiveId != "" && session.ReceiveId[0] == 'G' {
		if err := myredis.DelKey("user:group_session_list:" + ownerId); err != nil {
			zlog.Error(err.Error())
		}
	} else {
		if err := myredis.DelKey("user:session_list:" + ownerId); err != nil {
			zlog.Error(err.Error())
		}
	}

	return "删除成功", 0
}

// CheckOpenSessionAllowed 检查是否允许发起会话
func (s *sessionService) CheckOpenSessionAllowed(sendId, receiveId string) (string, bool, int) {
	if sendId == "" || receiveId == "" {
		return "参数错误", false, -2
	}
	contact, err := dao.Contact.GetUserContactByUserIdAndContactId(sendId, receiveId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "联系人关系不存在", false, -2
		}
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, false, -1
	}

	if contact.Status == contact_status_enum.BE_BLACK {
		return "已被对方拉黑，无法发起会话", false, -2
	} else if contact.Status == contact_status_enum.BLACK {
		return "已拉黑对方，先解除拉黑状态才能发起会话", false, -2
	}

	// 单聊
	if receiveId[0] == 'U' {
		user, err := dao.User.GetUserInfoByUuid(receiveId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "用户不存在", false, -2
			}
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, false, -1
		}
		if user.Status == user_status_enum.DISABLE {
			zlog.Info("对方已被禁用，无法发起会话")
			return "对方已被禁用，无法发起会话", false, -2
		}
	} else {
		// 群聊
		group, err := dao.Group.GetGroupInfoByGroupId(receiveId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "群聊不存在", false, -2
			}
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, false, -1
		}
		if group.Status == group_status_enum.DISABLE {
			zlog.Info("对方已被禁用，无法发起会话")
			return "对方已被禁用，无法发起会话", false, -2
		}
	}

	return "可以发起会话", true, 0
}
