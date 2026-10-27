package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	_ "wfagent/docs"
	"wfagent/internal/base/auth"
	"wfagent/internal/base/config"
	basefile "wfagent/internal/base/file"
	"wfagent/internal/base/middleware"
	"wfagent/internal/base/response"
	"wfagent/internal/base/system"
	"wfagent/internal/base/task"
	approval "wfagent/internal/module-approval"
	communication "wfagent/internal/module-communication"
	knowledge "wfagent/internal/module-knowledge"
)

// New registers middleware and routes. Runtime settings are read from config.C.
func New(db *gorm.DB, redisClient *redis.Client) *gin.Engine {
	if config.C.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.RequestID(), middleware.CORS(), middleware.RateLimit(redisClient), middleware.Audit(db))

	r.GET("/healthz", healthz)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/healthz")
	})

	v1 := r.Group("/v1")
	auth.RegisterRoutes(v1, db)
	communication.RegisterPublicIntegrationRoutes(v1, db)

	protected := v1.Group("")
	protected.Use(middleware.Auth())
	basefile.RegisterRoutes(protected, db)
	task.RegisterRoutes(protected, db, redisClient)
	system.RegisterRoutes(protected, db)
	approval.RegisterRoutes(protected, db)
	communication.RegisterRoutes(protected, db)
	knowledge.RegisterRoutes(protected, db)

	return r
}

// healthz godoc
// @Summary Health check
// @Tags Base
// @Produce json
// @Success 200 {object} response.Body{data=map[string]string}
// @Router /healthz [get]
func healthz(c *gin.Context) {
	response.Success(c, gin.H{"status": "ok", "app": config.C.App.Name, "env": config.C.App.Env})
}
