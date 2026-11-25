package draft

import (
	"context"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type draftService struct {
	draftRepo        ports.DraftRepository
	groupRepo        ports.GroupRepository
	userGroupService ports.UserGroupService
}

var _ ports.DraftService = (*draftService)(nil)

func NewDraftService(
	draftRepo ports.DraftRepository,
	groupRepo ports.GroupRepository,
	userGroupService ports.UserGroupService,
) ports.DraftService {
	return &draftService{
		draftRepo:        draftRepo,
		groupRepo:        groupRepo,
		userGroupService: userGroupService,
	}
}

func (s *draftService) CreateDraft(ctx context.Context, input domain.CreateDraftInput) (*domain.Draft, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if err := s.userGroupService.Authorize(ctx, input.UserID, input.DatabaseID, domain.AccessViewEdit); err != nil {
		return nil, domain.ErrForbidden
	}

	groups, err := s.groupRepo.FindAllByDatabaseID(ctx, input.DatabaseID)
	if err != nil {
		return nil, err
	}

	isReversible := true
	if input.IsReversible != nil {
		isReversible = *input.IsReversible
	}
	now := time.Now()

	draft := &domain.Draft{
		DatabaseID:         input.DatabaseID,
		UserID:             input.UserID,
		SqlUp:              input.SqlUp,
		SqlDown:            input.SqlDown,
		IsReversible:       isReversible,
		IrreversibleReason: input.IrreversibleReason,
		IsBatch:            input.IsBatch,
		BatchColumn:        input.BatchColumn,
		BatchSize:          input.BatchSize,
		Status:             domain.DraftStatusPendingApproval,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	var savedDraft *domain.Draft
	err = s.draftRepo.WithTransaction(ctx, func(txRepo ports.DraftRepository) error {
		var saveErr error
		savedDraft, saveErr = txRepo.Save(ctx, draft)
		if saveErr != nil {
			return saveErr
		}

		approvals := make([]domain.DraftApproval, 0, len(groups))
		for _, grp := range groups {
			var leaderID *uint
			if grp.LeaderID != 0 {
				lid := grp.LeaderID
				leaderID = &lid
			}
			approval := &domain.DraftApproval{
				DraftID:  savedDraft.ID,
				GroupID:  grp.ID,
				LeaderID: leaderID,
				Status:   domain.ApprovalStatusPending,
			}
			savedApproval, err := txRepo.SaveApproval(ctx, approval)
			if err != nil {
				return err
			}
			approvals = append(approvals, *savedApproval)
		}

		savedDraft.Approvals = approvals
		return nil
	})
	if err != nil {
		return nil, err
	}

	return savedDraft, nil
}

func (s *draftService) GetDraftsByDatabaseID(
	ctx context.Context,
	databaseID uint,
	userID uint,
	params pagination.PaginationParams,
) ([]*domain.Draft, int64, error) {
	if databaseID == 0 || userID == 0 {
		return nil, 0, domain.ErrInvalidDraftInput
	}

	if err := s.userGroupService.Authorize(ctx, userID, databaseID, domain.AccessViewOnly); err != nil {
		return nil, 0, domain.ErrForbidden
	}

	return s.draftRepo.FindAllByDatabaseID(ctx, databaseID, params)
}

func (s *draftService) GetDraftByID(ctx context.Context, draftID uint, userID uint) (*domain.Draft, error) {
	if draftID == 0 || userID == 0 {
		return nil, domain.ErrInvalidDraftInput
	}

	draft, err := s.draftRepo.FindByID(ctx, draftID)
	if err != nil {
		return nil, err
	}

	if err := s.userGroupService.Authorize(ctx, userID, draft.DatabaseID, domain.AccessViewOnly); err != nil {
		return nil, domain.ErrForbidden
	}

	return draft, nil
}

func (s *draftService) SubmitApproval(ctx context.Context, input domain.SubmitApprovalInput) (*domain.Draft, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	var resultDraft *domain.Draft
	var invalidated bool

	err := s.draftRepo.WithTransaction(ctx, func(txRepo ports.DraftRepository) error {
		draft, err := txRepo.FindByID(ctx, input.DraftID)
		if err != nil {
			return err
		}

		if draft.Status != domain.DraftStatusPendingApproval {
			return domain.ErrInvalidDraftState
		}

		if s.userGroupService != nil {
			if authErr := s.userGroupService.Authorize(ctx, draft.UserID, draft.DatabaseID, domain.AccessViewEdit); authErr != nil {
				draft.Status = domain.DraftStatusInvalidated
				draft.UpdatedAt = time.Now()
				if _, updErr := txRepo.Update(ctx, draft); updErr != nil {
					return updErr
				}
				invalidated = true
				return nil
			}
		}

		approvals, err := txRepo.FindApprovalsByDraftID(ctx, draft.ID)
		if err != nil {
			return err
		}

		var matchedApprovals []*domain.DraftApproval
		for _, app := range approvals {
			if app.LeaderID != nil && *app.LeaderID == input.UserID {
				matchedApprovals = append(matchedApprovals, app)
			}
		}

		if len(matchedApprovals) == 0 {
			return domain.ErrUnauthorized
		}

		now := time.Now()
		for _, app := range matchedApprovals {
			app.Status = input.Status
			app.RespondedAt = &now
			if _, err := txRepo.UpdateApproval(ctx, app); err != nil {
				return err
			}
		}

		if input.Status == domain.ApprovalStatusRejected {
			draft.Status = domain.DraftStatusRejected
			draft.UpdatedAt = now
			updatedDraft, err := txRepo.Update(ctx, draft)
			if err != nil {
				return err
			}
			resultDraft = updatedDraft
			return nil
		}

		updatedApprovals, err := txRepo.FindApprovalsByDraftID(ctx, draft.ID)
		if err != nil {
			return err
		}

		allApproved := len(updatedApprovals) > 0
		for _, app := range updatedApprovals {
			if app.Status != domain.ApprovalStatusApproved {
				allApproved = false
				break
			}
		}

		if allApproved {
			draft.Status = domain.DraftStatusApproved
			draft.UpdatedAt = now
			updatedDraft, err := txRepo.Update(ctx, draft)
			if err != nil {
				return err
			}
			resultDraft = updatedDraft
			return nil
		}

		draft.Approvals = make([]domain.DraftApproval, 0, len(updatedApprovals))
		for _, app := range updatedApprovals {
			draft.Approvals = append(draft.Approvals, *app)
		}
		resultDraft = draft
		return nil
	})
	if err != nil {
		return nil, err
	}
	if invalidated {
		return nil, domain.ErrInvalidDraftState
	}

	return resultDraft, nil
}

func (s *draftService) ValidateDraft(ctx context.Context, draftID uint) (*domain.Draft, error) {
	if draftID == 0 {
		return nil, domain.ErrInvalidDraftInput
	}

	draft, err := s.draftRepo.FindByID(ctx, draftID)
	if err != nil {
		return nil, err
	}

	if draft.IsTerminal() {
		return draft, nil
	}

	if err := s.userGroupService.Authorize(ctx, draft.UserID, draft.DatabaseID, domain.AccessViewEdit); err != nil {
		draft.Status = domain.DraftStatusInvalidated
		draft.UpdatedAt = time.Now()
		return s.draftRepo.Update(ctx, draft)
	}

	return draft, nil
}

func (s *draftService) ExecuteDraft(ctx context.Context, draftID uint, userID uint) error {
	if draftID == 0 || userID == 0 {
		return domain.ErrInvalidDraftInput
	}

	validatedDraft, err := s.ValidateDraft(ctx, draftID)
	if err != nil {
		return err
	}

	if validatedDraft.Status != domain.DraftStatusApproved {
		return domain.ErrDraftNotApproved
	}

	if err := s.userGroupService.Authorize(ctx, userID, validatedDraft.DatabaseID, domain.AccessViewEdit); err != nil {
		return domain.ErrForbidden
	}

	validatedDraft.Status = domain.DraftStatusExecuting
	validatedDraft.UpdatedAt = time.Now()
	_, err = s.draftRepo.Update(ctx, validatedDraft)
	return err
}
