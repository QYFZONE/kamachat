package https_server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	v1 "kama_chat_server/api/v1"
	"kama_chat_server/internal/config"
	"kama_chat_server/pkg/ssl"
)

var GE *gin.Engine

func init() {
	GE = gin.Default()

	// 1. 全局中间件（所有请求都经过）
	setupGlobalMiddleware()

	// 2. 静态资源（无需认证）
	setupStaticFiles()

	// 3. 路由分组注册
	setupPublicRoutes() // 公开接口（登录注册）
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
	cfg := config.GetConfig().MainConfig
	GE.Use(ssl.TlsHandler(cfg.Host, cfg.Port))

	// TODO: 如需统一认证，在这里添加 JWT 中间件
	// GE.Use(middleware.JWTAuth())
}

// setupStaticFiles 静态资源服务
func setupStaticFiles() {
	cfg := config.GetConfig()
	GE.Static("/static/avatars", cfg.StaticAvatarPath)
	GE.Static("/static/files", cfg.StaticFilePath)
}

func setupPublicRoutes() {
	auth := GE.Group("/auth")
	{
		auth.POST("/register", v1.Register)     // 注册
		auth.POST("/login", v1.Login)           // 账号密码登录
		auth.POST("/sms/login", v1.SmsLogin)    // 短信登录
		auth.POST("/email/login", v1.EmaiLogin) //邮箱登录
	}
}
