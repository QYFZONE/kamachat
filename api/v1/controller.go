package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func JsonBack(c *gin.Context, message string, ret int, data interface{}) {

	if ret == 0 {
		//ret 为 0 正常完成流程
		if data != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": message,
				"data":    data,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": message,
			})
		}
	} else if ret == -2 {
		//ret 为 -2 客户端错误
		c.JSON(http.StatusOK, gin.H{
			"code":    400,
			"message": message,
		})
	} else if ret == -1 {
		// ret 为 -1 服务器错误
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": message,
		})
	}
}
