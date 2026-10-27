package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"wfagent/internal/base/dto"
	"wfagent/internal/base/response"
)

func TestLoginSuccess(t *testing.T) {
	router := setupAuthTestRouter()

	recorder := postLogin(router, `{"username":"alice","password":"correct-password"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var body response.Body
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != response.OK.Code {
		t.Fatalf("code = %d, want %d", body.Code, response.OK.Code)
	}
	data, ok := body.Data.(map[string]any)
	if !ok {
		t.Fatalf("data is not object: %#v", body.Data)
	}
	if data["access_token"] == "" || data["refresh_token"] == "" {
		t.Fatalf("missing tokens: %#v", data)
	}
}

func TestLoginFail(t *testing.T) {
	router := setupAuthTestRouter()

	recorder := postLogin(router, `{"username":"alice","password":"wrong-password"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
}

func setupAuthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(fakeService{})
	router.POST("/v1/auth/login", handler.Login)
	return router
}

func postLogin(router *gin.Engine, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)
	return recorder
}

type fakeService struct{}

func (fakeService) Login(req dto.LoginRequest, _ string) (dto.TokenResponse, error) {
	if req.Username != "alice" || req.Password != "correct-password" {
		return dto.TokenResponse{}, ErrInvalidCredentials
	}
	return dto.TokenResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    7200,
		UserInfo: dto.UserInfoResponse{
			UserID:   "user_1",
			Username: req.Username,
			Roles:    []string{"SYS_ADMIN"},
		},
	}, nil
}

func (fakeService) Refresh(dto.RefreshTokenRequest) (dto.TokenResponse, error) {
	return dto.TokenResponse{}, nil
}

func (fakeService) Logout(string) error {
	return nil
}

func (fakeService) Me(string) (dto.UserInfoResponse, error) {
	return dto.UserInfoResponse{}, nil
}

func (fakeService) ChangePassword(string, dto.PasswordChangeRequest) error {
	return nil
}

func (fakeService) ForgotPassword(dto.ForgotPasswordRequest) (dto.ForgotPasswordResponse, error) {
	return dto.ForgotPasswordResponse{}, nil
}

func (fakeService) ResetPassword(dto.ResetPasswordRequest) error {
	return nil
}

func (fakeService) Register(req dto.RegisterRequest) (dto.RegisterResponse, error) {
	return dto.RegisterResponse{UserID: "user_" + req.Username, Username: req.Username, Email: req.Email}, nil
}
