package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"wfagent/internal/base/response"
	"wfagent/internal/base/security"
)

// Auth validates access tokens with the shared config.C JWT settings.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if raw == "" {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		claims, err := security.ParseAccessToken(raw)
		if err != nil || claims.TokenType != "access" {
			response.Error(c, response.ErrUnauthorized)
			return
		}
		blacklisted, err := security.IsTokenBlacklisted(raw)
		if err != nil {
			response.Error(c, response.ErrUnauthorized)
			return
		}
		if blacklisted {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actual, _ := c.Get("roles")
		userRoles, _ := actual.([]string)
		for _, userRole := range userRoles {
			if userRole == "SYS_ADMIN" || userRole == "*" {
				c.Next()
				return
			}
			for _, role := range roles {
				if userRole == role {
					c.Next()
					return
				}
			}
		}
		response.Error(c, response.ErrForbidden)
	}
}

func RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actual, _ := c.Get("permissions")
		userPermissions, _ := actual.([]string)
		for _, userPermission := range userPermissions {
			if userPermission == "*" {
				c.Next()
				return
			}
			for _, permission := range permissions {
				if userPermission == permission {
					c.Next()
					return
				}
			}
		}
		response.Error(c, response.ErrForbidden)
	}
}

func RequireResource(resource string, action string) gin.HandlerFunc {
	code := resource + ":" + action
	return RequirePermission(code, resource+":manage", "*")
}
