package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"wfagent/internal/base/config"
	"wfagent/internal/base/database"
)

func main() {
	direction := "up"
	if len(os.Args) > 1 {
		direction = strings.ToLower(os.Args[1])
	}
	if direction != "up" && direction != "down" {
		log.Fatalf("usage: go run ./cmd/migrate [up|down]")
	}
	if _, err := config.Load("."); err != nil {
		log.Fatal(err)
	}
	db, err := database.NewPostgres()
	if err != nil {
		log.Fatal(err)
	}
	if db == nil {
		log.Fatal("database is disabled; set DB_ENABLED=true before running migrations")
	}
	if err := ensureSchemaMigrations(db); err != nil {
		log.Fatal(err)
	}
	if err := runMigrations(db, direction); err != nil {
		log.Fatal(err)
	}
}

func ensureSchemaMigrations(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`).Error
}

func runMigrations(db *gorm.DB, direction string) error {
	pattern := filepath.Join("migrations", "*."+direction+".sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Strings(files)
	if direction == "down" {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}
	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), "."+direction+".sql")
		applied, err := isApplied(db, version)
		if err != nil {
			return err
		}
		if direction == "up" && applied {
			continue
		}
		if direction == "down" && !applied {
			continue
		}
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(sqlBytes)).Error; err != nil {
				return err
			}
			if direction == "up" {
				return tx.Exec("INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?) ON CONFLICT (version) DO NOTHING", version, time.Now()).Error
			}
			return tx.Exec("DELETE FROM schema_migrations WHERE version = ?", version).Error
		}); err != nil {
			return fmt.Errorf("migration %s failed: %w", file, err)
		}
		log.Printf("migration %s applied", file)
	}
	return nil
}

func isApplied(db *gorm.DB, version string) (bool, error) {
	var count int64
	err := db.Raw("SELECT COUNT(1) FROM schema_migrations WHERE version = ?", version).Scan(&count).Error
	return count > 0, err
}
