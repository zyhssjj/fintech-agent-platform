package auth

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"wfagent/internal/base/dto"
	"wfagent/internal/base/response"
)

// Handler carries authentication HTTP endpoints.
type Handler struct {
	service service
}

type service interface {
	Login(req dto.LoginRequest, clientIP string) (dto.TokenResponse, error)
	Refresh(req dto.RefreshTokenRequest) (dto.TokenResponse, error)
	Logout(accessToken string) error
	Me(username string) (dto.UserInfoResponse, error)
	ChangePassword(username string, req dto.PasswordChangeRequest) error
	ForgotPassword(req dto.ForgotPasswordRequest) (dto.ForgotPasswordResponse, error)
	ResetPassword(req dto.ResetPasswordRequest) error
	Register(req dto.RegisterRequest) (dto.RegisterResponse, error)
}

func NewHandler(service service) *Handler {
	return &Handler{service: service}
}

// Login godoc
// @Summary User login
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.LoginRequest true "Login credentials"
// @Success 200 {object} response.Body{data=dto.TokenResponse}
// @Failure 400 {object} response.Body
// @Failure 401 {object} response.Body
// @Router /v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	token, err := h.service.Login(req, c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(c, response.ErrUnauthorized, err.Error())
			return
		}
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.Success(c, token)
}

// Refresh godoc
// @Summary Refresh access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} response.Body{data=dto.TokenResponse}
// @Failure 400 {object} response.Body
// @Failure 401 {object} response.Body
// @Router /v1/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	token, err := h.service.Refresh(req)
	if err != nil {
		response.Error(c, response.ErrUnauthorized, err.Error())
		return
	}
	response.Success(c, token)
}

// Logout godoc
// @Summary Logout current user
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=map[string]string}
// @Failure 401 {object} response.Body
// @Router /v1/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if err := h.service.Logout(raw); err != nil {
		response.Error(c, response.ErrUnauthorized, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "logged_out"})
}

// Me godoc
// @Summary Get current user profile
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=dto.UserInfoResponse}
// @Failure 401 {object} response.Body
// @Router /v1/auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.GetString("username"))
	if err != nil {
		response.Error(c, response.ErrUnauthorized, err.Error())
		return
	}
	response.Success(c, user)
}

// ChangePassword godoc
// @Summary Change current user password
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.PasswordChangeRequest true "Password change request"
// @Success 200 {object} response.Body{data=map[string]string}
// @Failure 400 {object} response.Body
// @Failure 401 {object} response.Body
// @Router /v1/auth/password [put]
func (h *Handler) ChangePassword(c *gin.Context) {
	var req dto.PasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.service.ChangePassword(c.GetString("username"), req); err != nil {
		response.Error(c, authErrorCode(err), err.Error())
		return
	}
	response.Success(c, gin.H{"status": "changed"})
}

// ForgotPassword godoc
// @Summary Request password reset
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.ForgotPasswordRequest true "Forgot password request"
// @Success 200 {object} response.Body{data=dto.ForgotPasswordResponse}
// @Failure 400 {object} response.Body
// @Router /v1/auth/password/forgot [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	result, err := h.service.ForgotPassword(req)
	if err != nil {
		response.Error(c, authErrorCode(err), err.Error())
		return
	}
	response.Success(c, result)
}

// ResetPassword godoc
// @Summary Reset password with reset token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.ResetPasswordRequest true "Reset password request"
// @Success 200 {object} response.Body{data=map[string]string}
// @Failure 400 {object} response.Body
// @Failure 401 {object} response.Body
// @Router /v1/auth/password/reset [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.service.ResetPassword(req); err != nil {
		response.Error(c, authErrorCode(err), err.Error())
		return
	}
	response.Success(c, gin.H{"status": "reset"})
}

// Register godoc
// @Summary Register user
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.RegisterRequest true "Register request"
// @Success 201 {object} response.Body{data=dto.RegisterResponse}
// @Failure 400 {object} response.Body
// @Router /v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	result, err := h.service.Register(req)
	if err != nil {
		response.Error(c, authErrorCode(err), err.Error())
		return
	}
	response.Created(c, result)
}

func authErrorCode(err error) response.ErrorCode {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return response.ErrUnauthorized
	case errors.Is(err, ErrAlreadyExists):
		return response.ErrAlreadyExists
	case errors.Is(err, ErrValidation):
		return response.ErrValidation
	default:
		return response.ErrInternal
	}
}
