package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type DraftService interface {
	CreateDraft(ctx context.Context, input domain.CreateDraftInput) (*domain.Draft, error)
	CreateAutoDraft(ctx context.Context, databaseID, userID uint, operation string) (*domain.Draft, error)
	GetDraftsByDatabaseID(ctx context.Context, databaseID uint, userID uint, params pagination.PaginationParams) ([]*domain.Draft, int64, error)
	GetDraftByID(ctx context.Context, draftID uint, userID uint) (*domain.Draft, error)
	SubmitApproval(ctx context.Context, input domain.SubmitApprovalInput) (*domain.Draft, error)
	VoteDraft(ctx context.Context, databaseID, draftID, userID uint, approve bool) error
	ValidateDraft(ctx context.Context, draftID uint) (*domain.Draft, error)
	ExecuteDraft(ctx context.Context, draftID uint, userID uint) error
	ExecuteMigrationDraft(ctx context.Context, databaseID, draftID, userID uint) (*domain.Migration, error)
	UnlockDatabase(databaseID uint)
	ResumeBatch(ctx context.Context, draftID uint, userID uint) error
	RevertBatch(ctx context.Context, draftID uint, userID uint) error
	GetBatchProgress(ctx context.Context, draftID uint, userID uint) (*domain.BatchProgress, error)
}

type DraftRepository interface {
	WithTransaction(ctx context.Context, fn func(txRepo DraftRepository) error) error
	Save(ctx context.Context, draft *domain.Draft) (*domain.Draft, error)
	Update(ctx context.Context, draft *domain.Draft) (*domain.Draft, error)
	FindByID(ctx context.Context, id uint) (*domain.Draft, error)
	FindAllByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Draft, int64, error)
	SaveApproval(ctx context.Context, approval *domain.DraftApproval) (*domain.DraftApproval, error)
	UpdateApproval(ctx context.Context, approval *domain.DraftApproval) (*domain.DraftApproval, error)
	FindApprovalsByDraftID(ctx context.Context, draftID uint) ([]*domain.DraftApproval, error)
	FindApprovalsByDraftAndRound(ctx context.Context, draftID uint, round uint) ([]*domain.DraftApproval, error)
	FindApprovalByDraftAndGroup(ctx context.Context, draftID uint, groupID uint) (*domain.DraftApproval, error)
	FindApprovalByDraftGroupAndRound(ctx context.Context, draftID uint, groupID uint, round uint) (*domain.DraftApproval, error)
	FindDraftByStatusAndDatabaseID(ctx context.Context, databaseID uint, status string) (*domain.Draft, error)
	SaveCheckpoint(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error)
	UpdateCheckpoint(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error)
	FindCheckpointsByDraftID(ctx context.Context, draftID uint) ([]*domain.DraftCheckpoint, error)
	FindLastCommittedCheckpoint(ctx context.Context, draftID uint) (*domain.DraftCheckpoint, error)
	CreateApprovalsForRound(ctx context.Context, approvals []domain.DraftApproval) error
}
