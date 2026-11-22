package model

import "time"

type Draft struct {
	ID                 uint   `gorm:"primaryKey"`
	DatabaseID         uint   `gorm:"not null;index"`
	UserID             uint   `gorm:"not null;index"`
	SqlUp              string `gorm:"not null"`
	SqlDown            string `gorm:"not null;default:''"`
	IsReversible       bool   `gorm:"not null;default:true"`
	IrreversibleReason string `gorm:"type:text;not null;default:''"`
	IsBatch            bool   `gorm:"not null;default:false"`
	BatchColumn        string `gorm:"type:varchar(100);not null;default:''"`
	BatchSize          int    `gorm:"not null;default:0"`
	Status             string `gorm:"type:text;check:status IN ('pending_approval','approved','rejected','invalidated','executing','partially_reverted','applied','failed');not null;default:'pending_approval'"`
	CreatedAt          time.Time
	UpdatedAt          time.Time

	Database    Database          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User        User              `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Approvals   []DraftApproval   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Checkpoints []DraftCheckpoint `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type DraftApproval struct {
	ID          uint       `gorm:"primaryKey"`
	DraftID     uint       `gorm:"not null;uniqueIndex:idx_draft_approvals_draft_group,priority:1"`
	GroupID     uint       `gorm:"not null;uniqueIndex:idx_draft_approvals_draft_group,priority:2"`
	LeaderID    *uint      `gorm:"index"`
	Status      string     `gorm:"type:text;check:status IN ('pending','approved','rejected');not null;default:'pending'"`
	RespondedAt *time.Time `gorm:"default:null"`

	Draft  Draft  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Group  Group  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Leader *User  `gorm:"foreignKey:LeaderID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type DraftCheckpoint struct {
	ID           uint      `gorm:"primaryKey"`
	DraftID      uint      `gorm:"not null;index:idx_draft_checkpoints_draft_exec"`
	ExecutionID  uint      `gorm:"not null;index:idx_draft_checkpoints_draft_exec"`
	BatchNumber  int       `gorm:"not null"`
	CursorLastID string    `gorm:"type:varchar(255);not null"`
	RowsAffected int64     `gorm:"not null;default:0"`
	Status       string    `gorm:"type:text;check:status IN ('committed','reverted','failed');not null"`
	CreatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`

	Draft Draft `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
