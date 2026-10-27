package system

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wfagent/internal/base/middleware"
)

// RegisterRoutes registers base-system administration routes. They deliberately
// live outside business modules so every module shares one RBAC source of truth.
func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	handler := NewHandler(NewService(db))
	group := v1.Group("/system")
	group.GET("/roles", middleware.RequirePermission("system:role:manage"), handler.ListRoles)
	group.GET("/permissions", middleware.RequirePermission("system:role:manage"), handler.ListPermissions)
	group.PUT("/users/:user_id/roles", middleware.RequirePermission("system:user:manage"), handler.ReplaceUserRoles)
	group.PUT("/roles/:role_id/permissions", middleware.RequirePermission("system:role:manage"), handler.ReplaceRolePermissions)
	group.GET("/api-config", middleware.RequireRole("SYS_ADMIN"), handler.GetAPIConfig)
	group.PUT("/api-config", middleware.RequireRole("SYS_ADMIN"), handler.UpdateAPIConfig)
}
