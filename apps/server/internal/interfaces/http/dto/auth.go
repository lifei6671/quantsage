package dto

// LoginRequest 定义登录接口请求。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CurrentUser 定义当前登录用户响应。
type CurrentUser struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	Role        string `json:"role"`
	LastLoginAt string `json:"last_login_at,omitempty"`
}
