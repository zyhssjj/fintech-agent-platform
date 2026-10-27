package modulexxx

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes is the starter template for a business module.
func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	_ = db

	group := v1.Group("/module-xxx")
	_ = group
}
