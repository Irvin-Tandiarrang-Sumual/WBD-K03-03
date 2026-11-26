package migration

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
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
	params pagination.PaginationParams, hideRollback bool,
) ([]*domain.Migration, int64, error) {
	if databaseID == 0 || userID == 0 {
		return nil, 0, domain.ErrInvalidMigrationInput
	}
	if err := s.userGroupsService.Authorize(ctx, userID, databaseID, domain.AccessViewOnly); err != nil {
		return nil, 0, err
	}
	return s.repository.FindAllByDatabaseID(ctx, databaseID, params, hideRollback)
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

	firstSnapshot := ParseSchemaSnapshot(first.Snapshot)
	secondSnapshot := ParseSchemaSnapshot(second.Snapshot)
	schemaDiff := DiffSchemaSnapshots(firstSnapshot, secondSnapshot)

	return &domain.MigrationComparison{
		FirstMigration:  first,
		SecondMigration: second,
		SQLUpEqual:      first.SqlUp == second.SqlUp,
		SQLDownEqual:    first.SqlDown == second.SqlDown,
		SchemaDiff:      schemaDiff,
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

	var result *domain.Migration
	err = s.repository.WithPosition(ctx, input.DatabaseID, func(repo ports.MigrationRepository, pos domain.MigrationPosition) error {
		var applyErr error
		result, applyErr = s.applyMigration(ctx, repo, pos, dbWithCred, input)
		return applyErr
	})
	if err == nil && result != nil && result.Status == domain.MigrationStatusSnapshotPending {
		return nil, domain.ErrMigrationSnapshotUnavailable
	}
	return result, err
}

func (s *migrationService) applyMigration(ctx context.Context, repo ports.MigrationRepository, pos domain.MigrationPosition, dbWithCred domain.Database, input domain.CreateMigrationInput) (*domain.Migration, error) {
	if err := s.completePendingSnapshot(ctx, repo, dbWithCred); err != nil {
		return nil, err
	}
	latest, err := s.ensureBaseline(ctx, repo, dbWithCred, input.UserID)
	if err != nil {
		return nil, err
	}

	if pos.HeadID == nil && latest != nil && latest.Status == domain.MigrationStatusSuccess {
		pos.HeadID = &latest.ID
		if err := repo.UpdatePosition(ctx, input.DatabaseID, pos.HeadID, nil, nil, pos.ExecutionID); err != nil {
			return nil, err
		}
	}
	version, err := s.resolveNextVersion(latest)
	if err != nil {
		return nil, err
	}

	migration, err := s.createPendingMigration(ctx, repo, input, version)
	if err != nil {
		return nil, err
	}

	migration.ParentMigrationID = pos.HeadID
	migration.ExecutionID = pos.ExecutionID + 1
	migration.IsReversible, migration.IrreversibleReason = domain.ClassifyMigration(input.SqlUp)
	if migration.IsReversible {
		sqlDown, err := s.databaseDriver.GenerateSQLDown(ctx, dbWithCred, input.SqlUp)
		if err != nil {
			return s.failMigration(ctx, repo, migration, err)
		}
		migration.SqlDown = sqlDown
		if sqlDown == "" {
			migration.IsReversible = false
			migration.IrreversibleReason = "No inverse SQL available"
		}
	}
	if _, err := s.databaseDriver.ExecuteCommand(ctx, dbWithCred, input.SqlUp); err != nil {
		return s.failMigration(ctx, repo, migration, err)
	}

	s.captureAppliedSnapshot(ctx, dbWithCred, migration)
	saved, err := repo.Update(ctx, migration)
	if err != nil {
		return nil, err
	}
	if err = repo.UpdatePosition(ctx, input.DatabaseID, &saved.ID, pos.HeadID, &saved.ID, migration.ExecutionID); err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *migrationService) ensureBaseline(ctx context.Context, repo ports.MigrationRepository, db domain.Database, userID uint) (*domain.Migration, error) {
	latest, err := repo.FindLatestByDatabaseID(ctx, db.ID)
	if err != nil && !errors.Is(err, domain.ErrMigrationNotFound) {
		return nil, err
	}

	if latest != nil || db.Host == "" {
		return latest, nil
	}

	return s.createInitialBaseline(ctx, repo, db, userID)
}

func (s *migrationService) createInitialBaseline(ctx context.Context, repo ports.MigrationRepository, db domain.Database, userID uint) (*domain.Migration, error) {
	baselineBytes, baselineStruct, snapErr := CaptureDatabaseSnapshot(ctx, s.databaseDriver, db)
	if snapErr != nil {
		return nil, snapErr
	}
	if len(baselineBytes) == 0 {
		return nil, domain.ErrMigrationSnapshotUnavailable
	}

	baseline := &domain.Migration{
		DatabaseID:         db.ID,
		Version:            "1",
		IrreversibleReason: "Initial schema baseline",
		Status:             domain.MigrationStatusSuccess,
		SqlUp:              FormatSnapshotAsBaselineSQL(baselineStruct),
		SqlDown:            "-- Baseline schema snapshot (initial state)",
		Snapshot:           baselineBytes,
		UserID:             userID,
	}

	savedBaseline, err := repo.Save(ctx, baseline)
	if err != nil {
		return nil, err
	}
	return savedBaseline, nil
}

func (s *migrationService) createPendingMigration(
	ctx context.Context, repo ports.MigrationRepository, input domain.CreateMigrationInput,
	version string,
) (*domain.Migration, error) {
	migration := &domain.Migration{
		DatabaseID: input.DatabaseID,
		Version:    version,
		Status:     domain.MigrationStatusFail,
		SqlUp:      input.SqlUp,
		UserID:     input.UserID,
	}
	return repo.Save(ctx, migration)
}

func (s *migrationService) failMigration(
	ctx context.Context, repo ports.MigrationRepository, migration *domain.Migration,
	failureErr error,
) (*domain.Migration, error) {
	migration.ErrorMessage = failureErr.Error()
	return repo.Update(ctx, migration)
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

// The target and history databases commit independently. If metadata is
// temporarily unavailable, record the committed SQL and finish its snapshot
// under the database lock before allowing another change to the schema.
func (s *migrationService) captureAppliedSnapshot(ctx context.Context, db domain.Database, m *domain.Migration) {
	snapshot, _, err := CaptureDatabaseSnapshot(ctx, s.databaseDriver, db)
	if err != nil || len(snapshot) == 0 {
		m.Status = domain.MigrationStatusSnapshotPending
		return
	}
	m.Snapshot = snapshot
	m.Status = domain.MigrationStatusSuccess
}

func (s *migrationService) completePendingSnapshot(ctx context.Context, repo ports.MigrationRepository, db domain.Database) error {
	pending, err := repo.FindPendingSnapshot(ctx, db.ID)
	if err != nil || pending == nil {
		return err
	}
	s.captureAppliedSnapshot(ctx, db, pending)
	if pending.Status != domain.MigrationStatusSuccess {
		return domain.ErrMigrationSnapshotUnavailable
	}
	_, err = repo.Update(ctx, pending)
	return err
}
