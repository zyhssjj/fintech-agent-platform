package task

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"wfagent/internal/base/config"
)

const (
	redisQueueKey      = "task:queue"
	redisDeadLetterKey = "task:dead_letter"
)

type Broker interface {
	Submit(ctx context.Context, taskType string, payload map[string]any) (*AsyncTask, error)
	Query(ctx context.Context, taskID string) (*AsyncTask, error)
	Cancel(ctx context.Context, taskID string) (*AsyncTask, error)
}

type RedisBroker struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewBroker(db *gorm.DB, redisClient *redis.Client) *RedisBroker {
	return &RedisBroker{db: db, redis: redisClient}
}

func (b *RedisBroker) Submit(ctx context.Context, taskType string, payload map[string]any) (*AsyncTask, error) {
	if taskType == "" {
		return nil, errors.New("task_type is required")
	}
	if _, ok := registeredExecutor(taskType); !ok {
		return nil, fmt.Errorf("no executor is registered for task_type %q", taskType)
	}
	if b.redis == nil {
		return nil, errors.New("redis task broker is not available")
	}
	taskID := "task_" + randomHex(16)
	input := mustJSON(payload)
	record := &AsyncTask{
		TaskID:      taskID,
		TaskType:    taskType,
		Status:      "pending",
		Progress:    0,
		InputParams: input,
		RetryLimit:  config.C.Task.RetryLimit,
	}
	if b.db != nil {
		if err := b.db.WithContext(ctx).Create(record).Error; err != nil {
			return nil, err
		}
	}
	if b.db == nil {
		if err := b.writeRedisTask(ctx, record); err != nil {
			return nil, err
		}
	}
	if err := b.redis.LPush(ctx, redisQueueKey, taskID).Err(); err != nil {
		return nil, fmt.Errorf("enqueue task: %w", err)
	}
	return record, nil
}

func (b *RedisBroker) Query(ctx context.Context, taskID string) (*AsyncTask, error) {
	if taskID == "" {
		return nil, errors.New("task_id is required")
	}
	if b.db != nil {
		var record AsyncTask
		err := b.db.WithContext(ctx).Where("task_id = ? AND is_deleted = false", taskID).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return &record, err
	}
	return b.readRedisTask(ctx, taskID)
}

func (b *RedisBroker) Cancel(ctx context.Context, taskID string) (*AsyncTask, error) {
	record, err := b.Query(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if record.Status == "success" || record.Status == "failed" {
		return record, nil
	}
	now := time.Now()
	record.Status = "cancelled"
	record.FinishedAt = &now
	record.UpdateTime = now
	if b.db != nil {
		err = b.db.WithContext(ctx).Model(&AsyncTask{}).
			Where("task_id = ? AND status IN ?", taskID, []string{"pending", "running"}).
			Updates(map[string]any{"status": record.Status, "finished_at": now, "update_time": now}).Error
		return record, err
	}
	return record, b.writeRedisTask(ctx, record)
}

func (b *RedisBroker) markRunning(ctx context.Context, taskID string) (*AsyncTask, error) {
	record, err := b.Query(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if record.Status == "cancelled" {
		return record, nil
	}
	now := time.Now()
	record.Status = "running"
	record.Progress = 10
	record.StartedAt = &now
	record.UpdateTime = now
	if b.db != nil {
		err = b.db.WithContext(ctx).Model(&AsyncTask{}).Where("task_id = ?", taskID).Updates(map[string]any{
			"status":      record.Status,
			"progress":    record.Progress,
			"started_at":  now,
			"update_time": now,
		}).Error
	}
	if err == nil && b.db == nil {
		err = b.writeRedisTask(ctx, record)
	}
	return record, err
}

func (b *RedisBroker) markSuccess(ctx context.Context, record *AsyncTask, result map[string]any) error {
	if record == nil || record.Status == "cancelled" {
		return nil
	}
	now := time.Now()
	record.Status = "success"
	record.Progress = 100
	record.OutputResult = mustJSON(result)
	record.FinishedAt = &now
	record.UpdateTime = now
	if b.db != nil {
		return b.db.WithContext(ctx).Model(&AsyncTask{}).Where("task_id = ?", record.TaskID).Updates(map[string]any{
			"status":        record.Status,
			"progress":      record.Progress,
			"output_result": record.OutputResult,
			"finished_at":   now,
			"update_time":   now,
		}).Error
	}
	return b.writeRedisTask(ctx, record)
}

func (b *RedisBroker) markFailed(ctx context.Context, record *AsyncTask, cause error) error {
	if record == nil {
		return nil
	}
	record.RetryCount++
	if record.RetryCount <= record.RetryLimit {
		record.Status = "pending"
		record.Progress = 0
		record.ErrorMessage = cause.Error()
		record.UpdateTime = time.Now()
		if b.db != nil {
			if err := b.db.WithContext(ctx).Model(&AsyncTask{}).Where("task_id = ?", record.TaskID).Updates(map[string]any{
				"status":        record.Status,
				"progress":      record.Progress,
				"error_message": record.ErrorMessage,
				"retry_count":   record.RetryCount,
				"update_time":   record.UpdateTime,
			}).Error; err != nil {
				return err
			}
		} else if err := b.writeRedisTask(ctx, record); err != nil {
			return err
		}
		if b.redis != nil {
			return b.redis.LPush(ctx, redisQueueKey, record.TaskID).Err()
		}
		return nil
	}
	now := time.Now()
	record.Status = "failed"
	record.ErrorMessage = cause.Error()
	record.FinishedAt = &now
	record.UpdateTime = now
	if b.db != nil {
		if err := b.db.WithContext(ctx).Model(&AsyncTask{}).Where("task_id = ?", record.TaskID).Updates(map[string]any{
			"status":        record.Status,
			"error_message": record.ErrorMessage,
			"finished_at":   now,
			"update_time":   now,
		}).Error; err != nil {
			return err
		}
		return b.deadLetter(ctx, record)
	}
	if err := b.writeRedisTask(ctx, record); err != nil {
		return err
	}
	return b.deadLetter(ctx, record)
}

func (b *RedisBroker) deadLetter(ctx context.Context, record *AsyncTask) error {
	if b.redis == nil || record == nil {
		return nil
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return b.redis.LPush(ctx, redisDeadLetterKey, data).Err()
}

func (b *RedisBroker) writeRedisTask(ctx context.Context, task *AsyncTask) error {
	if b.redis == nil || task == nil {
		return nil
	}
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return b.redis.Set(ctx, redisTaskKey(task.TaskID), data, 7*24*time.Hour).Err()
}

func (b *RedisBroker) readRedisTask(ctx context.Context, taskID string) (*AsyncTask, error) {
	if b.redis == nil {
		return nil, gorm.ErrRecordNotFound
	}
	data, err := b.redis.Get(ctx, redisTaskKey(taskID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, gorm.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	var record AsyncTask
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func redisTaskKey(taskID string) string {
	return "task:record:" + taskID
}

func mustJSON(value any) datatypes.JSON {
	data, err := json.Marshal(value)
	if err != nil || len(data) == 0 {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(data)
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
