package testutil

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type MigrationRepository struct {
	ports.MigrationRepository
	FindAllByDatabaseIDFunc    func(context.Context, uint, pagination.PaginationParams, bool) ([]*domain.Migration, int64, error)
	FindLatestByDatabaseIDFunc func(context.Context, uint) (*domain.Migration, error)

	SaveFunc                  func(context.Context, *domain.Migration) (*domain.Migration, error)
	UpdateFunc                func(context.Context, *domain.Migration) (*domain.Migration, error)
	FindByIDAndDatabaseIDFunc func(context.Context, uint, uint) (*domain.Migration, error)
}

func (f *MigrationRepository) FindAllByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams, hideRollback bool) ([]*domain.Migration, int64, error) {
	if f.FindAllByDatabaseIDFunc != nil {
		return f.FindAllByDatabaseIDFunc(ctx, databaseID, params, hideRollback)
	}
	return nil, 0, nil
}

func (f *MigrationRepository) FindLatestByDatabaseID(ctx context.Context, databaseID uint) (*domain.Migration, error) {
	if f.FindLatestByDatabaseIDFunc != nil {
		return f.FindLatestByDatabaseIDFunc(ctx, databaseID)
	}
	return nil, nil
}

func (f *MigrationRepository) Save(ctx context.Context, migration *domain.Migration) (*domain.Migration, error) {
	if f.SaveFunc != nil {
		return f.SaveFunc(ctx, migration)
	}
	return migration, nil
}

func (f *MigrationRepository) Update(ctx context.Context, migration *domain.Migration) (*domain.Migration, error) {
	if f.UpdateFunc != nil {
		return f.UpdateFunc(ctx, migration)
	}
	return migration, nil
}

func (f *MigrationRepository) FindByIDAndDatabaseID(ctx context.Context, migrationID uint, databaseID uint) (*domain.Migration, error) {
	if f.FindByIDAndDatabaseIDFunc != nil {
		return f.FindByIDAndDatabaseIDFunc(ctx, migrationID, databaseID)
	}
	return nil, nil
}

func (f *MigrationRepository) WithPosition(ctx context.Context, id uint, fn func(ports.MigrationRepository, domain.MigrationPosition) error) error {
	return fn(f, domain.MigrationPosition{})
}
func (f *MigrationRepository) UpdatePosition(context.Context, uint, *uint, *uint, *uint, uint) error {
	return nil
}
func (f *MigrationRepository) FindPendingSnapshot(context.Context, uint) (*domain.Migration, error) {
	return nil, nil
}
