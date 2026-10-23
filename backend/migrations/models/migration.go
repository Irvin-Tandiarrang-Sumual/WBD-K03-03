package models

import (
	"time"

	"gorm.io/datatypes"
)

type Migration struct {
	DatabaseID int    `gorm:"primaryKey"`
	Version    string `gorm:"primaryKey"`

	MigratedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	Status     string         `gorm:"not null;default:'fail'"`
	SqlUp      string         `gorm:"not null"`
	SqlDown    string         `gorm:"not null"`
	Snapshot   datatypes.JSON `gorm:"type:jsonb"`
	UserID     int            `gorm:"not null"`

	User     User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Database Database `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}
