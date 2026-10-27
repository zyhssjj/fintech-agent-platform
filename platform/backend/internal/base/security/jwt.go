package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"wfagent/internal/base/config"
	"wfagent/internal/base/dto"
)

var redisClient *redis.Client

// Claims is the shared JWT payload used by access and refresh tokens.
type Claims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	TokenType   string   `json:"token_type"`
	jwt.RegisteredClaims
}

func SetRedisClient(client *redis.Client) {
	redisClient = client
}

func SignAccessToken(user dto.UserInfoResponse) (string, int, error) {
	jwtCfg := config.C.JWT
	expiresAt := time.Now().Add(jwtCfg.AccessTTL)
	token, err := sign(jwtCfg.AccessSecret, jwtCfg.Issuer, "access", expiresAt, user)
	return token, int(jwtCfg.AccessTTL.Seconds()), err
}

func SignRefreshToken(user dto.UserInfoResponse) (string, int, error) {
	jwtCfg := config.C.JWT
	expiresAt := time.Now().Add(jwtCfg.RefreshTTL)
	token, err := sign(jwtCfg.RefreshSecret, jwtCfg.Issuer, "refresh", expiresAt, user)
	return token, int(jwtCfg.RefreshTTL.Seconds()), err
}

func ParseAccessToken(raw string) (*Claims, error) {
	return ParseToken(config.C.JWT.AccessSecret, raw)
}

func ParseRefreshToken(raw string) (*Claims, error) {
	return ParseToken(config.C.JWT.RefreshSecret, raw)
}

func ParseToken(secret string, raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
			}
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(config.C.JWT.Issuer),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func BlacklistToken(raw string, ttl time.Duration) error {
	if redisClient == nil || raw == "" || ttl <= 0 {
		return nil
	}
	key := blacklistKey(raw)
	if err := redisClient.Set(context.Background(), key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("blacklist token: %w", err)
	}
	return nil
}

func IsTokenBlacklisted(raw string) (bool, error) {
	if redisClient == nil || raw == "" {
		return false, nil
	}
	value, err := redisClient.Exists(context.Background(), blacklistKey(raw)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		slog.Warn("redis token blacklist check failed", "error", err)
		return false, err
	}
	return value > 0, nil
}

func sign(secret, issuer, tokenType string, expiresAt time.Time, user dto.UserInfoResponse) (string, error) {
	claims := Claims{
		UserID:      user.UserID,
		Username:    user.Username,
		Roles:       user.Roles,
		Permissions: user.Permissions,
		TokenType:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   user.UserID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func blacklistKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return "jwt:blacklist:" + hex.EncodeToString(sum[:])
}
