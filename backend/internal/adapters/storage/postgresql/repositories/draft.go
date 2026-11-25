package repositories

import (
	"context"
	"errors"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type draftRepository struct {
	db   *gorm.DB
	inTx bool
}

var _ ports.DraftRepository = (*draftRepository)(nil)

func NewDraftRepository(db *gorm.DB) ports.DraftRepository {
	return &draftRepository{db: db}
}

func (r *draftRepository) WithTransaction(ctx context.Context, fn func(txRepo ports.DraftRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&draftRepository{db: tx, inTx: true})
	})
}

func (r *draftRepository) Save(ctx context.Context, draft *domain.Draft) (*domain.Draft, error) {
	if draft == nil {
		return nil, domain.ErrInvalidDraftInput
	}

	modelDraft := model.Draft{
		ID:                 draft.ID,
		DatabaseID:         draft.DatabaseID,
		UserID:             draft.UserID,
		SqlUp:              draft.SqlUp,
		SqlDown:            draft.SqlDown,
		IsReversible:       draft.IsReversible,
		IrreversibleReason: draft.IrreversibleReason,
		IsBatch:            draft.IsBatch,
		BatchColumn:        draft.BatchColumn,
		BatchSize:          draft.BatchSize,
		Status:             draft.Status,
		CreatedAt:          draft.CreatedAt,
		UpdatedAt:          draft.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&modelDraft).Error; err != nil {
		return nil, err
	}

	return toDomainDraft(&modelDraft), nil
}

func (r *draftRepository) Update(ctx context.Context, draft *domain.Draft) (*domain.Draft, error) {
	if draft == nil || draft.ID == 0 {
		return nil, domain.ErrInvalidDraftInput
	}

	updates := map[string]any{
		"status":              draft.Status,
		"sql_down":            draft.SqlDown,
		"is_reversible":       draft.IsReversible,
		"irreversible_reason": draft.IrreversibleReason,
		"is_batch":            draft.IsBatch,
		"batch_column":        draft.BatchColumn,
		"batch_size":          draft.BatchSize,
		"updated_at":          draft.UpdatedAt,
	}

	result := r.db.WithContext(ctx).Model(&model.Draft{}).Where("id = ?", draft.ID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrDraftNotFound
	}

	return r.FindByID(ctx, draft.ID)
}

func (r *draftRepository) FindByID(ctx context.Context, id uint) (*domain.Draft, error) {
	var modelDraft model.Draft
	query := r.db.WithContext(ctx)
	if r.inTx {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.
		Preload("Approvals").
		Where("id = ?", id).
		First(&modelDraft).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDraftNotFound
		}
		return nil, err
	}

	return toDomainDraft(&modelDraft), nil
}

func (r *draftRepository) FindAllByDatabaseID(
	ctx context.Context,
	databaseID uint,
	params pagination.PaginationParams,
) ([]*domain.Draft, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.Draft{}).
		Where("database_id = ?", databaseID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var modelDrafts []model.Draft
	err := r.db.WithContext(ctx).
		Preload("Approvals").
		Where("database_id = ?", databaseID).
		Order("id DESC").
		Offset(params.Offset()).
		Limit(params.Limit).
		Find(&modelDrafts).Error
	if err != nil {
		return nil, 0, err
	}

	results := make([]*domain.Draft, 0, len(modelDrafts))
	for i := range modelDrafts {
		results = append(results, toDomainDraft(&modelDrafts[i]))
	}

	return results, total, nil
}

func (r *draftRepository) SaveApproval(ctx context.Context, approval *domain.DraftApproval) (*domain.DraftApproval, error) {
	if approval == nil {
		return nil, domain.ErrInvalidDraftInput
	}

	modelApproval := model.DraftApproval{
		ID:          approval.ID,
		DraftID:     approval.DraftID,
		GroupID:     approval.GroupID,
		LeaderID:    approval.LeaderID,
		Status:      approval.Status,
		RespondedAt: approval.RespondedAt,
	}

	if err := r.db.WithContext(ctx).Create(&modelApproval).Error; err != nil {
		return nil, err
	}

	return toDomainDraftApproval(&modelApproval), nil
}

func (r *draftRepository) UpdateApproval(ctx context.Context, approval *domain.DraftApproval) (*domain.DraftApproval, error) {
	if approval == nil || approval.ID == 0 {
		return nil, domain.ErrInvalidDraftInput
	}

	updates := map[string]any{
		"status":       approval.Status,
		"responded_at": approval.RespondedAt,
		"leader_id":    approval.LeaderID,
	}

	result := r.db.WithContext(ctx).Model(&model.DraftApproval{}).Where("id = ?", approval.ID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrDraftNotFound
	}

	var updated model.DraftApproval
	if err := r.db.WithContext(ctx).First(&updated, approval.ID).Error; err != nil {
		return nil, err
	}

	return toDomainDraftApproval(&updated), nil
}

func (r *draftRepository) FindApprovalsByDraftID(ctx context.Context, draftID uint) ([]*domain.DraftApproval, error) {
	var modelApprovals []model.DraftApproval
	err := r.db.WithContext(ctx).
		Where("draft_id = ?", draftID).
		Order("id ASC").
		Find(&modelApprovals).Error
	if err != nil {
		return nil, err
	}

	results := make([]*domain.DraftApproval, 0, len(modelApprovals))
	for i := range modelApprovals {
		results = append(results, toDomainDraftApproval(&modelApprovals[i]))
	}

	return results, nil
}

func (r *draftRepository) FindApprovalByDraftAndGroup(ctx context.Context, draftID uint, groupID uint) (*domain.DraftApproval, error) {
	var modelApproval model.DraftApproval
	err := r.db.WithContext(ctx).
		Where("draft_id = ? AND group_id = ?", draftID, groupID).
		First(&modelApproval).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDraftNotFound
		}
		return nil, err
	}

	return toDomainDraftApproval(&modelApproval), nil
}

func toDomainDraft(m *model.Draft) *domain.Draft {
	if m == nil {
		return nil
	}

	approvals := make([]domain.DraftApproval, 0, len(m.Approvals))
	for i := range m.Approvals {
		approvals = append(approvals, *toDomainDraftApproval(&m.Approvals[i]))
	}

	return &domain.Draft{
		ID:                 m.ID,
		DatabaseID:         m.DatabaseID,
		UserID:             m.UserID,
		SqlUp:              m.SqlUp,
		SqlDown:            m.SqlDown,
		IsReversible:       m.IsReversible,
		IrreversibleReason: m.IrreversibleReason,
		IsBatch:            m.IsBatch,
		BatchColumn:        m.BatchColumn,
		BatchSize:          m.BatchSize,
		Status:             m.Status,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
		Approvals:          approvals,
	}
}

func toDomainDraftApproval(m *model.DraftApproval) *domain.DraftApproval {
	if m == nil {
		return nil
	}

	return &domain.DraftApproval{
		ID:          m.ID,
		DraftID:     m.DraftID,
		GroupID:     m.GroupID,
		LeaderID:    m.LeaderID,
		Status:      m.Status,
		RespondedAt: m.RespondedAt,
	}
}
