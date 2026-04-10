package request

// 登录请求 包含 电话和密码
type LoginRequest struct {
	Telephone string `json:"telephone"`
	Password  string `json:"password"`
}
