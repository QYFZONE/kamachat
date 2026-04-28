package https_server

import (
	v1 "kama_chat_server/api/v1"
	"kama_chat_server/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var GE *gin.Engine

func init() {
	GE = gin.Default()

	// 1. 全局中间件（所有请求都经过）
	setupGlobalMiddleware()

	// 2. 静态资源（无需认证）
	setupStaticFiles()

	// 3. 路由分组注册
	setupPublicRoutes()  // 公开接口（登录注册）
	setupUserRoutes()    // 用户模块
	setupGroupRoutes()   // 群组模块
	setupContactRoutes() // 联系人模块
	setupSessionRoutes() // 会话模块
	setupMessageRoutes() // 消息模块
	setupWebSocket()     // WebSocket（单独处理）
}

// setupGlobalMiddleware 全局中间件
func setupGlobalMiddleware() {
	// 跨域配置
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	GE.Use(cors.New(corsConfig))

	// HTTPS 强制跳转（HTTP → HTTPS）
	//cfg := config.GetConfig().MainConfig
	//GE.Use(ssl.TlsHandler(cfg.Host, cfg.Port))

	// TODO: 如需统一认证，在这里添加 JWT 中间件
	// GE.Use(middleware.JWTAuth())
}

// setupStaticFiles 静态资源服务
func setupStaticFiles() {
	cfg := config.GetConfig()
	GE.Static("/static/avatars", cfg.StaticAvatarPath)
	GE.Static("/static/files", cfg.StaticFilePath)
}

// setupPublicRoutes 公开接口（无需登录）
func setupPublicRoutes() {
	auth := GE.Group("/auth")
	{
		auth.POST("/register", v1.Register)     // 注册
		auth.POST("/login", v1.Login)           // 账号密码登录
		auth.POST("/sms/login", v1.SmsLogin)    // 短信登录
		auth.POST("/email/login", v1.EmaiLogin) //邮箱登录
	}
	// 短信验证码（公开）
	GE.POST("/sms/send", v1.SendSmsCode) //获取验证码
}

// setupUserRoutes 用户模块 /user/*
func setupUserRoutes() {
	user := GE.Group("/user")
	{
		user.POST("/getUserInfo", v1.GetUserInfo)         // 获取用户信息
		user.POST("/getUserInfoList", v1.GetUserInfoList) // 获取用户列表
		user.POST("/updateUserInfo", v1.UpdateUserInfo)   //更新用户信息
		user.POST("/ableUsers", v1.AbleUsers)             // 启用用户
		user.POST("/disableUsers", v1.DisableUsers)       // 禁用用户
		user.POST("/deleteUsers", v1.DeleteUsers)         // 删除用户
		user.POST("/setAdmin", v1.SetAdmin)               // 设置管理
	}
}

// setupGroupRoutes 群组模块 /group/*
func setupGroupRoutes() {
	group := GE.Group("/group")
	{
		group.POST("/createGroup", v1.CreateGroup)               // 创建群组
		group.POST("/loadMyGroup", v1.LoadMyGroup)               // LoadMyGroup 获取我创建的群聊
		group.POST("/checkGroupAddMode", v1.CheckGroupAddMode)   // 检查群组的加入方式（如是否需要验证、是否允许直接加入）
		group.POST("/enterGroupDirectly", v1.EnterGroupDirectly) // 无需审核，直接进入群组
		group.POST("/leaveGroup", v1.LeaveGroup)                 // 退出群组
		group.POST("/dismissGroup", v1.DismissGroup)             // 解散群组（通常只有群主可操作）
		group.POST("/getGroupInfo", v1.GetGroupInfo)             // 获取单个群组的详细信息
		group.POST("/getGroupInfoList", v1.GetGroupInfoList)     // 批量获取群组信息列表
		group.POST("/deleteGroups", v1.DeleteGroups)             // 批量删除群组（软删除或硬删除）
		group.POST("/setGroupsStatus", v1.SetGroupsStatus)       // 批量设置群组状态（如正常、禁言、封禁等）
		group.POST("/updateGroupInfo", v1.UpdateGroupInfo)       // 更新群组资料（名称、公告、头像等）
		group.POST("/getGroupMemberList", v1.GetGroupMemberList) // 获取群成员列表
		group.POST("/removeGroupMembers", v1.RemoveGroupMembers) // 批量移除群成员（踢人）
	}
}

// setupContactRoutes 联系人模块 /contact/*
func setupContactRoutes() {
	contact := GE.Group("/contact")
	{
		contact.POST("/getUserList", v1.GetUserList)                         // 获取用户联系人列表（好友、群聊等）
		contact.POST("/loadMyJoinedGroup", v1.LoadMyJoinedGroup)             // 获取我加入的群聊列表
		contact.POST("/getContactInfo", v1.GetContactInfo)                   // 获取联系人详情（用户或群聊）
		contact.POST("/deleteContact", v1.DeleteContact)                     // 删除联系人（好友关系）
		contact.POST("/applyContact", v1.ApplyContact)                       // 申请添加联系人（加好友 / 申请加群）
		contact.POST("/getNewContactList", v1.GetNewContactList)             // 获取新的联系人申请列表
		contact.POST("/passContactApply", v1.PassContactApply)               // 通过联系人申请
		contact.POST("/refuseContactApply", v1.RefuseContactApply)           // 拒绝联系人申请
		contact.POST("/blackContact", v1.BlackContact)                       // 拉黑联系人
		contact.POST("/cancelBlackContact", v1.CancelBlackContact)           // 取消拉黑联系人
		contact.POST("/getAddGroupList", v1.GetAddGroupList)                 // 获取新的群聊申请列表
		contact.POST("/blackApply", v1.BlackApply)                           // 拉黑申请人或屏蔽该申请
		contact.POST("/checkOpenSessionAllowed", v1.CheckOpenSessionAllowed) // 检查是否允许打开会话（如是否被拉黑、是否已删除关系等）
	}
}

// setupSessionRoutes 会话模块 /session/*
func setupSessionRoutes() {
	session := GE.Group("/session")
	{
		session.POST("/openSession", v1.OpenSession)                         // 打开会话（进入单聊或群聊会话）
		session.POST("/getUserSessionList", v1.GetUserSessionList)           // 获取用户单聊会话列表
		session.POST("/getGroupSessionList", v1.GetGroupSessionList)         // 获取群聊会话列表
		session.POST("/deleteSession", v1.DeleteSession)                     // 删除会话
		session.POST("/checkOpenSessionAllowed", v1.CheckOpenSessionAllowed) // 检查是否允许打开会话（如是否被拉黑、是否已删除关系等）
	}
}

// setupMessageRoutes 消息模块 /message/*
func setupMessageRoutes() {
	message := GE.Group("/message")
	{
		message.POST("/getMessageList", v1.GetMessageList)           //获取聊天记录
		message.POST("/getGroupMessageList", v1.GetGroupMessageList) //  获取群聊消息记录
		message.POST("/uploadAvatar", v1.UploadAvatar)               // 上传头像
		message.POST("/uploadFile", v1.UploadFile)                   // 上传文件
		//message.POST("/getCurContactListInChatRoom", v1.GetCurContactListInChatRoom)
	}
}

// setupWebSocket WebSocket 连接（单独处理，升级协议）
func setupWebSocket() {
	GE.GET("/wss", v1.WsLogin)
	GE.POST("/wsLogout", v1.WsLogout) // WS 登出
}
