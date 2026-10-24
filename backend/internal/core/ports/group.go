package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type GroupService interface {
	List(ctx context.Context) ([]domain.Group, error)
	Get(ctx context.Context, id uint) (domain.Group, error)

	Create(
		ctx context.Context,
		input domain.CreateGroupInput,
	) (domain.Group, error)

	Update(
		ctx context.Context,
		id uint,
		input domain.UpdateGroupInput,
	) (domain.Group, error)

	Delete(ctx context.Context, id uint) error

	AddUser(
		ctx context.Context,
		groupID uint,
		userID uint,
	) error

	RemoveUser(
		ctx context.Context,
		groupID uint,
		userID uint,
	) error

	EffectiveDatabaseAccess(
		ctx context.Context,
		userID uint,
		databaseID uint,
	) (domain.AccessType, error)
}

type GroupRepository interface {
	List(ctx context.Context) ([]domain.Group, error)
	FindByID(ctx context.Context, id uint) (domain.Group, error)
	ExistsAll(ctx context.Context, ids []uint) (bool, error)

	Create(
		ctx context.Context,
		group domain.Group,
	) (domain.Group, error)

	Update(
		ctx context.Context,
		group domain.Group,
	) (domain.Group, error)

	Delete(ctx context.Context, id uint) error

	AddUser(
		ctx context.Context,
		groupID uint,
		userID uint,
	) error

	RemoveUser(
		ctx context.Context,
		groupID uint,
		userID uint,
	) error

	CountUserGroups(
		ctx context.Context,
		userID uint,
	) (int64, error)

	ListUserIDs(
		ctx context.Context,
		groupID uint,
	) ([]uint, error)

	ListAccessTypes(
		ctx context.Context,
		userID uint,
		databaseID uint,
	) ([]domain.AccessType, error)
}
