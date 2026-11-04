package domain

import "time"

const (
	DefaultSessionIdleTimeout         = 30 * time.Minute
	DefaultSessionAbsoluteLifetime    = 24 * time.Hour
	DefaultSessionRotationInterval    = 15 * time.Minute
	DefaultSessionRotationGracePeriod = 30 * time.Second
)

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

func (s Session) IsExpired(now time.Time) bool {
	return s.isAbsoluteExpired(now) || s.isIdleExpired(now)
}

func (s Session) isAbsoluteExpired(now time.Time) bool {
	return !s.AbsoluteExpiration.IsZero() && !now.Before(s.AbsoluteExpiration)
}

func (s Session) isIdleExpired(now time.Time) bool {
	return s.IdleTimeout > 0 && now.Sub(s.LastActive) > s.IdleTimeout
}

func (s Session) IsPreviousToken(token string) bool {
	return s.PreviousOpaqueToken != "" &&
		token == s.PreviousOpaqueToken &&
		token != s.OpaqueToken
}

func (s Session) IsWithinGracePeriod(now time.Time) bool {
	return !s.GracePeriodUntil.IsZero() && now.Before(s.GracePeriodUntil)
}

func (s Session) NeedsRotation(now time.Time) bool {
	return !s.RotatedAt.IsZero() && now.Sub(s.RotatedAt) >= DefaultSessionRotationInterval
}
