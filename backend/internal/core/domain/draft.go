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
	DraftStatusPartial           = "partial"
	DraftStatusPartiallyReverted = "partially_reverted"
	DraftStatusApplied           = "applied"
	DraftStatusFailed            = "failed"

	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"

	ApprovalPhaseInitial         = "initial"
	ApprovalPhasePartialRecovery = "partial_recovery"
	ApprovalPhaseRevertFailure   = "revert_failure"

	VoteApproved = "approved"
	VoteRejected = "rejected"
	VoteResume   = "resume"
	VoteRevert   = "revert"
	VoteYes      = "yes"
	VoteNo       = "no"

	DraftOperationCustomSQL = "custom_sql"
	DraftOperationRollback  = "rollback"
	DraftOperationForward   = "forward"

	RecoveryActionResume  = "resume"
	RecoveryActionRevert  = "revert"
	RecoveryActionAborted = "aborted"
)

type Draft struct {
	ID                 uint
	DatabaseID         uint
	UserID             uint
	Operation          string
	TargetID           uint
	BaseExecutionID    uint
	SqlUp              string
	SqlDown            string
	IsReversible       bool
	IrreversibleReason string
	IsBatch            bool
	BatchColumn        string
	BatchSize          int
	CurrentRound       uint
	RecoveryAction     string
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
	Round       uint
	Phase       string
	Status      string
	Vote        string
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
	if d.IsBatch {
		if strings.TrimSpace(d.BatchColumn) == "" || d.BatchSize <= 0 {
			return ErrBatchMissingColumnOrSize
		}
	}
	return nil
}

type SubmitApprovalInput struct {
	DraftID uint
	UserID  uint
	Status  string
	Vote    string
}

func (s *SubmitApprovalInput) Validate() error {
	if s.DraftID == 0 || s.UserID == 0 {
		return ErrInvalidDraftInput
	}
	v := s.Vote
	if v == "" {
		v = s.Status
	}
	v = strings.ToLower(strings.TrimSpace(v))
	validVotes := map[string]bool{
		ApprovalStatusApproved: true,
		ApprovalStatusRejected: true,
		VoteResume:             true,
		VoteRevert:             true,
		VoteYes:                true,
		VoteNo:                 true,
	}
	if !validVotes[v] {
		return ErrInvalidDraftInput
	}
	return nil
}

type BatchProgress struct {
	CurrentBatch  int
	TotalBatches  int
	RowsProcessed int64
	TotalRows     int64
	Percentage    float64
	Status        string
}
