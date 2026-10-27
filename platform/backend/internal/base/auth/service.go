package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"wfagent/internal/base/config"
	"wfagent/internal/base/database"
	"wfagent/internal/base/dto"
	"wfagent/internal/base/security"
)

var ErrInvalidCredentials = errors.New("invalid username or password")
var ErrValidation = errors.New("validation failed")
var ErrAlreadyExists = errors.New("user already exists")

const passwordResetTTL = 15 * time.Minute

// Service contains authentication business logic. Config is read through config.C by lower layers.
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Login(req dto.LoginRequest, clientIP string) (dto.TokenResponse, error) {
	user, err := s.loginUser(req, clientIP)
	if err != nil {
		return dto.TokenResponse{}, err
	}
	access, expiresIn, err := security.SignAccessToken(user)
	if err != nil {
		return dto.TokenResponse{}, err
	}
	refresh, _, err := security.SignRefreshToken(user)
	if err != nil {
		return dto.TokenResponse{}, err
	}
	return dto.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		UserInfo:     user,
	}, nil
}

func (s *Service) Refresh(req dto.RefreshTokenRequest) (dto.TokenResponse, error) {
	claims, err := security.ParseRefreshToken(req.RefreshToken)
	if err != nil || claims.TokenType != "refresh" {
		return dto.TokenResponse{}, errors.New("invalid refresh token")
	}
	user, err := s.userInfoByUsername(claims.Username)
	if err != nil {
		return dto.TokenResponse{}, err
	}
	access, expiresIn, err := security.SignAccessToken(user)
	if err != nil {
		return dto.TokenResponse{}, err
	}
	return dto.TokenResponse{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}, nil
}

func (s *Service) Me(username string) (dto.UserInfoResponse, error) {
	return s.userInfoByUsername(username)
}

func (s *Service) Logout(accessToken string) error {
	if accessToken == "" {
		return errors.New("missing access token")
	}
	claims, err := security.ParseAccessToken(accessToken)
	if err != nil || claims.TokenType != "access" {
		return errors.New("invalid access token")
	}
	if claims.ExpiresAt == nil {
		return errors.New("invalid access token expiry")
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return errors.New("access token expired")
	}
	return security.BlacklistToken(accessToken, ttl)
}

func (s *Service) ChangePassword(username string, req dto.PasswordChangeRequest) error {
	if err := validatePasswordPair(req.NewPassword, req.ConfirmPassword); err != nil {
		return err
	}
	if username == "" {
		return ErrInvalidCredentials
	}
	if !config.C.Database.Enabled || s.db == nil {
		return nil
	}

	var user database.SysUser
	err := s.db.Where("username = ? AND is_deleted = false", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.Model(&database.SysUser{}).
		Where("id = ?", user.ID).
		Update("password", string(hashedPassword)).Error
}

func (s *Service) ForgotPassword(req dto.ForgotPasswordRequest) (dto.ForgotPasswordResponse, error) {
	account := strings.TrimSpace(req.Account)
	if account == "" {
		return dto.ForgotPasswordResponse{}, ErrValidation
	}

	username := account
	if config.C.Database.Enabled && s.db != nil {
		user, err := s.findActiveUserByAccount(account, false)
		if err != nil {
			return dto.ForgotPasswordResponse{}, err
		}
		username = user.Username
	}

	token, err := signPasswordResetToken(username, time.Now().Add(passwordResetTTL))
	if err != nil {
		return dto.ForgotPasswordResponse{}, err
	}
	return dto.ForgotPasswordResponse{
		ResetToken: token,
		ExpiresIn:  int(passwordResetTTL.Seconds()),
		Message:    "reset token generated",
	}, nil
}

func (s *Service) ResetPassword(req dto.ResetPasswordRequest) error {
	if err := validatePasswordPair(req.NewPassword, req.ConfirmPassword); err != nil {
		return err
	}
	username, err := parsePasswordResetToken(req.ResetToken)
	if err != nil {
		return ErrInvalidCredentials
	}
	if !config.C.Database.Enabled || s.db == nil {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	result := s.db.Model(&database.SysUser{}).
		Where("username = ? AND is_deleted = false", username).
		Update("password", string(hashedPassword))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInvalidCredentials
	}
	return nil
}

func (s *Service) Register(req dto.RegisterRequest) (dto.RegisterResponse, error) {
	if err := validatePasswordPair(req.Password, req.ConfirmPassword); err != nil {
		return dto.RegisterResponse{}, err
	}
	if !config.C.Database.Enabled || s.db == nil {
		return dto.RegisterResponse{
			UserID:   "user_" + req.Username,
			Username: req.Username,
			Email:    req.Email,
		}, nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return dto.RegisterResponse{}, err
	}

	user := database.SysUser{
		Username: strings.TrimSpace(req.Username),
		Password: string(hashedPassword),
		RealName: strings.TrimSpace(req.RealName),
		Email:    strings.TrimSpace(req.Email),
		Phone:    strings.TrimSpace(req.Phone),
		Status:   "active",
	}
	if user.Username == "" || user.RealName == "" || user.Email == "" {
		return dto.RegisterResponse{}, ErrValidation
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&database.SysUser{}).
			Where("is_deleted = false AND (username = ? OR email = ?)", user.Username, user.Email).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrAlreadyExists
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		var guestRole database.SysRole
		if err := tx.Where("role_code = ? AND is_deleted = false", "GUEST").First(&guestRole).Error; err == nil {
			return tx.Create(&database.SysUserRole{UserID: user.ID, RoleID: guestRole.ID}).Error
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return nil
	})
	if err != nil {
		return dto.RegisterResponse{}, err
	}

	return dto.RegisterResponse{
		UserID:   strconv.FormatUint(user.ID, 10),
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

func (s *Service) loginUser(req dto.LoginRequest, clientIP string) (dto.UserInfoResponse, error) {
	if !config.C.Database.Enabled || s.db == nil {
		return demoUser(req.Username), nil
	}

	user, err := s.findActiveUserByAccount(req.Username, true)
	if err != nil {
		return dto.UserInfoResponse{}, err
	}
	if user.Status != "" && user.Status != "active" {
		return dto.UserInfoResponse{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return dto.UserInfoResponse{}, ErrInvalidCredentials
	}

	now := time.Now()
	if err := s.db.Model(&database.SysUser{}).
		Where("id = ?", user.ID).
		Updates(map[string]any{
			"last_login_time": now,
			"last_login_ip":   clientIP,
		}).Error; err != nil {
		return dto.UserInfoResponse{}, err
	}
	user.LastLoginTime = &now
	user.LastLoginIP = clientIP

	return sysUserInfo(user), nil
}

func (s *Service) findActiveUserByAccount(account string, withRelations bool) (database.SysUser, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return database.SysUser{}, ErrInvalidCredentials
	}

	query := s.db
	if withRelations {
		query = preloadUser(query)
	}

	var user database.SysUser
	err := query.
		Where("is_deleted = false AND (LOWER(username) = ? OR LOWER(email) = ?)", strings.ToLower(account), strings.ToLower(account)).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.SysUser{}, ErrInvalidCredentials
	}
	if err != nil {
		return database.SysUser{}, err
	}
	if user.Status != "" && user.Status != "active" {
		return database.SysUser{}, ErrInvalidCredentials
	}
	return user, nil
}

func (s *Service) userInfoByUsername(username string) (dto.UserInfoResponse, error) {
	if !config.C.Database.Enabled || s.db == nil {
		return demoUser(username), nil
	}
	var user database.SysUser
	err := preloadUser(s.db).
		Where("username = ? AND is_deleted = false", username).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.UserInfoResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		return dto.UserInfoResponse{}, err
	}
	if user.Status != "" && user.Status != "active" {
		return dto.UserInfoResponse{}, ErrInvalidCredentials
	}
	return sysUserInfo(user), nil
}

func preloadUser(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Department").
		Preload("Roles", "is_deleted = false").
		Preload("Roles.Role", "is_deleted = false AND status = ?", "active").
		Preload("Roles.Role.Permissions", "is_deleted = false").
		Preload("Roles.Role.Permissions.Permission", "is_deleted = false")
}

func sysUserInfo(user database.SysUser) dto.UserInfoResponse {
	roles := make([]string, 0, len(user.Roles))
	permissions := make([]string, 0)
	for _, userRole := range user.Roles {
		if userRole.Role.RoleCode == "" {
			continue
		}
		roles = appendUnique(roles, userRole.Role.RoleCode)
		for _, rolePerm := range userRole.Role.Permissions {
			if rolePerm.Permission.PermCode != "" {
				permissions = appendUnique(permissions, rolePerm.Permission.PermCode)
			}
		}
	}

	departments := []string{}
	if user.Department != nil && user.Department.DeptName != "" {
		departments = append(departments, user.Department.DeptName)
	}

	lastLoginTime := ""
	if user.LastLoginTime != nil {
		lastLoginTime = user.LastLoginTime.Format(time.RFC3339)
	}

	return dto.UserInfoResponse{
		UserID:        strconv.FormatUint(user.ID, 10),
		Username:      user.Username,
		RealName:      user.RealName,
		Email:         user.Email,
		Phone:         user.Phone,
		Avatar:        user.Avatar,
		Roles:         roles,
		Permissions:   permissions,
		Departments:   departments,
		LastLoginTime: lastLoginTime,
		LastLoginIP:   user.LastLoginIP,
	}
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func demoUser(username string) dto.UserInfoResponse {
	if username == "" {
		username = "admin"
	}
	return dto.UserInfoResponse{
		UserID:        "user_demo_admin",
		Username:      username,
		RealName:      "System Administrator",
		Email:         username + "@example.com",
		Phone:         "",
		Avatar:        "",
		Roles:         []string{"SYS_ADMIN", "INVEST_MANAGER"},
		Permissions:   []string{"*"},
		Departments:   []string{"Technology Finance"},
		LastLoginTime: time.Now().Format(time.RFC3339),
		LastLoginIP:   "127.0.0.1",
	}
}

type passwordResetClaims struct {
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expires_at"`
}

func validatePasswordPair(password string, confirmPassword string) error {
	if strings.TrimSpace(password) == "" || password != confirmPassword || len(password) < 8 {
		return ErrValidation
	}
	return nil
}

func signPasswordResetToken(username string, expiresAt time.Time) (string, error) {
	claims := passwordResetClaims{Username: username, ExpiresAt: expiresAt.Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(config.C.JWT.AccessSecret))
	mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encodedPayload + "." + signature, nil
}

func parsePasswordResetToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", ErrInvalidCredentials
	}
	mac := hmac.New(sha256.New, []byte(config.C.JWT.AccessSecret))
	mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(actual, expected) {
		return "", ErrInvalidCredentials
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrInvalidCredentials
	}
	var claims passwordResetClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", ErrInvalidCredentials
	}
	if claims.Username == "" || time.Now().Unix() > claims.ExpiresAt {
		return "", ErrInvalidCredentials
	}
	return claims.Username, nil
}
