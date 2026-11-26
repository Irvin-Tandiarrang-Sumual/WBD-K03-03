package handlers

import (
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type CreateDraftRequest struct {
	SqlUp              string `json:"sqlUp" binding:"required"`
	SqlDown            string `json:"sqlDown"`
	IsReversible       *bool  `json:"isReversible"`
	IrreversibleReason string `json:"irreversibleReason"`
	IsBatch            bool   `json:"isBatch"`
	BatchColumn        string `json:"batchColumn"`
	BatchSize          int    `json:"batchSize"`
}

type SubmitApprovalRequest struct {
	Status string `json:"status" binding:"required"`
}

type DraftApprovalResponse struct {
	ID          uint       `json:"id"`
	DraftID     uint       `json:"draftId"`
	GroupID     uint       `json:"groupId"`
	LeaderID    *uint      `json:"leaderId"`
	Status      string     `json:"status"`
	Round       uint       `json:"round"`
	Phase       string     `json:"phase,omitempty"`
	Vote        string     `json:"vote,omitempty"`
	RespondedAt *time.Time `json:"respondedAt,omitempty"`
}

type DraftResponse struct {
	ID                 uint                    `json:"id"`
	DatabaseID         uint                    `json:"databaseId"`
	UserID             uint                    `json:"userId"`
	Operation          string                  `json:"operation,omitempty"`
	TargetID           uint                    `json:"targetId,omitempty"`
	BaseExecutionID    uint                    `json:"baseExecutionId,omitempty"`
	SqlUp              string                  `json:"sqlUp"`
	SqlDown            string                  `json:"sqlDown"`
	IsReversible       bool                    `json:"isReversible"`
	IrreversibleReason string                  `json:"irreversibleReason"`
	IsBatch            bool                    `json:"isBatch"`
	BatchColumn        string                  `json:"batchColumn"`
	BatchSize          int                     `json:"batchSize"`
	CurrentRound       uint                    `json:"currentRound"`
	RecoveryAction     string                  `json:"recoveryAction,omitempty"`
	Status             string                  `json:"status"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
	Approvals          []DraftApprovalResponse `json:"approvals"`
}

type BatchProgressResponse struct {
	CurrentBatch  int     `json:"current_batch"`
	TotalBatches  int     `json:"total_batches"`
	RowsProcessed int64   `json:"rows_processed"`
	TotalRows     int64   `json:"total_rows"`
	Percentage    float64 `json:"percentage"`
	Status        string  `json:"status"`
}

func toDraftResponse(d *domain.Draft) DraftResponse {
	if d == nil {
		return DraftResponse{}
	}

	approvals := make([]DraftApprovalResponse, 0, len(d.Approvals))
	for _, app := range d.Approvals {
		approvals = append(approvals, DraftApprovalResponse{
			ID:          app.ID,
			DraftID:     app.DraftID,
			GroupID:     app.GroupID,
			LeaderID:    app.LeaderID,
			Status:      app.Status,
			Round:       app.Round,
			Phase:       app.Phase,
			Vote:        app.Vote,
			RespondedAt: app.RespondedAt,
		})
	}

	return DraftResponse{
		ID:                 d.ID,
		DatabaseID:         d.DatabaseID,
		UserID:             d.UserID,
		Operation:          d.Operation,
		TargetID:           d.TargetID,
		BaseExecutionID:    d.BaseExecutionID,
		SqlUp:              d.SqlUp,
		SqlDown:            d.SqlDown,
		IsReversible:       d.IsReversible,
		IrreversibleReason: d.IrreversibleReason,
		IsBatch:            d.IsBatch,
		BatchColumn:        d.BatchColumn,
		BatchSize:          d.BatchSize,
		CurrentRound:       d.CurrentRound,
		RecoveryAction:     d.RecoveryAction,
		Status:             d.Status,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
		Approvals:          approvals,
	}
}
