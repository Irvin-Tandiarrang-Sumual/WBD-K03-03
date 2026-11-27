package draft

import (
	"context"
	"fmt"

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
		if target.DraftID != nil {
			source, err := s.draftRepo.FindByID(ctx, *target.DraftID)
			if err != nil {
				return err
			}
			draft.IsBatch = source.IsBatch
			draft.BatchColumn = source.BatchColumn
			draft.BatchSize = source.BatchSize
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

func (s *draftService) ExecuteMigrationDraft(ctx context.Context, dbID, draftID, userID uint) (string, error) {
	if s.migrationRepo == nil {
		return "", domain.ErrDatabaseNotFound
	}
	if s.userGroupService != nil {
		if err := s.userGroupService.Authorize(ctx, userID, dbID, domain.AccessViewEdit); err != nil {
			return "", err
		}
	}
	draft, err := s.migrationRepo.FindDraft(ctx, dbID, draftID)
	if err != nil {
		return "", err
	}
	return s.queueApprovedDraft(ctx, draft, userID)
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

func captureAppliedSnapshot(ctx context.Context, driver ports.DatabaseDriver, db domain.Database, m *domain.Migration) {
	snapshot, _, err := migration.CaptureDatabaseSnapshot(ctx, driver, db)
	if err != nil || len(snapshot) == 0 {
		m.Status = domain.MigrationStatusSnapshotPending
		return
	}
	m.Snapshot = snapshot
	m.Status = domain.MigrationStatusSuccess
}

func completePendingSnapshot(ctx context.Context, driver ports.DatabaseDriver, repo ports.MigrationRepository, db domain.Database) error {
	pending, err := repo.FindPendingSnapshot(ctx, db.ID)
	if err != nil || pending == nil {
		return err
	}
	captureAppliedSnapshot(ctx, driver, db, pending)
	if pending.Status != domain.MigrationStatusSuccess {
		return domain.ErrMigrationSnapshotUnavailable
	}
	_, err = repo.Update(ctx, pending)
	return err
}
