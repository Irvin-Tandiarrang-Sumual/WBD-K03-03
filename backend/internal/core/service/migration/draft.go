package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

func draftTarget(ctx context.Context, repo ports.MigrationRepository, dbID uint, pos domain.MigrationPosition, operation string) (*domain.Migration, error) {
	if pos.HeadID == nil {
		if operation == domain.OperationForward {
			return nil, domain.ErrNoForwardAvailable
		}
		return nil, domain.ErrNoActiveMigration
	}
	active, err := repo.FindByIDAndDatabaseID(ctx, *pos.HeadID, dbID)
	if err != nil {
		return nil, err
	}
	switch operation {
	case domain.OperationRollback:
		if !active.IsReversible {
			return nil, fmt.Errorf("%w: %s", domain.ErrMigrationIrreversible, active.IrreversibleReason)
		}
		if active.ParentMigrationID == nil || active.SqlDown == "" {
			return nil, domain.ErrMigrationRollbackUnavailable
		}
		return active, nil
	case domain.OperationForward:
		if active.ForwardMigrationID == nil {
			return nil, domain.ErrNoForwardAvailable
		}
		return repo.FindByIDAndDatabaseID(ctx, *active.ForwardMigrationID, dbID)
	default:
		return nil, domain.ErrInvalidMigrationInput
	}
}

func (s *migrationService) ProposeMigrationDraft(ctx context.Context, dbID, userID uint, operation string) (*domain.MigrationDraft, error) {
	if err := s.userGroupsService.Authorize(ctx, userID, dbID, domain.AccessViewEdit); err != nil {
		return nil, err
	}
	var draft *domain.MigrationDraft
	err := s.repository.WithPosition(ctx, dbID, func(repo ports.MigrationRepository, pos domain.MigrationPosition) error {
		target, err := draftTarget(ctx, repo, dbID, pos, operation)
		if err != nil {
			return err
		}
		draft = &domain.MigrationDraft{
			DatabaseID:      dbID,
			UserID:          userID,
			TargetID:        target.ID,
			Operation:       operation,
			BaseExecutionID: pos.ExecutionID,
			SqlUp:           target.SqlUp,
			SqlDown:         target.SqlDown,
			Status:          "pending_approval",
		}
		if operation == domain.OperationRollback {
			draft.SqlUp, draft.SqlDown = draft.SqlDown, draft.SqlUp
		}
		return repo.SaveDraft(ctx, draft)
	})
	if err != nil {
		return nil, err
	}
	return draft, nil
}

func (s *migrationService) VoteMigrationDraft(ctx context.Context, dbID, draftID, userID uint, approve bool) error {
	if err := s.userGroupsService.Authorize(ctx, userID, dbID, domain.AccessViewOnly); err != nil {
		return err
	}
	return s.repository.WithPosition(ctx, dbID, func(repo ports.MigrationRepository, pos domain.MigrationPosition) error {
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

func (s *migrationService) ExecuteMigrationDraft(ctx context.Context, dbID, draftID, userID uint) (*domain.Migration, error) {
	if err := s.userGroupsService.Authorize(ctx, userID, dbID, domain.AccessViewEdit); err != nil {
		return nil, err
	}
	db, err := s.databaseRepository.FindWithCredentialByID(ctx, dbID)
	if err != nil {
		return nil, err
	}
	var audit *domain.Migration
	var destination, parent, forward *uint
	err = s.repository.WithPosition(ctx, dbID, func(repo ports.MigrationRepository, pos domain.MigrationPosition) error {
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
		if err = s.userGroupsService.Authorize(ctx, draft.UserID, dbID, domain.AccessViewEdit); err != nil {
			return err
		}
		approved, err := repo.HasConsensus(ctx, dbID, draftID)
		if err != nil {
			return err
		}
		if !approved {
			return domain.ErrDraftConflict
		}
		node, err := draftTarget(ctx, repo, dbID, pos, draft.Operation)
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
		if draft.Operation == domain.OperationRollback {
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
		if statusErr := s.repository.UpdateDraftStatus(context.WithoutCancel(ctx), draftID, "failed"); statusErr != nil {
			return nil, fmt.Errorf("%w (recording failure: %v)", err, statusErr)
		}
		return nil, err
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	s.captureAppliedSnapshot(finishCtx, db, audit)
	err = s.repository.FinishDraft(finishCtx, draftID, audit, destination, parent, forward)
	if err != nil {
		return nil, err
	}
	if audit.Status == domain.MigrationStatusSnapshotPending {
		return nil, domain.ErrMigrationSnapshotUnavailable
	}
	return audit, nil
}
