package ports

import "context"

type SessionService interface {
	
	RevokeUser(ctx context.Context, userID uint) error
}

type SessionRepository interface {
}