package gorm

import (
	"encoding/json"
	"errors"
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
	"regexp"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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
		zlog.Error(err.Error())
	}
	return match
}

// checkUserIsAdminOrNot 检验用户是否为管理员
func (u *userInfoService) checkUserIsAdminOrNot(user model.UserInfo) int8 {
	return user.IsAdmin
}

// Login登录

func (u *userInfoService) Login(loginReq request.LoginRequest) (string, *respond.LoginRespond, int) {
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
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password))
	if err != nil {
		zlog.Error("密码错误: " + loginReq.Telephone)
		return "密码不正确，请重试", nil, -2
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
		CreatedAt: user.CreatedAt.Format("2006.01.02"),
	}

	return "登陆成功", loginRsp, 0
}

// SmsLogin 验证码登录
func (u *userInfoService) SmsLogin(req request.SmsLoginRequest) (string, *respond.LoginRespond, int) {
	user, err := dao.User.GetUserInfoByTelephone(req.Telephone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			message := "用户不存在，请注册"
			zlog.Error(message)
			return message, nil, -2
		}
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	key := "auth_code_" + req.Telephone
	code, err := myredis.GetKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	if code != req.SmsCode {
		message := "验证码不正确，请重试"
		zlog.Error(message)
		return message, nil, -2
	} else {
		if err := myredis.DelKey(key); err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
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
		CreatedAt: user.CreatedAt.Format("2006.01.02"),
	}
	return "登陆成功", loginRsp, 0
}

// SendSmsCode 发送短信验证码 - 验证码登录
func (u *userInfoService) SendSmsCode(telephone string) (string, int) {
	return sms.VerificationCode(telephone)
}

// checkTelephoneExist 检查手机号是否存在
func (u *userInfoService) checkTelephoneExist(telephone string) (string, int) {
	_, err := dao.User.GetUserInfoByTelephone(telephone)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		zlog.Info("该电话不存在，可以注册")
		return "", 0
	}
	zlog.Info("该电话已经存在，注册失败")
	return "该电话已经存在，注册失败", -2
}

// Register 注册，返回(message, register_respond_string, error)
func (u *userInfoService) Register(registerReq request.RegisterRequest) (string, *respond.RegisterRespond, int) {
	key := "auth_code_" + registerReq.Telephone
	code, err := myredis.GetKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	if code != registerReq.SmsCode {
		message := "验证码不正确，请重试"
		zlog.Error(message)
		return message, nil, -2
	} else {
		if err := myredis.DelKey(key); err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	// 不用校验手机号，前端校验
	// 判断电话是否已经被注册过了
	message, ret := u.checkTelephoneExist(registerReq.Telephone)
	if ret != 0 {
		return message, nil, ret
	}
	//密码加密存储
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(registerReq.Password), bcrypt.DefaultCost)
	// 新用户信息
	newUser := model.UserInfo{
		Uuid:      "U" + random.GetNowAndLenRandomString(11),
		Telephone: registerReq.Telephone,
		Password:  string(hashedPassword),
		Nickname:  registerReq.Nickname,
		Avatar:    "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png",
		CreatedAt: time.Now(),
		Status:    user_status_enum.NORMAL,
	}

	newUser.IsAdmin = u.checkUserIsAdminOrNot(newUser)
	err = dao.User.CreateNewUser(&newUser)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	registerRsp := &respond.RegisterRespond{
		Uuid:      newUser.Uuid,
		Telephone: newUser.Telephone,
		Nickname:  newUser.Nickname,
		Email:     newUser.Email,
		Avatar:    newUser.Avatar,
		Gender:    newUser.Gender,
		Birthday:  newUser.Birthday,
		Signature: newUser.Signature,
		IsAdmin:   newUser.IsAdmin,
		Status:    newUser.Status,
		CreatedAt: newUser.CreatedAt.Format("2006.01.02"),
	}
	return "注册成功", registerRsp, 0
}

// GetUserInfo 获取用户信息
func (u *userInfoService) GetUserInfo(uuid string) (string, *respond.GetUserInfoRespond, int) {
	// redis
	zlog.Info(uuid)
	cacheKey := "user_info_" + uuid
	repString, err := myredis.GetKeyNilIsErr(cacheKey)

	if err == nil {
		//在redis命中 解析
		var rep respond.GetUserInfoRespond
		if err := json.Unmarshal([]byte(repString), &rep); err != nil {
			zlog.Error("缓存数据解析失败: " + err.Error())
			// 解析失败，继续查数据库（视为缓存未命中）
		} else {
			return "获取用户信息成功", &rep, 0
		}
	} else if !errors.Is(err, redis.Nil) {
		// 非 Nil 错误（连接失败等）
		zlog.Error("Redis 查询异常: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 3. 缓存未命中，查数据库
	user, err := dao.User.GetUserInfoByUuid(uuid)
	if err != nil {
		zlog.Error("数据库查询失败: " + err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	// 5. 构造响应
	rsp := respond.GetUserInfoRespond{
		Uuid:      user.Uuid,
		Telephone: user.Telephone,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Birthday:  user.Birthday,
		Email:     user.Email,
		Gender:    user.Gender,
		Signature: user.Signature,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		IsAdmin:   user.IsAdmin,
		Status:    user.Status,
	}
	jsonData, err := json.Marshal(rsp)
	if err != nil {
		zlog.Error("JSON 序列化失败: " + err.Error())
	} else {
		if err := myredis.SetKeyEx(cacheKey, string(jsonData), 10*time.Minute); err != nil {
			zlog.Error("Redis 回填失败: " + err.Error())
			// 不影响主流程，仅记录日志
		}
	}

	return "获取用户信息成功", &rsp, 0
}

// GetUserInfoList 获取用户列表除了ownerId之外 - 管理员
// 管理员少，而且如果用户更改了，那么管理员会一直频繁删除redis，更新redis，比较麻烦，所以管理员暂时不使用redis缓存
func (u *userInfoService) GetUserInfoList(ownerId string) (string, []respond.GetUserListRespond, int) {
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

// UpdateUserInfo 修改用户信息
func (u *userInfoService) UpdateUserInfo(updateReq request.UpdateUserInfoRequest) (string, int) {
	user, err := dao.User.GetUserInfoByUuid(updateReq.Uuid)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	//更新
	if updateReq.Email != "" {
		user.Email = updateReq.Email
	}
	if updateReq.Nickname != "" {
		user.Nickname = updateReq.Nickname
	}
	if updateReq.Birthday != "" {
		user.Birthday = updateReq.Birthday
	}
	if updateReq.Signature != "" {
		user.Signature = updateReq.Signature
	}
	if updateReq.Avatar != "" {
		user.Avatar = updateReq.Avatar
	}

	if err := dao.User.SaveUser(user); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	cacheKey := "user_info_" + updateReq.Uuid
	if err := myredis.DelKey(cacheKey); err != nil {
		zlog.Error("删除缓存失败: " + err.Error())
		// 不影响主流程，但会导致缓存短暂不一致
	}

	return "修改用户信息成功", 0
}
