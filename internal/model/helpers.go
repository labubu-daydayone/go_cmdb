package model

// UPtr 将 int 转换为 *int
func UPtr(i int) *int {
	return &i
}

// UVal 安全地从 *int 获取值，如果为 nil 返回 0
func UVal(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
