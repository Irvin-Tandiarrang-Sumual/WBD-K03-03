package domain

import "time"

type Session struct {
	ID                 uint
	OpaqueToken        string
	IdleTimeout        time.Duration
	LastActive         time.Time
	AbsoluteExpiration time.Time
	UserID             uint
}
