package model

import (
	"time"

	"gorm.io/datatypes"
)

type Migration struct {
	ID         uint   `gorm:"primaryKey"`
	DatabaseID int    `gorm:"not null;uniqueIndex:idx_migrations_database_version,priority:1"`
	Version    string `gorm:"not null;uniqueIndex:idx_migrations_database_version,priority:2"`

	MigratedAt   time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	Status       string         `gorm:"not null;default:'fail'"`
	SqlUp        string         `gorm:"not null"`
	SqlDown      string         `gorm:"not null"`
	ErrorMessage string         `gorm:"not null;default:''"`
	Snapshot     datatypes.JSON `gorm:"type:jsonb"`
	UserID       int            `gorm:"not null"`

	User     User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Database Database `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}
