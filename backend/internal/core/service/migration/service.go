package migration

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type migrationService struct {
	repository         ports.MigrationRepository
	databaseRepository ports.DatabaseRepository
	databaseExecutor   ports.DatabaseExecutor
	groupService       ports.GroupService
}

func NewMigrationService(
	repository ports.MigrationRepository,
	databaseRepository ports.DatabaseRepository,
	databaseExecutor ports.DatabaseExecutor,
	groupService ports.GroupService,
) ports.MigrationService {
	return &migrationService{
		repository:         repository,
		databaseRepository: databaseRepository,
		databaseExecutor:   databaseExecutor,
		groupService:       groupService,
	}
}

func (s *migrationService) GetMigrationsByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error) {
	return s.repository.FindAllByDatabaseID(ctx, databaseID, params)
}

func (s *migrationService) CreateMigration(ctx context.Context, databaseID uint, userID uint, sqlUp string) (*domain.Migration, error) {
	if databaseID == 0 || userID == 0 || sqlUp == "" {
		return nil, domain.ErrInvalidMigrationInput
	}

	access, err := s.groupService.EffectiveDatabaseAccess(ctx, userID, databaseID)
	if err != nil {
		return nil, err
	}
	if !access.Allows(domain.AccessViewEdit) {
		return nil, domain.ErrDatabaseAccessDenied
	}
	_, err = s.databaseRepository.FindByIDAndUserID(ctx, databaseID, userID)
	if err != nil {
		return nil, err
	}
	databaseWithCredential, err := s.databaseRepository.FindWithCredentialByID(ctx, databaseID)
	if err != nil {
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

	migration := &domain.Migration{
		DatabaseID: databaseID,
		Version:    version,
		Status:     "fail",
		SqlUp:      sqlUp,
		UserID:     userID,
	}
	migration, err = s.repository.Save(ctx, migration)
	if err != nil {
		return nil, err
	}

	sqlDown, generationErr := generateSQLDown(ctx, s.databaseExecutor, databaseWithCredential, sqlUp)
	if generationErr != nil {
		migration.ErrorMessage = generationErr.Error()
		return s.repository.Update(ctx, migration)
	}
	migration.SqlDown = sqlDown

	if _, executionErr := s.databaseExecutor.ExecuteCommand(ctx, databaseWithCredential, sqlUp); executionErr != nil {
		migration.ErrorMessage = executionErr.Error()
		return s.repository.Update(ctx, migration)
	}

	migration.Status = "success"
	return s.repository.Update(ctx, migration)
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

func (s *migrationService) CompareMigrations(ctx context.Context, databaseID uint, userID uint, firstMigrationID uint, secondMigrationID uint) (*ports.MigrationComparison, error) {
	if databaseID == 0 || userID == 0 || firstMigrationID == 0 || secondMigrationID == 0 || firstMigrationID == secondMigrationID {
		return nil, domain.ErrInvalidMigrationComparison
	}
	if _, err := s.databaseRepository.FindByIDAndUserID(ctx, databaseID, userID); err != nil {
		return nil, err
	}
	firstMigration, err := s.repository.FindByIDAndDatabaseID(ctx, firstMigrationID, databaseID)
	if err != nil {
		return nil, err
	}
	secondMigration, err := s.repository.FindByIDAndDatabaseID(ctx, secondMigrationID, databaseID)
	if err != nil {
		return nil, err
	}
	return &ports.MigrationComparison{
		FirstMigration:  firstMigration,
		SecondMigration: secondMigration,
		SQLUpEqual:      firstMigration.SqlUp == secondMigration.SqlUp,
		SQLDownEqual:    firstMigration.SqlDown == secondMigration.SqlDown,
	}, nil
}

func (s *migrationService) RollbackMigrations(ctx context.Context, databaseID uint, userID uint, targetMigrationID uint) ([]*domain.Migration, error) {
	if databaseID == 0 || userID == 0 || targetMigrationID == 0 {
		return nil, domain.ErrInvalidMigrationInput
	}
	access, err := s.groupService.EffectiveDatabaseAccess(ctx, userID, databaseID)
	if err != nil {
		return nil, err
	}
	if !access.Allows(domain.AccessViewEdit) {
		return nil, domain.ErrDatabaseAccessDenied
	}
	if _, err := s.databaseRepository.FindByIDAndUserID(ctx, databaseID, userID); err != nil {
		return nil, err
	}
	database, err := s.databaseRepository.FindWithCredentialByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}
	target, err := s.repository.FindByIDAndDatabaseID(ctx, targetMigrationID, databaseID)
	if err != nil {
		return nil, err
	}
	if target.Status != "success" {
		return nil, domain.ErrMigrationRollbackUnavailable
	}
	targetVersion, err := strconv.Atoi(target.Version)
	if err != nil {
		return nil, domain.ErrMigrationVersionUnavailable
	}
	latest, err := s.repository.FindLatestSuccessfulByDatabaseID(ctx, databaseID)
	if err != nil {
		return nil, err
	}
	latestVersion, err := strconv.Atoi(latest.Version)
	if err != nil || targetVersion > latestVersion {
		return nil, domain.ErrMigrationRollbackUnavailable
	}

	rolledBack := make([]*domain.Migration, 0)
	for version := latestVersion; version > targetVersion; version-- {
		migration, err := s.repository.FindByVersionAndDatabaseID(ctx, strconv.Itoa(version), databaseID)
		if err != nil {
			return nil, err
		}
		if migration.Status != "success" {
			continue
		}
		if migration.SqlDown == "" {
			return nil, domain.ErrMigrationRollbackUnavailable
		}
		if _, err := s.databaseExecutor.ExecuteCommand(ctx, database, migration.SqlDown); err != nil {
			return nil, err
		}
		rolledBack = append(rolledBack, migration)
	}

	if len(rolledBack) == 0 {
		return rolledBack, nil
	}

	sqlUpStatements := make([]string, 0, len(rolledBack))
	sqlDownStatements := make([]string, 0, len(rolledBack))
	for _, migration := range rolledBack {
		sqlUpStatements = append(sqlUpStatements, migration.SqlDown)
	}
	for index := len(rolledBack) - 1; index >= 0; index-- {
		migration := rolledBack[index]
		sqlDownStatements = append(sqlDownStatements, migration.SqlUp)
	}

	latestAttempt, err := s.repository.FindLatestByDatabaseID(ctx, databaseID)
	if err != nil {
		return nil, err
	}
	latestAttemptVersion, err := strconv.Atoi(latestAttempt.Version)
	if err != nil {
		return nil, domain.ErrMigrationVersionUnavailable
	}

	rollbackMigration, err := s.repository.Save(ctx, &domain.Migration{
		DatabaseID: databaseID,
		Version:    strconv.Itoa(latestAttemptVersion + 1),
		Status:     "success",
		SqlUp:      strings.Join(sqlUpStatements, ";\n"),
		SqlDown:    strings.Join(sqlDownStatements, ";\n"),
		UserID:     userID,
	})
	if err != nil {
		return nil, err
	}
	rolledBack = append(rolledBack, rollbackMigration)
	return rolledBack, nil
}
