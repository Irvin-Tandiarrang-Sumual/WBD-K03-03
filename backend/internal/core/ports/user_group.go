package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type UserGroupService interface {
	AddUser(ctx context.Context, groupID uint, userID uint) error
	RemoveUser(ctx context.Context, groupID uint, userID uint) error
	GetEffectiveDatabaseAccess(ctx context.Context,
		userID uint, databaseID uint) (domain.AccessType, error)
}

type UserGroupRepository interface {
	AddUser(ctx context.Context, groupID uint, userID uint) error
	RemoveUser(ctx context.Context, groupID uint, userID uint) error
	CountByUserID(ctx context.Context, userID uint) (int64, error)
	FindAllUserIDsByGroupID(ctx context.Context, groupID uint) ([]uint, error)
	FindAllAccessTypes(ctx context.Context, userID uint, databaseID uint) ([]domain.AccessType, error)
	FindAllUserIDsByDatabaseID(ctx context.Context, databaseID uint) ([]uint, error)
	HasGroupWithOnlyThisDatabase(ctx context.Context, databaseID uint) (bool, error)
}
