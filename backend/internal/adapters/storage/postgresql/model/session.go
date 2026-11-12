package model

import "time"

type Session struct {
	ID uint `gorm:"primaryKey"`

	OpaqueToken         string        `gorm:"unique;not null"`
	PreviousOpaqueToken string        `gorm:"index"`
	GracePeriodUntil    *time.Time    `gorm:"default:null"`
	RotatedAt           time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP"`
	IdleTimeout         time.Duration `gorm:"not null"`
	LastActive          time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP"`
	AbsoluteExpiration  time.Time     `gorm:"not null"`
	CSRFToken           string        `gorm:"type:varchar(64);not null;default:''"`
	UserID              uint          `gorm:"not null;index"`

	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
