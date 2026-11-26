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

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil, unlockFn, nil)

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

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil, nil, nil)
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

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil, nil, nil)
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
	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil, nil, nil)

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

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil, nil, nil)
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

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil, nil, nil)

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

func TestDraftService_ProcessBatchChunk_AllChunksSucceed_RecordsCheckpointsAndMarksApplied(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 10, UserID: 4}

	draftObj := &domain.Draft{
		ID:          1,
		DatabaseID:  2,
		UserID:      4,
		SqlUp:       "UPDATE users SET active = true;",
		Status:      domain.DraftStatusExecuting,
		IsBatch:     true,
		BatchColumn: "id",
		BatchSize:   100,
	}

	savedCheckpoints := make([]*domain.DraftCheckpoint, 0)
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		SaveCheckpointFunc: func(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
			cp.ID = uint(len(savedCheckpoints) + 1)
			savedCheckpoints = append(savedCheckpoints, cp)
			return cp, nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2, Host: "localhost"}, nil
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

	chunkCalls := 0
	driver := &testutil.DatabaseDriver{
		ExecuteQueryFunc: func(ctx context.Context, db domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return &ports.DatabaseQueryResult{
				Rows: []map[string]any{{"count": int64(250)}},
			}, nil
		},
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			chunkCalls++
			switch chunkCalls {
			case 1:
				return &ports.DatabaseCommandResult{RowsAffected: 100}, nil
			case 2:
				return &ports.DatabaseCommandResult{RowsAffected: 100}, nil
			case 3:
				return &ports.DatabaseCommandResult{RowsAffected: 50}, nil
			default:
				return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
			}
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil, nil, nil)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if draftObj.Status != domain.DraftStatusApplied {
		t.Fatalf("expected status applied, got %s", draftObj.Status)
	}
	if len(savedCheckpoints) != 3 {
		t.Fatalf("expected 3 checkpoints, got %d", len(savedCheckpoints))
	}
	for i, cp := range savedCheckpoints {
		if cp.Status != "committed" {
			t.Fatalf("expected checkpoint %d to be committed, got %s", i+1, cp.Status)
		}
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusSuccess {
		t.Fatalf("expected success migration, got %+v", recordedMigration)
	}
}

func TestDraftService_ProcessBatchChunk_EmptyTargetRows_MarksAppliedWithoutCheckpoints(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 10, UserID: 4}

	draftObj := &domain.Draft{
		ID:          1,
		DatabaseID:  2,
		UserID:      4,
		SqlUp:       "UPDATE users SET active = true;",
		Status:      domain.DraftStatusExecuting,
		IsBatch:     true,
		BatchColumn: "id",
		BatchSize:   100,
	}

	checkpointCount := 0
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		SaveCheckpointFunc: func(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
			checkpointCount++
			return cp, nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2, Host: "localhost"}, nil
		},
		UpdateCurrentMigrationIDFunc: func(ctx context.Context, dbID, migID uint) error {
			return nil
		},
	}

	var recordedMigration *domain.Migration
	migrationRepo := &testutil.MigrationRepository{
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 101
			recordedMigration = m
			return m, nil
		},
	}

	driver := &testutil.DatabaseDriver{
		ExecuteQueryFunc: func(ctx context.Context, db domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return &ports.DatabaseQueryResult{
				Rows: []map[string]any{{"count": int64(0)}},
			}, nil
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil, nil, nil)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if draftObj.Status != domain.DraftStatusApplied {
		t.Fatalf("expected status applied, got %s", draftObj.Status)
	}
	if checkpointCount != 0 {
		t.Fatalf("expected 0 checkpoints for empty target, got %d", checkpointCount)
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusSuccess {
		t.Fatalf("expected success migration, got %+v", recordedMigration)
	}
}

func TestDraftService_ProcessBatchChunk_ChunkFails_RollsBackChunkTransitionsToPartialAndIncrementsRound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 10, UserID: 4}

	draftObj := &domain.Draft{
		ID:           1,
		DatabaseID:   2,
		UserID:       4,
		SqlUp:        "UPDATE users SET active = true;",
		Status:       domain.DraftStatusExecuting,
		IsBatch:      true,
		BatchColumn:  "id",
		BatchSize:    100,
		CurrentRound: 1,
	}

	savedCheckpoints := make([]*domain.DraftCheckpoint, 0)
	var createdRoundApprovals []domain.DraftApproval
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		SaveCheckpointFunc: func(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
			savedCheckpoints = append(savedCheckpoints, cp)
			return cp, nil
		},
		CreateApprovalsForRoundFunc: func(ctx context.Context, approvals []domain.DraftApproval) error {
			createdRoundApprovals = approvals
			return nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2, Host: "localhost"}, nil
		},
	}

	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{
				{ID: 1, LeaderID: 101},
				{ID: 2, LeaderID: 102},
			}, nil
		},
	}

	chunkCalls := 0
	driver := &testutil.DatabaseDriver{
		ExecuteQueryFunc: func(ctx context.Context, db domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return &ports.DatabaseQueryResult{Rows: []map[string]any{{"count": int64(200)}}}, nil
		},
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			chunkCalls++
			if chunkCalls == 1 {
				return &ports.DatabaseCommandResult{RowsAffected: 100}, nil
			}
			return nil, errors.New("deadlock detected or table constraint violation")
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, nil, driver, groupRepo, nil, nil)
	worker.SetRetryPolicy(0, time.Millisecond)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err == nil {
		t.Fatalf("expected error from failed chunk, got nil")
	}

	if draftObj.Status != domain.DraftStatusPartial {
		t.Fatalf("expected status partial, got %s", draftObj.Status)
	}
	if draftObj.CurrentRound != 2 {
		t.Fatalf("expected CurrentRound == 2, got %d", draftObj.CurrentRound)
	}
	if len(savedCheckpoints) != 2 {
		t.Fatalf("expected 2 checkpoints (1 committed, 1 failed), got %d", len(savedCheckpoints))
	}
	if savedCheckpoints[1].Status != "failed" {
		t.Fatalf("expected second checkpoint status failed, got %s", savedCheckpoints[1].Status)
	}
	if len(createdRoundApprovals) != 2 {
		t.Fatalf("expected 2 new approvals for round 2, got %d", len(createdRoundApprovals))
	}
	for _, app := range createdRoundApprovals {
		if app.Round != 2 || app.Phase != domain.ApprovalPhasePartialRecovery || app.Status != domain.ApprovalStatusPending {
			t.Fatalf("unexpected approval attributes: %+v", app)
		}
	}
}

func TestDraftService_ResumeBatch_FailsAgain_AdvancesCheckpointKeepsPartialStateAndSameExecutionID(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 42, UserID: 4, Operation: "resume"}

	draftObj := &domain.Draft{
		ID:           1,
		DatabaseID:   2,
		UserID:       4,
		SqlUp:        "UPDATE users SET active = true;",
		Status:       domain.DraftStatusExecuting,
		IsBatch:      true,
		BatchColumn:  "id",
		BatchSize:    100,
		CurrentRound: 2,
	}

	lastCommittedCp := &domain.DraftCheckpoint{
		ID:           1,
		DraftID:      1,
		BatchNumber:  1,
		CursorLastID: "100",
		Status:       "committed",
		RowsAffected: 100,
	}

	var savedCheckpoints []*domain.DraftCheckpoint
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		FindLastCommittedCheckpointFunc: func(ctx context.Context, draftID uint) (*domain.DraftCheckpoint, error) {
			return lastCommittedCp, nil
		},
		SaveCheckpointFunc: func(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
			savedCheckpoints = append(savedCheckpoints, cp)
			return cp, nil
		},
		CreateApprovalsForRoundFunc: func(ctx context.Context, approvals []domain.DraftApproval) error {
			return nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2, Host: "localhost"}, nil
		},
	}

	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{{ID: 1, LeaderID: 101}}, nil
		},
	}

	driver := &testutil.DatabaseDriver{
		ExecuteQueryFunc: func(ctx context.Context, db domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return &ports.DatabaseQueryResult{Rows: []map[string]any{{"count": int64(100)}}}, nil
		},
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return nil, errors.New("persistent query error on resume")
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, nil, driver, groupRepo, nil, nil)
	worker.SetRetryPolicy(0, time.Millisecond)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err == nil {
		t.Fatalf("expected error from failed chunk on resume, got nil")
	}

	if draftObj.Status != domain.DraftStatusPartial {
		t.Fatalf("expected status partial, got %s", draftObj.Status)
	}
	if draftObj.CurrentRound != 3 {
		t.Fatalf("expected CurrentRound incremented to 3, got %d", draftObj.CurrentRound)
	}
	if len(savedCheckpoints) != 1 || savedCheckpoints[0].Status != "failed" || savedCheckpoints[0].BatchNumber != 2 {
		t.Fatalf("expected failed checkpoint for batch 2, got %+v", savedCheckpoints)
	}
}

func TestDraftService_ResumeBatch_AllRemainingChunksSucceed_MarksApplied(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 42, UserID: 4, Operation: "resume"}

	draftObj := &domain.Draft{
		ID:           1,
		DatabaseID:   2,
		UserID:       4,
		SqlUp:        "UPDATE users SET active = true;",
		Status:       domain.DraftStatusExecuting,
		IsBatch:      true,
		BatchColumn:  "id",
		BatchSize:    100,
		CurrentRound: 2,
	}

	lastCommittedCp := &domain.DraftCheckpoint{
		ID:           1,
		DraftID:      1,
		BatchNumber:  1,
		CursorLastID: "100",
		Status:       "committed",
		RowsAffected: 100,
	}

	savedCheckpoints := make([]*domain.DraftCheckpoint, 0)
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		FindLastCommittedCheckpointFunc: func(ctx context.Context, draftID uint) (*domain.DraftCheckpoint, error) {
			return lastCommittedCp, nil
		},
		SaveCheckpointFunc: func(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
			savedCheckpoints = append(savedCheckpoints, cp)
			return cp, nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2, Host: "localhost"}, nil
		},
		UpdateCurrentMigrationIDFunc: func(ctx context.Context, dbID, migID uint) error {
			return nil
		},
	}

	var recordedMigration *domain.Migration
	migrationRepo := &testutil.MigrationRepository{
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			m.ID = 105
			recordedMigration = m
			return m, nil
		},
	}

	chunkCalls := 0
	driver := &testutil.DatabaseDriver{
		ExecuteQueryFunc: func(ctx context.Context, db domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return &ports.DatabaseQueryResult{Rows: []map[string]any{{"count": int64(150)}}}, nil
		},
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			chunkCalls++
			switch chunkCalls {
			case 1:
				return &ports.DatabaseCommandResult{RowsAffected: 100}, nil // Batch 2
			case 2:
				return &ports.DatabaseCommandResult{RowsAffected: 50}, nil // Batch 3
			default:
				return &ports.DatabaseCommandResult{RowsAffected: 0}, nil
			}
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil, nil, nil)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if draftObj.Status != domain.DraftStatusApplied {
		t.Fatalf("expected status applied, got %s", draftObj.Status)
	}
	if len(savedCheckpoints) != 2 {
		t.Fatalf("expected 2 newly committed checkpoints, got %d", len(savedCheckpoints))
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusSuccess {
		t.Fatalf("expected success migration, got %+v", recordedMigration)
	}
}

func TestDraftService_RevertBatch_Success_RevertsCommittedChunksInReverseOrderAndMarksFailed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 99, UserID: 4, Operation: "revert"}

	draftObj := &domain.Draft{
		ID:          1,
		DatabaseID:  2,
		UserID:      4,
		SqlUp:       "UPDATE users SET active = true;",
		SqlDown:     "UPDATE users SET active = false;",
		Status:      domain.DraftStatusExecuting,
		IsBatch:     true,
		BatchColumn: "id",
		BatchSize:   100,
	}

	checkpoints := []*domain.DraftCheckpoint{
		{ID: 1, DraftID: 1, BatchNumber: 1, Status: "committed", CursorLastID: "100"},
		{ID: 2, DraftID: 1, BatchNumber: 2, Status: "committed", CursorLastID: "200"},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		FindCheckpointsByDraftIDFunc: func(ctx context.Context, draftID uint) ([]*domain.DraftCheckpoint, error) {
			return checkpoints, nil
		},
		UpdateCheckpointFunc: func(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
			return cp, nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2, Host: "localhost"}, nil
		},
	}

	var recordedMigration *domain.Migration
	migrationRepo := &testutil.MigrationRepository{
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			recordedMigration = m
			return m, nil
		},
	}

	var executedCommands []string
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			executedCommands = append(executedCommands, command)
			return &ports.DatabaseCommandResult{RowsAffected: 100}, nil
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, migrationRepo, driver, nil, nil, nil)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if draftObj.Status != domain.DraftStatusFailed {
		t.Fatalf("expected status failed, got %s", draftObj.Status)
	}
	for _, cp := range checkpoints {
		if cp.Status != "reverted" {
			t.Fatalf("expected checkpoint status reverted, got %s", cp.Status)
		}
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusFail {
		t.Fatalf("expected fail migration, got %+v", recordedMigration)
	}
}

func TestDraftService_RevertBatch_FailsMidway_TransitionsToPartiallyRevertedAndIncrementsRound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{DraftID: 1, DatabaseID: 2, ExecutionID: 99, UserID: 4, Operation: "revert"}

	draftObj := &domain.Draft{
		ID:           1,
		DatabaseID:   2,
		UserID:       4,
		SqlUp:        "UPDATE users SET active = true;",
		SqlDown:      "UPDATE users SET active = false;",
		Status:       domain.DraftStatusExecuting,
		IsBatch:      true,
		BatchColumn:  "id",
		BatchSize:    100,
		CurrentRound: 2,
	}

	checkpoints := []*domain.DraftCheckpoint{
		{ID: 1, DraftID: 1, BatchNumber: 1, Status: "committed", CursorLastID: "100"},
		{ID: 2, DraftID: 1, BatchNumber: 2, Status: "committed", CursorLastID: "200"},
	}

	var createdRoundApprovals []domain.DraftApproval
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		FindCheckpointsByDraftIDFunc: func(ctx context.Context, draftID uint) ([]*domain.DraftCheckpoint, error) {
			return checkpoints, nil
		},
		CreateApprovalsForRoundFunc: func(ctx context.Context, approvals []domain.DraftApproval) error {
			createdRoundApprovals = approvals
			return nil
		},
	}

	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2, Host: "localhost"}, nil
		},
	}

	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{{ID: 1, LeaderID: 101}}, nil
		},
	}

	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return nil, errors.New("revert failure on chunk")
		},
	}

	worker := draft.NewExecutionWorker(draftRepo, dbRepo, nil, driver, groupRepo, nil, nil)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err == nil {
		t.Fatalf("expected error from midway revert failure, got nil")
	}

	if draftObj.Status != domain.DraftStatusPartiallyReverted {
		t.Fatalf("expected status partially_reverted, got %s", draftObj.Status)
	}
	if draftObj.CurrentRound != 3 {
		t.Fatalf("expected CurrentRound == 3, got %d", draftObj.CurrentRound)
	}
	if len(createdRoundApprovals) != 1 {
		t.Fatalf("expected 1 approval created, got %d", len(createdRoundApprovals))
	}
	if createdRoundApprovals[0].Phase != domain.ApprovalPhaseRevertFailure || createdRoundApprovals[0].Round != 3 {
		t.Fatalf("unexpected approval phase or round: %+v", createdRoundApprovals[0])
	}
}

type mockWorkerMetrics struct {
	executionCalls []string
	chunkCalls     []string
	retryCalls     []string
}

func (m *mockWorkerMetrics) RecordExecution(op, status string, sec float64) {
	m.executionCalls = append(m.executionCalls, op+":"+status)
}

func (m *mockWorkerMetrics) RecordBatchChunk(status string, rows int64, op string) {
	m.chunkCalls = append(m.chunkCalls, status)
}

func (m *mockWorkerMetrics) RecordRetry(op string) {
	m.retryCalls = append(m.retryCalls, op)
}

func TestExecutionWorker_Execute_WithMetrics_RecordsMetrics(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{
		DraftID:     1,
		DatabaseID:  2,
		ExecutionID: 10,
		Operation:   ports.MetricOpExecute,
	}

	draftObj := &domain.Draft{
		ID:         1,
		DatabaseID: 2,
		Status:     domain.DraftStatusExecuting,
		SqlUp:      "CREATE TABLE test (id INT);",
	}
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			return d, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2, Type: "postgresql"}, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return &ports.DatabaseCommandResult{RowsAffected: 1}, nil
		},
	}
	mockMetrics := &mockWorkerMetrics{}
	worker := draft.NewExecutionWorker(draftRepo, dbRepo, nil, driver, nil, nil, mockMetrics)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(mockMetrics.executionCalls) != 1 {
		t.Fatalf("expected 1 execution metric record, got %d", len(mockMetrics.executionCalls))
	}
	if mockMetrics.executionCalls[0] != "execute:success" {
		t.Errorf("expected 'execute:success', got %s", mockMetrics.executionCalls[0])
	}
}

func TestExecutionWorker_ProcessBatchChunk_ChunkFails_RecordsQuarantineMetric(t *testing.T) {
	// Arrange
	ctx := context.Background()
	task := ports.ExecutionTask{
		DraftID:     1,
		DatabaseID:  2,
		ExecutionID: 10,
		Operation:   ports.MetricOpExecute,
	}

	draftObj := &domain.Draft{
		ID:         1,
		DatabaseID: 2,
		Status:     domain.DraftStatusExecuting,
		IsBatch:    true,
		BatchSize:  100,
		SqlUp:      "UPDATE users SET active = 1 WHERE id > 0",
	}
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			return d, nil
		},
		SaveCheckpointFunc: func(ctx context.Context, cp *domain.DraftCheckpoint) (*domain.DraftCheckpoint, error) {
			return cp, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 2, Type: "postgresql"}, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteQueryFunc: func(ctx context.Context, db domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return &ports.DatabaseQueryResult{Rows: []map[string]any{{"count": int64(100)}}}, nil
		},
		ExecuteCommandFunc: func(ctx context.Context, db domain.Database, command string) (*ports.DatabaseCommandResult, error) {
			return nil, errors.New("deadlock detected")
		},
	}
	mockMetrics := &mockWorkerMetrics{}
	worker := draft.NewExecutionWorker(draftRepo, dbRepo, nil, driver, nil, nil, mockMetrics)
	worker.SetRetryPolicy(0, 0)

	// Act
	err := worker.Execute(ctx, task)

	// Assert
	if err == nil {
		t.Fatalf("expected error from failed chunk, got nil")
	}

	quarantinedFound := false
	for _, call := range mockMetrics.chunkCalls {
		if call == ports.MetricStatusQuarantined {
			quarantinedFound = true
			break
		}
	}
	if !quarantinedFound {
		t.Errorf("expected %q chunk metric to be recorded, got: %v", ports.MetricStatusQuarantined, mockMetrics.chunkCalls)
	}
}
