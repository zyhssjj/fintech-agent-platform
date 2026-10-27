package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"wfagent/internal/base/config"
	"wfagent/internal/base/database"
)

var auditFileMu sync.Mutex

// Audit records request metadata to com_operation_log and to a rotated JSONL file.
func Audit(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		entry := database.OperationLog{
			RequestID: c.GetString("request_id"),
			UserID:    c.GetString("user_id"),
			Username:  c.GetString("username"),
			Method:    c.Request.Method,
			Path:      c.FullPath(),
			ClientIP:  c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Status:    c.Writer.Status(),
			LatencyMS: time.Since(start).Milliseconds(),
			Params:    auditParams(c),
		}
		if entry.Path == "" {
			entry.Path = c.Request.URL.Path
		}

		writeAuditFile(entry)
		if db != nil && config.C.Audit.PersistToDB {
			go persistAuditLog(db, entry)
		}
	}
}

func auditParams(c *gin.Context) datatypes.JSON {
	payload := map[string]any{
		"query": c.Request.URL.Query(),
		"route": c.FullPath(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(raw)
}

func writeAuditFile(entry database.OperationLog) {
	cfg := config.C.Audit
	if cfg.LogFile == "" {
		return
	}
	record := map[string]any{
		"time":       time.Now().Format(time.RFC3339Nano),
		"request_id": entry.RequestID,
		"user_id":    entry.UserID,
		"username":   entry.Username,
		"method":     entry.Method,
		"path":       entry.Path,
		"status":     entry.Status,
		"ip":         entry.ClientIP,
		"user_agent": entry.UserAgent,
		"latency_ms": entry.LatencyMS,
	}
	line, err := json.Marshal(record)
	if err != nil {
		slog.Warn("marshal audit log failed", "error", err)
		return
	}

	auditFileMu.Lock()
	defer auditFileMu.Unlock()

	if err := rotateAuditFile(cfg.LogFile, cfg.MaxSizeMB, cfg.MaxBackups, int64(len(line)+1)); err != nil {
		slog.Warn("rotate audit log failed", "error", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(cfg.LogFile), 0o755); err != nil {
		slog.Warn("create audit log dir failed", "error", err)
		return
	}
	file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		slog.Warn("open audit log file failed", "error", err)
		return
	}
	defer file.Close()
	if _, err := file.Write(append(line, '\n')); err != nil {
		slog.Warn("write audit log file failed", "error", err)
	}
}

func rotateAuditFile(path string, maxSizeMB int64, maxBackups int, incomingBytes int64) error {
	if maxSizeMB <= 0 {
		maxSizeMB = 20
	}
	if maxBackups <= 0 {
		maxBackups = 5
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size()+incomingBytes <= maxSizeMB*1024*1024 {
		return nil
	}

	oldest := path + "." + strconv.Itoa(maxBackups)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return err
	}
	for i := maxBackups - 1; i >= 1; i-- {
		src := path + "." + strconv.Itoa(i)
		dst := path + "." + strconv.Itoa(i+1)
		if _, err := os.Stat(src); err == nil {
			if err := os.Rename(src, dst); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return os.Rename(path, path+".1")
}

func persistAuditLog(db *gorm.DB, entry database.OperationLog) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.WithContext(ctx).Create(&entry).Error; err != nil {
		slog.Warn("persist audit log failed", "error", err, "request_id", entry.RequestID)
	}
}
