package migration_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	migration "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

func TestGetMigrationsByDatabaseID_ZeroDatabaseID_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	migrations, total, err := svc.GetMigrationsByDatabaseID(context.Background(), 0, 1, pagination.PaginationParams{Page: 1, Limit: 10}, false)

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || migrations != nil || total != 0 {
		t.Fatalf("expected ErrInvalidMigrationInput, got err=%v, migrations=%v, total=%d", err, migrations, total)
	}
}

func TestGetMigrationsByDatabaseID_ZeroUserID_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	migrations, total, err := svc.GetMigrationsByDatabaseID(context.Background(), 1, 0, pagination.PaginationParams{Page: 1, Limit: 10}, false)

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || migrations != nil || total != 0 {
		t.Fatalf("expected ErrInvalidMigrationInput, got err=%v, migrations=%v, total=%d", err, migrations, total)
	}
}

func TestGetMigrationsByDatabaseID_UserNotAuthorized_ReturnsAuthorizationError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("unauthorized database access")
	ugSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, userID, databaseID uint, requiredAccessType domain.AccessType) error {
			return expectedErr
		},
	}
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		ugSvc,
	)

	// Act
	migrations, total, err := svc.GetMigrationsByDatabaseID(context.Background(), 10, 5, pagination.PaginationParams{Page: 1, Limit: 10}, false)

	// Assert
	if !errors.Is(err, expectedErr) || migrations != nil || total != 0 {
		t.Fatalf("expected %v, got err=%v, migrations=%v, total=%d", expectedErr, err, migrations, total)
	}
}

func TestGetMigrationsByDatabaseID_StorageDependencyFails_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &testutil.MigrationRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint, params pagination.PaginationParams, hideRollback bool) ([]*domain.Migration, int64, error) {
			return nil, 0, expectedErr
		},
	}
	svc := migration.NewMigrationService(
		repo,
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	migrations, total, err := svc.GetMigrationsByDatabaseID(context.Background(), 10, 5, pagination.PaginationParams{Page: 1, Limit: 10}, false)

	// Assert
	if !errors.Is(err, expectedErr) || migrations != nil || total != 0 {
		t.Fatalf("expected %v, got err=%v, migrations=%v, total=%d", expectedErr, err, migrations, total)
	}
}

func TestGetMigrationsByDatabaseID_ValidInput_ReturnsMigrationsAndTotal(t *testing.T) {
	// Arrange
	expectedMigrations := []*domain.Migration{
		{ID: 1, DatabaseID: 10, Version: "1", Status: domain.MigrationStatusSuccess},
		{ID: 2, DatabaseID: 10, Version: "2", Status: domain.MigrationStatusSuccess},
	}
	expectedTotal := int64(2)
	repo := &testutil.MigrationRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint, params pagination.PaginationParams, hideRollback bool) ([]*domain.Migration, int64, error) {
			return expectedMigrations, expectedTotal, nil
		},
	}
	svc := migration.NewMigrationService(
		repo,
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	migrations, total, err := svc.GetMigrationsByDatabaseID(context.Background(), 10, 5, pagination.PaginationParams{Page: 1, Limit: 10}, false)

	// Assert
	if err != nil || total != expectedTotal || len(migrations) != 2 {
		t.Fatalf("expected %d migrations with total %d, got err=%v, count=%d, total=%d", len(expectedMigrations), expectedTotal, err, len(migrations), total)
	}
}

func TestGetMigrationByIDAndDatabaseID_ZeroMigrationID_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	mig, count, err := svc.GetMigrationByIDAndDatabaseID(context.Background(), 0, 10, 5)

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || mig != nil || count != 0 {
		t.Fatalf("expected ErrInvalidMigrationInput, got err=%v, mig=%v, count=%d", err, mig, count)
	}
}

func TestGetMigrationByIDAndDatabaseID_ZeroDatabaseID_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	mig, count, err := svc.GetMigrationByIDAndDatabaseID(context.Background(), 1, 0, 5)

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || mig != nil || count != 0 {
		t.Fatalf("expected ErrInvalidMigrationInput, got err=%v, mig=%v, count=%d", err, mig, count)
	}
}

func TestGetMigrationByIDAndDatabaseID_ZeroUserID_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	mig, count, err := svc.GetMigrationByIDAndDatabaseID(context.Background(), 1, 10, 0)

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || mig != nil || count != 0 {
		t.Fatalf("expected ErrInvalidMigrationInput, got err=%v, mig=%v, count=%d", err, mig, count)
	}
}

func TestGetMigrationByIDAndDatabaseID_UserNotAuthorized_ReturnsAuthorizationError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("forbidden")
	ugSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, userID, databaseID uint, requiredAccessType domain.AccessType) error {
			return expectedErr
		},
	}
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		ugSvc,
	)

	// Act
	mig, count, err := svc.GetMigrationByIDAndDatabaseID(context.Background(), 1, 10, 5)

	// Assert
	if !errors.Is(err, expectedErr) || mig != nil || count != 0 {
		t.Fatalf("expected %v, got err=%v, mig=%v, count=%d", expectedErr, err, mig, count)
	}
}

func TestGetMigrationByIDAndDatabaseID_StorageDependencyFails_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("record not found")
	repo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return nil, expectedErr
		},
	}
	svc := migration.NewMigrationService(
		repo,
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	mig, count, err := svc.GetMigrationByIDAndDatabaseID(context.Background(), 1, 10, 5)

	// Assert
	if !errors.Is(err, expectedErr) || mig != nil || count != 0 {
		t.Fatalf("expected %v, got err=%v, mig=%v, count=%d", expectedErr, err, mig, count)
	}
}

func TestGetMigrationByIDAndDatabaseID_Found_ReturnsMigrationAndCountOne(t *testing.T) {
	// Arrange
	expectedMigration := &domain.Migration{ID: 1, DatabaseID: 10, Version: "1", Status: domain.MigrationStatusSuccess}
	repo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return expectedMigration, nil
		},
	}
	svc := migration.NewMigrationService(
		repo,
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	mig, count, err := svc.GetMigrationByIDAndDatabaseID(context.Background(), 1, 10, 5)

	// Assert
	if err != nil || mig != expectedMigration || count != 1 {
		t.Fatalf("expected migration %+v and count 1, got err=%v, mig=%+v, count=%d", expectedMigration, err, mig, count)
	}
}

func TestCompareMigrations_InvalidInput_ReturnsValidationError(t *testing.T) {
	// Arrange
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	comparison, err := svc.CompareMigrations(context.Background(), domain.CompareMigrationsInput{
		DatabaseID:        0,
		UserID:            1,
		FirstMigrationID:  1,
		SecondMigrationID: 2,
	})

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationComparison) || comparison != nil {
		t.Fatalf("expected ErrInvalidMigrationComparison, got err=%v, comparison=%v", err, comparison)
	}
}

func TestCompareMigrations_UserNotAuthorized_ReturnsAuthorizationError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("access denied")
	ugSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, userID, databaseID uint, requiredAccessType domain.AccessType) error {
			return expectedErr
		},
	}
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		ugSvc,
	)

	// Act
	comparison, err := svc.CompareMigrations(context.Background(), domain.CompareMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		FirstMigrationID:  1,
		SecondMigrationID: 2,
	})

	// Assert
	if !errors.Is(err, expectedErr) || comparison != nil {
		t.Fatalf("expected %v, got err=%v, comparison=%v", expectedErr, err, comparison)
	}
}

func TestCompareMigrations_FirstMigrationNotFound_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("first migration not found")
	repo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			if migrationID == 1 {
				return nil, expectedErr
			}
			return &domain.Migration{ID: 2}, nil
		},
	}
	svc := migration.NewMigrationService(
		repo,
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	comparison, err := svc.CompareMigrations(context.Background(), domain.CompareMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		FirstMigrationID:  1,
		SecondMigrationID: 2,
	})

	// Assert
	if !errors.Is(err, expectedErr) || comparison != nil {
		t.Fatalf("expected %v, got err=%v, comparison=%v", expectedErr, err, comparison)
	}
}

func TestCompareMigrations_SecondMigrationNotFound_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("second migration not found")
	repo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			if migrationID == 2 {
				return nil, expectedErr
			}
			return &domain.Migration{ID: 1}, nil
		},
	}
	svc := migration.NewMigrationService(
		repo,
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	comparison, err := svc.CompareMigrations(context.Background(), domain.CompareMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		FirstMigrationID:  1,
		SecondMigrationID: 2,
	})

	// Assert
	if !errors.Is(err, expectedErr) || comparison != nil {
		t.Fatalf("expected %v, got err=%v, comparison=%v", expectedErr, err, comparison)
	}
}

func TestCompareMigrations_BothFoundIdentical_ReturnsComparisonWithIsIdenticalTrue(t *testing.T) {
	// Arrange
	first := &domain.Migration{
		ID:       1,
		SqlUp:    "CREATE TABLE test (id INT);",
		SqlDown:  "DROP TABLE test;",
		Snapshot: []byte(`{"Tables":{}}`),
	}
	second := &domain.Migration{
		ID:       2,
		SqlUp:    "CREATE TABLE test (id INT);",
		SqlDown:  "DROP TABLE test;",
		Snapshot: []byte(`{"Tables":{}}`),
	}
	repo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			if migrationID == 1 {
				return first, nil
			}
			return second, nil
		},
	}
	svc := migration.NewMigrationService(
		repo,
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	comparison, err := svc.CompareMigrations(context.Background(), domain.CompareMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		FirstMigrationID:  1,
		SecondMigrationID: 2,
	})

	// Assert
	if err != nil || !comparison.SQLUpEqual || !comparison.SQLDownEqual || !comparison.SchemaDiff.IsIdentical {
		t.Fatalf("expected identical comparison, got err=%v, SQLUpEqual=%v, SQLDownEqual=%v, IsIdentical=%v", err, comparison.SQLUpEqual, comparison.SQLDownEqual, comparison.SchemaDiff.IsIdentical)
	}
}

func TestCompareMigrations_BothFoundDifferent_ReturnsComparisonWithDifferences(t *testing.T) {
	// Arrange
	first := &domain.Migration{
		ID:       1,
		SqlUp:    "CREATE TABLE first (id INT);",
		SqlDown:  "DROP TABLE first;",
		Snapshot: []byte(`{"Tables":{"first":{"Schema":"public","Columns":[{"Name":"id","Type":"int"}]}}}`),
	}
	second := &domain.Migration{
		ID:       2,
		SqlUp:    "CREATE TABLE second (id INT);",
		SqlDown:  "DROP TABLE second;",
		Snapshot: []byte(`{"Tables":{"second":{"Schema":"public","Columns":[{"Name":"id","Type":"int"}]}}}`),
	}
	repo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			if migrationID == 1 {
				return first, nil
			}
			return second, nil
		},
	}
	svc := migration.NewMigrationService(
		repo,
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	comparison, err := svc.CompareMigrations(context.Background(), domain.CompareMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		FirstMigrationID:  1,
		SecondMigrationID: 2,
	})

	// Assert
	if err != nil || comparison.SQLUpEqual || comparison.SQLDownEqual || comparison.SchemaDiff.IsIdentical {
		t.Fatalf("expected different comparison, got err=%v, SQLUpEqual=%v, SQLDownEqual=%v, IsIdentical=%v", err, comparison.SQLUpEqual, comparison.SQLDownEqual, comparison.SchemaDiff.IsIdentical)
	}
}

func TestCreateMigration_InvalidInput_ReturnsValidationError(t *testing.T) {
	// Arrange
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 0,
		UserID:     1,
		SqlUp:      "CREATE TABLE x (id INT);",
	})

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || result != nil {
		t.Fatalf("expected ErrInvalidMigrationInput, got err=%v, result=%v", err, result)
	}
}

func TestCreateMigration_UserNotAuthorized_ReturnsAuthorizationError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("insufficient permissions to edit database")
	ugSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, userID, databaseID uint, requiredAccessType domain.AccessType) error {
			return expectedErr
		},
	}
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		ugSvc,
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE x (id INT);",
	})

	// Assert
	if !errors.Is(err, expectedErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", expectedErr, err, result)
	}
}

func TestCreateMigration_ConcurrentMigrationRunning_ReturnsErrMigrationInProgress(t *testing.T) {
	// Arrange
	inProgress := make(chan struct{})
	release := make(chan struct{})

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			close(inProgress)
			<-release
			return domain.Database{ID: dbID}, nil
		},
	}
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	go func() {
		_, _ = svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
			DatabaseID: 10,
			UserID:     1,
			SqlUp:      "CREATE TABLE background (id INT);",
		})
	}()

	<-inProgress

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE concurrent (id INT);",
	})
	close(release)

	// Assert
	if !errors.Is(err, domain.ErrMigrationInProgress) || result != nil {
		t.Fatalf("expected ErrMigrationInProgress, got err=%v, result=%v", err, result)
	}
}

func TestCreateMigration_DatabaseNotFoundOrCredentialError_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database not found")
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{}, expectedErr
		},
	}
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE x (id INT);",
	})

	// Assert
	if !errors.Is(err, expectedErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", expectedErr, err, result)
	}
}

func TestCreateMigration_EnsureBaselineStorageDependencyFails_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("failure reading latest migration")
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID, Host: "localhost"}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return nil, expectedErr
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE x (id INT);",
	})

	// Assert
	if !errors.Is(err, expectedErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", expectedErr, err, result)
	}
}

func TestCreateMigration_InitialBaselineSaveFails_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("baseline save failed")
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID, Host: "localhost"}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return nil, domain.ErrMigrationNotFound
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			return nil, expectedErr
		},
	}
	driver := &testutil.DatabaseDriver{
		GetDatabaseMetadataFunc: func(ctx context.Context, db domain.Database) (*ports.DatabaseMetadata, error) {
			return &ports.DatabaseMetadata{
				Tables: []ports.DatabaseTableMetadata{{Name: "initial", Schema: "public"}},
			}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE x (id INT);",
	})

	// Assert
	if !errors.Is(err, expectedErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", expectedErr, err, result)
	}
}

func TestCreateMigration_CorruptedLatestVersionString_ReturnsErrMigrationVersionUnavailable(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID, Host: "localhost"}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "not-a-number"}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE x (id INT);",
	})

	// Assert
	if !errors.Is(err, domain.ErrMigrationVersionUnavailable) || result != nil {
		t.Fatalf("expected ErrMigrationVersionUnavailable, got err=%v, result=%v", err, result)
	}
}

func TestCreateMigration_PendingMigrationSaveFails_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("failed to save pending migration")
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID, Host: "localhost"}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1"}, nil
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			return nil, expectedErr
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE x (id INT);",
	})

	// Assert
	if !errors.Is(err, expectedErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", expectedErr, err, result)
	}
}

func TestCreateMigration_GenerateSQLDownFails_RecordsMigrationAsFailed(t *testing.T) {
	// Arrange
	downErr := errors.New("unsupported DDL construct for down generation")
	var updatedMigration *domain.Migration
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID, Host: "localhost"}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1"}, nil
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 2
			return m, nil
		},
		UpdateFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			updatedMigration = m
			return m, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		GenerateSQLDownFunc: func(ctx context.Context, db domain.Database, sqlUp string) (string, error) {
			return "", downErr
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE custom (id INT);",
	})

	// Assert
	if err != nil || result == nil || updatedMigration == nil || updatedMigration.Status != domain.MigrationStatusFail || updatedMigration.ErrorMessage != downErr.Error() {
		t.Fatalf("expected recorded failed migration without error, got err=%v, result=%v, updated=%+v", err, result, updatedMigration)
	}
}

func TestCreateMigration_ExecuteCommandFails_RecordsMigrationAsFailed(t *testing.T) {
	// Arrange
	execErr := errors.New("syntax error at or near 'INVALID'")
	var updatedMigration *domain.Migration
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID, Host: "localhost"}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1"}, nil
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 2
			return m, nil
		},
		UpdateFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			updatedMigration = m
			return m, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		GenerateSQLDownFunc: func(ctx context.Context, db domain.Database, sqlUp string) (string, error) {
			return "DROP TABLE invalid;", nil
		},
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return nil, execErr
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "INVALID SQL;",
	})

	// Assert
	if err != nil || result == nil || updatedMigration == nil || updatedMigration.Status != domain.MigrationStatusFail || updatedMigration.ErrorMessage != execErr.Error() {
		t.Fatalf("expected recorded failed migration without error, got err=%v, result=%v, updated=%+v", err, result, updatedMigration)
	}
}

func TestCreateMigration_FailMigrationStorageFails_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	updateErr := errors.New("update migration failed")
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID, Host: "localhost"}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1"}, nil
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 2
			return m, nil
		},
		UpdateFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			return nil, updateErr
		},
	}
	driver := &testutil.DatabaseDriver{
		GenerateSQLDownFunc: func(ctx context.Context, db domain.Database, sqlUp string) (string, error) {
			return "", errors.New("sql down failed")
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE custom (id INT);",
	})

	// Assert
	if !errors.Is(err, updateErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", updateErr, err, result)
	}
}

func TestCreateMigration_CaptureSnapshotFails_ReturnsSnapshotUnavailable(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID, Host: "localhost"}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1"}, nil
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 2
			return m, nil
		},
		UpdateFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			return m, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		GenerateSQLDownFunc: func(ctx context.Context, db domain.Database, sqlUp string) (string, error) {
			return "DROP TABLE test;", nil
		},
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
		GetDatabaseMetadataFunc: func(ctx context.Context, db domain.Database) (*ports.DatabaseMetadata, error) {
			return nil, errors.New("metadata query timeout")
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE test (id INT);",
	})

	// Assert
	if !errors.Is(err, domain.ErrMigrationSnapshotUnavailable) || result != nil {
		t.Fatalf("expected snapshot error, got err=%v, result=%+v", err, result)
	}
}

func TestCreateMigration_ValidInputWithExistingBaseline_SucceedsWithIncrementedVersion(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID, Host: "localhost"}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "2"}, nil
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 3
			return m, nil
		},
		UpdateFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			return m, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		GenerateSQLDownFunc: func(ctx context.Context, db domain.Database, sqlUp string) (string, error) {
			return "DROP TABLE products;", nil
		},
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
		GetDatabaseMetadataFunc: func(ctx context.Context, db domain.Database) (*ports.DatabaseMetadata, error) {
			return &ports.DatabaseMetadata{
				Tables: []ports.DatabaseTableMetadata{{Name: "products", Schema: "public"}},
			}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE products (id INT);",
	})

	// Assert
	if err != nil || result == nil || result.Version != "3" || result.Status != domain.MigrationStatusSuccess || len(result.Snapshot) == 0 {
		t.Fatalf("expected version 3 success migration with snapshot, got err=%v, result=%+v", err, result)
	}
}

func TestCreateMigration_ValidInputInitialBaselineCreated_CreatesBaselineThenMigration(t *testing.T) {
	// Arrange
	savedMigrations := make([]*domain.Migration, 0)
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID, Host: "localhost"}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return nil, domain.ErrMigrationNotFound
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = uint(len(savedMigrations) + 1)
			savedMigrations = append(savedMigrations, m)
			return m, nil
		},
		UpdateFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			return m, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		GenerateSQLDownFunc: func(ctx context.Context, db domain.Database, sqlUp string) (string, error) {
			return "DROP TABLE items;", nil
		},
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
		GetDatabaseMetadataFunc: func(ctx context.Context, db domain.Database) (*ports.DatabaseMetadata, error) {
			return &ports.DatabaseMetadata{
				Tables: []ports.DatabaseTableMetadata{{Name: "items", Schema: "public"}},
			}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.CreateMigration(context.Background(), domain.CreateMigrationInput{
		DatabaseID: 10,
		UserID:     1,
		SqlUp:      "CREATE TABLE items (id INT);",
	})

	// Assert
	if err != nil || result == nil || result.Version != "2" || len(savedMigrations) != 2 {
		t.Fatalf("expected baseline (v1) and migration (v2), got err=%v, result=%+v, savedCount=%d", err, result, len(savedMigrations))
	}
	if savedMigrations[0].Version != "1" || !strings.Contains(savedMigrations[0].SqlDown, "Baseline schema snapshot") {
		t.Fatalf("expected first saved migration to be baseline v1, got %+v", savedMigrations[0])
	}
}

func TestMigrationService_Classify_Irreversible_ReturnsFalseWithReason(t *testing.T) {
	// Arrange
	queries := []string{"DROP TABLE t", "alter table t drop column a", "drop database test", "TRUNCATE t", "CREATE TABLE t(id int); DELETE FROM t"}
	// Act
	for _, sql := range queries {
		reversible, reason := domain.ClassifyMigration(sql)
		// Assert
		if reversible != false || (strings.TrimSpace(reason) == "") != reversible {
			t.Errorf("%s: %t %q", sql, reversible, reason)
		}
	}
}

func TestMigrationService_Classify_Reversible_ReturnsTrue(t *testing.T) {
	// Arrange
	queries := []string{"CREATE TABLE t(id int)", "ALTER TABLE t ADD COLUMN foo int", "CREATE INDEX i ON t(id)", "ALTER TABLE t RENAME TO b", "CREATE TABLE t(s text DEFAULT 'DROP TABLE')"}
	// Act
	for _, sql := range queries {
		reversible, reason := domain.ClassifyMigration(sql)
		// Assert
		if reversible != true || (strings.TrimSpace(reason) == "") != reversible {
			t.Errorf("%s: %t %q", sql, reversible, reason)
		}
	}
}

func TestMigrationService_GetMigrations_HideRollbackTrue_FiltersRevertedPairs(t *testing.T) {
	// Arrange
	var received bool
	repo := &testutil.MigrationRepository{
		FindAllByDatabaseIDFunc: func(_ context.Context, _ uint, _ pagination.PaginationParams, hide bool) ([]*domain.Migration, int64, error) {
			received = true
			if hide != true {
				t.Fatalf("filter: %v", hide)
			}
			return nil, 0, nil
		},
	}
	svc := migration.NewMigrationService(repo, nil, nil, &testutil.UserGroupService{})
	// Act
	_, _, err := svc.GetMigrationsByDatabaseID(context.Background(), 1, 1, pagination.PaginationParams{}, true)
	// Assert
	if err != nil || !received {
		t.Fatalf("history: %v", err)
	}
}

func TestMigrationService_GetMigrations_HideRollbackFalse_ReturnsAll(t *testing.T) {
	// Arrange
	var received bool
	repo := &testutil.MigrationRepository{
		FindAllByDatabaseIDFunc: func(_ context.Context, _ uint, _ pagination.PaginationParams, hide bool) ([]*domain.Migration, int64, error) {
			received = true
			if hide != false {
				t.Fatalf("filter: %v", hide)
			}
			return nil, 0, nil
		},
	}
	svc := migration.NewMigrationService(repo, nil, nil, &testutil.UserGroupService{})
	// Act
	_, _, err := svc.GetMigrationsByDatabaseID(context.Background(), 1, 1, pagination.PaginationParams{}, false)
	// Assert
	if err != nil || !received {
		t.Fatalf("history: %v", err)
	}
}
