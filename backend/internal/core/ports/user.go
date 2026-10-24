package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type UserService interface {
}

type UserRepository interface {
	FindByID(ctx context.Context, id uint) (domain.User, error)
}