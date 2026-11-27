package model

import (
	"time"

	"gorm.io/datatypes"
)

type MigrationJob struct {
	ID           uint   `gorm:"primaryKey"`
	DatabaseID   uint   `gorm:"not null;uniqueIndex:idx_active_migration_job,where:status='pending' OR status='running' OR status='uncertain'"`
	DraftID      uint   `gorm:"not null;index"`
	ExecutionID  uint   `gorm:"not null"`
	UserID       uint   `gorm:"not null"`
	Operation    string `gorm:"not null"`
	Status       string `gorm:"not null;index"`
	TraceCarrier datatypes.JSON
	CreatedAt    time.Time
	UpdatedAt    time.Time

	Draft Draft `gorm:"constraint:OnDelete:CASCADE;"`
}
