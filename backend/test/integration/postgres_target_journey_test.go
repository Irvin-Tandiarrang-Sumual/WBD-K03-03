//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
)

func TestPostgresTarget_GivenUserWithReadOnlyAccess_WhenExecuteSelectQuery_ThenReturnRowsSuccessfully(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	// Register Target PostgreSQL in platform DB
	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target postgres: %v", err)
	}

	// Create Group with AccessViewOnly
	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "pg-analysts-group",
		LeaderID: createMigrationLeader(t, stack, "pg-analysts-group-leader").ID,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewOnly},
		},
	})
	if err != nil {
		t.Fatalf("failed to create view-only group: %v", err)
	}

	// Create user & login
	user := createRegularUser(t, stack, "pg_analyst_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act - Execute read-only SELECT
	queryReq := handlers.DatabaseQueryRequest{
		Query: "SELECT 1 AS col_val, 'pg_test' AS name_val;",
	}
	urlPath := fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID)
	rec := executeRequest(stack.engine, http.MethodPost, urlPath, queryReq, sessionCookie, csrfToken)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on SELECT query, got %d: %s", rec.Code, rec.Body.String())
	}

	res := parseResponseBody[*ports.DatabaseQueryResult](t, rec)
	if len(res.Data.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(res.Data.Rows))
	}
}

func TestPostgresTarget_GivenUserWithReadOnlyAccess_WhenExecuteMutatingCommand_ThenRejectWithForbidden(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target postgres: %v", err)
	}

	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "pg-readonly-enforcement",
		LeaderID: createMigrationLeader(t, stack, "pg-readonly-enforcement-leader").ID,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewOnly},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "pg_readonly_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act - Attempting to execute mutating DDL/DML
	queryReq := handlers.DatabaseQueryRequest{
		Query: "CREATE TABLE unauthorized_tbl (id INT);",
	}
	urlPath := fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID)
	rec := executeRequest(stack.engine, http.MethodPost, urlPath, queryReq, sessionCookie, csrfToken)

	// Assert - AccessViewOnly forbids mutations
	if rec.Code != http.StatusForbidden && rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 403 Forbidden or 400 on mutating query for read-only user, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostgresTarget_GivenUserWithReadWriteAccess_WhenExecuteExplainAndExplainAnalyze_ThenReturnValidExecutionPlan(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target postgres: %v", err)
	}

	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "pg-devs-group",
		LeaderID: createMigrationLeader(t, stack, "pg-devs-group-leader").ID,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "pg_dev_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act 1 - EXPLAIN
	explainReq := handlers.DatabaseQueryRequest{Query: "SELECT 1;"}
	explainURL := fmt.Sprintf("/api/v1/me/databases/%d/explain", createdDB.ID)
	recExplain := executeRequest(stack.engine, http.MethodPost, explainURL, explainReq, sessionCookie, csrfToken)

	// Assert 1
	if recExplain.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on EXPLAIN, got %d: %s", recExplain.Code, recExplain.Body.String())
	}

	// Act 2 - EXPLAIN ANALYZE
	explainAnalyzeURL := fmt.Sprintf("/api/v1/me/databases/%d/explain-analyze", createdDB.ID)
	recAnalyze := executeRequest(stack.engine, http.MethodPost, explainAnalyzeURL, explainReq, sessionCookie, csrfToken)

	// Assert 2
	if recAnalyze.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on EXPLAIN ANALYZE, got %d: %s", recAnalyze.Code, recAnalyze.Body.String())
	}
}

func TestPostgresTarget_GivenUserWithReadWriteAccess_WhenApplySQLUpMigration_ThenSchemaUpdatedAndSnapshotRecorded(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target postgres: %v", err)
	}

	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "pg-migrators-group",
		LeaderID: createMigrationLeader(t, stack, "pg-migrators-group-leader").ID,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "pg_migrator_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act - Apply SQL Up Migration
	sqlUp := `CREATE TABLE pg_products (
		id INT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		price NUMERIC(10,2) NOT NULL
	);
	INSERT INTO pg_products (id, name, price) VALUES (1, 'Widget', 19.99);`

	leader := signIn(t, stack, "pg-migrators-group-leader")
	draftURL := fmt.Sprintf("/api/v1/databases/%d/drafts", createdDB.ID)
	invalid := executeRequest(stack.engine, http.MethodPost, draftURL,
		handlers.CreateDraftRequest{SqlUp: "CREATE TABLE ("}, sessionCookie, csrfToken)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid SQL to be rejected before review: %s", invalid.Body.String())
	}
	migration := applyTestMigration(t, stack, createdDB.ID, testSession{sessionCookie, csrfToken},
		sqlUp, "DROP TABLE pg_products", leader)

	// Assert - Migration creation success
	if migration.Status != domain.MigrationStatusSuccess || migration.Version == "" || len(migration.Snapshot) == 0 {
		t.Fatalf("expected versioned migration with a snapshot, got %+v", migration)
	}
	reversible := true
	destructive := executeRequest(stack.engine, http.MethodPost, draftURL,
		handlers.CreateDraftRequest{SqlUp: "TRUNCATE pg_products", IsReversible: &reversible}, sessionCookie, csrfToken)
	if destructive.Code != http.StatusCreated {
		t.Fatalf("expected destructive draft to await approval: %s", destructive.Body.String())
	}
	destructiveDraft := parseResponseBody[handlers.DraftResponse](t, destructive).Data
	if destructiveDraft.IsReversible || destructiveDraft.IrreversibleReason == "" {
		t.Fatalf("expected a persisted irreversible draft, got %+v", destructiveDraft)
	}

	// Verify table physically exists in Target PostgreSQL
	verifyReq := handlers.DatabaseQueryRequest{Query: "SELECT name, price FROM pg_products WHERE id = 1;"}
	recVerify := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID), verifyReq, sessionCookie, csrfToken)
	if recVerify.Code != http.StatusOK {
		t.Fatalf("expected 200 OK verifying created table, got %d: %s", recVerify.Code, recVerify.Body.String())
	}
}

func TestPostgresTarget_GivenAppliedMigration_WhenRollbackTriggered_ThenCompensatingSQLDownRestoresPreviousSchema(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target postgres: %v", err)
	}

	leader := createMigrationLeader(t, stack, "pg_rollback_leader")
	group, err := stack.groupRepo.Save(ctx, domain.Group{
		LeaderID: leader.ID,
		Name:     "pg-rollback-group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	viewer := createMigrationLeader(t, stack, "pg_observer_leader")
	_, err = stack.groupRepo.Save(ctx, domain.Group{Name: "pg-observers", LeaderID: viewer.ID, DatabaseAccesses: []domain.DatabaseAccess{{DatabaseID: createdDB.ID, AccessType: domain.AccessViewOnly}}})
	if err != nil {
		t.Fatal(err)
	}
	viewerCookie, viewerToken := loginUser(t, stack.engine, viewer.Username, "password123")
	user := createRegularUser(t, stack, "pg_rollback_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	author := testSession{sessionCookie, csrfToken}
	leaderCookie, leaderToken := loginUser(t, stack.engine, leader.Username, "password123")
	voters := []testSession{{leaderCookie, leaderToken}}
	voters = append(voters, testSession{viewerCookie, viewerToken})

	// Step 1: Apply initial baseline migration 1
	previous := applyTestMigration(t, stack, createdDB.ID, author,
		"CREATE TABLE pg_base_tbl (id INT PRIMARY KEY);", "DROP TABLE pg_base_tbl", voters...)

	// Step 2: Apply second migration to be rolled back
	target := applyTestMigration(t, stack, createdDB.ID, author,
		"CREATE TABLE pg_rollback_tbl (id INT PRIMARY KEY, title VARCHAR(50));", "DROP TABLE pg_rollback_tbl", voters...)

	// Act
	base := fmt.Sprintf("/api/v1/me/databases/%d", createdDB.ID)
	proposal := executeRequest(stack.engine, http.MethodPost, base+"/rollback-draft", nil, sessionCookie, csrfToken)
	if proposal.Code != http.StatusCreated {
		t.Fatalf("propose: %s", proposal.Body.String())
	}
	draft := parseResponseBody[handlers.MigrationDraftResponse](t, proposal).Data
	path := fmt.Sprintf("%s/migration-drafts/%d", base, draft.ID)
	blocked := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, sessionCookie, csrfToken)
	if blocked.Code != http.StatusConflict {
		t.Fatalf("expected unapproved execution to fail: %s", blocked.Body.String())
	}
	vote := executeRequest(stack.engine, http.MethodPost, path+"/vote", map[string]bool{"approve": true}, leaderCookie, leaderToken)
	if vote.Code != http.StatusNoContent {
		t.Fatalf("vote: %s", vote.Body.String())
	}
	partial := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, sessionCookie, csrfToken)
	if partial.Code != http.StatusConflict {
		t.Fatal(partial.Body.String())
	}
	vote = executeRequest(stack.engine, http.MethodPost, path+"/vote", map[string]bool{"approve": true}, viewerCookie, viewerToken)
	if vote.Code != http.StatusNoContent {
		t.Fatal(vote.Body.String())
	}
	recRollback := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, sessionCookie, csrfToken)

	// Assert - Rollback execution success
	if recRollback.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted on rollback, got %d: %s", recRollback.Code, recRollback.Body.String())
	}

	rollback := waitForAppliedMigration(t, stack, createdDB.ID, draft.ID, author)

	// Verify table pg_rollback_tbl is dropped in PostgreSQL target
	verifyReq := handlers.DatabaseQueryRequest{Query: "SELECT * FROM pg_rollback_tbl;"}
	recVerify := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID), verifyReq, sessionCookie, csrfToken)
	if recVerify.Code == http.StatusOK {
		t.Fatalf("expected query to fail because table should be dropped by rollback, but got 200 OK")
	}

	if rollback.RevertsMigrationID == nil || *rollback.RevertsMigrationID != target.ID || rollback.ExecutionID == target.ExecutionID {
		t.Fatalf("rollback audit %+v; target %+v", rollback, target)
	}
	current := executeRequest(stack.engine, http.MethodGet, base, nil, sessionCookie, "")
	if current.Code != http.StatusOK {
		t.Fatal(current.Body.String())
	}
	databaseDetail := parseResponseBody[handlers.DatabaseResponse](t, current).Data
	if databaseDetail.CurrentMigrationID == nil || *databaseDetail.CurrentMigrationID != previous.ID || rollback.MigratedAt.IsZero() {
		t.Fatalf("expected the active parent and a rollback timestamp, got %+v and %+v", databaseDetail, rollback)
	}
	proposal = executeRequest(stack.engine, http.MethodPost, base+"/forward-draft", nil, sessionCookie, csrfToken)
	if proposal.Code != http.StatusCreated {
		t.Fatal(proposal.Body.String())
	}
	draft = parseResponseBody[handlers.MigrationDraftResponse](t, proposal).Data
	path = fmt.Sprintf("/api/v1/drafts/%d", draft.ID)
	approveTestDraft(t, stack, path, voters[0])
	partial = executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, sessionCookie, csrfToken)
	if partial.Code != http.StatusConflict {
		t.Fatal(partial.Body.String())
	}
	approveTestDraft(t, stack, path, voters[1])
	replay := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, sessionCookie, csrfToken)
	if replay.Code != http.StatusAccepted {
		t.Fatal(replay.Body.String())
	}
	forward := waitForAppliedMigration(t, stack, createdDB.ID, draft.ID, author)
	for _, hide := range []bool{false, true} {
		history := executeRequest(stack.engine, http.MethodGet, fmt.Sprintf("%s/migrations?hide_rollback=%t", base, hide), nil, sessionCookie, csrfToken)
		var page response.PagedResponse[handlers.MigrationResponse]
		if history.Code != http.StatusOK || json.Unmarshal(history.Body.Bytes(), &page) != nil {
			t.Fatal(history.Body.String())
		}
		seen := map[uint]bool{}
		for _, m := range page.Data {
			seen[m.ID] = true
		}
		if !seen[forward.ID] || seen[rollback.ID] == hide || seen[target.ID] == hide || page.Paging.TotalData != int64(len(page.Data)) {
			t.Fatalf("hide=%t: %s", hide, history.Body.String())
		}
	}
}
