package gorm

import (
	"encoding/json"
	"errors"
	"fmt"
	redis "github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"kama_chat_server/internal/dao"
	"kama_chat_server/internal/dto/request"
	"kama_chat_server/internal/dto/respond"
	"kama_chat_server/internal/model"
	myredis "kama_chat_server/internal/service/redis"
	"kama_chat_server/internal/service/sms"
	"kama_chat_server/pkg/constants"
	"kama_chat_server/pkg/enum/user_info/user_status_enum"
	"kama_chat_server/pkg/util/random"
	"kama_chat_server/pkg/zlog"
	"net/http"
	"regexp"
	"time"
)

type userInfoService struct{}

var UserInfoService = new(userInfoService)

// dao层加不了校验，在service层加
// checkTelephoneValid 检验电话是否合法 长度为11 只包含数字
func (u *userInfoService) checkTelephoneValid(telephone string) bool {
	pattern := `^1[3-9]\d{9}$`
	match, err := regexp.MatchString(pattern, telephone)
	if err != nil {
		zlog.Error(err.Error())
	}
	return match
}

// checkEmailValid 校验邮箱是否合法
func (u *userInfoService) checkEmailValid(email string) bool {
	pattern := `^[^\s@]+@[^\s@]+\.[^\s@]+$`
	match, err := regexp.MatchString(pattern, email)
	if err != nil {
		regexp.MatchString(pattern, email)
	}
	return match
}

// checkUserIsAdminOrNot 检验用户是否为管理员
func (u *userInfoService) checkUserIsAdminOrNot(user model.UserInfo) int8 {
	return user.IsAdmin
}

// Login登录

func (u *userInfoService) Login(loginReq request.LoginRequest) (string, *respond.LoginRespond, int) {
	password := loginReq.Password
	user, err := dao.User.GetUserInfoByTelephone(loginReq.Telephone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			message := "用户不存在，请注册"
			zlog.Error(message)
			return message, nil, -2
		}
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	if user.Password != password {
		message := "密码不正确，请重试"
		zlog.Error(message)
		return message, nil, -2
	}

	loginRsp := &respond.LoginRespond{
		Uuid:      user.Uuid,
		Telephone: user.Telephone,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Gender:    user.Gender,
		Birthday:  user.Birthday,
		Signature: user.Signature,
		IsAdmin:   user.IsAdmin,
		Status:    user.Status,
	}

	year, month, day := user.CreatedAt.Date()
	loginRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	return "登陆成功", loginRsp, 0
}

// Register 注册，返回(message, register_respond_string, error)
func (u *userInfoService) Register(registerReq request.RegisterRequest) (string, *respond.LoginRespond, int) {
	return "", nil, 0
}
