package v1

import (
	"fmt"
	_ "fmt"
	"kama_chat_server/internal/dto/request"
	"kama_chat_server/internal/service/gorm"
	"kama_chat_server/pkg/constants"
	"kama_chat_server/pkg/zlog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register 注册
func Register(c *gin.Context) {
	var registerReq request.RegisterRequest
	if err := c.BindJSON(&registerReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	fmt.Println(registerReq)
	message, userInfo, ret := gorm.UserInfoService.Register(registerReq)
	JsonBack(c, message, ret, userInfo)
}

// Login 登录
func Login(c *gin.Context) {
	var loginReq request.LoginRequest
	if err := c.Bind(&loginReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, userInfo, ret := gorm.UserInfoService.Login(loginReq)
	JsonBack(c, message, ret, userInfo)
}

func SmsLogin(c *gin.Context) {
	var req request.SmsLoginRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}

	message, userInfo, ret := gorm.UserInfoService.SmsLogin(req)
	JsonBack(c, message, ret, userInfo)
}

func EmaiLogin(c *gin.Context) {

}

// GetUserInfo 获取用户信息
func GetUserInfo(c *gin.Context) {
	var req request.GetUserInfoRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, UserInfo, ret := gorm.UserInfoService.GetUserInfo(req.Uuid)
	JsonBack(c, message, ret, UserInfo)
}

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
	message, userList, ret := gorm.UserInfoService.GetUserInfoList(req.OwnerId)
	JsonBack(c, message, ret, userList)
}

// UpdateUserInfo 修改用户信息
func UpdateUserInfo(c *gin.Context) {
	var req request.UpdateUserInfoRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserInfoService.UpdateUserInfo(req)
	JsonBack(c, message, ret, nil)
}

func AbleUsers(c *gin.Context) {

}

func DisableUsers(c *gin.Context) {

}

func DeleteUsers(c *gin.Context) {

}

func SetAdmin(c *gin.Context) {

}
