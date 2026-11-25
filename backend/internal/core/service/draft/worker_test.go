package draft_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/draft"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

func TestExecutionWorker_Execute_Success_RecordsSuccessMigrationAndMarksDraftApplied(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 3, UserID: 4}

	draftObj := &domain.Draft{
		ID:           1,
		DatabaseID:   2,
		UserID:       4,
		SqlUp:        "CREATE TABLE test (id INT);",
		Status:       domain.DraftStatusExecuting,
		IsReversible: true,
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2, Host: "localhost", User: "postgres"}, nil
		},
		UpdateCurrentMigrationIDFunc: func(ctx context.Context, dbID, migID uint) error {
			return nil
		},
	}

	var recordedMigration *domain.Migration
	migrationRepo := &testutil.MigrationRepository{
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 100
			recordedMigration = m
			return m, nil
		},
	}

	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
	}

	unlocked := false
	unlockFn := func(dbID uint) {
		unlocked = true
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, unlockFn)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !unlocked {
		t.Fatalf("expected database lock to be released")
	}
	if draftObj.Status != domain.DraftStatusApplied {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusApplied, draftObj.Status)
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusSuccess {
		t.Fatalf("expected migration status %q, got %+v", domain.MigrationStatusSuccess, recordedMigration)
	}
}

func TestExecutionWorker_Execute_TransientError_RetriesAndSucceeds(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 3, UserID: 4}

	draftObj := &domain.Draft{
		ID:         1,
		DatabaseID: 2,
		SqlUp:      "CREATE TABLE test (id INT);",
		Status:     domain.DraftStatusExecuting,
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2}, nil
		},
		UpdateCurrentMigrationIDFunc: func(ctx context.Context, dbID, migID uint) error {
			return nil
		},
	}

	var recordedMigration *domain.Migration
	migrationRepo := &testutil.MigrationRepository{
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			recordedMigration = m
			return m, nil
		},
	}

	attempts := 0
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			attempts++
			if attempts == 1 {
				return nil, errors.New("deadlock detected (retryable)")
			}
			return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil)
	worker.SetRetryPolicy(3, 1*time.Millisecond)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error after retry, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if draftObj.Status != domain.DraftStatusApplied {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusApplied, draftObj.Status)
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusSuccess {
		t.Fatalf("expected migration status success, got %+v", recordedMigration)
	}
}

func TestExecutionWorker_Execute_TransientErrorExceedsMaxRetries_RecordsFailMigrationAndMarksDraftFailed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 3, UserID: 4}

	draftObj := &domain.Draft{
		ID:         1,
		DatabaseID: 2,
		SqlUp:      "CREATE TABLE test (id INT);",
		Status:     domain.DraftStatusExecuting,
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2}, nil
		},
	}

	var recordedMigration *domain.Migration
	migrationRepo := &testutil.MigrationRepository{
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			recordedMigration = m
			return m, nil
		},
	}

	attempts := 0
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			attempts++
			return nil, errors.New("deadlock detected")
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil)
	worker.SetRetryPolicy(2, 1*time.Millisecond)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err == nil {
		t.Fatalf("expected error after exceeding retries, got nil")
	}
	if attempts != 3 { // attempt 0, 1, 2 = 3 tries
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if draftObj.Status != domain.DraftStatusFailed {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusFailed, draftObj.Status)
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusFail {
		t.Fatalf("expected migration status fail, got %+v", recordedMigration)
	}
}

func TestExecutionWorker_Execute_TargetDBUnreachable_RecordsFailMigrationAndMarksDraftFailed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 3, UserID: 4}

	draftObj := &domain.Draft{
		ID:         1,
		DatabaseID: 2,
		SqlUp:      "CREATE TABLE test (id INT);",
		Status:     domain.DraftStatusExecuting,
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{}, domain.ErrDatabaseConnectionFailed
		},
	}

	var recordedMigration *domain.Migration
	migrationRepo := &testutil.MigrationRepository{
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			recordedMigration = m
			return m, nil
		},
	}

	driver := &testutil.DatabaseDriver{}
	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if !errors.Is(err, domain.ErrDatabaseConnectionFailed) {
		t.Fatalf("expected ErrDatabaseConnectionFailed, got %v", err)
	}
	if draftObj.Status != domain.DraftStatusFailed {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusFailed, draftObj.Status)
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusFail {
		t.Fatalf("expected fail migration, got %+v", recordedMigration)
	}
}

func TestExecutionWorker_Execute_ContextTimeout_AbortsAndRecordsFailMigration(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 3, UserID: 4}
	draftObj := &domain.Draft{
		ID:         1,
		DatabaseID: 2,
		SqlUp:      "CREATE TABLE test (id INT);",
		Status:     domain.DraftStatusExecuting,
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2}, nil
		},
	}

	var recordedMigration *domain.Migration
	migrationRepo := &testutil.MigrationRepository{
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			recordedMigration = m
			return m, nil
		},
	}

	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			time.Sleep(50 * time.Millisecond) // exceeds context timeout
			return nil, ctx.Err()
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil)
	worker.SetRetryPolicy(0, time.Millisecond)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context timeout error, got %v", err)
	}
	if draftObj.Status != domain.DraftStatusFailed {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusFailed, draftObj.Status)
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusFail {
		t.Fatalf("expected fail migration, got %+v", recordedMigration)
	}
}

func TestExecutionWorker_Execute_WorkerPanics_RecoversGracefullyAndMarksDraftFailed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 3, UserID: 4}

	draftObj := &domain.Draft{
		ID:         1,
		DatabaseID: 2,
		SqlUp:      "CREATE TABLE test (id INT);",
		Status:     domain.DraftStatusExecuting,
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2}, nil
		},
	}

	var recordedMigration *domain.Migration
	migrationRepo := &testutil.MigrationRepository{
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			recordedMigration = m
			return m, nil
		},
	}

	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			panic("unexpected segmentation fault or nil pointer")
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err == nil {
		t.Fatalf("expected panic error to be returned, got nil")
	}
	if draftObj.Status != domain.DraftStatusFailed {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusFailed, draftObj.Status)
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusFail {
		t.Fatalf("expected fail migration, got %+v", recordedMigration)
	}
}
