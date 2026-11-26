package testutil

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type DraftRepository struct {
	ports.DraftRepository
	WithTransactionFunc             func(context.Context, func(ports.DraftRepository) error) error
	SaveFunc                        func(context.Context, *domain.Draft) (*domain.Draft, error)
	UpdateFunc                      func(context.Context, *domain.Draft) (*domain.Draft, error)
	FindByIDFunc                    func(context.Context, uint) (*domain.Draft, error)
	FindAllByDatabaseIDFunc         func(context.Context, uint, pagination.PaginationParams) ([]*domain.Draft, int64, error)
	SaveApprovalFunc                func(context.Context, *domain.DraftApproval) (*domain.DraftApproval, error)
	UpdateApprovalFunc              func(context.Context, *domain.DraftApproval) (*domain.DraftApproval, error)
	FindApprovalsByDraftIDFunc           func(context.Context, uint) ([]*domain.DraftApproval, error)
	FindApprovalsByDraftAndRoundFunc     func(context.Context, uint, uint) ([]*domain.DraftApproval, error)
	FindApprovalByDraftAndGroupFunc      func(context.Context, uint, uint) (*domain.DraftApproval, error)
	FindApprovalByDraftGroupAndRoundFunc func(context.Context, uint, uint, uint) (*domain.DraftApproval, error)
	FindDraftByStatusAndDatabaseIDFunc   func(context.Context, uint, string) (*domain.Draft, error)
	SaveCheckpointFunc                   func(context.Context, *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error)
	UpdateCheckpointFunc                 func(context.Context, *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error)
	FindCheckpointsByDraftIDFunc         func(context.Context, uint) ([]*domain.DraftCheckpoint, error)
	FindLastCommittedCheckpointFunc      func(context.Context, uint) (*domain.DraftCheckpoint, error)
	CreateApprovalsForRoundFunc          func(context.Context, []domain.DraftApproval) error
}

func (f *DraftRepository) FindApprovalsByDraftAndRound(ctx context.Context, draftID uint, round uint) ([]*domain.DraftApproval, error) {
	if f.FindApprovalsByDraftAndRoundFunc != nil {
		return f.FindApprovalsByDraftAndRoundFunc(ctx, draftID, round)
	}
	return nil, nil
}

func (f *DraftRepository) FindApprovalByDraftGroupAndRound(ctx context.Context, draftID uint, groupID uint, round uint) (*domain.DraftApproval, error) {
	if f.FindApprovalByDraftGroupAndRoundFunc != nil {
		return f.FindApprovalByDraftGroupAndRoundFunc(ctx, draftID, groupID, round)
	}
	return nil, nil
}

func (f *DraftRepository) SaveCheckpoint(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
	if f.SaveCheckpointFunc != nil {
		return f.SaveCheckpointFunc(ctx, cp)
	}
	return cp, nil
}

func (f *DraftRepository) UpdateCheckpoint(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
	if f.UpdateCheckpointFunc != nil {
		return f.UpdateCheckpointFunc(ctx, cp)
	}
	return cp, nil
}

func (f *DraftRepository) FindCheckpointsByDraftID(ctx context.Context, draftID uint) ([]*domain.DraftCheckpoint, error) {
	if f.FindCheckpointsByDraftIDFunc != nil {
		return f.FindCheckpointsByDraftIDFunc(ctx, draftID)
	}
	return nil, nil
}

func (f *DraftRepository) FindLastCommittedCheckpoint(ctx context.Context, draftID uint) (*domain.DraftCheckpoint, error) {
	if f.FindLastCommittedCheckpointFunc != nil {
		return f.FindLastCommittedCheckpointFunc(ctx, draftID)
	}
	return nil, nil
}

func (f *DraftRepository) CreateApprovalsForRound(ctx context.Context, approvals []domain.DraftApproval) error {
	if f.CreateApprovalsForRoundFunc != nil {
		return f.CreateApprovalsForRoundFunc(ctx, approvals)
	}
	return nil
}

func (f *DraftRepository) FindDraftByStatusAndDatabaseID(ctx context.Context, databaseID uint, status string) (*domain.Draft, error) {
	if f.FindDraftByStatusAndDatabaseIDFunc != nil {
		return f.FindDraftByStatusAndDatabaseIDFunc(ctx, databaseID, status)
	}
	return nil, nil
}

func (f *DraftRepository) WithTransaction(ctx context.Context, fn func(ports.DraftRepository) error) error {
	if f.WithTransactionFunc != nil {
		return f.WithTransactionFunc(ctx, fn)
	}
	return fn(f)
}

func (f *DraftRepository) Save(ctx context.Context, draft *domain.Draft) (*domain.Draft, error) {
	if f.SaveFunc != nil {
		return f.SaveFunc(ctx, draft)
	}
	return draft, nil
}

func (f *DraftRepository) Update(ctx context.Context, draft *domain.Draft) (*domain.Draft, error) {
	if f.UpdateFunc != nil {
		return f.UpdateFunc(ctx, draft)
	}
	return draft, nil
}

func (f *DraftRepository) FindByID(ctx context.Context, id uint) (*domain.Draft, error) {
	if f.FindByIDFunc != nil {
		return f.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (f *DraftRepository) FindAllByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Draft, int64, error) {
	if f.FindAllByDatabaseIDFunc != nil {
		return f.FindAllByDatabaseIDFunc(ctx, databaseID, params)
	}
	return nil, 0, nil
}

func (f *DraftRepository) SaveApproval(ctx context.Context, approval *domain.DraftApproval) (*domain.DraftApproval, error) {
	if f.SaveApprovalFunc != nil {
		return f.SaveApprovalFunc(ctx, approval)
	}
	return approval, nil
}

func (f *DraftRepository) UpdateApproval(ctx context.Context, approval *domain.DraftApproval) (*domain.DraftApproval, error) {
	if f.UpdateApprovalFunc != nil {
		return f.UpdateApprovalFunc(ctx, approval)
	}
	return approval, nil
}

func (f *DraftRepository) FindApprovalsByDraftID(ctx context.Context, draftID uint) ([]*domain.DraftApproval, error) {
	if f.FindApprovalsByDraftIDFunc != nil {
		return f.FindApprovalsByDraftIDFunc(ctx, draftID)
	}
	return nil, nil
}

func (f *DraftRepository) FindApprovalByDraftAndGroup(ctx context.Context, draftID uint, groupID uint) (*domain.DraftApproval, error) {
	if f.FindApprovalByDraftAndGroupFunc != nil {
		return f.FindApprovalByDraftAndGroupFunc(ctx, draftID, groupID)
	}
	return nil, nil
}

type DraftService struct {
	ports.DraftService
	CreateDraftFunc           func(context.Context, domain.CreateDraftInput) (*domain.Draft, error)
	GetDraftsByDatabaseIDFunc func(context.Context, uint, uint, pagination.PaginationParams) ([]*domain.Draft, int64, error)
	GetDraftByIDFunc          func(context.Context, uint, uint) (*domain.Draft, error)
	SubmitApprovalFunc        func(context.Context, domain.SubmitApprovalInput) (*domain.Draft, error)
	ValidateDraftFunc         func(context.Context, uint) (*domain.Draft, error)
	ExecuteDraftFunc          func(context.Context, uint, uint) error
	UnlockDatabaseFunc        func(databaseID uint)
	ResumeBatchFunc           func(context.Context, uint, uint) error
	RevertBatchFunc           func(context.Context, uint, uint) error
	GetBatchProgressFunc      func(context.Context, uint, uint) (*domain.BatchProgress, error)
}

func (f *DraftService) ResumeBatch(ctx context.Context, draftID uint, userID uint) error {
	if f.ResumeBatchFunc != nil {
		return f.ResumeBatchFunc(ctx, draftID, userID)
	}
	return nil
}

func (f *DraftService) RevertBatch(ctx context.Context, draftID uint, userID uint) error {
	if f.RevertBatchFunc != nil {
		return f.RevertBatchFunc(ctx, draftID, userID)
	}
	return nil
}

func (f *DraftService) GetBatchProgress(ctx context.Context, draftID uint, userID uint) (*domain.BatchProgress, error) {
	if f.GetBatchProgressFunc != nil {
		return f.GetBatchProgressFunc(ctx, draftID, userID)
	}
	return nil, nil
}

func (f *DraftService) CreateDraft(ctx context.Context, input domain.CreateDraftInput) (*domain.Draft, error) {
	if f.CreateDraftFunc != nil {
		return f.CreateDraftFunc(ctx, input)
	}
	return nil, nil
}

func (f *DraftService) GetDraftsByDatabaseID(ctx context.Context, databaseID uint, userID uint, params pagination.PaginationParams) ([]*domain.Draft, int64, error) {
	if f.GetDraftsByDatabaseIDFunc != nil {
		return f.GetDraftsByDatabaseIDFunc(ctx, databaseID, userID, params)
	}
	return nil, 0, nil
}

func (f *DraftService) GetDraftByID(ctx context.Context, draftID uint, userID uint) (*domain.Draft, error) {
	if f.GetDraftByIDFunc != nil {
		return f.GetDraftByIDFunc(ctx, draftID, userID)
	}
	return nil, nil
}

func (f *DraftService) SubmitApproval(ctx context.Context, input domain.SubmitApprovalInput) (*domain.Draft, error) {
	if f.SubmitApprovalFunc != nil {
		return f.SubmitApprovalFunc(ctx, input)
	}
	return nil, nil
}

func (f *DraftService) ValidateDraft(ctx context.Context, draftID uint) (*domain.Draft, error) {
	if f.ValidateDraftFunc != nil {
		return f.ValidateDraftFunc(ctx, draftID)
	}
	return nil, nil
}

func (f *DraftService) ExecuteDraft(ctx context.Context, draftID uint, userID uint) (string, error) {
	if f.ExecuteDraftFunc != nil {
		return "", f.ExecuteDraftFunc(ctx, draftID, userID)
	}
	return "", nil
}

func (f *DraftService) UnlockDatabase(databaseID uint) {
	if f.UnlockDatabaseFunc != nil {
		f.UnlockDatabaseFunc(databaseID)
	}
}
