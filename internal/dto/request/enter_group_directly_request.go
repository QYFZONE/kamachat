package request

type EnterGroupDirectlyRequest struct {
	UserId    string `json:"user_id"`
	ContactId string `json:"contact_id"`
}
