package task

import (
	"context"
	"testing"
)

func TestBaseEchoExecutor(t *testing.T) {
	executor, ok := registeredExecutor("base:echo")
	if !ok {
		t.Fatal("base:echo executor is not registered")
	}

	payload := map[string]any{"request_id": "test-1"}
	result, err := executor.Execute(context.Background(), payload)
	if err != nil {
		t.Fatalf("execute base echo task: %v", err)
	}
	echoed, ok := result["payload"].(map[string]any)
	if !ok || echoed["request_id"] != "test-1" {
		t.Fatalf("unexpected executor result: %#v", result)
	}
}

func TestRegisterExecutorRejectsDuplicateTaskType(t *testing.T) {
	taskType := "test:duplicate"
	if err := RegisterExecutor(taskType, ExecutorFunc(func(context.Context, map[string]any) (map[string]any, error) {
		return map[string]any{}, nil
	})); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	if err := RegisterExecutor(taskType, ExecutorFunc(func(context.Context, map[string]any) (map[string]any, error) {
		return map[string]any{}, nil
	})); err == nil {
		t.Fatal("expected duplicate task type to be rejected")
	}
}
