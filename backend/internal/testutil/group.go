package testutil

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type GroupRepository struct {
	ports.GroupRepository
	ports.UserGroupRepository
	FindAllFunc                      func(context.Context, pagination.PaginationParams) ([]domain.Group, int64, error)
	FindByIDFunc                     func(context.Context, uint) (domain.Group, error)
	AllExistFunc                     func(context.Context, []uint) (bool, error)
	SaveFunc                         func(context.Context, domain.Group) (domain.Group, error)
	UpdateFunc                       func(context.Context, domain.Group) (domain.Group, error)
	DeleteFunc                       func(context.Context, uint) error
	AddUserFunc                      func(context.Context, uint, uint) error
	RemoveUserFunc                   func(context.Context, uint, uint) error
	CountByUserIDFunc                func(context.Context, uint) (int64, error)
	FindAllUserIDsByGroupIDFunc      func(context.Context, uint) ([]uint, error)
	FindAllAccessTypesFunc           func(context.Context, uint, uint) ([]domain.AccessType, error)
	FindAllUserIDsByDatabaseIDFunc   func(context.Context, uint) ([]uint, error)
	HasGroupWithOnlyThisDatabaseFunc func(context.Context, uint) (bool, error)
}

func (f *GroupRepository) FindAll(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error) {
	if f.FindAllFunc != nil {
		return f.FindAllFunc(ctx, params)
	}
	return nil, 0, nil
}

func (f *GroupRepository) FindByID(ctx context.Context, id uint) (domain.Group, error) {
	if f.FindByIDFunc != nil {
		return f.FindByIDFunc(ctx, id)
	}
	return domain.Group{}, nil
}

func (f *GroupRepository) AllExist(ctx context.Context, ids []uint) (bool, error) {
	if f.AllExistFunc != nil {
		return f.AllExistFunc(ctx, ids)
	}
	return false, nil
}

func (f *GroupRepository) Save(ctx context.Context, group domain.Group) (domain.Group, error) {
	if f.SaveFunc != nil {
		return f.SaveFunc(ctx, group)
	}
	return domain.Group{}, nil
}

func (f *GroupRepository) Update(ctx context.Context, group domain.Group) (domain.Group, error) {
	if f.UpdateFunc != nil {
		return f.UpdateFunc(ctx, group)
	}
	return domain.Group{}, nil
}

func (f *GroupRepository) Delete(ctx context.Context, id uint) error {
	if f.DeleteFunc != nil {
		return f.DeleteFunc(ctx, id)
	}
	return nil
}

func (f *GroupRepository) AddUser(ctx context.Context, groupID uint, userID uint) error {
	if f.AddUserFunc != nil {
		return f.AddUserFunc(ctx, groupID, userID)
	}
	return nil
}

func (f *GroupRepository) RemoveUser(ctx context.Context, groupID uint, userID uint) error {
	if f.RemoveUserFunc != nil {
		return f.RemoveUserFunc(ctx, groupID, userID)
	}
	return nil
}

func (f *GroupRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	if f.CountByUserIDFunc != nil {
		return f.CountByUserIDFunc(ctx, userID)
	}
	return 0, nil
}

func (f *GroupRepository) FindAllUserIDsByGroupID(ctx context.Context, groupID uint) ([]uint, error) {
	if f.FindAllUserIDsByGroupIDFunc != nil {
		return f.FindAllUserIDsByGroupIDFunc(ctx, groupID)
	}
	return nil, nil
}

func (f *GroupRepository) FindAllAccessTypes(ctx context.Context, userID uint, databaseID uint) ([]domain.AccessType, error) {
	if f.FindAllAccessTypesFunc != nil {
		return f.FindAllAccessTypesFunc(ctx, userID, databaseID)
	}
	return nil, nil
}

func (f *GroupRepository) FindAllUserIDsByDatabaseID(ctx context.Context, databaseID uint) ([]uint, error) {
	if f.FindAllUserIDsByDatabaseIDFunc != nil {
		return f.FindAllUserIDsByDatabaseIDFunc(ctx, databaseID)
	}
	return nil, nil
}

func (f *GroupRepository) HasGroupWithOnlyThisDatabase(ctx context.Context, databaseID uint) (bool, error) {
	if f.HasGroupWithOnlyThisDatabaseFunc != nil {
		return f.HasGroupWithOnlyThisDatabaseFunc(ctx, databaseID)
	}
	return false, nil
}

type GroupService struct {
	ports.GroupService
	ports.UserGroupService
	GetAllGroupsFunc func(context.Context, pagination.PaginationParams) ([]domain.Group, int64, error)
	GetGroupByIDFunc func(context.Context, uint) (domain.Group, error)
	CreateGroupFunc  func(context.Context, domain.CreateGroupInput) (domain.Group, error)
	UpdateGroupFunc  func(context.Context, uint, domain.UpdateGroupInput) (domain.Group, error)
	DeleteGroupFunc  func(context.Context, uint) error
	AuthorizeFunc    func(context.Context, uint, uint, domain.AccessType) error
}

func (f *GroupService) GetAllGroups(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error) {
	if f.GetAllGroupsFunc != nil {
		return f.GetAllGroupsFunc(ctx, params)
	}
	return nil, 0, nil
}

func (f *GroupService) GetGroupByID(ctx context.Context, id uint) (domain.Group, error) {
	if f.GetGroupByIDFunc != nil {
		return f.GetGroupByIDFunc(ctx, id)
	}
	return domain.Group{}, nil
}

func (f *GroupService) CreateGroup(ctx context.Context, input domain.CreateGroupInput) (domain.Group, error) {
	if f.CreateGroupFunc != nil {
		return f.CreateGroupFunc(ctx, input)
	}
	return domain.Group{}, nil
}

func (f *GroupService) UpdateGroup(ctx context.Context, id uint, input domain.UpdateGroupInput) (domain.Group, error) {
	if f.UpdateGroupFunc != nil {
		return f.UpdateGroupFunc(ctx, id, input)
	}
	return domain.Group{}, nil
}

func (f *GroupService) DeleteGroup(ctx context.Context, id uint) error {
	if f.DeleteGroupFunc != nil {
		return f.DeleteGroupFunc(ctx, id)
	}
	return nil
}

func (f *GroupService) Authorize(ctx context.Context, userID uint, databaseID uint, requiredAccessType domain.AccessType) error {
	if f.AuthorizeFunc != nil {
		return f.AuthorizeFunc(ctx, userID, databaseID, requiredAccessType)
	}
	return nil
}
