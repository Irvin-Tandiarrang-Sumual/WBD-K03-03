package testutil

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type UserGroupRepository struct {
	ports.UserGroupRepository
	AddUserFunc                      func(context.Context, uint, uint) error
	RemoveUserFunc                   func(context.Context, uint, uint) error
	CountByUserIDFunc                func(context.Context, uint) (int64, error)
	FindAllUserIDsByGroupIDFunc      func(context.Context, uint) ([]uint, error)
	FindAllAccessTypesFunc           func(context.Context, uint, uint) ([]domain.AccessType, error)
	FindAllUserIDsByDatabaseIDFunc   func(context.Context, uint) ([]uint, error)
	HasGroupWithOnlyThisDatabaseFunc func(context.Context, uint) (bool, error)
}

func (f *UserGroupRepository) AddUser(ctx context.Context, groupID uint, userID uint) error {
	if f.AddUserFunc != nil {
		return f.AddUserFunc(ctx, groupID, userID)
	}
	return nil
}

func (f *UserGroupRepository) RemoveUser(ctx context.Context, groupID uint, userID uint) error {
	if f.RemoveUserFunc != nil {
		return f.RemoveUserFunc(ctx, groupID, userID)
	}
	return nil
}

func (f *UserGroupRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	if f.CountByUserIDFunc != nil {
		return f.CountByUserIDFunc(ctx, userID)
	}
	return 0, nil
}

func (f *UserGroupRepository) FindAllUserIDsByGroupID(ctx context.Context, groupID uint) ([]uint, error) {
	if f.FindAllUserIDsByGroupIDFunc != nil {
		return f.FindAllUserIDsByGroupIDFunc(ctx, groupID)
	}
	return nil, nil
}

func (f *UserGroupRepository) FindAllAccessTypes(ctx context.Context, userID uint, databaseID uint) ([]domain.AccessType, error) {
	if f.FindAllAccessTypesFunc != nil {
		return f.FindAllAccessTypesFunc(ctx, userID, databaseID)
	}
	return nil, nil
}

func (f *UserGroupRepository) FindAllUserIDsByDatabaseID(ctx context.Context, databaseID uint) ([]uint, error) {
	if f.FindAllUserIDsByDatabaseIDFunc != nil {
		return f.FindAllUserIDsByDatabaseIDFunc(ctx, databaseID)
	}
	return nil, nil
}

func (f *UserGroupRepository) HasGroupWithOnlyThisDatabase(ctx context.Context, databaseID uint) (bool, error) {
	if f.HasGroupWithOnlyThisDatabaseFunc != nil {
		return f.HasGroupWithOnlyThisDatabaseFunc(ctx, databaseID)
	}
	return false, nil
}

type UserGroupService struct {
	ports.UserGroupService
	AddUserFunc    func(context.Context, uint, uint) error
	RemoveUserFunc func(context.Context, uint, uint) error
	AuthorizeFunc  func(context.Context, uint, uint, domain.AccessType) error
}

func (f *UserGroupService) AddUser(ctx context.Context, groupID uint, userID uint) error {
	if f.AddUserFunc != nil {
		return f.AddUserFunc(ctx, groupID, userID)
	}
	return nil
}

func (f *UserGroupService) RemoveUser(ctx context.Context, groupID uint, userID uint) error {
	if f.RemoveUserFunc != nil {
		return f.RemoveUserFunc(ctx, groupID, userID)
	}
	return nil
}

func (f *UserGroupService) Authorize(ctx context.Context, userID uint, databaseID uint, requiredAccessType domain.AccessType) error {
	if f.AuthorizeFunc != nil {
		return f.AuthorizeFunc(ctx, userID, databaseID, requiredAccessType)
	}
	return nil
}

