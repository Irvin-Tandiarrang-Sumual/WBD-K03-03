package ports

import (
	"context"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type SessionService interface {
	Create(ctx context.Context, userID uint) (domain.Session, error)
	Resolve(ctx context.Context, opaqueToken string) (domain.Session, error)
	Revoke(ctx context.Context, opaqueToken string) error
	RevokeUser(ctx context.Context, userID uint) error
}

type SessionRepository interface {
	Create(ctx context.Context, session domain.Session) (domain.Session, error)
	FindByToken(ctx context.Context, opaqueToken string) (domain.Session, error)
	Touch(ctx context.Context, id uint, lastActive time.Time) error
	DeleteByToken(ctx context.Context, opaqueToken string) error
	DeleteByUserID(ctx context.Context, userID uint) error
}