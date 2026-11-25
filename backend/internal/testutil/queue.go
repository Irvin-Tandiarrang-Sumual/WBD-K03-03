package testutil

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type TaskQueue struct {
	ports.TaskQueue
	EnqueueFunc func(context.Context, ports.ExecutionTask) error
}

func (q *TaskQueue) Enqueue(ctx context.Context, task ports.ExecutionTask) error {
	if q.EnqueueFunc != nil {
		return q.EnqueueFunc(ctx, task)
	}
	return nil
}
