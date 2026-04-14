package request

type UpdateGroupInfoRequest struct {
	UserId  string `json:"user_id"`
	Uuid    string `json:"uuid"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
	AddMode int8   `json:"add_mode"`
	Notice  string `json:"notice"`
}
