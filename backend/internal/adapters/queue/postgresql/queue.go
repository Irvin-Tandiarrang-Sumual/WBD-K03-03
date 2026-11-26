package postgresql

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/metrics"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/gorm"
)

const (
	jobPending   = "pending"
	jobRunning   = "running"
	jobDone      = "done"
	jobFailed    = "failed"
	jobUncertain = "uncertain"
)

type Queue struct {
	db       *gorm.DB
	handler  ports.TaskHandler
	capacity int
	timeout  time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	workers  sync.WaitGroup
}

var _ ports.PersistentTaskQueue = (*Queue)(nil)

func NewQueue(db *gorm.DB, handler ports.TaskHandler, workers, capacity int, timeout time.Duration) *Queue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		db: db, handler: handler, capacity: max(1, capacity), timeout: timeout,
		ctx: ctx, cancel: cancel,
	}
	for range max(1, workers) {
		q.workers.Add(1)
		go q.run()
	}
	return q
}

func (q *Queue) Close() {
	q.cancel()
	q.workers.Wait()
}

func (q *Queue) run() {
	defer q.workers.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for q.ctx.Err() == nil {
		processed, err := q.processNext()
		if err != nil && q.ctx.Err() == nil {
			slog.Error("migration queue operation failed", "error", err)
		}
		if processed && err == nil {
			continue
		}
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (q *Queue) processNext() (bool, error) {
	var jobs []uint
	if err := q.db.WithContext(q.ctx).Model(&model.MigrationJob{}).
		Where("status IN ?", []string{jobPending, jobRunning}).
		Order("status ASC, id ASC").Pluck("id", &jobs).Error; err != nil {
		return false, err
	}
	for _, id := range jobs {
		if q.ctx.Err() != nil {
			return false, q.ctx.Err()
		}
		processed, err := q.processJob(id)
		if processed || err != nil {
			return processed, err
		}
	}
	return false, nil
}

func (q *Queue) processJob(id uint) (processed bool, err error) {
	err = q.db.WithContext(context.WithoutCancel(q.ctx)).Connection(func(conn *gorm.DB) (runErr error) {
		conn = conn.Session(&gorm.Session{NewDB: true})
		var locked bool
		if err := conn.Raw("SELECT pg_try_advisory_lock(hashtextextended(?, 0))", jobLockKey(id)).Scan(&locked).Error; err != nil || !locked {
			return err
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			unlockErr := conn.WithContext(ctx).Exec("SELECT pg_advisory_unlock(hashtextextended(?, 0))", jobLockKey(id)).Error
			runErr = errors.Join(runErr, unlockErr)
		}()

		var job model.MigrationJob
		if err := conn.Take(&job, id).Error; err != nil {
			return err
		}
		switch job.Status {
		case jobRunning:
			processed = true
			return finishJob(conn, job, nil)
		case jobPending:
			if err := q.ctx.Err(); err != nil {
				return err
			}
			processed = true
			return q.runJob(conn, job)
		default:
			return nil
		}
	})
	return
}

func jobLockKey(id uint) string {
	return "migration-job:" + strconv.FormatUint(uint64(id), 10)
}

func (q *Queue) runJob(conn *gorm.DB, job model.MigrationJob) error {
	task := ports.ExecutionTask{
		DraftID: job.DraftID, DatabaseID: job.DatabaseID, ExecutionID: job.ExecutionID,
		UserID: job.UserID, Operation: job.Operation, EnqueuedAt: job.CreatedAt,
		JobID: job.ID,
	}
	if err := json.Unmarshal(job.TraceCarrier, &task.TraceCarrier); err != nil {
		return err
	}
	if err := conn.Model(&job).Update("status", jobRunning).Error; err != nil {
		return err
	}
	metrics.QueueWaitDuration.Observe(time.Since(job.CreatedAt).Seconds())
	taskCtx, cancel := context.WithTimeout(context.Background(), q.timeout)
	defer cancel()
	return finishJob(conn, job, q.handler(taskCtx, task))
}

func finishJob(conn *gorm.DB, job model.MigrationJob, executionErr error) error {
	status := jobUncertain
	var draft model.Draft
	if err := conn.Select("status").Take(&draft, job.DraftID).Error; err != nil {
		return err
	}
	if !errors.Is(executionErr, domain.ErrMigrationOutcomeUnknown) {
		switch draft.Status {
		case domain.DraftStatusApplied:
			status = jobDone
		case domain.DraftStatusFailed, domain.DraftStatusInvalidated, domain.DraftStatusPartial, domain.DraftStatusPartiallyReverted:
			status = jobFailed
		}
	}
	if status == jobUncertain {
		slog.Warn("migration job requires commit verification", "job_id", job.ID, "database_id", job.DatabaseID)
	}
	return conn.Model(&job).Update("status", status).Error
}
