package draft

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/logger"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
)

type ExecutionWorker struct {
	draftRepo      ports.DraftRepository
	databaseRepo   ports.DatabaseRepository
	migrationRepo  ports.MigrationRepository
	databaseDriver ports.DatabaseDriver
	groupRepo      ports.GroupRepository
	unlockDatabase func(databaseID uint)
	maxRetries     int
	backoffBase    time.Duration
}

func NewExecutionWorker(
	draftRepo ports.DraftRepository,
	databaseRepo ports.DatabaseRepository,
	migrationRepo ports.MigrationRepository,
	databaseDriver ports.DatabaseDriver,
	groupRepo ports.GroupRepository,
	unlockDatabase func(databaseID uint),
) *ExecutionWorker {
	return &ExecutionWorker{
		draftRepo:      draftRepo,
		databaseRepo:   databaseRepo,
		migrationRepo:  migrationRepo,
		databaseDriver: databaseDriver,
		groupRepo:      groupRepo,
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

	ctx, _ = logger.EnsureCorrelationID(ctx)
	if task.UserID != 0 {
		ctx = logger.WithActor(ctx, logger.Actor{UserID: task.UserID})
	}

	slog.InfoContext(ctx, "task execution started",
		"event", "task.execution.start",
		"draft_id", task.DraftID,
		"database_id", task.DatabaseID,
		"execution_id", task.ExecutionID,
		"operation", task.Operation,
	)

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

	if task.Operation == "revert" {
		return w.executeRevert(ctx, task, draft, db)
	}

	if draft.IsBatch {
		return w.executeBatch(ctx, task, draft, db)
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
			backoffDelay := w.backoffBase * time.Duration(1<<attempt)
			slog.WarnContext(ctx, "transient error during execution, retrying with backoff",
				"event", "task.execution.retry",
				"draft_id", task.DraftID,
				"attempt", attempt+1,
				"max_retries", w.maxRetries,
				"backoff_ms", float64(backoffDelay.Nanoseconds())/1e6,
				"error", cmdErr.Error(),
			)
			select {
			case <-ctx.Done():
				w.handleFailure(context.Background(), task, ctx.Err().Error())
				return ctx.Err()
			case <-time.After(backoffDelay):
			}
		}
	}

	w.handleFailure(context.Background(), task, lastErr.Error())
	return lastErr
}

func (w *ExecutionWorker) executeBatch(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, db domain.Database) error {
	slog.InfoContext(ctx, "batch execution started",
		"event", "batch.execution.start",
		"draft_id", draft.ID,
		"database_id", task.DatabaseID,
		"execution_id", task.ExecutionID,
		"batch_size", draft.BatchSize,
		"batch_column", draft.BatchColumn,
		"operation", task.Operation,
	)

	tableName, err := sqlutil.ValidateBatchDML(draft.SqlUp)
	if err != nil {
		w.handleFailure(ctx, task, err.Error())
		return err
	}

	var startBatch int = 1
	var lastCursor string = ""

	if task.Operation == "resume" {
		lastCp, cpErr := w.draftRepo.FindLastCommittedCheckpoint(ctx, draft.ID)
		if cpErr == nil && lastCp != nil {
			startBatch = lastCp.BatchNumber + 1
			lastCursor = lastCp.CursorLastID
			slog.InfoContext(ctx, "resuming batch execution from checkpoint",
				"event", "batch.execution.resume",
				"draft_id", draft.ID,
				"start_batch", startBatch,
				"last_cursor", lastCursor,
			)
		}
	}

	batchSize := draft.BatchSize
	if batchSize <= 0 {
		batchSize = 1000
	}
	batchCol := draft.BatchColumn
	if batchCol == "" {
		batchCol = "id"
	}

	// Pre-flight boundary scan: check target row count
	countQuery := fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tableName)
	if lastCursor != "" {
		countQuery = fmt.Sprintf("SELECT COUNT(*) as count FROM %s WHERE %s > '%s'", tableName, batchCol, lastCursor)
	}
	countRes, queryErr := w.databaseDriver.ExecuteQuery(ctx, db, countQuery)
	if queryErr == nil && countRes != nil && len(countRes.Rows) > 0 {
		if countVal, ok := countRes.Rows[0]["count"]; ok {
			var totalRows int64
			switch v := countVal.(type) {
			case int64:
				totalRows = v
			case float64:
				totalRows = int64(v)
			case int:
				totalRows = int64(v)
			}
			slog.InfoContext(ctx, "batch boundary scan completed",
				"event", "batch.boundary_scan.complete",
				"draft_id", draft.ID,
				"total_rows", totalRows,
			)
			if totalRows == 0 && startBatch == 1 {
				// Empty target rows -> mark applied without creating checkpoints
				return w.handleSuccess(ctx, task, draft)
			}
		}
	}

	currentBatch := startBatch
	cursor := lastCursor

	for {
		select {
		case <-ctx.Done():
			w.handleBatchFailure(context.Background(), task, draft, currentBatch, cursor, ctx.Err().Error())
			return ctx.Err()
		default:
		}

		var rowsAffected int64
		var chunkErr error
		for attempt := 0; attempt <= w.maxRetries; attempt++ {
			select {
			case <-ctx.Done():
				w.handleBatchFailure(context.Background(), task, draft, currentBatch, cursor, ctx.Err().Error())
				return ctx.Err()
			default:
			}

			res, cmdErr := w.databaseDriver.ExecuteCommand(ctx, db, draft.SqlUp)
			if cmdErr == nil {
				if res != nil {
					rowsAffected = res.RowsAffected
				}
				chunkErr = nil
				break
			}
			chunkErr = cmdErr
			if !isTransientError(cmdErr) {
				break
			}
			if attempt < w.maxRetries {
				backoffDelay := w.backoffBase * time.Duration(1<<attempt)
				slog.WarnContext(ctx, "batch chunk execution transient error, retrying with backoff",
					"event", "batch.chunk.retry",
					"draft_id", draft.ID,
					"batch_number", currentBatch,
					"attempt", attempt+1,
					"max_retries", w.maxRetries,
					"backoff_ms", float64(backoffDelay.Nanoseconds())/1e6,
					"error", cmdErr.Error(),
				)
				time.Sleep(backoffDelay)
			}
		}

		if chunkErr != nil {
			w.handleBatchFailure(context.Background(), task, draft, currentBatch, cursor, chunkErr.Error())
			return chunkErr
		}

		if rowsAffected == 0 && currentBatch > 1 {
			break
		}

		newCursor := fmt.Sprintf("%d", currentBatch*batchSize)
		cp := &domain.DraftCheckpoint{
			DraftID:      draft.ID,
			ExecutionID:  task.ExecutionID,
			BatchNumber:  currentBatch,
			CursorLastID: newCursor,
			RowsAffected: rowsAffected,
			Status:       "committed",
			CreatedAt:    time.Now(),
		}
		if _, saveErr := w.draftRepo.SaveCheckpoint(ctx, cp); saveErr != nil {
			w.handleBatchFailure(context.Background(), task, draft, currentBatch, cursor, saveErr.Error())
			return saveErr
		}

		slog.InfoContext(ctx, "batch chunk committed",
			"event", "batch.chunk.complete",
			"draft_id", draft.ID,
			"batch_number", currentBatch,
			"rows_affected", rowsAffected,
			"cursor_last_id", newCursor,
		)

		cursor = newCursor
		currentBatch++

		if rowsAffected < int64(batchSize) {
			break
		}
	}

	return w.handleSuccess(ctx, task, draft)
}

func (w *ExecutionWorker) handleBatchFailure(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, batchNum int, cursor string, errMsg string) {
	slog.ErrorContext(ctx, "batch chunk failed, entering quarantine",
		"event", "batch.chunk.failed",
		"draft_id", draft.ID,
		"execution_id", task.ExecutionID,
		"batch_number", batchNum,
		"cursor", cursor,
		"error", errMsg,
		"quarantine_status", "partial",
	)

	now := time.Now()
	cp := &domain.DraftCheckpoint{
		DraftID:      draft.ID,
		ExecutionID:  task.ExecutionID,
		BatchNumber:  batchNum,
		CursorLastID: cursor,
		RowsAffected: 0,
		Status:       "failed",
		CreatedAt:    now,
	}
	_, _ = w.draftRepo.SaveCheckpoint(ctx, cp)

	draft.Status = domain.DraftStatusPartial
	draft.CurrentRound++
	draft.RecoveryAction = ""
	draft.UpdatedAt = now
	_, _ = w.draftRepo.Update(ctx, draft)

	if w.groupRepo != nil {
		groups, err := w.groupRepo.FindAllByDatabaseID(ctx, draft.DatabaseID)
		if err == nil && len(groups) > 0 {
			approvals := make([]domain.DraftApproval, 0, len(groups))
			for _, grp := range groups {
				var lid *uint
				if grp.LeaderID != 0 {
					l := grp.LeaderID
					lid = &l
				}
				approvals = append(approvals, domain.DraftApproval{
					DraftID:  draft.ID,
					GroupID:  grp.ID,
					LeaderID: lid,
					Round:    draft.CurrentRound,
					Phase:    domain.ApprovalPhasePartialRecovery,
					Status:   domain.ApprovalStatusPending,
					Vote:     domain.ApprovalStatusPending,
				})
			}
			_ = w.draftRepo.CreateApprovalsForRound(ctx, approvals)
		}
	}
}

func (w *ExecutionWorker) executeRevert(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, db domain.Database) error {
	checkpoints, err := w.draftRepo.FindCheckpointsByDraftID(ctx, draft.ID)
	if err != nil {
		w.handleFailure(ctx, task, err.Error())
		return err
	}

	committed := make([]*domain.DraftCheckpoint, 0)
	for i := len(checkpoints) - 1; i >= 0; i-- {
		if checkpoints[i].Status == "committed" {
			committed = append(committed, checkpoints[i])
		}
	}

	slog.InfoContext(ctx, "batch revert execution started",
		"event", "batch.revert.start",
		"draft_id", draft.ID,
		"execution_id", task.ExecutionID,
		"total_chunks_to_revert", len(committed),
	)

	for _, cp := range committed {
		var revertErr error
		for attempt := 0; attempt <= w.maxRetries; attempt++ {
			select {
			case <-ctx.Done():
				w.handleRevertFailure(context.Background(), task, draft, cp.BatchNumber, ctx.Err().Error())
				return ctx.Err()
			default:
			}

			sqlDown := draft.SqlDown
			if sqlDown == "" {
				sqlDown = "/* revert chunk */"
			}
			_, cmdErr := w.databaseDriver.ExecuteCommand(ctx, db, sqlDown)
			if cmdErr == nil {
				revertErr = nil
				break
			}
			revertErr = cmdErr
			if !isTransientError(cmdErr) {
				break
			}
			if attempt < w.maxRetries {
				time.Sleep(w.backoffBase * time.Duration(1<<attempt))
			}
		}

		if revertErr != nil {
			w.handleRevertFailure(context.Background(), task, draft, cp.BatchNumber, revertErr.Error())
			return revertErr
		}

		cp.Status = "reverted"
		_, _ = w.draftRepo.UpdateCheckpoint(ctx, cp)

		slog.InfoContext(ctx, "batch chunk reverted",
			"event", "batch.revert.chunk",
			"draft_id", draft.ID,
			"batch_number", cp.BatchNumber,
			"status", "reverted",
		)
	}

	now := time.Now()
	draft.Status = domain.DraftStatusFailed
	draft.UpdatedAt = now
	_, _ = w.draftRepo.Update(ctx, draft)

	mig := &domain.Migration{
		DatabaseID:   task.DatabaseID,
		UserID:       task.UserID,
		DraftID:      &task.DraftID,
		ExecutionID:  task.ExecutionID,
		Status:       domain.MigrationStatusFail,
		ErrorMessage: "Batch draft was reverted by author",
		MigratedAt:   now,
	}
	if w.migrationRepo != nil {
		_, _ = w.migrationRepo.Save(ctx, mig)
	}

	slog.InfoContext(ctx, "batch revert completed, quarantine lifted",
		"event", "batch.revert.complete",
		"draft_id", draft.ID,
		"execution_id", task.ExecutionID,
	)

	return nil
}

func (w *ExecutionWorker) handleRevertFailure(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, batchNum int, errMsg string) {
	slog.ErrorContext(ctx, "batch revert failed midway, entering partially_reverted quarantine",
		"event", "batch.revert.failed",
		"draft_id", draft.ID,
		"execution_id", task.ExecutionID,
		"batch_number", batchNum,
		"error", errMsg,
		"quarantine_status", "partially_reverted",
	)

	now := time.Now()
	draft.Status = domain.DraftStatusPartiallyReverted
	draft.CurrentRound++
	draft.RecoveryAction = ""
	draft.UpdatedAt = now
	_, _ = w.draftRepo.Update(ctx, draft)

	if w.groupRepo != nil {
		groups, err := w.groupRepo.FindAllByDatabaseID(ctx, draft.DatabaseID)
		if err == nil && len(groups) > 0 {
			approvals := make([]domain.DraftApproval, 0, len(groups))
			for _, grp := range groups {
				var lid *uint
				if grp.LeaderID != 0 {
					l := grp.LeaderID
					lid = &l
				}
				approvals = append(approvals, domain.DraftApproval{
					DraftID:  draft.ID,
					GroupID:  grp.ID,
					LeaderID: lid,
					Round:    draft.CurrentRound,
					Phase:    domain.ApprovalPhaseRevertFailure,
					Status:   domain.ApprovalStatusPending,
					Vote:     domain.ApprovalStatusPending,
				})
			}
			_ = w.draftRepo.CreateApprovalsForRound(ctx, approvals)
		}
	}
}

func (w *ExecutionWorker) handleSuccess(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft) error {
	slog.InfoContext(ctx, "migration execution completed successfully",
		"event", "migration.execute.success",
		"draft_id", draft.ID,
		"database_id", task.DatabaseID,
		"execution_id", task.ExecutionID,
	)

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
	slog.ErrorContext(ctx, "migration execution failed",
		"event", "migration.execute.failed",
		"draft_id", task.DraftID,
		"database_id", task.DatabaseID,
		"execution_id", task.ExecutionID,
		"error", errMsg,
	)

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
