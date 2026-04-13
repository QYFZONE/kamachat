package group_status_enum

const (
	NORMAL   = iota // 0 - 正常：群组可正常使用，可聊天
	DISABLE         // 1 - 禁用：群组被平台封禁或暂停（违规等）
	DISSOLVE        // 2 - 已解散：群组已被解散，逻辑上已不存在
)
