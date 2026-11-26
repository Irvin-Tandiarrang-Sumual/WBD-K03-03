package channel_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/queue/channel"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

func TestChannelQueue_Enqueue_QueueAvailable_ReturnsNil(t *testing.T) {
	// Arrange
	ctx := context.Background()
	var processedTask ports.ExecutionTask
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(taskCtx context.Context, task ports.ExecutionTask) error {
		processedTask = task
		wg.Done()
		return nil
	}

	q := channel.NewChannelQueue(handler, 1, 10, time.Second)
	defer q.Close()

	inputTask := ports.ExecutionTask{
		DraftID:     1,
		DatabaseID:  2,
		ExecutionID: 3,
		UserID:      4,
	}

	// Act
	err := q.Enqueue(ctx, inputTask)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	wg.Wait()
	if processedTask.DraftID != inputTask.DraftID || processedTask.ExecutionID != inputTask.ExecutionID {
		t.Fatalf("expected processed task %+v, got %+v", inputTask, processedTask)
	}
}

func TestChannelQueue_Enqueue_QueueFull_ReturnsErrQueueFull(t *testing.T) {
	// Arrange
	ctx := context.Background()
	blockForever := make(chan struct{})

	handler := func(taskCtx context.Context, task ports.ExecutionTask) error {
		select {
		case <-taskCtx.Done():
			return taskCtx.Err()
		case <-blockForever:
			return nil
		}
	}

	// 1 worker, queue buffer size 1
	q := channel.NewChannelQueue(handler, 1, 1, time.Second)
	defer func() {
		close(blockForever)
		q.Close()
	}()

	// Fill the worker
	_ = q.Enqueue(ctx, ports.ExecutionTask{DraftID: 1})
	time.Sleep(10 * time.Millisecond) // wait for worker to pick up

	// Fill the buffer
	_ = q.Enqueue(ctx, ports.ExecutionTask{DraftID: 2})

	// Act: third item should trigger ErrQueueFull
	err := q.Enqueue(ctx, ports.ExecutionTask{DraftID: 3})

	// Assert
	if !errors.Is(err, domain.ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
}

func TestChannelQueue_Enqueue_ContextCancelled_ReturnsContextCanceled(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately before enqueueing

	q := channel.NewChannelQueue(nil, 1, 10, time.Second)
	defer q.Close()

	// Act
	err := q.Enqueue(ctx, ports.ExecutionTask{DraftID: 1})

	// Assert
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
