package draft

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type ExecutionWorker struct {
	draftRepo      ports.DraftRepository
	databaseRepo   ports.DatabaseRepository
	migrationRepo  ports.MigrationRepository
	databaseDriver ports.DatabaseDriver
	unlockDatabase func(databaseID uint)
	maxRetries     int
	backoffBase    time.Duration
}

func NewExecutionWorker(
	draftRepo ports.DraftRepository,
	databaseRepo ports.DatabaseRepository,
	migrationRepo ports.MigrationRepository,
	databaseDriver ports.DatabaseDriver,
	unlockDatabase func(databaseID uint),
) *ExecutionWorker {
	return &ExecutionWorker{
		draftRepo:      draftRepo,
		databaseRepo:   databaseRepo,
		migrationRepo:  migrationRepo,
		databaseDriver: databaseDriver,
		unlockDatabase: unlockDatabase,
		maxRetries:     3,
		backoffBase:    10 * time.Millisecond,
	}
}

func (w *ExecutionWorker) SetRetryPolicy(maxRetries int, backoffBase time.Duration) {
	w.maxRetries = maxRetries
	w.backoffBase = backoffBase
}

func (w *ExecutionWorker) Execute(ctx context.Context, task ports.ExecutionTask) (err error) {
	if w.unlockDatabase != nil {
		defer w.unlockDatabase(task.DatabaseID)
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("worker panic: %v", r)
			w.handleFailure(context.Background(), task, panicErr.Error())
			err = panicErr
		}
	}()

	draft, err := w.draftRepo.FindByID(ctx, task.DraftID)
	if err != nil {
		w.handleFailure(ctx, task, err.Error())
		return err
	}

	db, err := w.databaseRepo.FindWithCredentialByID(ctx, task.DatabaseID)
	if err != nil {
		w.handleFailure(ctx, task, err.Error())
		return err
	}

	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			w.handleFailure(context.Background(), task, ctx.Err().Error())
			return ctx.Err()
		default:
		}

		_, cmdErr := w.databaseDriver.ExecuteCommand(ctx, db, draft.SqlUp)
		if cmdErr == nil {
			return w.handleSuccess(ctx, task, draft)
		}

		lastErr = cmdErr
		if !isTransientError(cmdErr) {
			break
		}

		if attempt < w.maxRetries {
			select {
			case <-ctx.Done():
				w.handleFailure(context.Background(), task, ctx.Err().Error())
				return ctx.Err()
			case <-time.After(w.backoffBase * time.Duration(1<<attempt)):
			}
		}
	}

	w.handleFailure(context.Background(), task, lastErr.Error())
	return lastErr
}

func (w *ExecutionWorker) handleSuccess(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft) error {
	now := time.Now()
	migration := &domain.Migration{
		DatabaseID:   task.DatabaseID,
		UserID:       task.UserID,
		DraftID:      &task.DraftID,
		ExecutionID:  task.ExecutionID,
		SqlUp:        draft.SqlUp,
		SqlDown:      draft.SqlDown,
		IsReversible: draft.IsReversible,
		Status:       domain.MigrationStatusSuccess,
		MigratedAt:   now,
	}

	if w.migrationRepo != nil {
		saved, err := w.migrationRepo.Save(ctx, migration)
		if err == nil && saved != nil && w.databaseRepo != nil {
			_ = w.databaseRepo.UpdateCurrentMigrationID(ctx, task.DatabaseID, saved.ID)
		}
	}

	draft.Status = domain.DraftStatusApplied
	draft.UpdatedAt = now
	_, err := w.draftRepo.Update(ctx, draft)
	return err
}

func (w *ExecutionWorker) handleFailure(ctx context.Context, task ports.ExecutionTask, errMsg string) {
	now := time.Now()
	migration := &domain.Migration{
		DatabaseID:   task.DatabaseID,
		UserID:       task.UserID,
		DraftID:      &task.DraftID,
		ExecutionID:  task.ExecutionID,
		Status:       domain.MigrationStatusFail,
		ErrorMessage: errMsg,
		MigratedAt:   now,
	}

	if w.migrationRepo != nil {
		_, _ = w.migrationRepo.Save(ctx, migration)
	}

	if draft, err := w.draftRepo.FindByID(ctx, task.DraftID); err == nil && draft != nil {
		draft.Status = domain.DraftStatusFailed
		draft.UpdatedAt = now
		_, _ = w.draftRepo.Update(ctx, draft)
	}
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "deadlock") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "lock wait timeout")
}
