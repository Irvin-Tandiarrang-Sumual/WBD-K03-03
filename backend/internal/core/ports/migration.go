package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type MigrationService interface {
	GetMigrationsByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error)
	CreateMigration(ctx context.Context, input domain.CreateMigrationInput) (*domain.Migration, error)
	GetMigrationByIDAndDatabaseID(ctx context.Context, migrationID uint, databaseID uint) (*domain.Migration, int64, error)
	CompareMigrations(ctx context.Context, input domain.CompareMigrationsInput) (*MigrationComparison, error)
	RollbackMigrations(ctx context.Context, input domain.RollbackMigrationsInput) ([]*domain.Migration, error)
}

type MigrationComparison struct {
	FirstMigration  *domain.Migration
	SecondMigration *domain.Migration
	SQLUpEqual      bool
	SQLDownEqual    bool
}

type MigrationRepository interface {
	FindAllByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error)
	FindLatestByDatabaseID(ctx context.Context, databaseID uint) (*domain.Migration, error)
	FindLatestSuccessfulByDatabaseID(ctx context.Context, databaseID uint) (*domain.Migration, error)
	Save(ctx context.Context, migration *domain.Migration) (*domain.Migration, error)
	Update(ctx context.Context, migration *domain.Migration) (*domain.Migration, error)
	FindByIDAndDatabaseID(ctx context.Context, migrationID uint, databaseID uint) (*domain.Migration, error)
	FindByVersionAndDatabaseID(ctx context.Context, version string, databaseID uint) (*domain.Migration, error)
}
