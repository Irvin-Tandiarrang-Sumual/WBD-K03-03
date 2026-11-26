package draft

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
)

type migrationExecution struct {
	audit   *domain.Migration
	head    *uint
	parent  *uint
	forward *uint
}

func requireConsensus(ctx context.Context, repo ports.MigrationRepository, draft *domain.Draft) error {
	if repo == nil {
		return nil
	}
	approved, err := repo.HasConsensus(ctx, draft.DatabaseID, draft.ID)
	if err != nil {
		return err
	}
	if !approved {
		return domain.ErrDraftNotApproved
	}
	return nil
}

func (w *ExecutionWorker) revalidateExecution(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft) error {
	if draft.DatabaseID != task.DatabaseID {
		return domain.ErrInvalidDraftState
	}
	if w.userGroupService != nil {
		if err := w.userGroupService.Authorize(ctx, draft.UserID, task.DatabaseID, domain.AccessViewEdit); err != nil {
			return err
		}
		if task.UserID != draft.UserID {
			if err := w.userGroupService.Authorize(ctx, task.UserID, task.DatabaseID, domain.AccessViewEdit); err != nil {
				return err
			}
		}
	}
	return requireConsensus(ctx, w.migrationRepo, draft)
}

func (w *ExecutionWorker) prepareExecution(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, db domain.Database) (*migrationExecution, error) {
	if err := completePendingSnapshot(ctx, w.databaseDriver, w.migrationRepo, db); err != nil {
		return nil, err
	}
	audit := &domain.Migration{
		DatabaseID:         task.DatabaseID,
		UserID:             task.UserID,
		DraftID:            &task.DraftID,
		ExecutionID:        task.ExecutionID,
		SqlUp:              draft.SqlUp,
		SqlDown:            draft.SqlDown,
		IsReversible:       draft.IsReversible,
		IrreversibleReason: draft.IrreversibleReason,
	}
	if isRollbackOrForward(draft) {
		return w.prepareTraversal(ctx, draft, db, audit)
	}

	latest, parent, err := w.findMigrationParent(ctx, draft.UserID, db)
	if err != nil {
		return nil, err
	}
	version, err := strconv.Atoi(latest.Version)
	if err != nil {
		return nil, domain.ErrMigrationVersionUnavailable
	}
	audit.Version = strconv.Itoa(version + 1)
	audit.ParentMigrationID = parent
	return &migrationExecution{audit: audit, parent: parent}, nil
}

func (w *ExecutionWorker) findMigrationParent(ctx context.Context, userID uint, db domain.Database) (*domain.Migration, *uint, error) {
	latest, err := w.migrationRepo.FindLatestByDatabaseID(ctx, db.ID)
	if err != nil && !errors.Is(err, domain.ErrMigrationNotFound) {
		return nil, nil, err
	}
	if latest == nil {
		latest, err = w.saveBaseline(ctx, userID, db)
		if err != nil {
			return nil, nil, err
		}
	}
	parent := db.CurrentMigrationID
	if parent == nil {
		parent = &latest.ID
	}
	return latest, parent, nil
}

func (w *ExecutionWorker) saveBaseline(ctx context.Context, userID uint, db domain.Database) (*domain.Migration, error) {
	snapshot, schema, err := migration.CaptureDatabaseSnapshot(ctx, w.databaseDriver, db)
	if err != nil {
		return nil, err
	}
	if len(snapshot) == 0 {
		return nil, domain.ErrMigrationSnapshotUnavailable
	}
	return w.migrationRepo.Save(ctx, &domain.Migration{
		DatabaseID:         db.ID,
		UserID:             userID,
		Version:            "1",
		Status:             domain.MigrationStatusSuccess,
		SqlUp:              migration.FormatSnapshotAsBaselineSQL(schema),
		Snapshot:           snapshot,
		IrreversibleReason: "Initial schema baseline",
	})
}

func (w *ExecutionWorker) prepareTraversal(ctx context.Context, draft *domain.Draft, db domain.Database, audit *domain.Migration) (*migrationExecution, error) {
	if db.LatestExecutionID != audit.ExecutionID || draft.BaseExecutionID+1 != audit.ExecutionID {
		return nil, domain.ErrDraftConflict
	}
	pos := domain.MigrationPosition{HeadID: db.CurrentMigrationID, ExecutionID: db.LatestExecutionID}
	node, err := resolveTargetMigration(ctx, w.migrationRepo, db.ID, pos, draft.Operation)
	if err != nil {
		return nil, err
	}
	if node.ID != draft.TargetID {
		return nil, domain.ErrDraftConflict
	}
	audit.Version = node.Version
	audit.SourceMigrationID = &node.ID
	audit.IsReversible = node.IsReversible
	audit.IrreversibleReason = node.IrreversibleReason

	if draft.Operation == domain.DraftOperationForward {
		return &migrationExecution{audit: audit, head: &node.ID}, nil
	}
	activeExecution, err := w.migrationRepo.FindActiveExecution(ctx, node.ID)
	if err != nil {
		return nil, err
	}
	audit.RevertsMigrationID = &activeExecution
	return &migrationExecution{audit: audit, head: node.ParentMigrationID, parent: node.ParentMigrationID, forward: &node.ID}, nil
}

func (w *ExecutionWorker) finishExecution(ctx context.Context, task ports.ExecutionTask, draft *domain.Draft, db domain.Database, plan *migrationExecution) error {
	if plan == nil {
		return w.handleSuccess(ctx, task, draft)
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	plan.audit.MigratedAt = time.Now()
	captureAppliedSnapshot(finishCtx, w.databaseDriver, db, plan.audit)
	return w.migrationRepo.FinishDraft(finishCtx, task.DraftID, plan.audit, plan.head, plan.parent, plan.forward)
}
