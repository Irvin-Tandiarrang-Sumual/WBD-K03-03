package domain

import "time"

type Session struct {
	ID                  uint
	OpaqueToken         string
	PreviousOpaqueToken string
	GracePeriodUntil    time.Time
	RotatedAt           time.Time
	IdleTimeout         time.Duration
	LastActive          time.Time
	AbsoluteExpiration  time.Time
	UserID              uint
}
