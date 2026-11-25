package domain

import (
	"strings"
	"time"
)

const (
	DraftStatusPendingApproval   = "pending_approval"
	DraftStatusApproved          = "approved"
	DraftStatusRejected          = "rejected"
	DraftStatusInvalidated       = "invalidated"
	DraftStatusExecuting         = "executing"
	DraftStatusPartiallyReverted = "partially_reverted"
	DraftStatusApplied           = "applied"
	DraftStatusFailed            = "failed"

	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"
)

type Draft struct {
	ID                 uint
	DatabaseID         uint
	UserID             uint
	SqlUp              string
	SqlDown            string
	IsReversible       bool
	IrreversibleReason string
	IsBatch            bool
	BatchColumn        string
	BatchSize          int
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time

	Approvals   []DraftApproval
	Checkpoints []DraftCheckpoint
}

func (d *Draft) IsTerminal() bool {
	return d.Status == DraftStatusRejected || d.Status == DraftStatusInvalidated
}

type DraftApproval struct {
	ID          uint
	DraftID     uint
	GroupID     uint
	LeaderID    *uint
	Status      string
	RespondedAt *time.Time
}

type DraftCheckpoint struct {
	ID           uint
	DraftID      uint
	ExecutionID  uint
	BatchNumber  int
	CursorLastID string
	RowsAffected int64
	Status       string
	CreatedAt    time.Time
}

type CreateDraftInput struct {
	DatabaseID         uint
	UserID             uint
	SqlUp              string
	SqlDown            string
	IsReversible       *bool
	IrreversibleReason string
	IsBatch            bool
	BatchColumn        string
	BatchSize          int
}

func (d *CreateDraftInput) Validate() error {
	if d.DatabaseID == 0 || d.UserID == 0 || strings.TrimSpace(d.SqlUp) == "" {
		return ErrInvalidDraftInput
	}
	return nil
}

type SubmitApprovalInput struct {
	DraftID uint
	UserID  uint
	Status  string
}

func (s *SubmitApprovalInput) Validate() error {
	if s.DraftID == 0 || s.UserID == 0 {
		return ErrInvalidDraftInput
	}
	if s.Status != ApprovalStatusApproved && s.Status != ApprovalStatusRejected {
		return ErrInvalidDraftInput
	}
	return nil
}
