package random

import (
	"math"
	"math/rand"
	"strconv"
	"time"
)

// 生成随机数和带时间戳的随机字符串
// 验证码、用户 ID、订单号或者临时凭证
func GetRandomInt(len int) int {
	return rand.Intn(9*int(math.Pow(10, float64(len-1)))) + int(math.Pow(10, float64(len-1)))
}

func GetNowAndLenRandomString(len int) string {
	return time.Now().Format("20060102") + strconv.Itoa(GetRandomInt(len))
}
