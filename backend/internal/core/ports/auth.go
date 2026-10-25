package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type AuthService interface {
	Login(ctx context.Context, username string, password string) (domain.User, domain.Session, error)
	Logout(ctx context.Context, opaqueToken string) error
}
