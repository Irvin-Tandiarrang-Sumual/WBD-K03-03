package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type GroupService interface {
	GetAllGroups(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error)
	GetGroupByID(ctx context.Context, id uint) (domain.Group, error)
	CreateGroup(ctx context.Context, input domain.CreateGroupInput) (domain.Group, error)
	UpdateGroup(ctx context.Context, id uint, input domain.UpdateGroupInput) (domain.Group, error)
	DeleteGroup(ctx context.Context, id uint) error
}

type GroupRepository interface {
	FindAll(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error)
	FindByID(ctx context.Context, id uint) (domain.Group, error)
	AllExist(ctx context.Context, ids []uint) (bool, error)
	Save(ctx context.Context, group domain.Group) (domain.Group, error)
	Update(ctx context.Context, group domain.Group) (domain.Group, error)
	Delete(ctx context.Context, id uint) error
}
