package task

import (
	"context"
	"errors"
	"sync"
)

// Executor executes one registered task type. Business modules register their
// own executors during bootstrap instead of implementing independent queues.
type Executor interface {
	Execute(ctx context.Context, payload map[string]any) (map[string]any, error)
}

type ExecutorFunc func(ctx context.Context, payload map[string]any) (map[string]any, error)

func (f ExecutorFunc) Execute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	return f(ctx, payload)
}

var executors = struct {
	sync.RWMutex
	items map[string]Executor
}{items: map[string]Executor{}}

func RegisterExecutor(taskType string, executor Executor) error {
	if taskType == "" || executor == nil {
		return errors.New("task type and executor are required")
	}
	executors.Lock()
	defer executors.Unlock()
	if _, exists := executors.items[taskType]; exists {
		return errors.New("task executor is already registered: " + taskType)
	}
	executors.items[taskType] = executor
	return nil
}

func registeredExecutor(taskType string) (Executor, bool) {
	executors.RLock()
	defer executors.RUnlock()
	executor, ok := executors.items[taskType]
	return executor, ok
}

func init() {
	// This executor provides an end-to-end base verification path. Business
	// modules must register a dedicated executor for their own task types.
	_ = RegisterExecutor("base:echo", ExecutorFunc(func(_ context.Context, payload map[string]any) (map[string]any, error) {
		return map[string]any{"payload": payload}, nil
	}))
}
