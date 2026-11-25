package draft

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type dbLock struct {
	mu     sync.Mutex
	locked atomic.Bool
}

func (l *dbLock) TryLock() bool {
	if l.mu.TryLock() {
		l.locked.Store(true)
		return true
	}
	return false
}

func (l *dbLock) Unlock() {
	if l.locked.CompareAndSwap(true, false) {
		l.mu.Unlock()
	}
}

type draftService struct {
	draftRepo        ports.DraftRepository
	groupRepo        ports.GroupRepository
	databaseRepo     ports.DatabaseRepository
	userGroupService ports.UserGroupService
	taskQueue        ports.TaskQueue
	dbLocks          sync.Map
}

var _ ports.DraftService = (*draftService)(nil)

func NewDraftService(
	draftRepo ports.DraftRepository,
	groupRepo ports.GroupRepository,
	userGroupService ports.UserGroupService,
	extra ...any,
) ports.DraftService {
	s := &draftService{
		draftRepo:        draftRepo,
		groupRepo:        groupRepo,
		userGroupService: userGroupService,
	}
	for _, dep := range extra {
		switch d := dep.(type) {
		case ports.DatabaseRepository:
			s.databaseRepo = d
		case ports.TaskQueue:
			s.taskQueue = d
		}
	}
	return s
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

	draft, err := s.draftRepo.FindByID(ctx, draftID)
	if err != nil {
		return err
	}

	if s.databaseRepo != nil {
		db, err := s.databaseRepo.FindById(ctx, draft.DatabaseID)
		if err != nil || db == nil {
			return domain.ErrDatabaseNotFound
		}
	}

	if draft.UserID != userID {
		return domain.ErrForbidden
	}

	if s.userGroupService != nil {
		if authErr := s.userGroupService.Authorize(ctx, draft.UserID, draft.DatabaseID, domain.AccessViewEdit); authErr != nil {
			draft.Status = domain.DraftStatusInvalidated
			draft.UpdatedAt = time.Now()
			_, _ = s.draftRepo.Update(ctx, draft)
			return domain.ErrDraftNotApproved
		}
	}

	if draft.Status != domain.DraftStatusApproved {
		return domain.ErrDraftNotApproved
	}

	if s.draftRepo != nil {
		partiallyReverted, err := s.draftRepo.FindDraftByStatusAndDatabaseID(ctx, draft.DatabaseID, domain.DraftStatusPartiallyReverted)
		if err == nil && partiallyReverted != nil && partiallyReverted.ID != 0 {
			return domain.ErrDatabaseMustResolvePartiallyReverted
		}

		executingDraft, err := s.draftRepo.FindDraftByStatusAndDatabaseID(ctx, draft.DatabaseID, domain.DraftStatusExecuting)
		if err == nil && executingDraft != nil && executingDraft.ID != 0 {
			return domain.ErrMigrationInProgress
		}
	}

	val, _ := s.dbLocks.LoadOrStore(draft.DatabaseID, &dbLock{})
	lock := val.(*dbLock)
	if !lock.TryLock() {
		return domain.ErrMigrationInProgress
	}

	var execID uint
	if s.databaseRepo != nil {
		incID, err := s.databaseRepo.IncrementExecutionID(ctx, draft.DatabaseID)
		if err != nil {
			lock.Unlock()
			return err
		}
		execID = incID
	}

	draft.Status = domain.DraftStatusExecuting
	draft.UpdatedAt = time.Now()
	if _, err := s.draftRepo.Update(ctx, draft); err != nil {
		lock.Unlock()
		return err
	}

	if s.taskQueue != nil {
		task := ports.ExecutionTask{
			DraftID:     draft.ID,
			DatabaseID:  draft.DatabaseID,
			ExecutionID: execID,
			UserID:      userID,
		}
		if err := s.taskQueue.Enqueue(ctx, task); err != nil {
			lock.Unlock()
			return err
		}
	}

	return nil
}

func (s *draftService) UnlockDatabase(databaseID uint) {
	if val, ok := s.dbLocks.Load(databaseID); ok {
		lock := val.(*dbLock)
		lock.Unlock()
	}
}
