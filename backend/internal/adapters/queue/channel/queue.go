package channel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

const (
	DefaultTaskTimeout = 15 * time.Minute
	DefaultQueueSize   = 100
	DefaultWorkerCount = 5
)

type ChannelQueue struct {
	tasks       chan ports.ExecutionTask
	handler     ports.TaskHandler
	workerCount int
	timeout     time.Duration
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	isClosed    bool
}

var _ ports.TaskQueue = (*ChannelQueue)(nil)

func NewChannelQueue(handler ports.TaskHandler, workerCount int, queueSize int, timeout time.Duration) *ChannelQueue {
	if workerCount <= 0 {
		workerCount = DefaultWorkerCount
	}
	if queueSize <= 0 {
		queueSize = DefaultQueueSize
	}
	if timeout <= 0 {
		timeout = DefaultTaskTimeout
	}

	ctx, cancel := context.WithCancel(context.Background())
	q := &ChannelQueue{
		tasks:       make(chan ports.ExecutionTask, queueSize),
		handler:     handler,
		workerCount: workerCount,
		timeout:     timeout,
		ctx:         ctx,
		cancel:      cancel,
	}

	q.startWorkers()
	return q
}

func (q *ChannelQueue) startWorkers() {
	for i := 0; i < q.workerCount; i++ {
		q.wg.Add(1)
		go q.worker()
	}
}

func (q *ChannelQueue) worker() {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return
		case task, ok := <-q.tasks:
			if !ok {
				return
			}
			q.processTask(task)
		}
	}
}

func (q *ChannelQueue) processTask(task ports.ExecutionTask) {
	defer func() {
		if r := recover(); r != nil {
			// Panic recovery ensures unhandled panic inside task execution does not kill the worker pool.
			_ = r
		}
	}()

	taskCtx, cancel := context.WithTimeout(context.Background(), q.timeout)
	defer cancel()

	if q.handler != nil {
		_ = q.handler(taskCtx, task)
	}
}

func (q *ChannelQueue) Enqueue(ctx context.Context, task ports.ExecutionTask) error {
	q.mu.Lock()
	if q.isClosed {
		q.mu.Unlock()
		return fmt.Errorf("queue is closed")
	}
	q.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.tasks <- task:
		return nil
	default:
		return domain.ErrQueueFull
	}
}

func (q *ChannelQueue) Close() {
	q.mu.Lock()
	if q.isClosed {
		q.mu.Unlock()
		return
	}
	q.isClosed = true
	q.mu.Unlock()

	q.cancel()
	close(q.tasks)
	q.wg.Wait()
}
