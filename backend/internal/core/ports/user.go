package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type UserService interface {
	GetAllUsers(ctx context.Context, params pagination.PaginationParams) ([]domain.User, int64, error)
	GetUserByID(ctx context.Context, id uint) (domain.User, error)
	CreateUser(ctx context.Context, input domain.CreateUserInput) (domain.User, error)
	UpdateUser(ctx context.Context, id uint, input domain.UpdateUserInput) (domain.User, error)
	DeleteUser(ctx context.Context, id uint) error
	UpdateUserRole(ctx context.Context, id uint, role string) (domain.User, error)
}

type UserRepository interface {
	FindAll(ctx context.Context, params pagination.PaginationParams) ([]domain.User, int64, error)
	FindByID(ctx context.Context, id uint) (domain.User, error)
	CountByRole(ctx context.Context, role string) (int64, error)
	Save(ctx context.Context, user domain.User, groupIDs []uint) (domain.User, error)
	Update(ctx context.Context, id uint, username *string, passwordHash *string) (domain.User, error)
	Delete(ctx context.Context, id uint) error

	FindByUsernameWithPassword(ctx context.Context, username string) (domain.User, error)
	UpdateRole(ctx context.Context, id uint, role string) (domain.User, error)
}
