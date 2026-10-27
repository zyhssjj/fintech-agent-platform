package auth

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wfagent/internal/base/middleware"
)

// RegisterRoutes registers authentication endpoints.
func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	handler := NewHandler(NewService(db))
	group := v1.Group("/auth")
	group.POST("/login", handler.Login)
	group.POST("/refresh", handler.Refresh)
	group.POST("/register", handler.Register)
	group.POST("/password/forgot", handler.ForgotPassword)
	group.POST("/password/reset", handler.ResetPassword)

	protected := group.Group("")
	protected.Use(middleware.Auth())
	protected.POST("/logout", handler.Logout)
	protected.GET("/me", handler.Me)
	protected.PUT("/password", handler.ChangePassword)
}
