package draft

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
)

func batchExecution(draft *domain.Draft, statement string) (ports.BatchExecution, error) {
	column := strings.ToLower(strings.TrimSpace(draft.BatchColumn))
	table, sql, err := sqlutil.BindBatchUpdate(statement, column)
	if err != nil || draft.BatchSize <= 0 || draft.BatchSize > 10000 {
		return ports.BatchExecution{}, domain.ErrInvalidDraftInput
	}
	return ports.BatchExecution{
		Key:   fmt.Sprintf("batch:%d:%d:%d", draft.DatabaseID, draft.ID, draft.CreatedAt.UnixNano()),
		Table: table, Column: column, Statement: sql, Size: draft.BatchSize,
	}, nil
}

func validateBatchStatements(db domain.Database, input domain.CreateDraftInput) error {
	draft := &domain.Draft{BatchColumn: input.BatchColumn, BatchSize: input.BatchSize}
	up, err := batchExecution(draft, input.SqlUp)
	if err != nil || db.Type != domain.DatabaseTypePostgreSQL {
		return domain.ErrInvalidDraftInput
	}
	if strings.TrimSpace(input.SqlDown) == "" {
		return nil
	}
	down, err := batchExecution(draft, input.SqlDown)
	if err != nil || down.Table != up.Table {
		return domain.ErrInvalidDraftInput
	}
	return nil
}

func (w *ExecutionWorker) executePersistedBatch(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, db domain.Database, executor ports.BatchExecutor) error {
	logBatchStart(ctx, task, draft)
	up, err := batchExecution(draft, draft.SqlUp)
	if err != nil {
		w.handleFailure(ctx, task, err.Error())
		return err
	}
	if task.Operation == ports.MetricOpRevert {
		return w.revertPersistedBatch(ctx, task, draft, db, executor, up)
	}
	plan, err := w.prepareExecution(ctx, task, draft, db)
	if err != nil {
		return err
	}
	if strings.TrimSpace(draft.SqlDown) == "" {
		plan.audit.IsReversible = false
		plan.audit.IrreversibleReason = "No batch SQL down was approved"
	}
	up.Resume = task.Operation == ports.MetricOpResume
	if isRollbackOrForward(draft) {
		source, err := w.batchTraversalSource(ctx, draft)
		if err != nil {
			return err
		}
		up.SourceKey = source.Key
		up.Reverse = draft.Operation == domain.DraftOperationRollback
	}
	total, err := executor.PrepareBatch(ctx, db, up)
	if err != nil {
		if !up.Resume {
			w.handleFailure(context.WithoutCancel(ctx), task, err.Error())
		}
		return err
	}
	if err := w.applyPersistedBatches(ctx, task, draft, func(number int) (*ports.BatchReceipt, error) {
		return executor.ExecuteBatchChunk(ctx, db, up, number, false)
	}, int((total+int64(up.Size)-1)/int64(up.Size))); err != nil {
		return err
	}
	return w.finishExecution(ctx, task, draft, db, plan)
}

func (w *ExecutionWorker) batchTraversalSource(ctx context.Context, draft *domain.Draft) (ports.BatchExecution, error) {
	node, err := w.migrationRepo.FindByIDAndDatabaseID(ctx, draft.TargetID, draft.DatabaseID)
	if err != nil {
		return ports.BatchExecution{}, err
	}
	if node.DraftID == nil {
		return ports.BatchExecution{}, domain.ErrNoCheckpointToResume
	}
	original, err := w.draftRepo.FindByID(ctx, *node.DraftID)
	if err != nil {
		return ports.BatchExecution{}, err
	}
	return batchExecution(original, original.SqlUp)
}

func (w *ExecutionWorker) applyPersistedBatches(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, apply func(int) (*ports.BatchReceipt, error), count int) error {
	checkpoints, err := w.draftRepo.FindCheckpointsByDraftID(ctx, draft.ID)
	if err != nil {
		return err
	}
	committed := make(map[int]bool, len(checkpoints))
	cursor := ""
	for _, checkpoint := range checkpoints {
		if checkpoint.Status == "committed" {
			committed[checkpoint.BatchNumber] = true
			cursor = checkpoint.CursorLastID
		}
	}
	for number := 1; number <= count; number++ {
		if committed[number] {
			continue
		}
		receipt, err := w.retryPersistedChunk(ctx, func() (*ports.BatchReceipt, error) { return apply(number) })
		if err != nil {
			if !outcomeUnknown(err) {
				w.handleBatchFailure(context.WithoutCancel(ctx), task, draft, number, cursor, err.Error())
			}
			return err
		}
		if receipt == nil {
			return domain.ErrMigrationOutcomeUnknown
		}
		cursor = receipt.Cursor
		if err := w.saveBatchReceipt(ctx, task, receipt); err != nil {
			return err
		}
	}
	return nil
}

func (w *ExecutionWorker) saveBatchReceipt(ctx context.Context, task ports.ExecutionTask, receipt *ports.BatchReceipt) error {
	_, err := w.draftRepo.SaveCheckpoint(ctx, &domain.DraftCheckpoint{
		DraftID: task.DraftID, ExecutionID: task.ExecutionID, BatchNumber: receipt.Number,
		CursorLastID: receipt.Cursor, RowsAffected: receipt.Rows, Status: "committed",
	})
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrMigrationOutcomeUnknown, err)
	}
	w.metrics.RecordBatchChunk(ports.MetricStatusCommitted, receipt.Rows, task.Operation)
	return nil
}

func (w *ExecutionWorker) revertPersistedBatch(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, db domain.Database, executor ports.BatchExecutor, up ports.BatchExecution) error {
	down, err := batchExecution(draft, draft.SqlDown)
	if err != nil || !draft.IsReversible || up.Table != down.Table {
		w.handleRevertFailure(context.WithoutCancel(ctx), task, draft, 0, domain.ErrMigrationIrreversible.Error())
		return domain.ErrMigrationIrreversible
	}
	checkpoints, err := w.draftRepo.FindCheckpointsByDraftID(ctx, draft.ID)
	if err != nil {
		return err
	}
	for _, checkpoint := range filterCommittedCheckpointsReverse(checkpoints) {
		_, err := w.retryPersistedChunk(ctx, func() (*ports.BatchReceipt, error) {
			return executor.ExecuteBatchChunk(ctx, db, down, checkpoint.BatchNumber, true)
		})
		if err != nil {
			if !outcomeUnknown(err) {
				w.handleRevertFailure(context.WithoutCancel(ctx), task, draft, checkpoint.BatchNumber, err.Error())
			}
			return err
		}
		checkpoint.Status = "reverted"
		if _, err := w.draftRepo.UpdateCheckpoint(ctx, checkpoint); err != nil {
			return fmt.Errorf("%w: %w", domain.ErrMigrationOutcomeUnknown, err)
		}
		w.metrics.RecordBatchChunk(ports.MetricStatusReverted, checkpoint.RowsAffected, task.Operation)
	}
	return w.finalizeRevertSuccess(ctx, task, draft)
}

func (w *ExecutionWorker) retryPersistedChunk(ctx context.Context, run func() (*ports.BatchReceipt, error)) (*ports.BatchReceipt, error) {
	for attempt := 0; ; attempt++ {
		receipt, err := run()
		if !isTransientError(err) || attempt >= w.maxRetries {
			if err != nil && outcomeUnknown(err) {
				return nil, fmt.Errorf("%w: %w", domain.ErrMigrationOutcomeUnknown, err)
			}
			return receipt, err
		}
		timer := time.NewTimer(w.backoffBase * time.Duration(1<<attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
