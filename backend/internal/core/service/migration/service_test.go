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
	migrations, total, err := svc.GetMigrationsByDatabaseID(context.Background(), 0, 1, pagination.PaginationParams{Page: 1, Limit: 10})

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
	migrations, total, err := svc.GetMigrationsByDatabaseID(context.Background(), 1, 0, pagination.PaginationParams{Page: 1, Limit: 10})

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
	migrations, total, err := svc.GetMigrationsByDatabaseID(context.Background(), 10, 5, pagination.PaginationParams{Page: 1, Limit: 10})

	// Assert
	if !errors.Is(err, expectedErr) || migrations != nil || total != 0 {
		t.Fatalf("expected %v, got err=%v, migrations=%v, total=%d", expectedErr, err, migrations, total)
	}
}

func TestGetMigrationsByDatabaseID_StorageDependencyFails_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &testutil.MigrationRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error) {
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
	migrations, total, err := svc.GetMigrationsByDatabaseID(context.Background(), 10, 5, pagination.PaginationParams{Page: 1, Limit: 10})

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
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error) {
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
	migrations, total, err := svc.GetMigrationsByDatabaseID(context.Background(), 10, 5, pagination.PaginationParams{Page: 1, Limit: 10})

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

func TestCreateMigration_CaptureSnapshotFails_StillSucceedsWithoutSnapshot(t *testing.T) {
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
	if err != nil || result == nil || result.Status != domain.MigrationStatusSuccess || len(result.Snapshot) != 0 {
		t.Fatalf("expected successful migration with empty snapshot, got err=%v, result=%+v", err, result)
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

func TestRollbackMigrations_InvalidInput_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	svc := migration.NewMigrationService(
		&testutil.MigrationRepository{},
		&testutil.DatabaseRepository{},
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        0,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || result != nil {
		t.Fatalf("expected ErrInvalidMigrationInput, got err=%v, result=%v", err, result)
	}
}

func TestRollbackMigrations_UserNotAuthorized_ReturnsAuthorizationError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("unauthorized rollback attempt")
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
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, expectedErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", expectedErr, err, result)
	}
}

func TestRollbackMigrations_DatabaseLocked_ReturnsErrMigrationInProgress(t *testing.T) {
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
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})
	close(release)

	// Assert
	if !errors.Is(err, domain.ErrMigrationInProgress) || result != nil {
		t.Fatalf("expected ErrMigrationInProgress, got err=%v, result=%v", err, result)
	}
}

func TestRollbackMigrations_DatabaseCredentialFails_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("db credential lookup failed")
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
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, expectedErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", expectedErr, err, result)
	}
}

func TestRollbackMigrations_TargetMigrationNotFound_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return nil, domain.ErrMigrationNotFound
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 999, // non-existent migration ID
	})

	// Assert
	if !errors.Is(err, domain.ErrMigrationNotFound) || result != nil {
		t.Fatalf("expected ErrMigrationNotFound, got err=%v, result=%v", err, result)
	}
}

func TestRollbackMigrations_TargetMigrationStatusNotSuccess_ReturnsErrMigrationRollbackUnavailable(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: migrationID, Status: domain.MigrationStatusFail}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, domain.ErrMigrationRollbackUnavailable) || result != nil {
		t.Fatalf("expected ErrMigrationRollbackUnavailable, got err=%v, result=%v", err, result)
	}
}

func TestRollbackMigrations_LatestSuccessfulNotFound_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("no successful migrations")
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: migrationID, Version: "1", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
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
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, expectedErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", expectedErr, err, result)
	}
}

func TestRollbackMigrations_TargetVersionInvalid_ReturnsErrMigrationVersionUnavailable(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: migrationID, Version: "invalid-target", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "2", Status: domain.MigrationStatusSuccess}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, domain.ErrMigrationVersionUnavailable) || result != nil {
		t.Fatalf("expected ErrMigrationVersionUnavailable, got err=%v, result=%v", err, result)
	}
}

func TestRollbackMigrations_LatestVersionInvalid_ReturnsErrMigrationVersionUnavailable(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: migrationID, Version: "1", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "invalid-latest", Status: domain.MigrationStatusSuccess}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, domain.ErrMigrationVersionUnavailable) || result != nil {
		t.Fatalf("expected ErrMigrationVersionUnavailable, got err=%v, result=%v", err, result)
	}
}

func TestRollbackMigrations_TargetVersionEqualToLatest_ReturnsErrMigrationRollbackUnavailable(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: migrationID, Version: "2", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "2", Status: domain.MigrationStatusSuccess}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 2,
	})

	// Assert
	if !errors.Is(err, domain.ErrMigrationRollbackUnavailable) || result != nil {
		t.Fatalf("expected ErrMigrationRollbackUnavailable, got err=%v, result=%v", err, result)
	}
}

func TestRollbackMigrations_TargetVersionHigherThanLatest_ReturnsErrMigrationRollbackUnavailable(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			// Target version 5 has not occurred yet (future / non-existent higher version)
			return &domain.Migration{ID: migrationID, Version: "5", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			// Current latest version is only 2
			return &domain.Migration{ID: 2, Version: "2", Status: domain.MigrationStatusSuccess}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		&testutil.DatabaseDriver{},
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 5,
	})

	// Assert
	if !errors.Is(err, domain.ErrMigrationRollbackUnavailable) || result != nil {
		t.Fatalf("expected ErrMigrationRollbackUnavailable when rolling back to future non-existent version, got err=%v, result=%v", err, result)
	}
}

func TestRollbackMigrations_MigrationInRollbackRangeNotFound_ExecutesCompensatingStatementsAndReturnsError(t *testing.T) {
	// Arrange
	rangeErr := errors.New("migration in range not found")
	var compensatingExecuted []string
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 3, Version: "3", Status: domain.MigrationStatusSuccess}, nil
		},
		FindByVersionAndDatabaseIDFunc: func(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
			if version == "3" {
				return &domain.Migration{
					ID:      3,
					Version: "3",
					Status:  domain.MigrationStatusSuccess,
					SqlUp:   "CREATE TABLE test3 (id INT);",
					SqlDown: "DROP TABLE test3;",
				}, nil
			}
			return nil, rangeErr
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			compensatingExecuted = append(compensatingExecuted, command)
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, rangeErr) || result != nil || len(compensatingExecuted) < 2 {
		t.Fatalf("expected error %v and compensating execution, got err=%v, result=%v, executed=%v", rangeErr, err, result, compensatingExecuted)
	}
	if compensatingExecuted[0] != "DROP TABLE test3" || compensatingExecuted[1] != "CREATE TABLE test3 (id INT)" {
		t.Fatalf("expected down statement then compensating up, got %v", compensatingExecuted)
	}
}

func TestRollbackMigrations_MissingSQLDownInRollbackRange_ExecutesCompensatingStatementsAndReturnsErrMigrationRollbackUnavailable(t *testing.T) {
	// Arrange
	var compensatingExecuted []string
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 3, Version: "3", Status: domain.MigrationStatusSuccess}, nil
		},
		FindByVersionAndDatabaseIDFunc: func(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
			if version == "3" {
				return &domain.Migration{
					ID:      3,
					Version: "3",
					Status:  domain.MigrationStatusSuccess,
					SqlUp:   "CREATE TABLE test3 (id INT);",
					SqlDown: "DROP TABLE test3;",
				}, nil
			}
			return &domain.Migration{
				ID:      2,
				Version: "2",
				Status:  domain.MigrationStatusSuccess,
				SqlUp:   "CREATE TABLE test2 (id INT);",
				SqlDown: "", // Missing SQL Down!
			}, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			compensatingExecuted = append(compensatingExecuted, command)
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, domain.ErrMigrationRollbackUnavailable) || result != nil || len(compensatingExecuted) < 2 {
		t.Fatalf("expected ErrMigrationRollbackUnavailable, got err=%v, result=%v, executed=%v", err, result, compensatingExecuted)
	}
}

func TestRollbackMigrations_ExecuteRollbackDownFails_ExecutesCompensatingStatementsAndReturnsExecutionError(t *testing.T) {
	// Arrange
	execErr := errors.New("foreign key constraint prevents table drop")
	var executedCommands []string
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "2", Status: domain.MigrationStatusSuccess}, nil
		},
		FindByVersionAndDatabaseIDFunc: func(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{
				ID:      2,
				Version: "2",
				Status:  domain.MigrationStatusSuccess,
				SqlUp:   "CREATE TABLE test (id INT);",
				SqlDown: "DROP TABLE test;",
			}, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			executedCommands = append(executedCommands, command)
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
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, execErr) || result != nil {
		t.Fatalf("expected execution error %v, got err=%v, result=%v", execErr, err, result)
	}
}

func TestRollbackMigrations_RecordRollbackMigrationFindLatestFails_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("failure finding latest attempt")
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "2", Status: domain.MigrationStatusSuccess}, nil
		},
		FindByVersionAndDatabaseIDFunc: func(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{
				ID:      2,
				Version: "2",
				Status:  domain.MigrationStatusSuccess,
				SqlUp:   "CREATE TABLE test (id INT);",
				SqlDown: "DROP TABLE test;",
			}, nil
		},
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return nil, expectedErr
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, expectedErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", expectedErr, err, result)
	}
}

func TestRollbackMigrations_RecordRollbackMigrationSaveFails_ReturnsRepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("audit rollback migration save failed")
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "2", Status: domain.MigrationStatusSuccess}, nil
		},
		FindByVersionAndDatabaseIDFunc: func(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{
				ID:      2,
				Version: "2",
				Status:  domain.MigrationStatusSuccess,
				SqlUp:   "CREATE TABLE test (id INT);",
				SqlDown: "DROP TABLE test;",
			}, nil
		},
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "2"}, nil
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			return nil, expectedErr
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, expectedErr) || result != nil {
		t.Fatalf("expected %v, got err=%v, result=%v", expectedErr, err, result)
	}
}

func TestRollbackMigrations_ValidRange_ExecutesDownStatementsAndRecordsAuditMigration(t *testing.T) {
	// Arrange
	var executedDowns []string
	var savedAudit *domain.Migration
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{
				ID:       1,
				Version:  "1",
				Status:   domain.MigrationStatusSuccess,
				Snapshot: []byte(`{"Tables":{"target":{"Schema":"public"}}}`),
			}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 3, Version: "3", Status: domain.MigrationStatusSuccess}, nil
		},
		FindByVersionAndDatabaseIDFunc: func(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
			if version == "3" {
				return &domain.Migration{
					ID:      3,
					Version: "3",
					Status:  domain.MigrationStatusSuccess,
					SqlUp:   "CREATE TABLE test3 (id INT);",
					SqlDown: "DROP TABLE test3;",
				}, nil
			}
			return &domain.Migration{
				ID:      2,
				Version: "2",
				Status:  domain.MigrationStatusSuccess,
				SqlUp:   "CREATE TABLE test2 (id INT);",
				SqlDown: "DROP TABLE test2;",
			}, nil
		},
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 3, Version: "3"}, nil
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 4
			savedAudit = m
			return m, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			executedDowns = append(executedDowns, command)
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if err != nil || len(result) != 3 || savedAudit == nil {
		t.Fatalf("expected 2 rolled back + 1 audit migration, got err=%v, count=%d, savedAudit=%+v", err, len(result), savedAudit)
	}
	if savedAudit.Version != "4" || savedAudit.Status != domain.MigrationStatusSuccess {
		t.Fatalf("expected audit migration version 4 success, got %+v", savedAudit)
	}
	if len(executedDowns) != 2 || executedDowns[0] != "DROP TABLE test3" || executedDowns[1] != "DROP TABLE test2" {
		t.Fatalf("expected down statements executed in reverse chronological order, got %v", executedDowns)
	}
}

func TestRollbackMigrations_RangeContainsFailedMigration_SkipsFailedMigrationAndRollsBackSuccesses(t *testing.T) {
	// Arrange
	var executedDowns []string
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{
				ID:      1,
				Version: "1",
				Status:  domain.MigrationStatusSuccess,
			}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 3, Version: "3", Status: domain.MigrationStatusSuccess}, nil
		},
		FindByVersionAndDatabaseIDFunc: func(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
			if version == "3" {
				return &domain.Migration{
					ID:      3,
					Version: "3",
					Status:  domain.MigrationStatusSuccess,
					SqlUp:   "CREATE TABLE test3 (id INT);",
					SqlDown: "DROP TABLE test3;",
				}, nil
			}
			// Version 2 was a failed migration attempt in the database
			return &domain.Migration{
				ID:      2,
				Version: "2",
				Status:  domain.MigrationStatusFail,
				SqlUp:   "CREATE TABLE test2_failed (id INT);",
				SqlDown: "DROP TABLE test2_failed;",
			}, nil
		},
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 3, Version: "3"}, nil
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 4
			return m, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			executedDowns = append(executedDowns, command)
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if err != nil || len(result) != 2 { // 1 success rollback + 1 audit migration
		t.Fatalf("expected 1 rolled back migration + 1 audit migration, got err=%v, count=%d", err, len(result))
	}
	if len(executedDowns) != 1 || executedDowns[0] != "DROP TABLE test3" {
		t.Fatalf("expected only successful migration v3 to be executed for down, got %v", executedDowns)
	}
}

func TestRollbackMigrations_CompensatingMultiStatementsExecutedInReverseOrder_ExecutesAllCompensatingStatements(t *testing.T) {
	// Arrange
	execErr := errors.New("migration 2 down execution failed")
	var compensatingCommands []string
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 3, Version: "3", Status: domain.MigrationStatusSuccess}, nil
		},
		FindByVersionAndDatabaseIDFunc: func(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
			if version == "3" {
				return &domain.Migration{
					ID:      3,
					Version: "3",
					Status:  domain.MigrationStatusSuccess,
					SqlUp:   "CREATE TABLE part1 (id INT);\nCREATE TABLE part2 (id INT);",
					SqlDown: "DROP TABLE part2;\nDROP TABLE part1;",
				}, nil
			}
			return &domain.Migration{
				ID:      2,
				Version: "2",
				Status:  domain.MigrationStatusSuccess,
				SqlUp:   "CREATE TABLE part0 (id INT);",
				SqlDown: "DROP TABLE part0;",
			}, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			if strings.Contains(command, "DROP TABLE part0") {
				// Migration 2 down fails
				return nil, execErr
			}
			if strings.Contains(command, "CREATE TABLE part") {
				// Compensating commands executed by rollbackExecutedStatements
				compensatingCommands = append(compensatingCommands, command)
			}
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, execErr) || result != nil || len(compensatingCommands) < 2 {
		t.Fatalf("expected execution error and multi-statement compensating execution, got err=%v, result=%v, compensating=%v", err, result, compensatingCommands)
	}
	if compensatingCommands[0] != "CREATE TABLE part1 (id INT)" || compensatingCommands[1] != "CREATE TABLE part2 (id INT)" {
		t.Fatalf("expected compensating statements executed in order, got %v", compensatingCommands)
	}
}

func TestRollbackMigrations_RecordRollbackMigrationCorruptVersion_ReturnsErrMigrationVersionUnavailable(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 1, Version: "1", Status: domain.MigrationStatusSuccess}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "2", Status: domain.MigrationStatusSuccess}, nil
		},
		FindByVersionAndDatabaseIDFunc: func(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{
				ID:      2,
				Version: "2",
				Status:  domain.MigrationStatusSuccess,
				SqlUp:   "CREATE TABLE test (id INT);",
				SqlDown: "DROP TABLE test;",
			}, nil
		},
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "NaN"}, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
	}
	svc := migration.NewMigrationService(
		migRepo,
		dbRepo,
		driver,
		&testutil.UserGroupService{},
	)

	// Act
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if !errors.Is(err, domain.ErrMigrationVersionUnavailable) || result != nil {
		t.Fatalf("expected ErrMigrationVersionUnavailable, got err=%v, result=%v", err, result)
	}
}

func TestRollbackMigrations_CapturesFreshSnapshotOnSuccess_SavesFreshSnapshotInAuditRecord(t *testing.T) {
	// Arrange
	var savedAudit *domain.Migration
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, dbID uint) (domain.Database, error) {
			return domain.Database{ID: dbID}, nil
		},
	}
	migRepo := &testutil.MigrationRepository{
		FindByIDAndDatabaseIDFunc: func(ctx context.Context, migrationID, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{
				ID:       1,
				Version:  "1",
				Status:   domain.MigrationStatusSuccess,
				Snapshot: []byte(`{"Tables":{"old_target":{"Schema":"public"}}}`),
			}, nil
		},
		FindLatestSuccessfulByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "2", Status: domain.MigrationStatusSuccess}, nil
		},
		FindByVersionAndDatabaseIDFunc: func(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
			return &domain.Migration{
				ID:      2,
				Version: "2",
				Status:  domain.MigrationStatusSuccess,
				SqlUp:   "CREATE TABLE test (id INT);",
				SqlDown: "DROP TABLE test;",
			}, nil
		},
		FindLatestByDatabaseIDFunc: func(ctx context.Context, dbID uint) (*domain.Migration, error) {
			return &domain.Migration{ID: 2, Version: "2"}, nil
		},
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 3
			savedAudit = m
			return m, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
		GetDatabaseMetadataFunc: func(ctx context.Context, db domain.Database) (*ports.DatabaseMetadata, error) {
			return &ports.DatabaseMetadata{
				Tables: []ports.DatabaseTableMetadata{
					{Name: "restored_table", Schema: "public"},
				},
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
	result, err := svc.RollbackMigrations(context.Background(), domain.RollbackMigrationsInput{
		DatabaseID:        10,
		UserID:            1,
		TargetMigrationID: 1,
	})

	// Assert
	if err != nil || len(result) != 2 || savedAudit == nil || len(savedAudit.Snapshot) == 0 {
		t.Fatalf("expected fresh snapshot saved in audit record, got err=%v, count=%d, savedAudit=%+v", err, len(result), savedAudit)
	}
	if !strings.Contains(string(savedAudit.Snapshot), "restored_table") {
		t.Fatalf("expected audit snapshot to contain fresh 'restored_table', got %s", string(savedAudit.Snapshot))
	}
}
