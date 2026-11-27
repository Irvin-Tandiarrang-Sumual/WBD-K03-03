//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func TestRaceCondition_SimultaneousExecuteDraft_OnlyOneSucceedsOthersConflict(t *testing.T) {
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	targetDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target DB: %v", err)
	}

	leadEng := createRegularUser(t, stack, "lead_race_exec", "password123", nil)
	groupEng, err := stack.groupRepo.Save(ctx, domain.Group{
		LeaderID: leadEng.ID,
		Name:     "eng_race_exec_group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: targetDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create eng group: %v", err)
	}

	alice := createRegularUser(t, stack, "alice_race_exec", "password123", []uint{groupEng.ID})
	aliceCookie, aliceCSRF := loginUser(t, stack.engine, alice.Username, "password123")
	leadCookie, leadCSRF := loginUser(t, stack.engine, leadEng.Username, "password123")

	isRev := true
	createReq := handlers.CreateDraftRequest{
		SqlUp:        "CREATE TABLE race_single_exec (id SERIAL PRIMARY KEY, val TEXT);",
		SqlDown:      "DROP TABLE race_single_exec;",
		IsReversible: &isRev,
	}
	createPath := fmt.Sprintf("/api/v1/me/databases/%d/drafts", targetDB.ID)
	recCreate := executeRequest(stack.engine, http.MethodPost, createPath, createReq, aliceCookie, aliceCSRF)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("failed to create draft: %s", recCreate.Body.String())
	}
	draft := parseResponseBody[handlers.DraftResponse](t, recCreate).Data

	votePath := fmt.Sprintf("/api/v1/me/drafts/%d/approvals", draft.ID)
	recVote := executeRequest(stack.engine, http.MethodPost, votePath, handlers.SubmitApprovalRequest{Status: "approved"}, leadCookie, leadCSRF)
	if recVote.Code != http.StatusOK {
		t.Fatalf("failed to approve draft: %s", recVote.Body.String())
	}

	const concurrentCallers = 10
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})
	var acceptedCount int32
	var conflictCount int32

	execPath := fmt.Sprintf("/api/v1/me/drafts/%d/execute", draft.ID)

	for i := 0; i < concurrentCallers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startBarrier
			rec := executeRequest(stack.engine, http.MethodPost, execPath, nil, aliceCookie, aliceCSRF)
			if rec.Code == http.StatusAccepted {
				atomic.AddInt32(&acceptedCount, 1)
			} else if rec.Code == http.StatusConflict {
				atomic.AddInt32(&conflictCount, 1)
			}
		}()
	}

	close(startBarrier)
	wg.Wait()

	if acceptedCount != 1 {
		t.Fatalf("expected exactly 1 accepted execution request, got %d", acceptedCount)
	}
	if conflictCount != int32(concurrentCallers-1) {
		t.Fatalf("expected exactly %d conflict responses, got %d", concurrentCallers-1, conflictCount)
	}

	var jobCount int64
	err = stack.db.WithContext(ctx).Model(&model.MigrationJob{}).
		Where("database_id = ? AND draft_id = ?", targetDB.ID, draft.ID).
		Count(&jobCount).Error
	if err != nil {
		t.Fatalf("failed to query migration jobs: %v", err)
	}
	if jobCount != 1 {
		t.Fatalf("expected exactly 1 queued migration job, got %d", jobCount)
	}
}

func TestRaceCondition_ConcurrentApproveAndRejectVotes_ConsensusAlwaysRejects(t *testing.T) {
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	targetDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target DB: %v", err)
	}

	leadEng := createRegularUser(t, stack, "lead_eng_race_vote", "password123", nil)
	groupEng, err := stack.groupRepo.Save(ctx, domain.Group{
		LeaderID: leadEng.ID,
		Name:     "eng_race_vote_group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: targetDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create eng group: %v", err)
	}

	leadQA := createRegularUser(t, stack, "lead_qa_race_vote", "password123", nil)
	groupQA, err := stack.groupRepo.Save(ctx, domain.Group{
		LeaderID: leadQA.ID,
		Name:     "qa_race_vote_group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: targetDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create qa group: %v", err)
	}
	_ = groupQA

	alice := createRegularUser(t, stack, "alice_race_vote", "password123", []uint{groupEng.ID})
	aliceCookie, aliceCSRF := loginUser(t, stack.engine, alice.Username, "password123")

	isRev := true
	createReq := handlers.CreateDraftRequest{
		SqlUp:        "CREATE TABLE race_vote_test (id SERIAL PRIMARY KEY);",
		SqlDown:      "DROP TABLE race_vote_test;",
		IsReversible: &isRev,
	}
	createPath := fmt.Sprintf("/api/v1/me/databases/%d/drafts", targetDB.ID)
	recCreate := executeRequest(stack.engine, http.MethodPost, createPath, createReq, aliceCookie, aliceCSRF)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("failed to create draft: %s", recCreate.Body.String())
	}
	draft := parseResponseBody[handlers.DraftResponse](t, recCreate).Data

	leadEngCookie, leadEngCSRF := loginUser(t, stack.engine, leadEng.Username, "password123")
	leadQACookie, leadQACSRF := loginUser(t, stack.engine, leadQA.Username, "password123")

	votePath := fmt.Sprintf("/api/v1/me/drafts/%d/approvals", draft.ID)

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	wg.Add(2)
	go func() {
		defer wg.Done()
		<-startBarrier
		executeRequest(stack.engine, http.MethodPost, votePath, handlers.SubmitApprovalRequest{Status: "approved"}, leadEngCookie, leadEngCSRF)
	}()
	go func() {
		defer wg.Done()
		<-startBarrier
		executeRequest(stack.engine, http.MethodPost, votePath, handlers.SubmitApprovalRequest{Status: "rejected"}, leadQACookie, leadQACSRF)
	}()

	close(startBarrier)
	wg.Wait()

	draftRepo := repositories.NewDraftRepository(stack.db)
	finalDraft, err := draftRepo.FindByID(ctx, draft.ID)
	if err != nil {
		t.Fatalf("failed to retrieve draft after concurrent voting: %v", err)
	}

	if finalDraft.Status != domain.DraftStatusRejected {
		t.Fatalf("expected single veto to force rejected status under concurrent voting, got %s", finalDraft.Status)
	}
}

func TestRaceCondition_SimultaneousExecutionDifferentDraftsSameDatabase_SerializedByDatabaseLock(t *testing.T) {
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	targetDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target DB: %v", err)
	}

	leader := createRegularUser(t, stack, "lead_race_dblock", "password123", nil)
	group, err := stack.groupRepo.Save(ctx, domain.Group{
		LeaderID: leader.ID,
		Name:     "race_dblock_group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: targetDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	author := createRegularUser(t, stack, "author_race_dblock", "password123", []uint{group.ID})
	authorCookie, authorCSRF := loginUser(t, stack.engine, author.Username, "password123")
	leaderCookie, leaderCSRF := loginUser(t, stack.engine, leader.Username, "password123")

	createDraft := func(tableName string) uint {
		isRev := true
		req := handlers.CreateDraftRequest{
			SqlUp:        fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY);", tableName),
			SqlDown:      fmt.Sprintf("DROP TABLE %s;", tableName),
			IsReversible: &isRev,
		}
		rec := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/drafts", targetDB.ID), req, authorCookie, authorCSRF)
		if rec.Code != http.StatusCreated {
			t.Fatalf("failed to create draft: %s", rec.Body.String())
		}
		d := parseResponseBody[handlers.DraftResponse](t, rec).Data
		voteRec := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/drafts/%d/approvals", d.ID), handlers.SubmitApprovalRequest{Status: "approved"}, leaderCookie, leaderCSRF)
		if voteRec.Code != http.StatusOK {
			t.Fatalf("failed to approve draft: %s", voteRec.Body.String())
		}
		return d.ID
	}

	draft1ID := createDraft("race_lock_table_a")
	draft2ID := createDraft("race_lock_table_b")

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})
	statusCodes := make([]int, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		<-startBarrier
		rec := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/drafts/%d/execute", draft1ID), nil, authorCookie, authorCSRF)
		statusCodes[0] = rec.Code
	}()
	go func() {
		defer wg.Done()
		<-startBarrier
		rec := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/drafts/%d/execute", draft2ID), nil, authorCookie, authorCSRF)
		statusCodes[1] = rec.Code
	}()

	close(startBarrier)
	wg.Wait()

	hasAccepted := statusCodes[0] == http.StatusAccepted || statusCodes[1] == http.StatusAccepted
	hasConflict := statusCodes[0] == http.StatusConflict || statusCodes[1] == http.StatusConflict

	if !hasAccepted || !hasConflict {
		t.Fatalf("expected exactly one 202 Accepted and one 409 Conflict across simultaneous executions on same database, got: %v", statusCodes)
	}
}
