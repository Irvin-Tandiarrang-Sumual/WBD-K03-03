//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

func TestDraftJourney_GivenCustomSQLDraft_WhenFullConsensusReached_ThenExecuteAndPhysicallyApplyToTarget(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	targetDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target DB: %v", err)
	}

	leadEng := createRegularUser(t, stack, "lead_eng", "password123", nil)
	groupEng, err := stack.groupRepo.Save(ctx, domain.Group{
		LeaderID: leadEng.ID,
		Name:     "eng_group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: targetDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create eng group: %v", err)
	}

	leadQA := createRegularUser(t, stack, "lead_qa", "password123", nil)
	groupQA, err := stack.groupRepo.Save(ctx, domain.Group{
		LeaderID: leadQA.ID,
		Name:     "qa_group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: targetDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create qa group: %v", err)
	}
	_ = groupQA
	draftRepo := repositories.NewDraftRepository(stack.db)

	alice := createRegularUser(t, stack, "dev_alice", "password123", []uint{groupEng.ID})
	aliceCookie, aliceCSRF := loginUser(t, stack.engine, alice.Username, "password123")

	// Act 1: Alice creates a Custom SQL Draft
	isRev := true
	createReq := handlers.CreateDraftRequest{
		SqlUp:        "CREATE TABLE e2e_features (id SERIAL PRIMARY KEY, feature_name VARCHAR(100) NOT NULL);",
		SqlDown:      "DROP TABLE e2e_features;",
		IsReversible: &isRev,
	}
	createPath := fmt.Sprintf("/api/v1/me/databases/%d/drafts", targetDB.ID)
	recCreate := executeRequest(stack.engine, http.MethodPost, createPath, createReq, aliceCookie, aliceCSRF)

	// Assert 1: HTTP 201 Created with status pending_approval and 2 pending group approvals
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on CreateDraft, got %d: %s", recCreate.Code, recCreate.Body.String())
	}
	draftRes := parseResponseBody[handlers.DraftResponse](t, recCreate).Data
	if draftRes.Status != domain.DraftStatusPendingApproval {
		t.Fatalf("expected status %s, got %s", domain.DraftStatusPendingApproval, draftRes.Status)
	}
	if len(draftRes.Approvals) != 2 {
		t.Fatalf("expected 2 approvals for 2 database groups, got %d", len(draftRes.Approvals))
	}

	// Act 2: Alice attempts to execute before approvals are gathered
	execPath := fmt.Sprintf("/api/v1/me/drafts/%d/execute", draftRes.ID)
	recPrematureExec := executeRequest(stack.engine, http.MethodPost, execPath, nil, aliceCookie, aliceCSRF)
	if recPrematureExec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when executing unapproved draft, got %d: %s", recPrematureExec.Code, recPrematureExec.Body.String())
	}

	// Act 3: Leader Eng approves
	leadEngCookie, leadEngCSRF := loginUser(t, stack.engine, leadEng.Username, "password123")
	votePath := fmt.Sprintf("/api/v1/me/drafts/%d/approvals", draftRes.ID)
	recVoteEng := executeRequest(stack.engine, http.MethodPost, votePath, handlers.SubmitApprovalRequest{Status: "approved"}, leadEngCookie, leadEngCSRF)
	if recVoteEng.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on leader eng approval, got %d: %s", recVoteEng.Code, recVoteEng.Body.String())
	}
	draftEngRes := parseResponseBody[handlers.DraftResponse](t, recVoteEng).Data
	// Status still pending_approval because leadQA has not approved yet
	if draftEngRes.Status != domain.DraftStatusPendingApproval {
		t.Fatalf("expected draft to remain pending_approval with partial consensus, got %s", draftEngRes.Status)
	}

	// Act 4: Leader QA approves -> Full Consensus reached!
	leadQACookie, leadQACSRF := loginUser(t, stack.engine, leadQA.Username, "password123")
	recVoteQA := executeRequest(stack.engine, http.MethodPost, votePath, handlers.SubmitApprovalRequest{Status: "approved"}, leadQACookie, leadQACSRF)
	if recVoteQA.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on leader qa approval, got %d: %s", recVoteQA.Code, recVoteQA.Body.String())
	}
	draftQARes := parseResponseBody[handlers.DraftResponse](t, recVoteQA).Data
	if draftQARes.Status != domain.DraftStatusApproved {
		t.Fatalf("expected draft to become approved after full consensus, got %s", draftQARes.Status)
	}

	// Act 5: Non-author (leadEng) tries to execute Alice's draft -> Rejected with 403 Forbidden
	recNonAuthorExec := executeRequest(stack.engine, http.MethodPost, execPath, nil, leadEngCookie, leadEngCSRF)
	if recNonAuthorExec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when non-author attempts execution, got %d: %s", recNonAuthorExec.Code, recNonAuthorExec.Body.String())
	}

	// Act 6: Alice (author) executes her approved draft
	recExec := executeRequest(stack.engine, http.MethodPost, execPath, nil, aliceCookie, aliceCSRF)
	if recExec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted on draft execution, got %d: %s", recExec.Code, recExec.Body.String())
	}

	// Assert 6: Poll for worker execution completion and verify physical table in Target PostgreSQL
	var applied bool
	for attempt := 0; attempt < 15; attempt++ {
		time.Sleep(200 * time.Millisecond)
		d, err := draftRepo.FindByID(ctx, draftRes.ID)
		if err == nil && d.Status == domain.DraftStatusApplied {
			applied = true
			break
		}
	}
	if !applied {
		t.Fatalf("draft execution did not reach applied status within timeout")
	}

	verifyReq := handlers.DatabaseQueryRequest{
		Query: "SELECT table_name FROM information_schema.tables WHERE table_name = 'e2e_features';",
	}
	recVerify := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/query", targetDB.ID), verifyReq, aliceCookie, aliceCSRF)
	if recVerify.Code != http.StatusOK {
		t.Fatalf("expected 200 OK verifying created table, got %d: %s", recVerify.Code, recVerify.Body.String())
	}
	resVerify := parseResponseBody[*ports.DatabaseQueryResult](t, recVerify)
	if len(resVerify.Data.Rows) != 1 {
		t.Fatalf("expected 1 row confirming e2e_features table creation, got %d", len(resVerify.Data.Rows))
	}
}

func TestDraftJourney_GivenCustomSQLDraft_WhenSingleLeaderVetoes_ThenDraftRejectedImmediately(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	targetDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target DB: %v", err)
	}

	leadEng := createRegularUser(t, stack, "lead_eng_veto", "password123", nil)
	groupEng, err := stack.groupRepo.Save(ctx, domain.Group{
		LeaderID: leadEng.ID,
		Name:     "eng_group_veto",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: targetDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	alice := createRegularUser(t, stack, "dev_alice_veto", "password123", []uint{groupEng.ID})
	aliceCookie, aliceCSRF := loginUser(t, stack.engine, alice.Username, "password123")

	// Act 1: Alice creates draft
	isRev := true
	createReq := handlers.CreateDraftRequest{
		SqlUp:        "CREATE TABLE risky_table (id INT);",
		SqlDown:      "DROP TABLE risky_table;",
		IsReversible: &isRev,
	}
	createPath := fmt.Sprintf("/api/v1/me/databases/%d/drafts", targetDB.ID)
	recCreate := executeRequest(stack.engine, http.MethodPost, createPath, createReq, aliceCookie, aliceCSRF)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("failed to create draft: %s", recCreate.Body.String())
	}
	draft := parseResponseBody[handlers.DraftResponse](t, recCreate).Data

	// Act 2: Leader Eng casts single VETO rejection
	leadEngCookie, leadEngCSRF := loginUser(t, stack.engine, leadEng.Username, "password123")
	votePath := fmt.Sprintf("/api/v1/me/drafts/%d/approvals", draft.ID)
	recVeto := executeRequest(stack.engine, http.MethodPost, votePath, handlers.SubmitApprovalRequest{Status: "rejected"}, leadEngCookie, leadEngCSRF)
	if recVeto.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on veto vote, got %d: %s", recVeto.Code, recVeto.Body.String())
	}

	// Assert: Status becomes rejected immediately
	draftVetoed := parseResponseBody[handlers.DraftResponse](t, recVeto).Data
	if draftVetoed.Status != domain.DraftStatusRejected {
		t.Fatalf("expected draft status to be rejected after single veto, got %s", draftVetoed.Status)
	}

	// Act 3: Execution attempt on rejected draft must be blocked
	execPath := fmt.Sprintf("/api/v1/me/drafts/%d/execute", draft.ID)
	recExec := executeRequest(stack.engine, http.MethodPost, execPath, nil, aliceCookie, aliceCSRF)
	if recExec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when attempting to execute rejected draft, got %d: %s", recExec.Code, recExec.Body.String())
	}
}

func TestDraftJourney_GivenBatchDraftExecutionFailure_WhenQuarantined_ThenBlocksNewDraftExecutionUntilResolved(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	targetDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target DB: %v", err)
	}

	leader := createRegularUser(t, stack, "lead_batch", "password123", nil)
	group, err := stack.groupRepo.Save(ctx, domain.Group{
		LeaderID: leader.ID,
		Name:     "batch_eng_group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: targetDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	alice := createRegularUser(t, stack, "batch_alice", "password123", []uint{group.ID})
	aliceCookie, aliceCSRF := loginUser(t, stack.engine, alice.Username, "password123")
	draftRepo := repositories.NewDraftRepository(stack.db)

	// Seed an existing failed/partially executed batch draft on targetDB to trigger Quarantine
	failingBatchDraft := &domain.Draft{
		DatabaseID:     targetDB.ID,
		UserID:         alice.ID,
		SqlUp:          "UPDATE accounts SET balance = balance + 10 WHERE id BETWEEN ? AND ?;",
		IsBatch:        true,
		BatchColumn:    "id",
		BatchSize:      100,
		Status:         domain.DraftStatusPartial, // Causes quarantine!
		CurrentRound:   1,
		RecoveryAction: "",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	savedDraft, err := draftRepo.Save(ctx, failingBatchDraft)
	if err != nil {
		t.Fatalf("failed to save partial batch draft: %v", err)
	}

	// Act 1: Alice creates a new normal draft and gets it approved
	isRev := true
	createReq := handlers.CreateDraftRequest{
		SqlUp:        "CREATE TABLE test_quarantine (id INT);",
		SqlDown:      "DROP TABLE test_quarantine;",
		IsReversible: &isRev,
	}
	recCreate := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/drafts", targetDB.ID), createReq, aliceCookie, aliceCSRF)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("failed to create draft: %s", recCreate.Body.String())
	}
	newDraft := parseResponseBody[handlers.DraftResponse](t, recCreate).Data

	leaderCookie, leaderCSRF := loginUser(t, stack.engine, leader.Username, "password123")
	recApprove := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/drafts/%d/approvals", newDraft.ID), handlers.SubmitApprovalRequest{Status: "approved"}, leaderCookie, leaderCSRF)
	if recApprove.Code != http.StatusOK {
		t.Fatalf("failed to approve draft: %s", recApprove.Body.String())
	}

	// Act 2: Attempting to execute new draft while database is in quarantine must return HTTP 423 Locked!
	recExec := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/drafts/%d/execute", newDraft.ID), nil, aliceCookie, aliceCSRF)
	if recExec.Code != http.StatusLocked {
		t.Fatalf("expected 423 Locked (Quarantine) when database has unresolved partial batch, got %d: %s", recExec.Code, recExec.Body.String())
	}

	// Act 3: Leader votes for recovery action (Round approval for resume) on the quarantined batch draft
	// Setup round 1 approval record for leader
	_, _ = draftRepo.SaveApproval(ctx, &domain.DraftApproval{
		DraftID:  savedDraft.ID,
		GroupID:  group.ID,
		LeaderID: &leader.ID,
		Round:    1,
		Status:   domain.ApprovalStatusPending,
	})
	recRecoveryVote := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/drafts/%d/approvals", savedDraft.ID), handlers.SubmitApprovalRequest{Status: "resume"}, leaderCookie, leaderCSRF)
	if recRecoveryVote.Code != http.StatusOK {
		t.Fatalf("failed to vote resume on quarantined draft: %s", recRecoveryVote.Body.String())
	}
	recoveredDraft := parseResponseBody[handlers.DraftResponse](t, recRecoveryVote).Data
	if recoveredDraft.RecoveryAction != domain.VoteResume {
		t.Fatalf("expected recoveryAction to be resume, got %s", recoveredDraft.RecoveryAction)
	}
}
