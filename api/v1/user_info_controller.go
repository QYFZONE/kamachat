package v1

import (
	"fmt"
	_ "fmt"
	"github.com/gin-gonic/gin"
	"kama_chat_server/internal/dto/request"
	"kama_chat_server/internal/service/gorm"
	"kama_chat_server/pkg/constants"
	"kama_chat_server/pkg/zlog"
	"net/http"
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

func SmsLogin(context *gin.Context) {

}

func EmaiLogin(context *gin.Context) {

}
