package database

import (
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"wfagent/internal/base/config"
)

// NewPostgres initializes GORM from the shared config.C.Database settings.
func NewPostgres() (*gorm.DB, error) {
	cfg := config.C.Database
	if !cfg.Enabled {
		slog.Warn("database disabled; repository implementations should stay in placeholder mode")
		return nil, nil
	}

	db, err := gorm.Open(postgres.Open(cfg.BuildDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return db, nil
}
