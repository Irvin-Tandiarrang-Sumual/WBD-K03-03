package ports

import "context"

type ExecutionTask struct {
	DraftID     uint
	DatabaseID  uint
	ExecutionID uint
	UserID      uint
}

type TaskQueue interface {
	Enqueue(ctx context.Context, task ExecutionTask) error
}

type TaskHandler func(ctx context.Context, task ExecutionTask) error
