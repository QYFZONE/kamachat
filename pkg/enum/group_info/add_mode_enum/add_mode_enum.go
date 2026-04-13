package add_mode_enum

const (
	DIRECT = iota // 0 - 直接加入：任何人无需审核可直接进群
	AUDIT         // 1 - 审核加入：需群主/管理员同意后才能进群
)
