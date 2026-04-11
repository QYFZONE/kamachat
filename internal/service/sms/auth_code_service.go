package sms

import (
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"kama_chat_server/internal/config"
	"kama_chat_server/internal/service/redis"
	"kama_chat_server/pkg/constants"
	"kama_chat_server/pkg/util/random"
	"kama_chat_server/pkg/zlog"
	"strconv"
	"sync"
	"time"
)

var (
	smsClient *dysmsapi20170525.Client
	once      sync.Once
)

// createClient 使用AK&SK初始化账号Client
func createClient() (result *dysmsapi20170525.Client, err error) {
	once.Do(func() {
		cfg := &openapi.Config{
			AccessKeyId:     tea.String(config.GetConfig().AccessKeyID),
			AccessKeySecret: tea.String(config.GetConfig().AccessKeySecret),
			Endpoint:        tea.String("dysmsapi.aliyuncs.com"),
		}
		smsClient, err = dysmsapi20170525.NewClient(cfg)
	})
	return smsClient, err
}

// VerificationCode 发送短信验证码
func VerificationCode(telephone string) (string, int) {
	client, err := createClient()
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	key := "auth_code_" + telephone
	code, err := redis.GetKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	if code != "" {
		// 直接返回，验证码还没过期，用户应该去输验证码
		message := "目前还不能发送验证码，请输入已发送的验证码"
		zlog.Error(message)
		return message, -2
	}

	// 验证码过期，重新生成
	code = strconv.Itoa(random.GetRandomInt(6))
	fmt.Println(code)
	err = redis.SetKeyEx(key, code, time.Minute) // 1分钟有效
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
		SignName:      tea.String("阿里云短信测试"),
		TemplateCode:  tea.String("SMS_154950909"), // 短信模板
		PhoneNumbers:  tea.String(telephone),
		TemplateParam: tea.String("{\"code\":\"" + code + "\"}"),
	}

	runtime := &util.RuntimeOptions{}
	// 目前使用的是测试专用签名，签名必须是“阿里云短信测试”，模板code为“SMS_154950909”
	rsp, err := client.SendSmsWithOptions(sendSmsRequest, runtime)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	zlog.Info(*util.ToJSONString(rsp))
	return "验证码发送成功，请及时在对应电话查收短信", 0
}
