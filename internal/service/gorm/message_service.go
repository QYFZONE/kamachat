package gorm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kama_chat_server/internal/config"
	"kama_chat_server/internal/dao"
	"kama_chat_server/internal/dto/respond"
	myredis "kama_chat_server/internal/service/redis"
	"kama_chat_server/pkg/constants"
	"kama_chat_server/pkg/util/random"
	"kama_chat_server/pkg/zlog"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type messageService struct {
}

var MessageService = new(messageService)

// GetMessageList 获取聊天记录
func (m *messageService) GetMessageList(userOneId, userTwoId string) (string, []respond.GetMessageListRespond, int) {
	if userOneId == "" || userTwoId == "" {
		return "参数错误", nil, -2
	}

	// 单聊消息缓存 key 规范化，避免 A_B 和 B_A 两套缓存
	cacheLeft, cacheRight := userOneId, userTwoId
	if cacheLeft > cacheRight {
		cacheLeft, cacheRight = cacheRight, cacheLeft
	}
	cacheKey := "message:list:" + cacheLeft + ":" + cacheRight

	// 先查缓存
	rspString, err := myredis.GetKey(cacheKey)
	if err == nil {
		var rsp []respond.GetMessageListRespond
		if err := json.Unmarshal([]byte(rspString), &rsp); err == nil {
			return "获取聊天记录成功", rsp, 0
		} else {
			zlog.Error(err.Error())
		}
		// 缓存解析失败，当未命中处理
	} else if !errors.Is(err, redis.Nil) {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 查数据库
	messageList, err := dao.Message.GetMessageListByUserIds(userOneId, userTwoId)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	rspList := make([]respond.GetMessageListRespond, 0, len(messageList))

	for _, message := range messageList {
		rspList = append(rspList, respond.GetMessageListRespond{
			SendId:     message.SendId,
			SendName:   message.SendName,
			SendAvatar: message.SendAvatar,
			ReceiveId:  message.ReceiveId,
			Content:    message.Content,
			Url:        message.Url,
			Type:       message.Type,
			FileType:   message.FileType,
			FileName:   message.FileName,
			FileSize:   message.FileSize,
			CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 回填缓存
	jsonData, err := json.Marshal(rspList)
	if err != nil {
		zlog.Error(err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			zlog.Error(err.Error())
		}
	}

	return "获取聊天记录成功", rspList, 0
}

// GetGroupMessageList 获取群聊消息记录
func (m *messageService) GetGroupMessageList(groupId string) (string, []respond.GetGroupMessageListRespond, int) {
	if groupId == "" {
		return "参数错误", nil, -2
	}

	cacheKey := "message:group_list:" + groupId

	// 先查缓存
	rspString, err := myredis.GetKeyNilIsErr(cacheKey)
	if err == nil {
		var rsp []respond.GetGroupMessageListRespond
		if err := json.Unmarshal([]byte(rspString), &rsp); err == nil {
			return "获取聊天记录成功", rsp, 0
		} else {
			zlog.Error(err.Error())
		}
		// 缓存解析失败，当未命中处理
	} else if !errors.Is(err, redis.Nil) {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 查数据库
	messageList, err := dao.Message.GetGroupMessageListByGroupId(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	rspList := make([]respond.GetGroupMessageListRespond, 0, len(messageList))
	for _, message := range messageList {
		rspList = append(rspList, respond.GetGroupMessageListRespond{
			SendId:     message.SendId,
			SendName:   message.SendName,
			SendAvatar: message.SendAvatar,
			ReceiveId:  message.ReceiveId,
			Content:    message.Content,
			Url:        message.Url,
			Type:       message.Type,
			FileType:   message.FileType,
			FileName:   message.FileName,
			FileSize:   message.FileSize,
			CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 回填缓存
	jsonData, err := json.Marshal(rspList)
	if err != nil {
		zlog.Error(err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			zlog.Error(err.Error())
		}
	}

	return "获取聊天记录成功", rspList, 0
}

// UploadAvatar 上传头像
func (m *messageService) UploadAvatar(c *gin.Context) (string, int) {
	if err := c.Request.ParseMultipartForm(constants.FILE_MAX_SIZE); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	ownerId := c.PostForm("owner_id")
	if ownerId == "" {
		return "参数错误", -2
	}

	mForm := c.Request.MultipartForm
	for key := range mForm.File {
		file, fileHeader, err := c.Request.FormFile(key)
		if err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		defer file.Close()

		zlog.Info(fmt.Sprintf("文件名：%s，文件大小：%d", fileHeader.Filename, fileHeader.Size))

		ext := filepath.Ext(fileHeader.Filename)
		zlog.Info(ext)

		// 头像文件名统一为 avatar_ownerId.xxx
		fileName := "avatar_" + ownerId + ext
		localFileName := filepath.Join(config.GetConfig().StaticAvatarPath, fileName)

		out, err := os.Create(localFileName)
		if err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}

		zlog.Info("完成文件上传")
		return "上传成功", 0
	}

	return "未找到上传文件", -2
}

// UploadFile 上传文件
func (m *messageService) UploadFile(c *gin.Context) (string, string, int) {
	if err := c.Request.ParseMultipartForm(constants.FILE_MAX_SIZE); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, "", -1
	}

	mForm := c.Request.MultipartForm
	if mForm == nil || len(mForm.File) == 0 {
		return "未找到上传文件", "", -2
	}

	for key := range mForm.File {
		file, fileHeader, err := c.Request.FormFile(key)
		if err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, "", -1
		}
		defer file.Close()

		zlog.Info(fmt.Sprintf("文件名：%s，文件大小：%d", fileHeader.Filename, fileHeader.Size))

		// 保留原文件后缀，文件名改成随机名，避免重名覆盖
		ext := filepath.Ext(fileHeader.Filename)
		fileName := "file_" + random.GetNowAndLenRandomString(11) + ext
		localFileName := filepath.Join(config.GetConfig().StaticFilePath, fileName)

		out, err := os.Create(localFileName)
		if err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, "", -1
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, "", -1
		}

		zlog.Info("完成文件上传")
		return "上传成功", "/static/files/" + fileName, 0
	}

	return "未找到上传文件", "", -2
}
