package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type UserService interface {
	List(ctx context.Context) ([]domain.User, error)
	Get(ctx context.Context, id uint) (domain.User, error)
	Create(ctx context.Context, input domain.CreateUserInput) (domain.User, error)
	Update(ctx context.Context, id uint, input domain.UpdateUserInput) (domain.User, error)
	Delete(ctx context.Context, id uint) error
	SetRole(ctx context.Context, id uint, role string) (domain.User, error)
}

type UserRepository interface {
	FindByID(ctx context.Context, id uint) (domain.User, error)
}