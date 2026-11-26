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

func executeBatchFixture(t *testing.T, stack *testStack, target domain.Database, statement string) {
	t.Helper()
	if _, err := stack.databaseDriver.ExecuteCommand(context.Background(), target, statement); err != nil {
		t.Fatal(err)
	}
}
