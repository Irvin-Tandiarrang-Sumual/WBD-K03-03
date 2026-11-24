package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type MigrationService interface {
	GetMigrationsByDatabaseID(ctx context.Context, databaseID uint, userID uint, params pagination.PaginationParams, hideRollback bool) ([]*domain.Migration, int64, error)
	CreateMigration(ctx context.Context, input domain.CreateMigrationInput) (*domain.Migration, error)
	GetMigrationByIDAndDatabaseID(ctx context.Context, migrationID uint, databaseID uint, userID uint) (*domain.Migration, int64, error)
	CompareMigrations(ctx context.Context, input domain.CompareMigrationsInput) (*domain.MigrationComparison, error)
	ProposeMigrationDraft(ctx context.Context, databaseID, userID uint, operation string) (*domain.MigrationDraft, error)
	VoteMigrationDraft(ctx context.Context, databaseID, draftID, userID uint, approve bool) error
	ExecuteMigrationDraft(ctx context.Context, databaseID, draftID, userID uint) (*domain.Migration, error)
}

type MigrationRepository interface {
	WithPosition(ctx context.Context, databaseID uint, fn func(MigrationRepository, domain.MigrationPosition) error) error
	UpdatePosition(ctx context.Context, databaseID uint, head, parent, forward *uint, executionID uint) error
	SaveDraft(ctx context.Context, draft *domain.MigrationDraft) error
	FindDraft(ctx context.Context, databaseID, draftID uint) (*domain.MigrationDraft, error)
	UpdateDraftStatus(ctx context.Context, draftID uint, status string) error
	Vote(ctx context.Context, draftID, userID uint, approve bool) error
	HasConsensus(ctx context.Context, databaseID, draftID uint) (bool, error)
	FinishDraft(ctx context.Context, draftID uint, audit *domain.Migration, head, parent, forward *uint) error
	FindPendingSnapshot(ctx context.Context, dbID uint) (*domain.Migration, error)
	FindActiveExecution(ctx context.Context, nodeID uint) (uint, error)
	FindAllByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams, hideRollback bool) ([]*domain.Migration, int64, error)
	FindLatestByDatabaseID(ctx context.Context, databaseID uint) (*domain.Migration, error)

	Save(ctx context.Context, migration *domain.Migration) (*domain.Migration, error)
	Update(ctx context.Context, migration *domain.Migration) (*domain.Migration, error)
	FindByIDAndDatabaseID(ctx context.Context, migrationID uint, databaseID uint) (*domain.Migration, error)
}
