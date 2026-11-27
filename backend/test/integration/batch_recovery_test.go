//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

func TestBatchMigration_GivenSparseKeysAndBatchFailure_WhenResumed_ThenCommittedRowsAreNotRepeated(t *testing.T) {
	// Arrange
	ctx := context.Background()
	target := requireTargetPostgres(t)
	stack := setupTestStack(t)
	database, err := stack.databaseRepo.Save(ctx, &target)
	if err != nil {
		t.Fatal(err)
	}
	target.ID = database.ID
	executeBatchFixture(t, stack, target, `
		CREATE TABLE batch_resume_items (
			id bigint PRIMARY KEY, score int NOT NULL,
			CONSTRAINT allow_batch_score CHECK (id <> 30 OR score <= 0)
		);
		INSERT INTO batch_resume_items VALUES (10, 0), (20, 0), (30, 0), (40, 0), (50, 0)`)
	leader := createMigrationLeader(t, stack, "batch_resume_leader")
	_, err = stack.groupRepo.Save(ctx, domain.Group{
		Name: "batch_resume_team", LeaderID: leader.ID,
		DatabaseAccesses: []domain.DatabaseAccess{{DatabaseID: database.ID, AccessType: domain.AccessViewEdit}},
	})
	if err != nil {
		t.Fatal(err)
	}
	author := signIn(t, stack, leader.Username)
	request := handlers.CreateDraftRequest{
		SqlUp:   "UPDATE batch_resume_items SET score = score + 1 WHERE id BETWEEN ? AND ?",
		SqlDown: "UPDATE batch_resume_items SET score = score - 1 WHERE id BETWEEN ? AND ?",
		IsBatch: true, BatchColumn: "id", BatchSize: 2,
	}
	proposal := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/databases/%d/drafts", database.ID), request, author.cookie, author.csrf)
	if proposal.Code != http.StatusCreated {
		t.Fatal(proposal.Body.String())
	}
	draft := parseResponseBody[handlers.DraftResponse](t, proposal).Data
	path := fmt.Sprintf("/api/v1/drafts/%d", draft.ID)
	approveTestDraft(t, stack, path, author)
	execution := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, author.cookie, author.csrf)
	if execution.Code != http.StatusAccepted {
		t.Fatal(execution.Body.String())
	}
	job := parseResponseBody[handlers.DraftExecutionResponse](t, execution).Data
	jobID, err := strconv.ParseUint(job.JobID, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	waitForJobStatus(t, stack.db, uint(jobID), "failed")
	checkpoint := executeRequest(stack.engine, http.MethodGet, path+"/progress", nil, author.cookie, "")
	progress := parseResponseBody[handlers.BatchProgressResponse](t, checkpoint).Data
	if progress.CurrentBatch != 1 || progress.RowsProcessed != 2 || progress.Status != domain.DraftStatusPartial {
		t.Fatalf("expected the first two rows committed, got %+v", progress)
	}
	executeBatchFixture(t, stack, target, "ALTER TABLE batch_resume_items DROP CONSTRAINT allow_batch_score")

	// Act
	vote := executeRequest(stack.engine, http.MethodPost, path+"/approvals", handlers.SubmitApprovalRequest{Status: domain.VoteResume}, author.cookie, author.csrf)
	if vote.Code != http.StatusOK {
		t.Fatal(vote.Body.String())
	}
	resume := executeRequest(stack.engine, http.MethodPost, path+"/resume", nil, author.cookie, author.csrf)
	if resume.Code != http.StatusAccepted {
		t.Fatal(resume.Body.String())
	}

	// Assert
	migration := waitForAppliedMigration(t, stack, database.ID, draft.ID, author)
	if migration.Status != domain.MigrationStatusSuccess || len(migration.Snapshot) == 0 {
		t.Fatalf("expected versioned migration with schema snapshot, got %+v", migration)
	}
	rows, err := stack.databaseDriver.ExecuteQuery(ctx, target, "SELECT id FROM batch_resume_items WHERE score = 1 ORDER BY id")
	if err != nil || len(rows.Rows) != 5 {
		t.Fatalf("expected five rows updated exactly once, got %+v, error %v", rows, err)
	}
	executeBatchFixture(t, stack, target, "INSERT INTO batch_resume_items VALUES (60, 7)")
	traverseAppliedBatch(t, stack, database.ID, author, domain.DraftOperationRollback)
	rows, err = stack.databaseDriver.ExecuteQuery(ctx, target, "SELECT id FROM batch_resume_items WHERE (id < 60 AND score = 0) OR (id = 60 AND score = 7)")
	if err != nil || len(rows.Rows) != 6 {
		t.Fatalf("rollback changed the wrong rows: %+v, error %v", rows, err)
	}
	traverseAppliedBatch(t, stack, database.ID, author, domain.DraftOperationForward)
	rows, err = stack.databaseDriver.ExecuteQuery(ctx, target, "SELECT id FROM batch_resume_items WHERE (id < 60 AND score = 1) OR (id = 60 AND score = 7)")
	if err != nil || len(rows.Rows) != 6 {
		t.Fatalf("forward changed the wrong rows: %+v, error %v", rows, err)
	}
}

func traverseAppliedBatch(t *testing.T, stack *testStack, databaseID uint, author testSession, operation string) {
	t.Helper()
	path := fmt.Sprintf("/api/v1/me/databases/%d/%s-draft", databaseID, operation)
	proposal := executeRequest(stack.engine, http.MethodPost, path, nil, author.cookie, author.csrf)
	if proposal.Code != http.StatusCreated {
		t.Fatal(proposal.Body.String())
	}
	draft := parseResponseBody[handlers.MigrationDraftResponse](t, proposal).Data
	path = fmt.Sprintf("/api/v1/drafts/%d", draft.ID)
	approveTestDraft(t, stack, path, author)
	execution := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, author.cookie, author.csrf)
	if execution.Code != http.StatusAccepted {
		t.Fatal(execution.Body.String())
	}
	job := parseResponseBody[handlers.DraftExecutionResponse](t, execution).Data
	jobID, err := strconv.ParseUint(job.JobID, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	waitForJobStatus(t, stack.db, uint(jobID), "done")
}

func TestBatchMigration_GivenCommittedKeys_WhenRevertedTwice_ThenOnlyOriginalAffectedRowsAreRestored(t *testing.T) {
	// Arrange
	ctx := context.Background()
	target := requireTargetPostgres(t)
	stack := setupTestStack(t)
	executeBatchFixture(t, stack, target, "CREATE TABLE batch_revert_items (id bigint PRIMARY KEY, score int); INSERT INTO batch_revert_items VALUES (10, 4), (30, 0), (50, 4)")
	executor := stack.databaseDriver.(ports.BatchExecutor)
	up := ports.BatchExecution{Key: fmt.Sprintf("revert-test:%d", time.Now().UnixNano()), Table: "batch_revert_items", Column: "id", Size: 2,
		Statement: "UPDATE batch_revert_items SET score = score + 1 WHERE score = 4 AND id IN ?"}
	if _, err := executor.PrepareBatch(ctx, target, up); err != nil {
		t.Fatal(err)
	}
	if _, err := executor.ExecuteBatchChunk(ctx, target, up, 1, false); err != nil {
		t.Fatal(err)
	}
	down := up
	down.Statement = "UPDATE batch_revert_items SET score = score - 1 WHERE id IN ?"

	// Act
	for range 2 {
		if _, err := executor.ExecuteBatchChunk(ctx, target, down, 1, true); err != nil {
			t.Fatal(err)
		}
	}

	// Assert
	rows, err := stack.databaseDriver.ExecuteQuery(ctx, target, "SELECT id FROM batch_revert_items WHERE (id IN (10, 50) AND score = 4) OR (id = 30 AND score = 0)")
	if err != nil || len(rows.Rows) != 3 {
		t.Fatalf("expected all rows to retain original values, got %+v, error %v", rows, err)
	}
}

func TestBatchMigration_GivenLargeDataset_WhenExecutedInMultipleChunks_ThenProgressTrackedAndAllRowsTransformed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	target := requireTargetPostgres(t)
	stack := setupTestStack(t)
	database, err := stack.databaseRepo.Save(ctx, &target)
	if err != nil {
		t.Fatal(err)
	}
	target.ID = database.ID

	var insertStatements string
	for i := 1; i <= 50; i++ {
		insertStatements += fmt.Sprintf("INSERT INTO batch_chunk_items VALUES (%d, 10);", i)
	}
	executeBatchFixture(t, stack, target, `
		CREATE TABLE batch_chunk_items (
			id bigint PRIMARY KEY,
			val int NOT NULL
		); `+insertStatements)

	leader := createMigrationLeader(t, stack, "batch_chunk_leader")
	_, err = stack.groupRepo.Save(ctx, domain.Group{
		Name: "batch_chunk_team", LeaderID: leader.ID,
		DatabaseAccesses: []domain.DatabaseAccess{{DatabaseID: database.ID, AccessType: domain.AccessViewEdit}},
	})
	if err != nil {
		t.Fatal(err)
	}
	author := signIn(t, stack, leader.Username)

	// Act: Submit batch draft with BatchSize = 10 (5 chunks across 50 rows)
	request := handlers.CreateDraftRequest{
		SqlUp:   "UPDATE batch_chunk_items SET val = val + 50 WHERE id BETWEEN ? AND ?",
		SqlDown: "UPDATE batch_chunk_items SET val = val - 50 WHERE id BETWEEN ? AND ?",
		IsBatch: true, BatchColumn: "id", BatchSize: 10,
	}
	proposal := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/databases/%d/drafts", database.ID), request, author.cookie, author.csrf)
	if proposal.Code != http.StatusCreated {
		t.Fatal(proposal.Body.String())
	}
	draft := parseResponseBody[handlers.DraftResponse](t, proposal).Data
	path := fmt.Sprintf("/api/v1/drafts/%d", draft.ID)
	approveTestDraft(t, stack, path, author)

	execution := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, author.cookie, author.csrf)
	if execution.Code != http.StatusAccepted {
		t.Fatal(execution.Body.String())
	}
	job := parseResponseBody[handlers.DraftExecutionResponse](t, execution).Data
	jobID, err := strconv.ParseUint(job.JobID, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	waitForJobStatus(t, stack.db, uint(jobID), "done")

	// Assert: Progress endpoint tracks full completion and all rows are updated
	checkpoint := executeRequest(stack.engine, http.MethodGet, path+"/progress", nil, author.cookie, "")
	progress := parseResponseBody[handlers.BatchProgressResponse](t, checkpoint).Data
	if progress.RowsProcessed != 50 || progress.Status != domain.DraftStatusApplied {
		t.Fatalf("expected 50 rows processed with applied status, got %+v", progress)
	}

	rows, err := stack.databaseDriver.ExecuteQuery(ctx, target, "SELECT id FROM batch_chunk_items WHERE val = 60 ORDER BY id")
	if err != nil || len(rows.Rows) != 50 {
		t.Fatalf("expected 50 updated rows with val=60, got %+v, error %v", rows, err)
	}
}

func TestBatchMigration_GivenMidBatchError_WhenTargetQuarantined_ThenSubsequentExecutionsBlockedUntilResolved(t *testing.T) {
	// Arrange
	ctx := context.Background()
	target := requireTargetPostgres(t)
	stack := setupTestStack(t)
	database, err := stack.databaseRepo.Save(ctx, &target)
	if err != nil {
		t.Fatal(err)
	}
	target.ID = database.ID

	var insertStatements string
	for i := 1; i <= 40; i++ {
		insertStatements += fmt.Sprintf("INSERT INTO batch_quarantine_items VALUES (%d, 5);", i)
	}
	executeBatchFixture(t, stack, target, `
		CREATE TABLE batch_quarantine_items (
			id bigint PRIMARY KEY,
			score int NOT NULL,
			CONSTRAINT chk_quarantine_limit CHECK (id <> 25 OR score <= 5)
		); `+insertStatements)

	leader := createMigrationLeader(t, stack, "batch_quarantine_leader")
	_, err = stack.groupRepo.Save(ctx, domain.Group{
		Name: "batch_quarantine_team", LeaderID: leader.ID,
		DatabaseAccesses: []domain.DatabaseAccess{{DatabaseID: database.ID, AccessType: domain.AccessViewEdit}},
	})
	if err != nil {
		t.Fatal(err)
	}
	author := signIn(t, stack, leader.Username)

	// Act 1: Submit batch draft that will fail at row 25 during chunk 3
	request := handlers.CreateDraftRequest{
		SqlUp:   "UPDATE batch_quarantine_items SET score = score + 10 WHERE id BETWEEN ? AND ?",
		SqlDown: "UPDATE batch_quarantine_items SET score = score - 10 WHERE id BETWEEN ? AND ?",
		IsBatch: true, BatchColumn: "id", BatchSize: 10,
	}
	proposal := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/databases/%d/drafts", database.ID), request, author.cookie, author.csrf)
	if proposal.Code != http.StatusCreated {
		t.Fatal(proposal.Body.String())
	}
	failingDraft := parseResponseBody[handlers.DraftResponse](t, proposal).Data
	failingPath := fmt.Sprintf("/api/v1/drafts/%d", failingDraft.ID)
	approveTestDraft(t, stack, failingPath, author)

	execution := executeRequest(stack.engine, http.MethodPost, failingPath+"/execute", nil, author.cookie, author.csrf)
	if execution.Code != http.StatusAccepted {
		t.Fatal(execution.Body.String())
	}
	job := parseResponseBody[handlers.DraftExecutionResponse](t, execution).Data
	jobID, err := strconv.ParseUint(job.JobID, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	waitForJobStatus(t, stack.db, uint(jobID), "failed")

	// Assert 1: Draft status is partial
	checkpoint := executeRequest(stack.engine, http.MethodGet, failingPath+"/progress", nil, author.cookie, "")
	progress := parseResponseBody[handlers.BatchProgressResponse](t, checkpoint).Data
	if progress.Status != domain.DraftStatusPartial {
		t.Fatalf("expected draft status %s, got %s", domain.DraftStatusPartial, progress.Status)
	}

	// Act 2: Attempting to execute any new draft on this quarantined database must be blocked with Conflict / Locked
	newDraftReq := handlers.CreateDraftRequest{
		SqlUp:   "CREATE TABLE blocked_table (id INT);",
		SqlDown: "DROP TABLE blocked_table;",
	}
	newProposal := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/databases/%d/drafts", database.ID), newDraftReq, author.cookie, author.csrf)
	if newProposal.Code == http.StatusCreated {
		newDraft := parseResponseBody[handlers.DraftResponse](t, newProposal).Data
		approveTestDraft(t, stack, fmt.Sprintf("/api/v1/drafts/%d", newDraft.ID), author)
		blockedExec := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/drafts/%d/execute", newDraft.ID), nil, author.cookie, author.csrf)
		if blockedExec.Code != http.StatusConflict && blockedExec.Code != http.StatusLocked {
			t.Fatalf("expected new draft execution to be rejected with 409 or 423 while database is quarantined, got %d: %s", blockedExec.Code, blockedExec.Body.String())
		}
	}
}

func executeBatchFixture(t *testing.T, stack *testStack, target domain.Database, statement string) {
	t.Helper()
	if _, err := stack.databaseDriver.ExecuteCommand(context.Background(), target, statement); err != nil {
		t.Fatal(err)
	}
}
