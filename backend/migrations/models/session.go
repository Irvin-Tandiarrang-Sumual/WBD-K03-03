package models

import (
	"time"

	"gorm.io/gorm"
)

type Session struct {
	gorm.Model

	OpaqueToken        string        `gorm:"unique; not null"`
	IdleTimeout        time.Duration `gorm:"not null"`
	LastActive         time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP"`
	AbsoluteExpiration time.Time     `gorm:"not null"`
	UserID             int           `gorm:"not null"`

	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
