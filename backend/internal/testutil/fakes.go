package testutil

import (
	"context"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type UserRepository struct {
	ports.UserRepository
	ListFunc                       func(context.Context, pagination.PaginationParams) ([]domain.User, int64, error)
	FindByIDFunc                   func(context.Context, uint) (domain.User, error)
	FindByUsernameWithPasswordFunc func(context.Context, string) (domain.User, error)
	CountByRoleFunc                func(context.Context, string) (int64, error)
	CreateFunc                     func(context.Context, domain.User, []uint) (domain.User, error)
	UpdateFunc                     func(context.Context, uint, *string, *string) (domain.User, error)
	DeleteFunc                     func(context.Context, uint) error
	UpdateUserRoleFunc             func(context.Context, uint, string) (domain.User, error)
}

func (f *UserRepository) List(ctx context.Context, params pagination.PaginationParams) ([]domain.User, int64, error) {
	return f.ListFunc(ctx, params)
}

func (f *UserRepository) FindByID(ctx context.Context, id uint) (domain.User, error) {
	return f.FindByIDFunc(ctx, id)
}

func (f *UserRepository) FindByUsernameWithPassword(ctx context.Context, username string) (domain.User, error) {
	return f.FindByUsernameWithPasswordFunc(ctx, username)
}

func (f *UserRepository) CountByRole(ctx context.Context, role string) (int64, error) {
	if f.CountByRoleFunc != nil {
		return f.CountByRoleFunc(ctx, role)
	}
	return 2, nil
}

func (f *UserRepository) Create(ctx context.Context, user domain.User, groupIDs []uint) (domain.User, error) {
	return f.CreateFunc(ctx, user, groupIDs)
}

func (f *UserRepository) Update(ctx context.Context, id uint, username *string, passwordHash *string) (domain.User, error) {
	return f.UpdateFunc(ctx, id, username, passwordHash)
}

func (f *UserRepository) Delete(ctx context.Context, id uint) error {
	return f.DeleteFunc(ctx, id)
}

func (f *UserRepository) SetRole(ctx context.Context, id uint, role string) (domain.User, error) {
	if f.UpdateUserRoleFunc != nil {
		return f.UpdateUserRoleFunc(ctx, id, role)
	}
	return domain.User{}, nil
}

func (f *UserRepository) FindAll(ctx context.Context, params pagination.PaginationParams) ([]domain.User, int64, error) {
	if f.ListFunc != nil {
		return f.ListFunc(ctx, params)
	}
	return nil, 0, nil
}

func (f *UserRepository) Save(ctx context.Context, user domain.User, groupIDs []uint) (domain.User, error) {
	if f.CreateFunc != nil {
		return f.CreateFunc(ctx, user, groupIDs)
	}
	return domain.User{}, nil
}

func (f *UserRepository) UpdateRole(ctx context.Context, id uint, role string) (domain.User, error) {
	if f.UpdateUserRoleFunc != nil {
		return f.UpdateUserRoleFunc(ctx, id, role)
	}
	return domain.User{}, nil
}

type GroupRepository struct {
	ports.GroupRepository
	ports.UserGroupRepository
	ListFunc                         func(context.Context, pagination.PaginationParams) ([]domain.Group, int64, error)
	FindAllFunc                      func(context.Context, pagination.PaginationParams) ([]domain.Group, int64, error)
	FindByIDFunc                     func(context.Context, uint) (domain.Group, error)
	ExistsAllFunc                    func(context.Context, []uint) (bool, error)
	AllExistFunc                     func(context.Context, []uint) (bool, error)
	CreateFunc                       func(context.Context, domain.Group) (domain.Group, error)
	SaveFunc                         func(context.Context, domain.Group) (domain.Group, error)
	UpdateFunc                       func(context.Context, domain.Group) (domain.Group, error)
	DeleteFunc                       func(context.Context, uint) error
	AddUserFunc                      func(context.Context, uint, uint) error
	RemoveUserFunc                   func(context.Context, uint, uint) error
	CountUserGroupsFunc              func(context.Context, uint) (int64, error)
	CountByUserIDFunc                func(context.Context, uint) (int64, error)
	ListUserIDsFunc                  func(context.Context, uint) ([]uint, error)
	FindAllUserIDsByGroupIDFunc      func(context.Context, uint) ([]uint, error)
	ListAccessTypesFunc              func(context.Context, uint, uint) ([]domain.AccessType, error)
	FindAllAccessTypesFunc           func(context.Context, uint, uint) ([]domain.AccessType, error)
	ListUserIDsByDatabaseFunc        func(context.Context, uint) ([]uint, error)
	FindAllUserIDsByDatabaseIDFunc   func(context.Context, uint) ([]uint, error)
	HasGroupWithOnlyDatabaseFunc     func(context.Context, uint) (bool, error)
	HasGroupWithOnlyThisDatabaseFunc func(context.Context, uint) (bool, error)
}

func (f *GroupRepository) List(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error) {
	return f.ListFunc(ctx, params)
}

func (f *GroupRepository) FindByID(ctx context.Context, id uint) (domain.Group, error) {
	return f.FindByIDFunc(ctx, id)
}

func (f *GroupRepository) ExistsAll(ctx context.Context, ids []uint) (bool, error) {
	return f.ExistsAllFunc(ctx, ids)
}

func (f *GroupRepository) Create(ctx context.Context, group domain.Group) (domain.Group, error) {
	return f.CreateFunc(ctx, group)
}

func (f *GroupRepository) Update(ctx context.Context, group domain.Group) (domain.Group, error) {
	return f.UpdateFunc(ctx, group)
}

func (f *GroupRepository) Delete(ctx context.Context, id uint) error {
	return f.DeleteFunc(ctx, id)
}

func (f *GroupRepository) AddUser(ctx context.Context, groupID uint, userID uint) error {
	return f.AddUserFunc(ctx, groupID, userID)
}

func (f *GroupRepository) RemoveUser(ctx context.Context, groupID uint, userID uint) error {
	return f.RemoveUserFunc(ctx, groupID, userID)
}

func (f *GroupRepository) CountUserGroups(ctx context.Context, userID uint) (int64, error) {
	return f.CountUserGroupsFunc(ctx, userID)
}

func (f *GroupRepository) ListUserIDs(ctx context.Context, groupID uint) ([]uint, error) {
	return f.ListUserIDsFunc(ctx, groupID)
}

func (f *GroupRepository) ListAccessTypes(ctx context.Context, userID uint, databaseID uint) ([]domain.AccessType, error) {
	return f.ListAccessTypesFunc(ctx, userID, databaseID)
}

func (f *GroupRepository) ListUserIDsByDatabase(ctx context.Context, databaseID uint) ([]uint, error) {
	return f.ListUserIDsByDatabaseFunc(ctx, databaseID)
}

func (f *GroupRepository) HasGroupWithOnlyDatabase(ctx context.Context, databaseID uint) (bool, error) {
	if f.HasGroupWithOnlyDatabaseFunc != nil {
		return f.HasGroupWithOnlyDatabaseFunc(ctx, databaseID)
	}
	if f.HasGroupWithOnlyThisDatabaseFunc != nil {
		return f.HasGroupWithOnlyThisDatabaseFunc(ctx, databaseID)
	}
	return false, nil
}

func (f *GroupRepository) FindAll(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error) {
	if f.FindAllFunc != nil {
		return f.FindAllFunc(ctx, params)
	}
	if f.ListFunc != nil {
		return f.ListFunc(ctx, params)
	}
	return nil, 0, nil
}

func (f *GroupRepository) AllExist(ctx context.Context, ids []uint) (bool, error) {
	if f.AllExistFunc != nil {
		return f.AllExistFunc(ctx, ids)
	}
	if f.ExistsAllFunc != nil {
		return f.ExistsAllFunc(ctx, ids)
	}
	return false, nil
}

func (f *GroupRepository) Save(ctx context.Context, group domain.Group) (domain.Group, error) {
	if f.SaveFunc != nil {
		return f.SaveFunc(ctx, group)
	}
	if f.CreateFunc != nil {
		return f.CreateFunc(ctx, group)
	}
	return domain.Group{}, nil
}

func (f *GroupRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	if f.CountByUserIDFunc != nil {
		return f.CountByUserIDFunc(ctx, userID)
	}
	if f.CountUserGroupsFunc != nil {
		return f.CountUserGroupsFunc(ctx, userID)
	}
	return 0, nil
}

func (f *GroupRepository) FindAllUserIDsByGroupID(ctx context.Context, groupID uint) ([]uint, error) {
	if f.FindAllUserIDsByGroupIDFunc != nil {
		return f.FindAllUserIDsByGroupIDFunc(ctx, groupID)
	}
	if f.ListUserIDsFunc != nil {
		return f.ListUserIDsFunc(ctx, groupID)
	}
	return nil, nil
}

func (f *GroupRepository) FindAllAccessTypes(ctx context.Context, userID uint, databaseID uint) ([]domain.AccessType, error) {
	if f.FindAllAccessTypesFunc != nil {
		return f.FindAllAccessTypesFunc(ctx, userID, databaseID)
	}
	if f.ListAccessTypesFunc != nil {
		return f.ListAccessTypesFunc(ctx, userID, databaseID)
	}
	return nil, nil
}

func (f *GroupRepository) FindAllUserIDsByDatabaseID(ctx context.Context, databaseID uint) ([]uint, error) {
	if f.FindAllUserIDsByDatabaseIDFunc != nil {
		return f.FindAllUserIDsByDatabaseIDFunc(ctx, databaseID)
	}
	if f.ListUserIDsByDatabaseFunc != nil {
		return f.ListUserIDsByDatabaseFunc(ctx, databaseID)
	}
	return nil, nil
}

func (f *GroupRepository) HasGroupWithOnlyThisDatabase(ctx context.Context, databaseID uint) (bool, error) {
	if f.HasGroupWithOnlyThisDatabaseFunc != nil {
		return f.HasGroupWithOnlyThisDatabaseFunc(ctx, databaseID)
	}
	if f.HasGroupWithOnlyDatabaseFunc != nil {
		return f.HasGroupWithOnlyDatabaseFunc(ctx, databaseID)
	}
	return false, nil
}

type DatabaseRepository struct {
	ports.DatabaseRepository
	FindByIdFunc                    func(context.Context, uint) (*domain.Database, error)
	FindAllFunc                     func(context.Context, pagination.PaginationParams) ([]*domain.Database, int64, error)
	SaveFunc                        func(context.Context, *domain.Database) (*domain.Database, error)
	FindAllByUserIDFunc             func(context.Context, uint, pagination.PaginationParams) ([]*domain.Database, int64, error)
	FindByIDAndUserIDFunc           func(context.Context, uint, uint) (*domain.Database, error)
	FindByIDFunc                    func(context.Context, uint) (domain.Database, error)
	FindWithCredentialByIDFunc      func(context.Context, uint) (domain.Database, error)
	FindCredentialByHostAndPortFunc func(context.Context, string, uint, string) (string, error)
	GetDistinctHostsFunc            func(context.Context) ([]domain.DatabaseHost, error)
	ExistsAllFunc                   func(context.Context, []uint) (bool, error)
	CreateFunc                      func(context.Context, domain.Database) (domain.Database, error)
	CreateDatabaseFunc              func(context.Context, *domain.Database) (*domain.Database, error)
	DeleteFunc                      func(context.Context, uint) error
}

func (f *DatabaseRepository) FindById(ctx context.Context, id uint) (*domain.Database, error) {
	if f.FindByIdFunc != nil {
		return f.FindByIdFunc(ctx, id)
	}
	database, err := f.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &database, nil
}

func (f *DatabaseRepository) FindAll(ctx context.Context, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	return f.FindAllFunc(ctx, params)
}

func (f *DatabaseRepository) Save(ctx context.Context, database *domain.Database) (*domain.Database, error) {
	return f.SaveFunc(ctx, database)
}

func (f *DatabaseRepository) FindAllByUserID(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	return f.FindAllByUserIDFunc(ctx, userID, params)
}

func (f *DatabaseRepository) FindByIDAndUserID(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error) {
	return f.FindByIDAndUserIDFunc(ctx, databaseID, userID)
}

func (f *DatabaseRepository) FindWithCredentialByID(ctx context.Context, id uint) (domain.Database, error) {
	if f.FindWithCredentialByIDFunc != nil {
		return f.FindWithCredentialByIDFunc(ctx, id)
	}
	database, err := f.FindById(ctx, id)
	if err != nil {
		return domain.Database{}, err
	}
	return *database, nil
}

func (f *DatabaseRepository) FindByID(ctx context.Context, id uint) (domain.Database, error) {
	return f.FindByIDFunc(ctx, id)
}

func (f *DatabaseRepository) ExistsAll(ctx context.Context, ids []uint) (bool, error) {
	return f.ExistsAllFunc(ctx, ids)
}

func (f *DatabaseRepository) Create(ctx context.Context, database domain.Database) (domain.Database, error) {
	return f.CreateFunc(ctx, database)
}

func (f *DatabaseRepository) CreateDatabase(ctx context.Context, database *domain.Database) (*domain.Database, error) {
	if f.CreateDatabaseFunc != nil {
		return f.CreateDatabaseFunc(ctx, database)
	}
	createdDatabase, err := f.Create(ctx, *database)
	if err != nil {
		return nil, err
	}
	return &createdDatabase, nil
}

func (f *DatabaseRepository) FindCredentialByHostAndPort(ctx context.Context, host string, port uint, user string) (string, error) {
	if f.FindCredentialByHostAndPortFunc != nil {
		return f.FindCredentialByHostAndPortFunc(ctx, host, port, user)
	}
	return "", nil
}

func (f *DatabaseRepository) GetDistinctHosts(ctx context.Context) ([]domain.DatabaseHost, error) {
	if f.GetDistinctHostsFunc != nil {
		return f.GetDistinctHostsFunc(ctx)
	}
	return nil, nil
}

func (f *DatabaseRepository) Delete(ctx context.Context, id uint) error {
	return f.DeleteFunc(ctx, id)
}

type SessionRepository struct {
	ports.SessionRepository
	CreateFunc         func(context.Context, domain.Session) (domain.Session, error)
	FindByTokenFunc    func(context.Context, string) (domain.Session, error)
	TouchFunc          func(context.Context, uint, time.Time) error
	RotateFunc         func(context.Context, domain.Session) error
	DeleteFunc         func(context.Context, uint) error
	DeleteByTokenFunc  func(context.Context, string) error
	DeleteByUserIDFunc func(context.Context, uint) error
}

func (f *SessionRepository) Create(ctx context.Context, session domain.Session) (domain.Session, error) {
	return f.CreateFunc(ctx, session)
}

func (f *SessionRepository) Save(ctx context.Context, session domain.Session) (domain.Session, error) {
	if f.CreateFunc != nil {
		return f.CreateFunc(ctx, session)
	}
	return domain.Session{}, nil
}

func (f *SessionRepository) DeleteByUserIDs(ctx context.Context, userIDs []uint) error {
	if f.DeleteByUserIDFunc != nil {
		for _, id := range userIDs {
			if err := f.DeleteByUserIDFunc(ctx, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *SessionRepository) FindByToken(ctx context.Context, token string) (domain.Session, error) {
	return f.FindByTokenFunc(ctx, token)
}

func (f *SessionRepository) Touch(ctx context.Context, id uint, lastActive time.Time) error {
	return f.TouchFunc(ctx, id, lastActive)
}

func (f *SessionRepository) Rotate(ctx context.Context, session domain.Session) error {
	if f.RotateFunc != nil {
		return f.RotateFunc(ctx, session)
	}
	if f.TouchFunc != nil {
		return f.TouchFunc(ctx, session.ID, session.LastActive)
	}
	return nil
}

func (f *SessionRepository) Delete(ctx context.Context, id uint) error {
	return f.DeleteFunc(ctx, id)
}

func (f *SessionRepository) DeleteByToken(ctx context.Context, token string) error {
	return f.DeleteByTokenFunc(ctx, token)
}

func (f *SessionRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	return f.DeleteByUserIDFunc(ctx, userID)
}

type SessionService struct {
	ports.SessionService
	CreateFunc      func(context.Context, uint) (domain.Session, error)
	ResolveFunc     func(context.Context, string) (domain.Session, error)
	RevokeFunc      func(context.Context, string) error
	RevokeUserFunc  func(context.Context, uint) error
	RevokeUsersFunc func(context.Context, []uint) error
}

func (f *SessionService) Create(ctx context.Context, userID uint) (domain.Session, error) {
	return f.CreateFunc(ctx, userID)
}

func (f *SessionService) CreateSession(ctx context.Context, userID uint) (domain.Session, error) {
	if f.CreateFunc != nil {
		return f.CreateFunc(ctx, userID)
	}
	return domain.Session{}, nil
}

func (f *SessionService) Resolve(ctx context.Context, token string) (domain.Session, error) {
	return f.ResolveFunc(ctx, token)
}

func (f *SessionService) Revoke(ctx context.Context, token string) error {
	return f.RevokeFunc(ctx, token)
}

func (f *SessionService) RevokeUser(ctx context.Context, userID uint) error {
	return f.RevokeUserFunc(ctx, userID)
}

func (f *SessionService) RevokeUsers(ctx context.Context, userIDs []uint) error {
	if f.RevokeUsersFunc != nil {
		return f.RevokeUsersFunc(ctx, userIDs)
	}
	if f.RevokeUserFunc != nil {
		for _, userID := range userIDs {
			if err := f.RevokeUserFunc(ctx, userID); err != nil {
				return err
			}
		}
	}
	return nil
}

type DatabaseConnectionTester struct {
	ports.DatabaseConnectionTester
	TestFunc func(context.Context, domain.Database) error
}

func (f *DatabaseConnectionTester) Test(ctx context.Context, database domain.Database) error {
	return f.TestFunc(ctx, database)
}

type DatabasePoolInvalidator struct {
	ports.DatabasePoolInvalidator
	InvalidateFunc func(uint) error
}

func (f *DatabasePoolInvalidator) Invalidate(databaseID uint) error {
	return f.InvalidateFunc(databaseID)
}

type UserService struct {
	ports.UserService
	ListFunc    func(context.Context, pagination.PaginationParams) ([]domain.User, int64, error)
	GetFunc     func(context.Context, uint) (domain.User, error)
	CreateFunc  func(context.Context, domain.CreateUserInput) (domain.User, error)
	UpdateFunc  func(context.Context, uint, domain.UpdateUserInput) (domain.User, error)
	DeleteFunc  func(context.Context, uint) error
	SetRoleFunc func(context.Context, uint, string) (domain.User, error)
}

func (f *UserService) List(ctx context.Context, params pagination.PaginationParams) ([]domain.User, int64, error) {
	return f.ListFunc(ctx, params)
}

func (f *UserService) Get(ctx context.Context, id uint) (domain.User, error) {
	return f.GetFunc(ctx, id)
}

func (f *UserService) Create(ctx context.Context, input domain.CreateUserInput) (domain.User, error) {
	return f.CreateFunc(ctx, input)
}

func (f *UserService) Update(ctx context.Context, id uint, input domain.UpdateUserInput) (domain.User, error) {
	return f.UpdateFunc(ctx, id, input)
}

func (f *UserService) Delete(ctx context.Context, id uint) error {
	return f.DeleteFunc(ctx, id)
}

func (f *UserService) SetRole(ctx context.Context, id uint, role string) (domain.User, error) {
	if f.SetRoleFunc != nil {
		return f.SetRoleFunc(ctx, id, role)
	}
	return domain.User{}, nil
}

func (f *UserService) GetAllUsers(ctx context.Context, params pagination.PaginationParams) ([]domain.User, int64, error) {
	if f.ListFunc != nil {
		return f.ListFunc(ctx, params)
	}
	return nil, 0, nil
}

func (f *UserService) GetUserByID(ctx context.Context, id uint) (domain.User, error) {
	if f.GetFunc != nil {
		return f.GetFunc(ctx, id)
	}
	return domain.User{}, nil
}

func (f *UserService) CreateUser(ctx context.Context, input domain.CreateUserInput) (domain.User, error) {
	if f.CreateFunc != nil {
		return f.CreateFunc(ctx, input)
	}
	return domain.User{}, nil
}

func (f *UserService) UpdateUser(ctx context.Context, id uint, input domain.UpdateUserInput) (domain.User, error) {
	if f.UpdateFunc != nil {
		return f.UpdateFunc(ctx, id, input)
	}
	return domain.User{}, nil
}

func (f *UserService) DeleteUser(ctx context.Context, id uint) error {
	if f.DeleteFunc != nil {
		return f.DeleteFunc(ctx, id)
	}
	return nil
}

func (f *UserService) UpdateUserRole(ctx context.Context, id uint, role string) (domain.User, error) {
	if f.SetRoleFunc != nil {
		return f.SetRoleFunc(ctx, id, role)
	}
	return domain.User{}, nil
}

type GroupService struct {
	ports.GroupService
	ports.UserGroupService
	ListFunc                       func(context.Context, pagination.PaginationParams) ([]domain.Group, int64, error)
	GetAllGroupsFunc               func(context.Context, pagination.PaginationParams) ([]domain.Group, int64, error)
	GetFunc                        func(context.Context, uint) (domain.Group, error)
	GetGroupByIDFunc               func(context.Context, uint) (domain.Group, error)
	CreateFunc                     func(context.Context, domain.CreateGroupInput) (domain.Group, error)
	CreateGroupFunc                func(context.Context, domain.CreateGroupInput) (domain.Group, error)
	UpdateFunc                     func(context.Context, uint, domain.UpdateGroupInput) (domain.Group, error)
	UpdateGroupFunc                func(context.Context, uint, domain.UpdateGroupInput) (domain.Group, error)
	DeleteFunc                     func(context.Context, uint) error
	DeleteGroupFunc                func(context.Context, uint) error
	AddUserFunc                    func(context.Context, uint, uint) error
	RemoveUserFunc                 func(context.Context, uint, uint) error
	EffectiveDatabaseAccessFunc    func(context.Context, uint, uint) (domain.AccessType, error)
	GetEffectiveDatabaseAccessFunc func(context.Context, uint, uint) (domain.AccessType, error)
}

func (f *GroupService) List(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error) {
	return f.ListFunc(ctx, params)
}

func (f *GroupService) Get(ctx context.Context, id uint) (domain.Group, error) {
	return f.GetFunc(ctx, id)
}

func (f *GroupService) Create(ctx context.Context, input domain.CreateGroupInput) (domain.Group, error) {
	return f.CreateFunc(ctx, input)
}

func (f *GroupService) Update(ctx context.Context, id uint, input domain.UpdateGroupInput) (domain.Group, error) {
	return f.UpdateFunc(ctx, id, input)
}

func (f *GroupService) Delete(ctx context.Context, id uint) error {
	return f.DeleteFunc(ctx, id)
}

func (f *GroupService) AddUser(ctx context.Context, groupID uint, userID uint) error {
	return f.AddUserFunc(ctx, groupID, userID)
}

func (f *GroupService) RemoveUser(ctx context.Context, groupID uint, userID uint) error {
	return f.RemoveUserFunc(ctx, groupID, userID)
}

func (f *GroupService) EffectiveDatabaseAccess(ctx context.Context, userID uint, databaseID uint) (domain.AccessType, error) {
	if f.EffectiveDatabaseAccessFunc != nil {
		return f.EffectiveDatabaseAccessFunc(ctx, userID, databaseID)
	}
	if f.GetEffectiveDatabaseAccessFunc != nil {
		return f.GetEffectiveDatabaseAccessFunc(ctx, userID, databaseID)
	}
	return "", nil
}

func (f *GroupService) GetAllGroups(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error) {
	if f.GetAllGroupsFunc != nil {
		return f.GetAllGroupsFunc(ctx, params)
	}
	if f.ListFunc != nil {
		return f.ListFunc(ctx, params)
	}
	return nil, 0, nil
}

func (f *GroupService) GetGroupByID(ctx context.Context, id uint) (domain.Group, error) {
	if f.GetGroupByIDFunc != nil {
		return f.GetGroupByIDFunc(ctx, id)
	}
	if f.GetFunc != nil {
		return f.GetFunc(ctx, id)
	}
	return domain.Group{}, nil
}

func (f *GroupService) CreateGroup(ctx context.Context, input domain.CreateGroupInput) (domain.Group, error) {
	if f.CreateGroupFunc != nil {
		return f.CreateGroupFunc(ctx, input)
	}
	if f.CreateFunc != nil {
		return f.CreateFunc(ctx, input)
	}
	return domain.Group{}, nil
}

func (f *GroupService) UpdateGroup(ctx context.Context, id uint, input domain.UpdateGroupInput) (domain.Group, error) {
	if f.UpdateGroupFunc != nil {
		return f.UpdateGroupFunc(ctx, id, input)
	}
	if f.UpdateFunc != nil {
		return f.UpdateFunc(ctx, id, input)
	}
	return domain.Group{}, nil
}

func (f *GroupService) DeleteGroup(ctx context.Context, id uint) error {
	if f.DeleteGroupFunc != nil {
		return f.DeleteGroupFunc(ctx, id)
	}
	if f.DeleteFunc != nil {
		return f.DeleteFunc(ctx, id)
	}
	return nil
}

func (f *GroupService) GetEffectiveDatabaseAccess(ctx context.Context, userID uint, databaseID uint) (domain.AccessType, error) {
	if f.GetEffectiveDatabaseAccessFunc != nil {
		return f.GetEffectiveDatabaseAccessFunc(ctx, userID, databaseID)
	}
	if f.EffectiveDatabaseAccessFunc != nil {
		return f.EffectiveDatabaseAccessFunc(ctx, userID, databaseID)
	}
	return "", nil
}

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
	AddUserFunc                    func(context.Context, uint, uint) error
	RemoveUserFunc                 func(context.Context, uint, uint) error
	GetEffectiveDatabaseAccessFunc func(context.Context, uint, uint) (domain.AccessType, error)
	EffectiveDatabaseAccessFunc    func(context.Context, uint, uint) (domain.AccessType, error)
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

func (f *UserGroupService) GetEffectiveDatabaseAccess(ctx context.Context, userID uint, databaseID uint) (domain.AccessType, error) {
	if f.GetEffectiveDatabaseAccessFunc != nil {
		return f.GetEffectiveDatabaseAccessFunc(ctx, userID, databaseID)
	}
	if f.EffectiveDatabaseAccessFunc != nil {
		return f.EffectiveDatabaseAccessFunc(ctx, userID, databaseID)
	}
	return "", nil
}

func (f *UserGroupService) EffectiveDatabaseAccess(ctx context.Context, userID uint, databaseID uint) (domain.AccessType, error) {
	return f.GetEffectiveDatabaseAccess(ctx, userID, databaseID)
}

type DatabaseService struct {
	ports.DatabaseService
	GetDatabaseByIDFunc          func(context.Context, uint) (*domain.Database, error)
	GetAllDatabasesFunc          func(context.Context, pagination.PaginationParams) ([]*domain.Database, int64, error)
	GetDatabaseHostsFunc         func(context.Context) ([]domain.DatabaseHost, error)
	CreateDatabaseFunc           func(context.Context, *ports.DatabaseInput) (*domain.Database, error)
	DeleteDatabaseFunc           func(context.Context, uint) error
	GetDatabasesByUserIDFunc     func(context.Context, uint, pagination.PaginationParams) ([]*domain.Database, int64, error)
	GetDatabaseByIDAndUserIDFunc func(context.Context, uint, uint) (*domain.Database, error)
}

func (f *DatabaseService) GetDatabaseByID(ctx context.Context, databaseID uint) (*domain.Database, error) {
	return f.GetDatabaseByIDFunc(ctx, databaseID)
}

func (f *DatabaseService) GetAllDatabases(ctx context.Context, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	return f.GetAllDatabasesFunc(ctx, params)
}

func (f *DatabaseService) GetDatabaseHosts(ctx context.Context) ([]domain.DatabaseHost, error) {
	if f.GetDatabaseHostsFunc != nil {
		return f.GetDatabaseHostsFunc(ctx)
	}
	return nil, nil
}

func (f *DatabaseService) CreateDatabase(ctx context.Context, input *ports.DatabaseInput) (*domain.Database, error) {
	return f.CreateDatabaseFunc(ctx, input)
}

func (f *DatabaseService) DeleteDatabase(ctx context.Context, databaseID uint) error {
	return f.DeleteDatabaseFunc(ctx, databaseID)
}

func (f *DatabaseService) GetDatabasesByUserID(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	return f.GetDatabasesByUserIDFunc(ctx, userID, params)
}

func (f *DatabaseService) GetDatabaseByIDAndUserID(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error) {
	return f.GetDatabaseByIDAndUserIDFunc(ctx, databaseID, userID)
}

type MigrationService struct {
	ports.MigrationService
	GetMigrationsByDatabaseIDFunc     func(context.Context, uint, pagination.PaginationParams) ([]*domain.Migration, int64, error)
	CreateMigrationFunc               func(context.Context, uint, uint, string) (*domain.Migration, error)
	GetMigrationByIDAndDatabaseIDFunc func(context.Context, uint, uint) (*domain.Migration, int64, error)
	CompareMigrationsFunc             func(context.Context, uint, uint, uint, uint) (*ports.MigrationComparison, error)
	RollbackMigrationsFunc            func(context.Context, uint, uint, uint) ([]*domain.Migration, error)
}

func (f *MigrationService) GetMigrationsByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error) {
	return f.GetMigrationsByDatabaseIDFunc(ctx, databaseID, params)
}

func (f *MigrationService) CreateMigration(ctx context.Context, databaseID uint, userID uint, sqlUp string) (*domain.Migration, error) {
	return f.CreateMigrationFunc(ctx, databaseID, userID, sqlUp)
}

func (f *MigrationService) GetMigrationByIDAndDatabaseID(ctx context.Context, migrationID uint, databaseID uint) (*domain.Migration, int64, error) {
	return f.GetMigrationByIDAndDatabaseIDFunc(ctx, migrationID, databaseID)
}

func (f *MigrationService) CompareMigrations(ctx context.Context, databaseID uint, userID uint, firstMigrationID uint, secondMigrationID uint) (*ports.MigrationComparison, error) {
	return f.CompareMigrationsFunc(ctx, databaseID, userID, firstMigrationID, secondMigrationID)
}

func (f *MigrationService) RollbackMigrations(ctx context.Context, databaseID uint, userID uint, targetMigrationID uint) ([]*domain.Migration, error) {
	return f.RollbackMigrationsFunc(ctx, databaseID, userID, targetMigrationID)
}
