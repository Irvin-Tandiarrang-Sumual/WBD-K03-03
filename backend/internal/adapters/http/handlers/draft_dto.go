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
	RespondedAt *time.Time `json:"respondedAt,omitempty"`
}

type DraftResponse struct {
	ID                 uint                    `json:"id"`
	DatabaseID         uint                    `json:"databaseId"`
	UserID             uint                    `json:"userId"`
	SqlUp              string                  `json:"sqlUp"`
	SqlDown            string                  `json:"sqlDown"`
	IsReversible       bool                    `json:"isReversible"`
	IrreversibleReason string                  `json:"irreversibleReason"`
	IsBatch            bool                    `json:"isBatch"`
	BatchColumn        string                  `json:"batchColumn"`
	BatchSize          int                     `json:"batchSize"`
	Status             string                  `json:"status"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
	Approvals          []DraftApprovalResponse `json:"approvals"`
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
			RespondedAt: app.RespondedAt,
		})
	}

	return DraftResponse{
		ID:                 d.ID,
		DatabaseID:         d.DatabaseID,
		UserID:             d.UserID,
		SqlUp:              d.SqlUp,
		SqlDown:            d.SqlDown,
		IsReversible:       d.IsReversible,
		IrreversibleReason: d.IrreversibleReason,
		IsBatch:            d.IsBatch,
		BatchColumn:        d.BatchColumn,
		BatchSize:          d.BatchSize,
		Status:             d.Status,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
		Approvals:          approvals,
	}
}
