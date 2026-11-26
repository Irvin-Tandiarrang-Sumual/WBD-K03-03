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
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ExecutionWorker struct {
	draftRepo      ports.DraftRepository
	databaseRepo   ports.DatabaseRepository
	migrationRepo  ports.MigrationRepository
	databaseDriver ports.DatabaseDriver
	groupRepo      ports.GroupRepository
	unlockDatabase func(databaseID uint)
	metrics        ports.MigrationMetrics
	maxRetries     int
	backoffBase    time.Duration

	userGroupService ports.UserGroupService
}

func NewExecutionWorker(
	draftRepo ports.DraftRepository,
	databaseRepo ports.DatabaseRepository,
	migrationRepo ports.MigrationRepository,
	databaseDriver ports.DatabaseDriver,
	groupRepo ports.GroupRepository,
	unlockDatabase func(databaseID uint),
	metrics ports.MigrationMetrics,
	authorizers ...ports.UserGroupService,
) *ExecutionWorker {
	if metrics == nil {
		metrics = ports.NoopMigrationMetrics{}
	}
	worker := &ExecutionWorker{
		draftRepo:      draftRepo,
		databaseRepo:   databaseRepo,
		migrationRepo:  migrationRepo,
		databaseDriver: databaseDriver,
		groupRepo:      groupRepo,
		unlockDatabase: unlockDatabase,
		metrics:        metrics,
		maxRetries:     3,
		backoffBase:    10 * time.Millisecond,
	}
	if len(authorizers) > 0 {
		worker.userGroupService = authorizers[0]
	}
	return worker
}

func (w *ExecutionWorker) SetRetryPolicy(maxRetries int, backoffBase time.Duration) {
	w.maxRetries = maxRetries
	w.backoffBase = backoffBase
}

func (w *ExecutionWorker) Execute(ctx context.Context, task ports.ExecutionTask) (err error) {
	if w.unlockDatabase != nil {
		defer w.unlockDatabase(task.DatabaseID)
	}

	ctx = telemetry.Extract(ctx, task.TraceCarrier)
	tr := telemetry.Tracer("migration.worker")
	ctx, span := tr.Start(ctx, "worker.execute_task",
		trace.WithAttributes(
			attribute.Int("migration.draft_id", int(task.DraftID)),
			attribute.Int("migration.database_id", int(task.DatabaseID)),
			attribute.Int("migration.execution_id", int(task.ExecutionID)),
			attribute.String("migration.operation", task.Operation),
		),
	)
	start := time.Now()
	op := task.Operation
	if op == "" {
		op = ports.MetricOpExecute
	}

	defer func() {
		duration := time.Since(start).Seconds()
		if err != nil {
			w.metrics.RecordExecution(op, ports.MetricStatusFailed, duration)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			w.metrics.RecordExecution(op, ports.MetricStatusSuccess, duration)
			span.SetStatus(codes.Ok, "success")
		}
		span.End()
	}()

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
			if task.JobID != 0 {
				err = fmt.Errorf("%w: %v", domain.ErrMigrationOutcomeUnknown, panicErr)
				return
			}
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

	if task.Operation == ports.MetricOpExecute && draft.Status != domain.DraftStatusExecuting {
		return domain.ErrInvalidDraftState
	}
	if task.Operation == ports.MetricOpExecute {
		if err := w.revalidateExecution(ctx, task, draft); err != nil {
			w.handleFailure(ctx, task, err.Error())
			return err
		}
	}
	if batches, ok := w.databaseDriver.(ports.BatchExecutor); ok && draft.IsBatch {
		if task.Operation != ports.MetricOpExecute {
			if err := w.revalidateExecution(ctx, task, draft); err != nil {
				return err
			}
		}
		return w.executePersistedBatch(ctx, task, draft, db, batches)
	}
	if task.Operation == "revert" {
		return w.executeRevert(ctx, task, draft, db)
	}

	if draft.IsBatch {
		return w.executeBatch(ctx, task, draft, db)
	}

	if w.migrationRepo == nil {
		return w.executeSingleMigrationWithRetry(ctx, task, draft, db, nil)
	}
	plan, err := w.prepareExecution(ctx, task, draft, db)
	if err != nil {
		w.handleFailure(ctx, task, err.Error())
		return err
	}
	return w.executeSingleMigrationWithRetry(ctx, task, draft, db, plan)
}

func (w *ExecutionWorker) executeSingleMigrationWithRetry(
	ctx context.Context,
	task ports.ExecutionTask,
	draft *domain.Draft,
	db domain.Database,
	plan *migrationExecution,
) error {
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
			return w.finishExecution(ctx, task, draft, db, plan)
		}
		if outcomeUnknown(cmdErr) {
			return fmt.Errorf("%w: %w", domain.ErrMigrationOutcomeUnknown, cmdErr)
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

type batchPlan struct {
	startBatch    int
	lastCursor    string
	batchSize     int
	emptyTerminal bool
}

func (w *ExecutionWorker) executeBatch(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, db domain.Database) error {
	logBatchStart(ctx, task, draft)

	plan, err := w.prepareBatchPlan(ctx, task, draft, db)
	if err != nil {
		w.handleFailure(ctx, task, err.Error())
		return err
	}
	if plan.emptyTerminal {
		return w.handleSuccess(ctx, task, draft)
	}

	currentBatch := plan.startBatch
	cursor := plan.lastCursor

	for {
		if err := ctx.Err(); err != nil {
			w.handleBatchFailure(context.Background(), task, draft, currentBatch, cursor, err.Error())
			return err
		}

		_, newCursor, isTerminal, chunkErr := w.executeAndCommitChunk(ctx, task, draft, db, currentBatch, plan.batchSize)
		if chunkErr != nil {
			if outcomeUnknown(chunkErr) {
				return fmt.Errorf("%w: %w", domain.ErrMigrationOutcomeUnknown, chunkErr)
			}
			w.handleBatchFailure(context.Background(), task, draft, currentBatch, cursor, chunkErr.Error())
			return chunkErr
		}

		cursor = newCursor
		currentBatch++

		if isTerminal {
			break
		}
	}

	return w.handleSuccess(ctx, task, draft)
}

func logBatchStart(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft) {
	slog.InfoContext(ctx, "batch execution started",
		"event", "batch.execution.start",
		"draft_id", draft.ID,
		"database_id", task.DatabaseID,
		"execution_id", task.ExecutionID,
		"batch_size", draft.BatchSize,
		"batch_column", draft.BatchColumn,
		"operation", task.Operation,
	)
}

func (w *ExecutionWorker) executeAndCommitChunk(
	ctx context.Context,
	task ports.ExecutionTask,
	draft *domain.Draft,
	db domain.Database,
	currentBatch int,
	batchSize int,
) (rowsAffected int64, newCursor string, isTerminal bool, err error) {
	rowsAffected, chunkSpan, chunkErr := w.executeChunkWithRetry(ctx, draft, db, currentBatch)
	if chunkErr != nil {
		return 0, "", false, chunkErr
	}

	if rowsAffected == 0 && currentBatch > 1 {
		chunkSpan.SetStatus(codes.Ok, "empty_chunk_terminal")
		chunkSpan.End()
		return 0, "", true, nil
	}

	newCursor = fmt.Sprintf("%d", currentBatch*batchSize)
	if saveErr := w.recordChunkCheckpoint(ctx, task, draft, currentBatch, rowsAffected, newCursor, chunkSpan); saveErr != nil {
		return 0, "", false, fmt.Errorf("%w: %w", domain.ErrMigrationOutcomeUnknown, saveErr)
	}

	return rowsAffected, newCursor, rowsAffected < int64(batchSize), nil
}

func (w *ExecutionWorker) prepareBatchPlan(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, db domain.Database) (*batchPlan, error) {
	tableName, err := sqlutil.ValidateBatchDML(draft.SqlUp)
	if err != nil {
		return nil, err
	}

	startBatch := 1
	lastCursor := ""

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
				return &batchPlan{emptyTerminal: true}, nil
			}
		}
	}

	return &batchPlan{
		startBatch: startBatch,
		lastCursor: lastCursor,
		batchSize:  batchSize,
	}, nil
}

func (w *ExecutionWorker) executeChunkWithRetry(
	ctx context.Context,
	draft *domain.Draft,
	db domain.Database,
	currentBatch int,
) (int64, trace.Span, error) {
	tr := telemetry.Tracer("migration.worker")
	chunkCtx, chunkSpan := tr.Start(ctx, "migration.batch_chunk",
		trace.WithAttributes(
			attribute.Int("migration.batch_number", currentBatch),
			attribute.Int("migration.draft_id", int(draft.ID)),
		),
	)

	var rowsAffected int64
	var chunkErr error

	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		select {
		case <-chunkCtx.Done():
			chunkSpan.RecordError(chunkCtx.Err())
			chunkSpan.SetStatus(codes.Error, chunkCtx.Err().Error())
			chunkSpan.End()
			return 0, nil, chunkCtx.Err()
		default:
		}

		res, cmdErr := w.databaseDriver.ExecuteCommand(chunkCtx, db, draft.SqlUp)
		if cmdErr == nil {
			if res != nil {
				rowsAffected = res.RowsAffected
			}
			return rowsAffected, chunkSpan, nil
		}

		chunkErr = cmdErr
		if !isTransientError(cmdErr) {
			break
		}

		if attempt < w.maxRetries {
			w.metrics.RecordRetry(ports.MetricOpExecute)
			backoffDelay := w.backoffBase * time.Duration(1<<attempt)
			slog.WarnContext(chunkCtx, "batch chunk execution transient error, retrying with backoff",
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

	w.metrics.RecordBatchChunk(ports.MetricStatusFailed, 0, ports.MetricOpExecute)
	chunkSpan.RecordError(chunkErr)
	chunkSpan.SetStatus(codes.Error, chunkErr.Error())
	chunkSpan.End()
	return 0, nil, chunkErr
}

func (w *ExecutionWorker) recordChunkCheckpoint(
	ctx context.Context,
	task ports.ExecutionTask,
	draft *domain.Draft,
	currentBatch int,
	rowsAffected int64,
	newCursor string,
	chunkSpan trace.Span,
) error {
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
		w.metrics.RecordBatchChunk(ports.MetricStatusFailed, 0, ports.MetricOpExecute)
		chunkSpan.RecordError(saveErr)
		chunkSpan.SetStatus(codes.Error, saveErr.Error())
		chunkSpan.End()
		return saveErr
	}

	chunkSpan.SetAttributes(attribute.Int64("migration.rows_affected", rowsAffected))
	chunkSpan.SetStatus(codes.Ok, "committed")
	chunkSpan.End()

	w.metrics.RecordBatchChunk(ports.MetricStatusCommitted, rowsAffected, ports.MetricOpExecute)

	slog.InfoContext(ctx, "batch chunk committed",
		"event", "batch.chunk.complete",
		"draft_id", draft.ID,
		"batch_number", currentBatch,
		"rows_affected", rowsAffected,
		"cursor_last_id", newCursor,
	)

	return nil
}

func (w *ExecutionWorker) handleBatchFailure(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, batchNum int, cursor string, errMsg string) {
	w.metrics.RecordBatchChunk(ports.MetricStatusQuarantined, 0, task.Operation)
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

	committed := filterCommittedCheckpointsReverse(checkpoints)
	slog.InfoContext(ctx, "batch revert execution started",
		"event", "batch.revert.start",
		"draft_id", draft.ID,
		"execution_id", task.ExecutionID,
		"total_chunks_to_revert", len(committed),
	)

	for _, cp := range committed {
		if err := w.revertSingleChunk(ctx, draft, db, cp); err != nil {
			if outcomeUnknown(err) {
				return fmt.Errorf("%w: %w", domain.ErrMigrationOutcomeUnknown, err)
			}
			w.handleRevertFailure(context.Background(), task, draft, cp.BatchNumber, err.Error())
			return err
		}
	}

	return w.finalizeRevertSuccess(ctx, task, draft)
}

func filterCommittedCheckpointsReverse(checkpoints []*domain.DraftCheckpoint) []*domain.DraftCheckpoint {
	committed := make([]*domain.DraftCheckpoint, 0)
	for i := len(checkpoints) - 1; i >= 0; i-- {
		if checkpoints[i].Status == "committed" {
			committed = append(committed, checkpoints[i])
		}
	}
	return committed
}

func (w *ExecutionWorker) revertSingleChunk(
	ctx context.Context,
	draft *domain.Draft,
	db domain.Database,
	cp *domain.DraftCheckpoint,
) error {
	sqlDown := draft.SqlDown
	if sqlDown == "" {
		sqlDown = "/* revert chunk */"
	}

	var revertErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
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
		return revertErr
	}

	cp.Status = "reverted"
	if _, err := w.draftRepo.UpdateCheckpoint(ctx, cp); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrMigrationOutcomeUnknown, err)
	}

	w.metrics.RecordBatchChunk(ports.MetricStatusReverted, 0, ports.MetricOpRevert)
	slog.InfoContext(ctx, "batch chunk reverted",
		"event", "batch.revert.chunk",
		"draft_id", draft.ID,
		"batch_number", cp.BatchNumber,
		"status", "reverted",
	)

	return nil
}

func (w *ExecutionWorker) finalizeRevertSuccess(
	ctx context.Context,
	task ports.ExecutionTask,
	draft *domain.Draft,
) error {
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

	draft, err := w.draftRepo.FindByID(ctx, task.DraftID)
	if err == nil && draft != nil {
		migration.SqlUp, migration.SqlDown = draft.SqlUp, draft.SqlDown
		migration.IsReversible = draft.IsReversible
		migration.IrreversibleReason = draft.IrreversibleReason
	}
	if w.migrationRepo != nil {
		_, _ = w.migrationRepo.Save(ctx, migration)
	}

	if err == nil && draft != nil {
		draft.Status = domain.DraftStatusFailed
		draft.UpdatedAt = now
		_, _ = w.draftRepo.Update(ctx, draft)
	}
}

func isTransientError(err error) bool {
	if err == nil || outcomeUnknown(err) {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "deadlock") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "lock wait timeout")
}
