package model

import (
	"time"

	"gorm.io/datatypes"
)

type Migration struct {
	ID                 uint   `gorm:"primaryKey"`
	DatabaseID         int    `gorm:"not null;index"`
	Version            string
	RevertsMigrationID *uint  `gorm:"index"`

	MigratedAt         time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	Status             string         `gorm:"not null;default:'fail'"`
	SqlUp              string         `gorm:"not null"`
	SqlDown            string         `gorm:"not null;default:''"`
	ErrorMessage       string         `gorm:"not null;default:''"`
	Snapshot           datatypes.JSON `gorm:"type:jsonb"`
	UserID             int            `gorm:"not null"`
	DraftID            *uint          `gorm:"index"`
	ExecutionID        uint           `gorm:"not null;default:0;index"`
	IsReversible       bool           `gorm:"not null;default:true"`
	IrreversibleReason string         `gorm:"type:text;not null;default:''"`
	ForwardMigrationID *uint          `gorm:"index"`

	RevertsMigration *Migration `gorm:"foreignKey:RevertsMigrationID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	User             User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Database         Database   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Draft            *Draft     `gorm:"foreignKey:DraftID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	ForwardMigration *Migration `gorm:"foreignKey:ForwardMigrationID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}
