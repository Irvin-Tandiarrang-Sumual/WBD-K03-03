package testutil

import (
	"context"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

// Setiap fake meng-embed interface aslinya. Method yang benar-benar dibutuhkan
// test dioverride lewat function field. Kalau code yang diuji tiba-tiba memanggil
// dependency lain yang tidak kita siapkan, test akan gagal alih-alih diam-diam lolos.

type UserRepository struct {
	ports.UserRepository
	ListFunc                       func(context.Context, pagination.PaginationParams) ([]domain.User, int64, error)
	FindByIDFunc                   func(context.Context, uint) (domain.User, error)
	FindByUsernameWithPasswordFunc func(context.Context, string) (domain.User, error)
	CreateFunc                     func(context.Context, domain.User, []uint) (domain.User, error)
	UpdateFunc                     func(context.Context, uint, *string, *string) (domain.User, error)
	DeleteFunc                     func(context.Context, uint) error
	SetRoleFunc                    func(context.Context, uint, string) (domain.User, error)
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
	return f.SetRoleFunc(ctx, id, role)
}

type GroupRepository struct {
	ports.GroupRepository
	ListFunc                     func(context.Context, pagination.PaginationParams) ([]domain.Group, int64, error)
	FindByIDFunc                 func(context.Context, uint) (domain.Group, error)
	ExistsAllFunc                func(context.Context, []uint) (bool, error)
	CreateFunc                   func(context.Context, domain.Group) (domain.Group, error)
	UpdateFunc                   func(context.Context, domain.Group) (domain.Group, error)
	DeleteFunc                   func(context.Context, uint) error
	AddUserFunc                  func(context.Context, uint, uint) error
	RemoveUserFunc               func(context.Context, uint, uint) error
	CountUserGroupsFunc          func(context.Context, uint) (int64, error)
	ListUserIDsFunc              func(context.Context, uint) ([]uint, error)
	ListAccessTypesFunc          func(context.Context, uint, uint) ([]domain.AccessType, error)
	ListUserIDsByDatabaseFunc    func(context.Context, uint) ([]uint, error)
	HasGroupWithOnlyDatabaseFunc func(context.Context, uint) (bool, error)
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
	return f.HasGroupWithOnlyDatabaseFunc(ctx, databaseID)
}

type DatabaseRepository struct {
	ports.DatabaseRepository
	FindByIdFunc          func(context.Context, uint) (*domain.Database, error)
	FindAllFunc           func(context.Context, pagination.PaginationParams) ([]*domain.Database, int64, error)
	SaveFunc              func(context.Context, *domain.Database) (*domain.Database, error)
	FindAllByUserIDFunc   func(context.Context, uint, pagination.PaginationParams) ([]*domain.Database, int64, error)
	FindByIDAndUserIDFunc func(context.Context, uint, uint) (*domain.Database, error)
	FindByIDFunc          func(context.Context, uint) (domain.Database, error)
	ExistsAllFunc         func(context.Context, []uint) (bool, error)
	CreateFunc            func(context.Context, domain.Database) (domain.Database, error)
	DeleteFunc            func(context.Context, uint) error
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
	createdDatabase, err := f.Create(ctx, *database)
	if err != nil {
		return nil, err
	}
	return &createdDatabase, nil
}

func (f *DatabaseRepository) Delete(ctx context.Context, id uint) error {
	return f.DeleteFunc(ctx, id)
}

type SessionRepository struct {
	ports.SessionRepository
	CreateFunc         func(context.Context, domain.Session) (domain.Session, error)
	FindByTokenFunc    func(context.Context, string) (domain.Session, error)
	TouchFunc          func(context.Context, uint, time.Time) error
	DeleteFunc         func(context.Context, uint) error
	DeleteByTokenFunc  func(context.Context, string) error
	DeleteByUserIDFunc func(context.Context, uint) error
}

func (f *SessionRepository) Create(ctx context.Context, session domain.Session) (domain.Session, error) {
	return f.CreateFunc(ctx, session)
}

func (f *SessionRepository) FindByToken(ctx context.Context, token string) (domain.Session, error) {
	return f.FindByTokenFunc(ctx, token)
}

func (f *SessionRepository) Touch(ctx context.Context, id uint, lastActive time.Time) error {
	return f.TouchFunc(ctx, id, lastActive)
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
	CreateFunc     func(context.Context, uint) (domain.Session, error)
	ResolveFunc    func(context.Context, string) (domain.Session, error)
	RevokeFunc     func(context.Context, string) error
	RevokeUserFunc func(context.Context, uint) error
}

func (f *SessionService) Create(ctx context.Context, userID uint) (domain.Session, error) {
	return f.CreateFunc(ctx, userID)
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
	GetFunc func(context.Context, uint) (domain.User, error)
}

func (f *UserService) Get(ctx context.Context, id uint) (domain.User, error) {
	return f.GetFunc(ctx, id)
}

type GroupService struct {
	ports.GroupService
	EffectiveDatabaseAccessFunc func(context.Context, uint, uint) (domain.AccessType, error)
}

func (f *GroupService) EffectiveDatabaseAccess(ctx context.Context, userID uint, databaseID uint) (domain.AccessType, error) {
	return f.EffectiveDatabaseAccessFunc(ctx, userID, databaseID)
}
