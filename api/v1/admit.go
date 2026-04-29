package v1

import (
	"kama_chat_server/internal/dto/request"
	"kama_chat_server/internal/service/gorm"
	"kama_chat_server/pkg/constants"
	"kama_chat_server/pkg/zlog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUserInfoList 获取用户列表
func GetUserInfoList(c *gin.Context) {
	var req request.GetUserInfoListRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, userList, ret := gorm.AdmitService.GetUserInfoList(req.OwnerId)
	JsonBack(c, message, ret, userList)
}

func AbleUsers(c *gin.Context) {
	var req request.AbleUsersRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.AdmitService.AbleUsers(req.UuidList)
	JsonBack(c, message, ret, nil)
}

func DisableUsers(c *gin.Context) {
	var req request.DisAbleUsersRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.AdmitService.DisableUsers(req.UuidList)
	JsonBack(c, message, ret, nil)
}

func GetGroupInfoList(c *gin.Context) {
	message, groupList, ret := gorm.AdmitService.GetGroupInfoList()
	JsonBack(c, message, ret, groupList)
}

func AbleGroups(c *gin.Context) {
	var req request.AbleGroupsRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.AdmitService.AbleGroups(req.UuidList)
	JsonBack(c, message, ret, nil)
}

func DisAbleGroups(c *gin.Context) {
	var req request.DisAbleGroupsRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.AdmitService.DisableGroups(req.UuidList)
	JsonBack(c, message, ret, nil)
}
