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
		Operation:          draft.Operation,
		TargetID:           draft.TargetID,
		BaseExecutionID:    draft.BaseExecutionID,
		SqlUp:              draft.SqlUp,
		SqlDown:            draft.SqlDown,
		IsReversible:       draft.IsReversible,
		IrreversibleReason: draft.IrreversibleReason,
		IsBatch:            draft.IsBatch,
		BatchColumn:        draft.BatchColumn,
		BatchSize:          draft.BatchSize,
		CurrentRound:       draft.CurrentRound,
		RecoveryAction:     draft.RecoveryAction,
		Status:             draft.Status,
		CreatedAt:          draft.CreatedAt,
		UpdatedAt:          draft.UpdatedAt,
	}
	if modelDraft.CurrentRound == 0 {
		modelDraft.CurrentRound = 1
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
		"operation":           draft.Operation,
		"target_id":           draft.TargetID,
		"base_execution_id":   draft.BaseExecutionID,
		"status":              draft.Status,
		"sql_down":            draft.SqlDown,
		"is_reversible":       draft.IsReversible,
		"irreversible_reason": draft.IrreversibleReason,
		"is_batch":            draft.IsBatch,
		"batch_column":        draft.BatchColumn,
		"batch_size":          draft.BatchSize,
		"current_round":       draft.CurrentRound,
		"recovery_action":     draft.RecoveryAction,
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
		Preload("Checkpoints").
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

	round := approval.Round
	if round == 0 {
		round = 1
	}
	phase := approval.Phase
	if phase == "" {
		phase = domain.ApprovalPhaseInitial
	}
	vote := approval.Vote
	if vote == "" {
		vote = approval.Status
	}

	modelApproval := model.DraftApproval{
		ID:          approval.ID,
		DraftID:     approval.DraftID,
		GroupID:     approval.GroupID,
		Round:       round,
		Phase:       phase,
		LeaderID:    approval.LeaderID,
		Status:      approval.Status,
		Vote:        vote,
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
		"vote":         approval.Vote,
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

func (r *draftRepository) FindApprovalsByDraftAndRound(ctx context.Context, draftID uint, round uint) ([]*domain.DraftApproval, error) {
	var modelApprovals []model.DraftApproval
	err := r.db.WithContext(ctx).
		Where("draft_id = ? AND round = ?", draftID, round).
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
		Order("round DESC, id DESC").
		First(&modelApproval).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDraftNotFound
		}
		return nil, err
	}

	return toDomainDraftApproval(&modelApproval), nil
}

func (r *draftRepository) FindApprovalByDraftGroupAndRound(ctx context.Context, draftID uint, groupID uint, round uint) (*domain.DraftApproval, error) {
	var modelApproval model.DraftApproval
	err := r.db.WithContext(ctx).
		Where("draft_id = ? AND group_id = ? AND round = ?", draftID, groupID, round).
		First(&modelApproval).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDraftNotFound
		}
		return nil, err
	}

	return toDomainDraftApproval(&modelApproval), nil
}

func (r *draftRepository) CreateApprovalsForRound(ctx context.Context, approvals []domain.DraftApproval) error {
	if len(approvals) == 0 {
		return nil
	}
	models := make([]model.DraftApproval, len(approvals))
	for i, a := range approvals {
		models[i] = model.DraftApproval{
			DraftID:     a.DraftID,
			GroupID:     a.GroupID,
			Round:       a.Round,
			Phase:       a.Phase,
			LeaderID:    a.LeaderID,
			Status:      a.Status,
			Vote:        a.Vote,
			RespondedAt: a.RespondedAt,
		}
	}
	return r.db.WithContext(ctx).Create(&models).Error
}

func (r *draftRepository) SaveCheckpoint(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
	if cp == nil {
		return nil, domain.ErrInvalidDraftInput
	}
	m := model.DraftCheckpoint{
		ID:           cp.ID,
		DraftID:      cp.DraftID,
		ExecutionID:  cp.ExecutionID,
		BatchNumber:  cp.BatchNumber,
		CursorLastID: cp.CursorLastID,
		RowsAffected: cp.RowsAffected,
		Status:       cp.Status,
		CreatedAt:    cp.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return toDomainDraftCheckpoint(&m), nil
}

func (r *draftRepository) UpdateCheckpoint(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
	if cp == nil || cp.ID == 0 {
		return nil, domain.ErrInvalidDraftInput
	}
	updates := map[string]any{
		"cursor_last_id": cp.CursorLastID,
		"rows_affected":  cp.RowsAffected,
		"status":         cp.Status,
	}
	if err := r.db.WithContext(ctx).Model(&model.DraftCheckpoint{}).Where("id = ?", cp.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return cp, nil
}

func (r *draftRepository) FindCheckpointsByDraftID(ctx context.Context, draftID uint) ([]*domain.DraftCheckpoint, error) {
	var models []model.DraftCheckpoint
	err := r.db.WithContext(ctx).
		Where("draft_id = ?", draftID).
		Order("batch_number ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	results := make([]*domain.DraftCheckpoint, len(models))
	for i := range models {
		results[i] = toDomainDraftCheckpoint(&models[i])
	}
	return results, nil
}

func (r *draftRepository) FindLastCommittedCheckpoint(ctx context.Context, draftID uint) (*domain.DraftCheckpoint, error) {
	var m model.DraftCheckpoint
	err := r.db.WithContext(ctx).
		Where("draft_id = ? AND status = ?", draftID, "committed").
		Order("batch_number DESC").
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainDraftCheckpoint(&m), nil
}

func (r *draftRepository) FindDraftByStatusAndDatabaseID(ctx context.Context, databaseID uint, status string) (*domain.Draft, error) {
	var modelDraft model.Draft
	err := r.db.WithContext(ctx).
		Where("database_id = ? AND status = ?", databaseID, status).
		First(&modelDraft).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDraftNotFound
		}
		return nil, err
	}
	return toDomainDraft(&modelDraft), nil
}

func toDomainDraft(m *model.Draft) *domain.Draft {
	if m == nil {
		return nil
	}

	approvals := make([]domain.DraftApproval, 0, len(m.Approvals))
	for i := range m.Approvals {
		approvals = append(approvals, *toDomainDraftApproval(&m.Approvals[i]))
	}

	checkpoints := make([]domain.DraftCheckpoint, 0, len(m.Checkpoints))
	for i := range m.Checkpoints {
		checkpoints = append(checkpoints, *toDomainDraftCheckpoint(&m.Checkpoints[i]))
	}

	return &domain.Draft{
		ID:                 m.ID,
		DatabaseID:         m.DatabaseID,
		UserID:             m.UserID,
		Operation:          m.Operation,
		TargetID:           m.TargetID,
		BaseExecutionID:    m.BaseExecutionID,
		SqlUp:              m.SqlUp,
		SqlDown:            m.SqlDown,
		IsReversible:       m.IsReversible,
		IrreversibleReason: m.IrreversibleReason,
		IsBatch:            m.IsBatch,
		BatchColumn:        m.BatchColumn,
		BatchSize:          m.BatchSize,
		CurrentRound:       m.CurrentRound,
		RecoveryAction:     m.RecoveryAction,
		Status:             m.Status,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
		Approvals:          approvals,
		Checkpoints:        checkpoints,
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
		Round:       m.Round,
		Phase:       m.Phase,
		LeaderID:    m.LeaderID,
		Status:      m.Status,
		Vote:        m.Vote,
		RespondedAt: m.RespondedAt,
	}
}

func toDomainDraftCheckpoint(m *model.DraftCheckpoint) *domain.DraftCheckpoint {
	if m == nil {
		return nil
	}

	return &domain.DraftCheckpoint{
		ID:           m.ID,
		DraftID:      m.DraftID,
		ExecutionID:  m.ExecutionID,
		BatchNumber:  m.BatchNumber,
		CursorLastID: m.CursorLastID,
		RowsAffected: m.RowsAffected,
		Status:       m.Status,
		CreatedAt:    m.CreatedAt,
	}
}
