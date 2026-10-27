package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"wfagent/internal/base/config"
)

// CORS applies cross-origin settings from config.C.CORS.
func CORS() gin.HandlerFunc {
	cfg := config.C.CORS
	allowAll := contains(cfg.AllowOrigins, "*")
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && contains(cfg.AllowOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ","))
		c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ","))
		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
