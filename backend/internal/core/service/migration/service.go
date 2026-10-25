package migration

import (
	"context"
	"errors"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type migrationService struct {
	repository         ports.MigrationRepository
	databaseRepository ports.DatabaseRepository
	databaseExecutor   ports.DatabaseExecutor
}

func NewMigrationService(
	repository ports.MigrationRepository,
	databaseRepository ports.DatabaseRepository,
	databaseExecutor ports.DatabaseExecutor,
) ports.MigrationService {
	return &migrationService{
		repository:         repository,
		databaseRepository: databaseRepository,
		databaseExecutor:   databaseExecutor,
	}
}

func (s *migrationService) GetMigrationsByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error) {
	return s.repository.FindAllByDatabaseID(ctx, databaseID, params)
}

func (s *migrationService) CreateMigration(ctx context.Context, databaseID uint, userID uint, sqlUp string) (*domain.Migration, error) {
	if databaseID == 0 || userID == 0 || sqlUp == "" {
		return nil, domain.ErrInvalidMigrationInput
	}

	_, err := s.databaseRepository.FindByIDAndUserID(ctx, databaseID, userID)
	if err != nil {
		return nil, err
	}
	databaseWithCredential, err := s.databaseRepository.FindWithCredentialByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}
	sqlDown, err := generateSQLDown(ctx, s.databaseExecutor, databaseWithCredential, sqlUp)
	if err != nil {
		return nil, err
	}
	if _, err := s.databaseExecutor.ExecuteCommand(ctx, databaseWithCredential, sqlUp); err != nil {
		return nil, err
	}

	latest, err := s.repository.FindLatestByDatabaseID(ctx, databaseID)
	if err != nil && !errors.Is(err, domain.ErrMigrationNotFound) {
		return nil, err
	}

	version := "1"
	if latest != nil {
		latestVersion, parseErr := strconv.Atoi(latest.Version)
		if parseErr != nil {
			return nil, domain.ErrMigrationVersionUnavailable
		}
		version = strconv.Itoa(latestVersion + 1)
	}

	return s.repository.Save(ctx, &domain.Migration{
		DatabaseID: databaseID,
		Version:    version,
		Status:     "success",
		SqlUp:      sqlUp,
		SqlDown:    sqlDown,
		UserID:     userID,
	})
}

func (s *migrationService) GetMigrationByIDAndDatabaseID(ctx context.Context, migrationID uint, databaseID uint) (*domain.Migration, int64, error) {
	if migrationID == 0 || databaseID == 0 {
		return nil, 0, domain.ErrInvalidMigrationInput
	}

	migration, err := s.repository.FindByIDAndDatabaseID(ctx, migrationID, databaseID)
	if err != nil {
		return nil, 0, err
	}
	return migration, 1, nil
}
