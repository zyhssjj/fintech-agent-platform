package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"wfagent/internal/base/config"
)

type WorkerPool struct {
	broker *RedisBroker
}

func StartWorkers(ctx context.Context, db *gorm.DB, redisClient *redis.Client) *WorkerPool {
	if redisClient == nil || config.C.Task.WorkerCount <= 0 {
		slog.Warn("async task workers disabled; redis is not available or worker count is zero")
		return &WorkerPool{broker: NewBroker(db, redisClient)}
	}
	pool := &WorkerPool{broker: NewBroker(db, redisClient)}
	for i := 0; i < config.C.Task.WorkerCount; i++ {
		go pool.run(ctx, i+1)
	}
	return pool
}

func (p *WorkerPool) run(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		item, err := p.broker.redis.BRPop(ctx, 5*time.Second, redisQueueKey).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Warn("async task worker pop failed", "worker_id", workerID, "error", err)
			continue
		}
		if len(item) < 2 {
			continue
		}
		p.process(ctx, workerID, item[1])
	}
}

func (p *WorkerPool) process(ctx context.Context, workerID int, taskID string) {
	record, err := p.broker.markRunning(ctx, taskID)
	if err != nil {
		slog.Warn("async task mark running failed", "worker_id", workerID, "task_id", taskID, "error", err)
		return
	}
	if record == nil || record.Status == "cancelled" {
		return
	}

	executor, ok := registeredExecutor(record.TaskType)
	if !ok {
		p.fail(ctx, workerID, record, fmt.Errorf("no executor is registered for task_type %q", record.TaskType))
		return
	}
	payload := map[string]any{}
	if err := json.Unmarshal(record.InputParams, &payload); err != nil {
		p.fail(ctx, workerID, record, fmt.Errorf("decode task payload: %w", err))
		return
	}
	result, err := executor.Execute(ctx, payload)
	if err != nil {
		p.fail(ctx, workerID, record, err)
		return
	}
	result["worker_id"] = workerID
	if err := p.broker.markSuccess(ctx, record, result); err != nil {
		p.fail(ctx, workerID, record, err)
	}
}

func (p *WorkerPool) fail(ctx context.Context, workerID int, record *AsyncTask, cause error) {
	if err := p.broker.markFailed(ctx, record, cause); err != nil {
		slog.Warn("async task failure update failed", "worker_id", workerID, "task_id", record.TaskID, "error", err)
		return
	}
	slog.Warn("async task execution failed", "worker_id", workerID, "task_id", record.TaskID, "error", cause)
}
