package dto

// 本文件定义认证授权接口 AUTH-001~AUTH-006 的请求和响应结构。
type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Captcha    string `json:"captcha"`
	CaptchaKey string `json:"captcha_key"`
}

type TokenResponse struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token,omitempty"`
	TokenType    string           `json:"token_type"`
	ExpiresIn    int              `json:"expires_in"`
	UserInfo     UserInfoResponse `json:"user_info,omitempty"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type PasswordChangeRequest struct {
	OldPassword     string `json:"old_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

type ForgotPasswordRequest struct {
	Account string `json:"account" binding:"required"`
}

type ForgotPasswordResponse struct {
	ResetToken string `json:"reset_token,omitempty"`
	ExpiresIn  int    `json:"expires_in"`
	Message    string `json:"message"`
}

type ResetPasswordRequest struct {
	ResetToken      string `json:"reset_token" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

type RegisterRequest struct {
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
	Phone           string `json:"phone"`
	RealName        string `json:"real_name" binding:"required"`
	InvitationCode  string `json:"invitation_code"`
}

type RegisterResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}
