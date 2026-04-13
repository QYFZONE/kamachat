package contact_status_enum

const (
	NORMAL         = iota // 0 - 正常状态（好友关系正常，无任何异常）
	BE_BLACK              // 1 - 被对方拉黑（我是受害者）
	BLACK                 // 2 - 拉黑对方（我是主动方）
	BE_DELETE             // 3 - 被对方删除（我是受害者）
	DELETE                // 4 - 删除对方（我是主动方）
	SILENCE               // 5 - 被禁言（在群组中无法发言）
	QUIT_GROUP            // 6 - 主动退出群组
	KICK_OUT_GROUP        // 7 - 被踢出群组（被动离开）
)
