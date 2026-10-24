package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type MigrationService interface {
	GetMigrationsByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error)
	CreateMigration(ctx context.Context, databaseID uint, userID uint, sqlUp string) (*domain.Migration, error)
	GetMigrationByIDAndDatabaseID(ctx context.Context, migrationID uint, databaseID uint) (*domain.Migration, int64, error)
}

type MigrationRepository interface {
	FindAllByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error)
	FindLatestByDatabaseID(ctx context.Context, databaseID uint) (*domain.Migration, error)
	Save(ctx context.Context, migration *domain.Migration) (*domain.Migration, error)
	FindByIDAndDatabaseID(ctx context.Context, migrationID uint, databaseID uint) (*domain.Migration, error)
}
