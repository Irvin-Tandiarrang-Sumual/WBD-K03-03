package draft

import (
	"context"
	"fmt"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
)

func (s *draftService) CreateAutoDraft(ctx context.Context, dbID, userID uint, operation string) (*domain.Draft, error) {
	if s.userGroupService != nil {
		if err := s.userGroupService.Authorize(ctx, userID, dbID, domain.AccessViewEdit); err != nil {
			return nil, err
		}
	}
	if s.migrationRepo == nil {
		return nil, domain.ErrDatabaseNotFound
	}
	var draft *domain.Draft
	err := s.migrationRepo.WithPosition(ctx, dbID, func(repo ports.MigrationRepository, pos domain.MigrationPosition) error {
		target, err := resolveTargetMigration(ctx, repo, dbID, pos, operation)
		if err != nil {
			return err
		}
		draft = &domain.Draft{
			DatabaseID:      dbID,
			UserID:          userID,
			TargetID:        target.ID,
			Operation:       operation,
			BaseExecutionID: pos.ExecutionID,
			SqlUp:           target.SqlUp,
			SqlDown:         target.SqlDown,
			IsReversible:    target.IsReversible,
			Status:          domain.DraftStatusPendingApproval,
		}
		if operation == domain.DraftOperationRollback {
			draft.SqlUp, draft.SqlDown = draft.SqlDown, draft.SqlUp
		}
		return repo.SaveDraft(ctx, draft)
	})
	if err != nil {
		return nil, err
	}
	return draft, nil
}

func (s *draftService) VoteDraft(ctx context.Context, dbID, draftID, userID uint, approve bool) error {
	if s.userGroupService != nil {
		if err := s.userGroupService.Authorize(ctx, userID, dbID, domain.AccessViewOnly); err != nil {
			return err
		}
	}
	if s.migrationRepo == nil {
		return domain.ErrDatabaseNotFound
	}
	return s.migrationRepo.WithPosition(ctx, dbID, func(repo ports.MigrationRepository, pos domain.MigrationPosition) error {
		draft, err := repo.FindDraft(ctx, dbID, draftID)
		if err != nil {
			return err
		}
		if draft.Status != "pending_approval" || draft.BaseExecutionID != pos.ExecutionID {
			return domain.ErrDraftConflict
		}
		if err := repo.Vote(ctx, draftID, userID, approve); err != nil {
			return err
		}
		if !approve {
			return repo.UpdateDraftStatus(ctx, draftID, "rejected")
		}
		approved, err := repo.HasConsensus(ctx, dbID, draftID)
		if err != nil || !approved {
			return err
		}
		return repo.UpdateDraftStatus(ctx, draftID, "approved")
	})
}

func (s *draftService) ExecuteMigrationDraft(ctx context.Context, dbID, draftID, userID uint) (*domain.Migration, error) {
	if s.userGroupService != nil {
		if err := s.userGroupService.Authorize(ctx, userID, dbID, domain.AccessViewEdit); err != nil {
			return nil, err
		}
	}
	if s.databaseRepo == nil || s.databaseDriver == nil || s.migrationRepo == nil {
		return nil, domain.ErrDatabaseNotFound
	}
	db, err := s.databaseRepo.FindWithCredentialByID(ctx, dbID)
	if err != nil {
		return nil, err
	}
	var audit *domain.Migration
	var destination, parent, forward *uint
	err = s.migrationRepo.WithPosition(ctx, dbID, func(repo ports.MigrationRepository, pos domain.MigrationPosition) error {
		if err := s.completePendingSnapshot(ctx, repo, db); err != nil {
			return err
		}
		draft, err := repo.FindDraft(ctx, dbID, draftID)
		if err != nil {
			return err
		}
		if draft.Status != "approved" || draft.BaseExecutionID != pos.ExecutionID {
			return domain.ErrDraftConflict
		}
		if s.userGroupService != nil {
			if err = s.userGroupService.Authorize(ctx, draft.UserID, dbID, domain.AccessViewEdit); err != nil {
				return err
			}
		}
		approved, err := repo.HasConsensus(ctx, dbID, draftID)
		if err != nil {
			return err
		}
		if !approved {
			return domain.ErrDraftConflict
		}
		node, err := resolveTargetMigration(ctx, repo, dbID, pos, draft.Operation)
		if err != nil {
			return err
		}
		if node.ID != draft.TargetID {
			return domain.ErrDraftConflict
		}
		destination = &node.ID
		audit = &domain.Migration{
			DatabaseID:         dbID,
			UserID:             userID,
			Version:            node.Version,
			SourceMigrationID:  &node.ID,
			DraftID:            &draft.ID,
			ExecutionID:        pos.ExecutionID + 1,
			SqlUp:              draft.SqlUp,
			SqlDown:            draft.SqlDown,
			Status:             domain.MigrationStatusSuccess,
			IsReversible:       node.IsReversible,
			IrreversibleReason: node.IrreversibleReason,
		}
		if draft.Operation == domain.DraftOperationRollback {
			id, err := repo.FindActiveExecution(ctx, node.ID)
			if err != nil {
				return err
			}
			audit.RevertsMigrationID = &id
			destination, parent, forward = node.ParentMigrationID, node.ParentMigrationID, &node.ID
		}
		return repo.UpdateDraftStatus(ctx, draftID, "executing")
	})
	if err != nil {
		return nil, err
	}
	_, err = s.databaseDriver.ExecuteCommand(ctx, db, audit.SqlUp)
	if err != nil {
		if statusErr := s.migrationRepo.UpdateDraftStatus(context.WithoutCancel(ctx), draftID, "failed"); statusErr != nil {
			return nil, fmt.Errorf("%w (recording failure: %v)", err, statusErr)
		}
		return nil, err
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	s.captureAppliedSnapshot(finishCtx, db, audit)
	err = s.migrationRepo.FinishDraft(finishCtx, draftID, audit, destination, parent, forward)
	if err != nil {
		return nil, err
	}
	if audit.Status == domain.MigrationStatusSnapshotPending {
		return nil, domain.ErrMigrationSnapshotUnavailable
	}
	return audit, nil
}

func resolveTargetMigration(ctx context.Context, repo ports.MigrationRepository, dbID uint, pos domain.MigrationPosition, operation string) (*domain.Migration, error) {
	if pos.HeadID == nil {
		if operation == domain.DraftOperationForward {
			return nil, domain.ErrNoForwardAvailable
		}
		return nil, domain.ErrNoActiveMigration
	}
	active, err := repo.FindByIDAndDatabaseID(ctx, *pos.HeadID, dbID)
	if err != nil {
		return nil, err
	}
	switch operation {
	case domain.DraftOperationRollback:
		if !active.IsReversible {
			return nil, fmt.Errorf("%w: %s", domain.ErrMigrationIrreversible, active.IrreversibleReason)
		}
		if active.ParentMigrationID == nil || active.SqlDown == "" {
			return nil, domain.ErrMigrationRollbackUnavailable
		}
		return active, nil
	case domain.DraftOperationForward:
		if active.ForwardMigrationID == nil {
			return nil, domain.ErrNoForwardAvailable
		}
		return repo.FindByIDAndDatabaseID(ctx, *active.ForwardMigrationID, dbID)
	default:
		return nil, domain.ErrInvalidMigrationInput
	}
}

func (s *draftService) captureAppliedSnapshot(ctx context.Context, db domain.Database, m *domain.Migration) {
	snapshot, _, err := migration.CaptureDatabaseSnapshot(ctx, s.databaseDriver, db)
	if err != nil || len(snapshot) == 0 {
		m.Status = domain.MigrationStatusSnapshotPending
		return
	}
	m.Snapshot = snapshot
	m.Status = domain.MigrationStatusSuccess
}

func (s *draftService) completePendingSnapshot(ctx context.Context, repo ports.MigrationRepository, db domain.Database) error {
	pending, err := repo.FindPendingSnapshot(ctx, db.ID)
	if err != nil || pending == nil {
		return err
	}
	s.captureAppliedSnapshot(ctx, db, pending)
	if pending.Status != domain.MigrationStatusSuccess {
		return domain.ErrMigrationSnapshotUnavailable
	}
	_, err = repo.Update(ctx, pending)
	return err
}
