package statemachine

const (
	TaskPending   = "pending"
	TaskRunning   = "running"
	TaskSuccess   = "success"
	TaskFailed    = "failed"
	TaskCancelled = "cancelled"
)

var taskTransitions = map[string][]string{
	TaskPending: {TaskRunning, TaskCancelled},
	TaskRunning: {TaskSuccess, TaskFailed, TaskCancelled},
	TaskFailed:  {TaskPending},
}

func CanTransitionTask(from, to string) bool {
	return canTransition(taskTransitions, from, to)
}
