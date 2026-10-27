package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"wfagent/internal/base/config"
	"wfagent/internal/base/database"
	baserouter "wfagent/internal/base/router"
	"wfagent/internal/base/security"
	"wfagent/internal/base/task"
	approval "wfagent/internal/module-approval"
	communication "wfagent/internal/module-communication"
)

// App wires the backend runtime after config.Load has populated config.C.
type App struct {
	DB     *gorm.DB
	Redis  *redis.Client
	Router *gin.Engine
	root   string
}

func New(root string) (*App, error) {
	if _, err := config.Load(root); err != nil {
		return nil, err
	}

	db, err := database.NewPostgres()
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrateBase(db); err != nil {
		return nil, err
	}
	if err := database.SeedInitialAdmin(db); err != nil {
		return nil, err
	}
	if err := task.AutoMigrate(db); err != nil {
		return nil, err
	}
	if err := approval.AutoMigrate(db); err != nil {
		return nil, err
	}
	if err := communication.AutoMigrate(db); err != nil {
		return nil, err
	}
	redisClient := database.NewRedis()
	security.SetRedisClient(redisClient)
	if err := approval.SeedDefaults(db); err != nil {
		return nil, err
	}
	if err := communication.SeedDemoData(db); err != nil {
		return nil, err
	}

	return &App{
		DB:     db,
		Redis:  redisClient,
		Router: baserouter.New(db, redisClient),
		root:   root,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	runCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	go config.Watch(runCtx, a.root, 5*time.Second, nil)
	task.StartWorkers(runCtx, a.DB, a.Redis)
	communication.StartMailboxAutoSync(runCtx, a.DB)

	server := &http.Server{
		Addr:              config.C.Server.Address(),
		Handler:           a.Router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("backend listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-stop:
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if a.Redis != nil {
		return a.Redis.Close()
	}
	return nil
}
