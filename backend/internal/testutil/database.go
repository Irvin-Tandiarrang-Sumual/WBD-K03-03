package testutil

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type DatabaseRepository struct {
	ports.DatabaseRepository
	FindByIdFunc                    func(context.Context, uint) (*domain.Database, error)
	FindAllFunc                     func(context.Context, pagination.PaginationParams) ([]*domain.Database, int64, error)
	SaveFunc                        func(context.Context, *domain.Database) (*domain.Database, error)
	DeleteFunc                      func(context.Context, uint) error
	FindAllByUserIDFunc             func(context.Context, uint, pagination.PaginationParams) ([]*domain.Database, int64, error)
	FindByIDAndUserIDFunc           func(context.Context, uint, uint) (*domain.Database, error)
	FindWithCredentialByIDFunc      func(context.Context, uint) (domain.Database, error)
	FindCredentialByHostAndPortFunc func(context.Context, string, uint, string) (string, error)
	GetDistinctHostsFunc            func(context.Context) ([]domain.DatabaseHost, error)
	ExistsAllFunc                   func(context.Context, []uint) (bool, error)
}

func (f *DatabaseRepository) FindById(ctx context.Context, id uint) (*domain.Database, error) {
	if f.FindByIdFunc != nil {
		return f.FindByIdFunc(ctx, id)
	}
	return nil, nil
}

func (f *DatabaseRepository) FindAll(ctx context.Context, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	if f.FindAllFunc != nil {
		return f.FindAllFunc(ctx, params)
	}
	return nil, 0, nil
}

func (f *DatabaseRepository) Save(ctx context.Context, database *domain.Database) (*domain.Database, error) {
	if f.SaveFunc != nil {
		return f.SaveFunc(ctx, database)
	}
	return database, nil
}

func (f *DatabaseRepository) Delete(ctx context.Context, id uint) error {
	if f.DeleteFunc != nil {
		return f.DeleteFunc(ctx, id)
	}
	return nil
}

func (f *DatabaseRepository) FindAllByUserID(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	if f.FindAllByUserIDFunc != nil {
		return f.FindAllByUserIDFunc(ctx, userID, params)
	}
	return nil, 0, nil
}

func (f *DatabaseRepository) FindByIDAndUserID(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error) {
	if f.FindByIDAndUserIDFunc != nil {
		return f.FindByIDAndUserIDFunc(ctx, databaseID, userID)
	}
	return nil, nil
}

func (f *DatabaseRepository) FindWithCredentialByID(ctx context.Context, id uint) (domain.Database, error) {
	if f.FindWithCredentialByIDFunc != nil {
		return f.FindWithCredentialByIDFunc(ctx, id)
	}
	return domain.Database{}, nil
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

func (f *DatabaseRepository) ExistsAll(ctx context.Context, ids []uint) (bool, error) {
	if f.ExistsAllFunc != nil {
		return f.ExistsAllFunc(ctx, ids)
	}
	return false, nil
}

type DatabaseConnectionTester struct {
	ports.DatabaseConnectionTester
	TestFunc func(context.Context, domain.Database) error
}

func (f *DatabaseConnectionTester) Test(ctx context.Context, database domain.Database) error {
	if f.TestFunc != nil {
		return f.TestFunc(ctx, database)
	}
	return nil
}

type DatabasePoolInvalidator struct {
	ports.DatabasePoolInvalidator
	InvalidateFunc func(uint) error
}

func (f *DatabasePoolInvalidator) Invalidate(databaseID uint) error {
	if f.InvalidateFunc != nil {
		return f.InvalidateFunc(databaseID)
	}
	return nil
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

func (f *DatabaseService) GetDatabaseByID(ctx context.Context, id uint) (*domain.Database, error) {
	if f.GetDatabaseByIDFunc != nil {
		return f.GetDatabaseByIDFunc(ctx, id)
	}
	return nil, nil
}

func (f *DatabaseService) GetAllDatabases(ctx context.Context, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	if f.GetAllDatabasesFunc != nil {
		return f.GetAllDatabasesFunc(ctx, params)
	}
	return nil, 0, nil
}

func (f *DatabaseService) GetDatabaseHosts(ctx context.Context) ([]domain.DatabaseHost, error) {
	if f.GetDatabaseHostsFunc != nil {
		return f.GetDatabaseHostsFunc(ctx)
	}
	return nil, nil
}

func (f *DatabaseService) CreateDatabase(ctx context.Context, input *ports.DatabaseInput) (*domain.Database, error) {
	if f.CreateDatabaseFunc != nil {
		return f.CreateDatabaseFunc(ctx, input)
	}
	return nil, nil
}

func (f *DatabaseService) DeleteDatabase(ctx context.Context, id uint) error {
	if f.DeleteDatabaseFunc != nil {
		return f.DeleteDatabaseFunc(ctx, id)
	}
	return nil
}

func (f *DatabaseService) GetDatabasesByUserID(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	if f.GetDatabasesByUserIDFunc != nil {
		return f.GetDatabasesByUserIDFunc(ctx, userID, params)
	}
	return nil, 0, nil
}

func (f *DatabaseService) GetDatabaseByIDAndUserID(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error) {
	if f.GetDatabaseByIDAndUserIDFunc != nil {
		return f.GetDatabaseByIDAndUserIDFunc(ctx, databaseID, userID)
	}
	return nil, nil
}

