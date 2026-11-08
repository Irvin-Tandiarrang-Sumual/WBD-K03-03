package migration

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
)

type migrationService struct {
	repository         ports.MigrationRepository
	databaseRepository ports.DatabaseRepository
	databaseDriver     ports.DatabaseDriver
	userGroupsService  ports.UserGroupService
	dbLocks            sync.Map
}

func NewMigrationService(
	repository ports.MigrationRepository,
	databaseRepository ports.DatabaseRepository,
	databaseDriver ports.DatabaseDriver,
	userGroupService ports.UserGroupService,
) ports.MigrationService {
	return &migrationService{
		repository:         repository,
		databaseRepository: databaseRepository,
		databaseDriver:     databaseDriver,
		userGroupsService:  userGroupService,
	}
}

func (s *migrationService) GetMigrationsByDatabaseID(
	ctx context.Context,
	databaseID uint,
	userID uint,
	params pagination.PaginationParams,
) ([]*domain.Migration, int64, error) {
	if databaseID == 0 || userID == 0 {
		return nil, 0, domain.ErrInvalidMigrationInput
	}
	if err := s.userGroupsService.Authorize(ctx, userID, databaseID, domain.AccessViewOnly); err != nil {
		return nil, 0, err
	}
	return s.repository.FindAllByDatabaseID(ctx, databaseID, params)
}

func (s *migrationService) GetMigrationByIDAndDatabaseID(
	ctx context.Context,
	migrationID uint,
	databaseID uint,
	userID uint,
) (*domain.Migration, int64, error) {
	if migrationID == 0 || databaseID == 0 || userID == 0 {
		return nil, 0, domain.ErrInvalidMigrationInput
	}
	if err := s.userGroupsService.Authorize(ctx, userID, databaseID, domain.AccessViewOnly); err != nil {
		return nil, 0, err
	}

	migration, err := s.repository.FindByIDAndDatabaseID(ctx, migrationID, databaseID)
	if err != nil {
		return nil, 0, err
	}
	return migration, 1, nil
}

func (s *migrationService) CompareMigrations(
	ctx context.Context,
	input domain.CompareMigrationsInput,
) (*domain.MigrationComparison, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if err := s.userGroupsService.Authorize(ctx, input.UserID, input.DatabaseID, domain.AccessViewOnly); err != nil {
		return nil, err
	}

	first, err := s.repository.FindByIDAndDatabaseID(ctx, input.FirstMigrationID, input.DatabaseID)
	if err != nil {
		return nil, err
	}

	second, err := s.repository.FindByIDAndDatabaseID(ctx, input.SecondMigrationID, input.DatabaseID)
	if err != nil {
		return nil, err
	}

	return &domain.MigrationComparison{
		FirstMigration:  first,
		SecondMigration: second,
		SQLUpEqual:      first.SqlUp == second.SqlUp,
		SQLDownEqual:    first.SqlDown == second.SqlDown,
	}, nil
}

func (s *migrationService) CreateMigration(ctx context.Context, input domain.CreateMigrationInput) (*domain.Migration, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if err := s.userGroupsService.Authorize(ctx, input.UserID, input.DatabaseID, domain.AccessViewEdit); err != nil {
		return nil, err
	}

	unlock, ok := s.tryLockDatabase(input.DatabaseID)
	if !ok {
		return nil, domain.ErrMigrationInProgress
	}
	defer unlock()

	dbWithCred, err := s.databaseRepository.FindWithCredentialByID(ctx, input.DatabaseID)
	if err != nil {
		return nil, err
	}

	latest, err := s.ensureBaseline(ctx, dbWithCred, input.UserID)
	if err != nil {
		return nil, err
	}

	version, err := s.resolveNextVersion(latest)
	if err != nil {
		return nil, err
	}

	migration, err := s.createPendingMigration(ctx, input, version)
	if err != nil {
		return nil, err
	}

	sqlDown, err := s.databaseDriver.GenerateSQLDown(ctx, dbWithCred, input.SqlUp)
	if err != nil {
		return s.failMigration(ctx, migration, err)
	}
	migration.SqlDown = sqlDown

	if _, err := s.databaseDriver.ExecuteCommand(ctx, dbWithCred, input.SqlUp); err != nil {
		return s.failMigration(ctx, migration, err)
	}

	if snapshotBytes, _, err := captureDatabaseSnapshot(ctx, s.databaseDriver, dbWithCred); err == nil && len(snapshotBytes) > 0 {
		migration.Snapshot = snapshotBytes
	}

	migration.Status = domain.MigrationStatusSuccess
	return s.repository.Update(ctx, migration)
}

func (s *migrationService) ensureBaseline(ctx context.Context, db domain.Database, userID uint) (*domain.Migration, error) {
	latest, err := s.repository.FindLatestByDatabaseID(ctx, db.ID)
	if err != nil && !errors.Is(err, domain.ErrMigrationNotFound) {
		return nil, err
	}

	if latest != nil || db.Host == "" {
		return latest, nil
	}

	return s.createInitialBaseline(ctx, db, userID)
}

func (s *migrationService) createInitialBaseline(ctx context.Context, db domain.Database, userID uint) (*domain.Migration, error) {
	baselineBytes, baselineStruct, snapErr := captureDatabaseSnapshot(ctx, s.databaseDriver, db)
	if snapErr != nil || len(baselineBytes) == 0 {
		return nil, nil
	}

	baseline := &domain.Migration{
		DatabaseID: db.ID,
		Version:    "1",
		Status:     domain.MigrationStatusSuccess,
		SqlUp:      formatSnapshotAsBaselineSQL(baselineStruct),
		SqlDown:    "-- Baseline schema snapshot (initial state)",
		Snapshot:   baselineBytes,
		UserID:     userID,
	}

	savedBaseline, err := s.repository.Save(ctx, baseline)
	if err != nil {
		return nil, err
	}
	return savedBaseline, nil
}

func (s *migrationService) RollbackMigrations(ctx context.Context, input domain.RollbackMigrationsInput) ([]*domain.Migration, error) {
	if err := input.Validate(); err != nil {
		return nil, domain.ErrInvalidMigrationInput
	}

	if err := s.userGroupsService.Authorize(ctx, input.UserID, input.DatabaseID, domain.AccessViewEdit); err != nil {
		return nil, err
	}

	unlock, ok := s.tryLockDatabase(input.DatabaseID)
	if !ok {
		return nil, domain.ErrMigrationInProgress
	}
	defer unlock()

	dbWithCred, err := s.databaseRepository.FindWithCredentialByID(ctx, input.DatabaseID)
	if err != nil {
		return nil, err
	}

	target, err := s.resolveTargetMigration(ctx, input.TargetMigrationID, input.DatabaseID)
	if err != nil {
		return nil, err
	}

	latest, err := s.repository.FindLatestSuccessfulByDatabaseID(ctx, input.DatabaseID)
	if err != nil {
		return nil, err
	}

	if err := s.validateRollbackOrder(target, latest); err != nil {
		return nil, err
	}

	rolledBack, err := s.executeRollbackRange(ctx, dbWithCred, target, latest)
	if err != nil {
		return nil, err
	}

	if len(rolledBack) == 0 {
		return rolledBack, nil
	}

	auditMigration, err := s.recordRollbackMigration(ctx, dbWithCred, input.UserID, target, rolledBack)
	if err != nil {
		return nil, err
	}

	return append(rolledBack, auditMigration), nil
}

func (s *migrationService) resolveTargetMigration(ctx context.Context, targetID, dbID uint) (*domain.Migration, error) {
	target, err := s.repository.FindByIDAndDatabaseID(ctx, targetID, dbID)
	if err != nil {
		return nil, err
	}
	if target.Status != domain.MigrationStatusSuccess {
		return nil, domain.ErrMigrationRollbackUnavailable
	}
	return target, nil
}

func (s *migrationService) validateRollbackOrder(target, latest *domain.Migration) error {
	targetVersion, err := strconv.Atoi(target.Version)
	if err != nil {
		return domain.ErrMigrationVersionUnavailable
	}

	latestVersion, err := strconv.Atoi(latest.Version)
	if err != nil {
		return domain.ErrMigrationVersionUnavailable
	}

	if targetVersion >= latestVersion {
		return domain.ErrMigrationRollbackUnavailable
	}

	return nil
}

func (s *migrationService) executeRollbackRange(
	ctx context.Context,
	db domain.Database,
	target, latest *domain.Migration,
) ([]*domain.Migration, error) {
	targetVer, _ := strconv.Atoi(target.Version)
	latestVer, _ := strconv.Atoi(latest.Version)

	rolledBack := make([]*domain.Migration, 0)
	compensatingUps := make([]string, 0)

	for version := latestVer; version > targetVer; version-- {
		migration, err := s.repository.FindByVersionAndDatabaseID(ctx, strconv.Itoa(version), db.ID)
		if err != nil {
			rollbackExecutedStatements(ctx, s.databaseDriver, db, compensatingUps)
			return nil, err
		}

		if migration.Status != domain.MigrationStatusSuccess {
			continue
		}

		if migration.SqlDown == "" {
			rollbackExecutedStatements(ctx, s.databaseDriver, db, compensatingUps)
			return nil, domain.ErrMigrationRollbackUnavailable
		}

		upsToCompensate, err := s.applyRollbackStatement(ctx, db, migration)
		if err != nil {
			rollbackExecutedStatements(ctx, s.databaseDriver, db, compensatingUps)
			return nil, err
		}

		compensatingUps = append(compensatingUps, upsToCompensate...)
		rolledBack = append(rolledBack, migration)
	}

	return rolledBack, nil
}

func (s *migrationService) applyRollbackStatement(
	ctx context.Context, db domain.Database,
	migration *domain.Migration,
) ([]string, error) {
	downStatements := sqlutil.SplitStatements(migration.SqlDown)
	upStatements := sqlutil.SplitStatements(migration.SqlUp)
	executedUps := make([]string, 0, len(downStatements))

	for idx, downStmt := range downStatements {
		if _, err := s.databaseDriver.ExecuteCommand(ctx, db, downStmt); err != nil {
			return nil, err
		}

		correspondingUpIndex := len(upStatements) - 1 - idx
		if correspondingUpIndex >= 0 && correspondingUpIndex < len(upStatements) {
			executedUps = append(executedUps, upStatements[correspondingUpIndex])
		}
	}

	return executedUps, nil
}

func (s *migrationService) recordRollbackMigration(
	ctx context.Context, db domain.Database, userID uint,
	target *domain.Migration, rolledBack []*domain.Migration,
) (*domain.Migration, error) {
	latestAttempt, err := s.repository.FindLatestByDatabaseID(ctx, db.ID)
	if err != nil {
		return nil, err
	}

	nextVersion, err := s.resolveNextVersion(latestAttempt)
	if err != nil {
		return nil, err
	}

	snapshot := s.resolveRollbackSnapshot(ctx, db, target)

	return s.repository.Save(ctx, &domain.Migration{
		DatabaseID: db.ID,
		Version:    nextVersion,
		Status:     domain.MigrationStatusSuccess,
		SqlUp:      buildRollbackSQLUp(rolledBack),
		SqlDown:    buildRollbackSQLDown(rolledBack),
		Snapshot:   snapshot,
		UserID:     userID,
	})
}

func (s *migrationService) resolveRollbackSnapshot(ctx context.Context, db domain.Database, target *domain.Migration) []byte {
	snapshotBytes, _, _ := captureDatabaseSnapshot(ctx, s.databaseDriver, db)
	if len(snapshotBytes) == 0 && len(target.Snapshot) > 0 {
		return target.Snapshot
	}
	return snapshotBytes
}

func buildRollbackSQLUp(rolledBack []*domain.Migration) string {
	statements := make([]string, 0, len(rolledBack))
	for _, m := range rolledBack {
		statements = append(statements, m.SqlDown)
	}
	return strings.Join(statements, ";\n")
}

func buildRollbackSQLDown(rolledBack []*domain.Migration) string {
	statements := make([]string, 0, len(rolledBack))
	for i := len(rolledBack) - 1; i >= 0; i-- {
		statements = append(statements, rolledBack[i].SqlUp)
	}
	return strings.Join(statements, ";\n")
}

func (s *migrationService) createPendingMigration(
	ctx context.Context, input domain.CreateMigrationInput,
	version string,
) (*domain.Migration, error) {
	migration := &domain.Migration{
		DatabaseID: input.DatabaseID,
		Version:    version,
		Status:     domain.MigrationStatusFail,
		SqlUp:      input.SqlUp,
		UserID:     input.UserID,
	}
	return s.repository.Save(ctx, migration)
}

func (s *migrationService) failMigration(
	ctx context.Context, migration *domain.Migration,
	failureErr error,
) (*domain.Migration, error) {
	migration.ErrorMessage = failureErr.Error()
	return s.repository.Update(ctx, migration)
}

func (s *migrationService) resolveNextVersion(latest *domain.Migration) (string, error) {
	if latest == nil {
		return "1", nil
	}
	latestVersion, err := strconv.Atoi(latest.Version)
	if err != nil {
		return "", domain.ErrMigrationVersionUnavailable
	}
	return strconv.Itoa(latestVersion + 1), nil
}

func (s *migrationService) tryLockDatabase(databaseID uint) (func(), bool) {
	val, _ := s.dbLocks.LoadOrStore(databaseID, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	if !mu.TryLock() {
		return nil, false
	}
	return mu.Unlock, true
}


func rollbackExecutedStatements(
	ctx context.Context, driver ports.DatabaseDriver,
	database domain.Database, executedDowns []string,
) {
	for index := len(executedDowns) - 1; index >= 0; index-- {
		downScript := executedDowns[index]
		subStatements := sqlutil.SplitStatements(downScript)
		if len(subStatements) <= 1 {
			if strings.TrimSpace(downScript) != "" {
				_, _ = driver.ExecuteCommand(ctx, database, downScript)
			}
			continue
		}
		for _, subStmt := range subStatements {
			_, _ = driver.ExecuteCommand(ctx, database, subStmt)
		}
	}
}
