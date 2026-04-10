package ssl

import (
	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure" // 安全中间件库
	"kama_chat_server/pkg/zlog"  // 项目自定义日志
	"strconv"
)

// TlsHandler 返回一个 Gin 中间件函数
// host: 域名，如 "example.com"
// port: HTTPS 端口，如 443
// 完成http 到 https的转换
func TlsHandler(host string, port int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建 secure 中间件实例
		secureMiddleware := secure.New(secure.Options{
			SSLRedirect: true,                            // ← 关键：启用 HTTPS 强制跳转
			SSLHost:     host + ":" + strconv.Itoa(port), // ← 目标地址
		})
		// 处理当前请求
		err := secureMiddleware.Process(c.Writer, c.Request)

		// If there was an error, do not continue.
		if err != nil {
			zlog.Fatal(err.Error())
			return
		}

		c.Next()
	}
}
