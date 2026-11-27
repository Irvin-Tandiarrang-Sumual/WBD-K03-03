package ports

import (
	"context"
	"time"
)

type ExecutionTask struct {
	DraftID      uint
	DatabaseID   uint
	ExecutionID  uint
	UserID       uint
	Operation    string
	EnqueuedAt   time.Time
	TraceCarrier map[string]string
	JobID        uint
}

type TaskQueue interface {
	Enqueue(ctx context.Context, task ExecutionTask) error
}

type PersistentTaskQueue interface {
	TaskQueue
	Submit(ctx context.Context, task ExecutionTask) (string, error)
}

type TaskHandler func(ctx context.Context, task ExecutionTask) error
