package draft_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/draft"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

// --- Draft Creation & Access Guard ---

func TestDraftService_CreateDraft_ValidEditAccess_CreatesDraftAndApprovalsForAllLeaders(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(10)
	databaseID := uint(1)
	leader1ID := uint(101)
	leader2ID := uint(102)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			if uID == authorID && dbID == databaseID && reqAccess == domain.AccessViewEdit {
				return nil
			}
			return domain.ErrDatabaseAccessDenied
		},
	}

	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{
				{ID: 1, Name: "Frontend Group", LeaderID: leader1ID},
				{ID: 2, Name: "Backend Group", LeaderID: leader2ID},
			}, nil
		},
	}

	savedApprovals := make([]*domain.DraftApproval, 0)
	draftRepo := &testutil.DraftRepository{
		SaveFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			d.ID = 1
			return d, nil
		},
		SaveApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			app.ID = uint(len(savedApprovals) + 1)
			savedApprovals = append(savedApprovals, app)
			return app, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, groupRepo, userGroupSvc)

	input := domain.CreateDraftInput{
		DatabaseID: databaseID,
		UserID:     authorID,
		SqlUp:      "CREATE TABLE users (id INT PRIMARY KEY);",
	}

	// Act
	createdDraft, err := svc.CreateDraft(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if createdDraft.Status != domain.DraftStatusPendingApproval {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusPendingApproval, createdDraft.Status)
	}
	if len(createdDraft.Approvals) != 2 {
		t.Fatalf("expected 2 approvals, got %d", len(createdDraft.Approvals))
	}
	for _, app := range createdDraft.Approvals {
		if app.Status != domain.ApprovalStatusPending {
			t.Fatalf("expected approval status %q, got %q", domain.ApprovalStatusPending, app.Status)
		}
	}
}

func TestDraftService_CreateDraft_ViewOnlyAccess_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(10)
	databaseID := uint(1)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			// User only has view_only, rejecting view_edit
			return domain.ErrDatabaseAccessDenied
		},
	}

	groupRepo := &testutil.GroupRepository{}
	draftRepo := &testutil.DraftRepository{}

	svc := draft.NewDraftService(draftRepo, groupRepo, userGroupSvc)

	input := domain.CreateDraftInput{
		DatabaseID: databaseID,
		UserID:     authorID,
		SqlUp:      "CREATE TABLE users (id INT PRIMARY KEY);",
	}

	// Act
	_, err := svc.CreateDraft(ctx, input)

	// Assert
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDraftService_CreateDraft_NoGroupAccess_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(99)
	databaseID := uint(1)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return domain.ErrAccessDenied
		},
	}

	groupRepo := &testutil.GroupRepository{}
	draftRepo := &testutil.DraftRepository{}

	svc := draft.NewDraftService(draftRepo, groupRepo, userGroupSvc)

	input := domain.CreateDraftInput{
		DatabaseID: databaseID,
		UserID:     authorID,
		SqlUp:      "CREATE TABLE users (id INT PRIMARY KEY);",
	}

	// Act
	_, err := svc.CreateDraft(ctx, input)

	// Assert
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDraftService_CreateDraft_AnotherDraftAlreadyApproved_AllowsCreationAndVoting(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(10)
	databaseID := uint(1)
	leaderID := uint(101)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}

	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{{ID: 1, Name: "Main Group", LeaderID: leaderID}}, nil
		},
	}

	// Simulate Draft 1 is already approved in the database while Draft 2 is created and voted on
	draftsStore := map[uint]*domain.Draft{
		1: {ID: 1, DatabaseID: databaseID, UserID: authorID, Status: domain.DraftStatusApproved},
	}
	approvalsStore := map[uint][]*domain.DraftApproval{}

	draftRepo := &testutil.DraftRepository{
		SaveFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			d.ID = 2
			draftsStore[d.ID] = d
			return d, nil
		},
		SaveApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			app.ID = 2
			approvalsStore[app.DraftID] = append(approvalsStore[app.DraftID], app)
			return app, nil
		},
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			if d, ok := draftsStore[id]; ok {
				return d, nil
			}
			return nil, domain.ErrDraftNotFound
		},
		FindApprovalsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftApproval, error) {
			return approvalsStore[dID], nil
		},
		UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			for _, a := range approvalsStore[app.DraftID] {
				if a.ID == app.ID {
					a.Status = app.Status
					a.RespondedAt = app.RespondedAt
				}
			}
			return app, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftsStore[d.ID] = d
			return d, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, groupRepo, userGroupSvc)

	createInput := domain.CreateDraftInput{
		DatabaseID: databaseID,
		UserID:     authorID,
		SqlUp:      "CREATE TABLE products (id INT PRIMARY KEY);",
	}

	// Act
	newDraft, err := svc.CreateDraft(ctx, createInput)
	if err != nil {
		t.Fatalf("expected nil error on CreateDraft, got %v", err)
	}

	votedDraft, voteErr := svc.SubmitApproval(ctx, domain.SubmitApprovalInput{
		DraftID: newDraft.ID,
		UserID:  leaderID,
		Status:  domain.ApprovalStatusApproved,
	})

	// Assert
	if voteErr != nil {
		t.Fatalf("expected nil error on SubmitApproval, got %v", voteErr)
	}
	if newDraft.ID != 2 {
		t.Fatalf("expected new draft id 2, got %d", newDraft.ID)
	}
	if votedDraft.Status != domain.DraftStatusApproved {
		t.Fatalf("expected voted draft status %q, got %q", domain.DraftStatusApproved, votedDraft.Status)
	}
	if draftsStore[1].Status != domain.DraftStatusApproved {
		t.Fatalf("expected existing draft 1 to remain %q, got %q", domain.DraftStatusApproved, draftsStore[1].Status)
	}
}

// --- Approval Consensus ---

func TestDraftService_SubmitApproval_UserNotGroupLeader_ReturnsErrUnauthorized(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	nonLeaderUserID := uint(999)
	leaderID := uint(100)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return &domain.Draft{
				ID:         draftID,
				DatabaseID: 1,
				UserID:     authorID,
				Status:     domain.DraftStatusPendingApproval,
			}, nil
		},
		FindApprovalsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftApproval, error) {
			return []*domain.DraftApproval{
				{ID: 1, DraftID: draftID, GroupID: 1, LeaderID: &leaderID, Status: domain.ApprovalStatusPending},
			}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	input := domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  nonLeaderUserID,
		Status:  domain.ApprovalStatusApproved,
	}

	// Act
	_, err := svc.SubmitApproval(ctx, input)

	// Assert
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestDraftService_SubmitApproval_SingleLeaderRejects_TransitionsToRejected(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	leader1ID := uint(101)
	leader2ID := uint(102)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: 1,
		UserID:     authorID,
		Status:     domain.DraftStatusPendingApproval,
	}

	approvals := []*domain.DraftApproval{
		{ID: 1, DraftID: draftID, GroupID: 1, LeaderID: &leader1ID, Status: domain.ApprovalStatusApproved},
		{ID: 2, DraftID: draftID, GroupID: 2, LeaderID: &leader2ID, Status: domain.ApprovalStatusPending},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
		FindApprovalsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftApproval, error) {
			return approvals, nil
		},
		UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			for _, a := range approvals {
				if a.ID == app.ID {
					a.Status = app.Status
					a.RespondedAt = app.RespondedAt
				}
			}
			return app, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			activeDraft = d
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	input := domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  leader2ID,
		Status:  domain.ApprovalStatusRejected,
	}

	// Act
	updatedDraft, err := svc.SubmitApproval(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if updatedDraft.Status != domain.DraftStatusRejected {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusRejected, updatedDraft.Status)
	}
}

func TestDraftService_SubmitApproval_AllLeadersApprove_TransitionsToApproved(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	leader1ID := uint(101)
	leader2ID := uint(102)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: 1,
		UserID:     authorID,
		Status:     domain.DraftStatusPendingApproval,
	}

	approvals := []*domain.DraftApproval{
		{ID: 1, DraftID: draftID, GroupID: 1, LeaderID: &leader1ID, Status: domain.ApprovalStatusApproved},
		{ID: 2, DraftID: draftID, GroupID: 2, LeaderID: &leader2ID, Status: domain.ApprovalStatusPending},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
		FindApprovalsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftApproval, error) {
			return approvals, nil
		},
		UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			for _, a := range approvals {
				if a.ID == app.ID {
					a.Status = app.Status
					a.RespondedAt = app.RespondedAt
				}
			}
			return app, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			activeDraft = d
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	input := domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  leader2ID,
		Status:  domain.ApprovalStatusApproved,
	}

	// Act
	updatedDraft, err := svc.SubmitApproval(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if updatedDraft.Status != domain.DraftStatusApproved {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusApproved, updatedDraft.Status)
	}
}

func TestDraftService_SubmitApproval_DraftNotPendingApproval_ReturnsErrInvalidDraftState(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	leaderID := uint(101)

	testCases := []string{
		domain.DraftStatusApproved,
		domain.DraftStatusRejected,
		domain.DraftStatusInvalidated,
		domain.DraftStatusExecuting,
		domain.DraftStatusApplied,
		domain.DraftStatusFailed,
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			draftRepo := &testutil.DraftRepository{
				FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
					return &domain.Draft{
						ID:     draftID,
						Status: tc,
					}, nil
				},
			}

			svc := draft.NewDraftService(draftRepo, nil, nil)

			input := domain.SubmitApprovalInput{
				DraftID: draftID,
				UserID:  leaderID,
				Status:  domain.ApprovalStatusApproved,
			}

			// Act
			_, err := svc.SubmitApproval(ctx, input)

			// Assert
			if !errors.Is(err, domain.ErrInvalidDraftState) {
				t.Fatalf("expected ErrInvalidDraftState for status %s, got %v", tc, err)
			}
		})
	}
}

func TestDraftService_SubmitApproval_AuthorLostEditAccess_InvalidatesDraftAndReturnsErrInvalidDraftState(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	leaderID := uint(101)
	databaseID := uint(1)

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: databaseID,
		UserID:     authorID,
		Status:     domain.DraftStatusPendingApproval,
	}

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return domain.ErrDatabaseAccessDenied
		},
	}

	updateCalled := false
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			updateCalled = true
			activeDraft = d
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	input := domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  leaderID,
		Status:  domain.ApprovalStatusApproved,
	}

	// Act
	_, err := svc.SubmitApproval(ctx, input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidDraftState) {
		t.Fatalf("expected ErrInvalidDraftState, got %v", err)
	}
	if !updateCalled {
		t.Fatal("expected draftRepo.Update to be called to persist invalidated state")
	}
	if activeDraft.Status != domain.DraftStatusInvalidated {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusInvalidated, activeDraft.Status)
	}
}

// --- Auto-Invalidation Edge Cases ---

func TestDraftService_ValidateDraft_AuthorRemovedFromGroup_TransitionsToInvalidated(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	databaseID := uint(1)

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: databaseID,
		UserID:     authorID,
		Status:     domain.DraftStatusPendingApproval,
	}

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			// Author was removed from group, so authorize fails
			return domain.ErrAccessDenied
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			activeDraft = d
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	// Act
	validatedDraft, err := svc.ValidateDraft(ctx, draftID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if validatedDraft.Status != domain.DraftStatusInvalidated {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusInvalidated, validatedDraft.Status)
	}
}

func TestDraftService_ValidateDraft_GroupDetachedFromDatabase_TransitionsToInvalidated(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	databaseID := uint(1)

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: databaseID,
		UserID:     authorID,
		Status:     domain.DraftStatusApproved,
	}

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			// Group was detached from database
			return domain.ErrDatabaseAccessDenied
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			activeDraft = d
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	// Act
	validatedDraft, err := svc.ValidateDraft(ctx, draftID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if validatedDraft.Status != domain.DraftStatusInvalidated {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusInvalidated, validatedDraft.Status)
	}
}

func TestDraftService_ValidateDraft_GroupAccessDowngradedToViewOnly_TransitionsToInvalidated(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	databaseID := uint(1)

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: databaseID,
		UserID:     authorID,
		Status:     domain.DraftStatusApproved,
	}

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			// Group access was downgraded from view_edit to view_only, rejecting view_edit
			if reqAccess == domain.AccessViewEdit {
				return domain.ErrDatabaseAccessDenied
			}
			return nil
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			activeDraft = d
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	// Act
	validatedDraft, err := svc.ValidateDraft(ctx, draftID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if validatedDraft.Status != domain.DraftStatusInvalidated {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusInvalidated, validatedDraft.Status)
	}
}

func TestDraftService_ExecuteDraft_DraftInvalidated_ReturnsErrDraftNotApproved(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	databaseID := uint(1)

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: databaseID,
		UserID:     authorID,
		Status:     domain.DraftStatusApproved, // initially approved
	}

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			// Access lost right before execution!
			return domain.ErrDatabaseAccessDenied
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			activeDraft = d
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, authorID)

	// Assert
	if !errors.Is(err, domain.ErrDraftNotApproved) {
		t.Fatalf("expected ErrDraftNotApproved, got %v", err)
	}
	if activeDraft.Status != domain.DraftStatusInvalidated {
		t.Fatalf("expected draft status to be updated to %q, got %q", domain.DraftStatusInvalidated, activeDraft.Status)
	}
}

func TestDraftService_ExecuteDraft_CallerLacksEditAccess_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	callerID := uint(99)
	databaseID := uint(1)

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: databaseID,
		UserID:     authorID,
		Status:     domain.DraftStatusApproved,
	}

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			if uID == authorID {
				return nil
			}
			return domain.ErrDatabaseAccessDenied
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, callerID)

	// Assert
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if activeDraft.Status != domain.DraftStatusApproved {
		t.Fatalf("expected draft status to remain %q, got %q", domain.DraftStatusApproved, activeDraft.Status)
	}
}

// --- Additional Edge Cases & Workflow ---

func TestDraftService_ExecuteDraft_Approved_TransitionsToExecuting(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	databaseID := uint(1)

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: databaseID,
		UserID:     authorID,
		Status:     domain.DraftStatusApproved,
	}

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			activeDraft = d
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, authorID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if activeDraft.Status != domain.DraftStatusExecuting {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusExecuting, activeDraft.Status)
	}
}

func TestDraftService_ValidateDraft_StillValid_ReturnsDraftWithoutChanges(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	databaseID := uint(1)

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: databaseID,
		UserID:     authorID,
		Status:     domain.DraftStatusPendingApproval,
	}

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	// Act
	validated, err := svc.ValidateDraft(ctx, draftID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if validated.Status != domain.DraftStatusPendingApproval {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusPendingApproval, validated.Status)
	}
}

func TestDraftService_ValidateDraft_AlreadyTerminal_ReturnsDraftDirectly(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)

	activeDraft := &domain.Draft{
		ID:     draftID,
		Status: domain.DraftStatusRejected,
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	validated, err := svc.ValidateDraft(ctx, draftID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if validated.Status != domain.DraftStatusRejected {
		t.Fatalf("expected status %q, got %q", domain.DraftStatusRejected, validated.Status)
	}
}

func TestDraftService_SubmitApproval_PartialApproval_RemainsPendingApproval(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	leader1ID := uint(101)
	leader2ID := uint(102)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}

	activeDraft := &domain.Draft{
		ID:         draftID,
		DatabaseID: 1,
		UserID:     authorID,
		Status:     domain.DraftStatusPendingApproval,
	}

	approvals := []*domain.DraftApproval{
		{ID: 1, DraftID: draftID, GroupID: 1, LeaderID: &leader1ID, Status: domain.ApprovalStatusPending},
		{ID: 2, DraftID: draftID, GroupID: 2, LeaderID: &leader2ID, Status: domain.ApprovalStatusPending},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return activeDraft, nil
		},
		FindApprovalsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftApproval, error) {
			return approvals, nil
		},
		UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			for _, a := range approvals {
				if a.ID == app.ID {
					a.Status = app.Status
					a.RespondedAt = app.RespondedAt
				}
			}
			return app, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	input := domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  leader1ID,
		Status:  domain.ApprovalStatusApproved,
	}

	// Act
	res, err := svc.SubmitApproval(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.Status != domain.DraftStatusPendingApproval {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusPendingApproval, res.Status)
	}
}

// --- Querying Drafts ---

func TestDraftService_GetDraftsByDatabaseID_ValidAccess_ReturnsDrafts(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uint(10)
	databaseID := uint(1)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint, params pagination.PaginationParams) ([]*domain.Draft, int64, error) {
			return []*domain.Draft{{ID: 1, DatabaseID: databaseID, Status: domain.DraftStatusApproved}}, 1, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	// Act
	drafts, total, err := svc.GetDraftsByDatabaseID(ctx, databaseID, userID, pagination.PaginationParams{Page: 1, Limit: 10})

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if total != 1 || len(drafts) != 1 {
		t.Fatalf("expected 1 draft, got %d with total %d", len(drafts), total)
	}
}

func TestDraftService_GetDraftsByDatabaseID_NoAccess_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uint(10)
	databaseID := uint(1)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return domain.ErrAccessDenied
		},
	}

	svc := draft.NewDraftService(nil, nil, userGroupSvc)

	// Act
	_, _, err := svc.GetDraftsByDatabaseID(ctx, databaseID, userID, pagination.PaginationParams{Page: 1, Limit: 10})

	// Assert
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDraftService_GetDraftByID_ValidAccess_ReturnsDraft(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uint(10)
	draftID := uint(1)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return &domain.Draft{ID: draftID, DatabaseID: 1, Status: domain.DraftStatusApproved}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	// Act
	d, err := svc.GetDraftByID(ctx, draftID, userID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if d.ID != draftID {
		t.Fatalf("expected draft id %d, got %d", draftID, d.ID)
	}
}

func TestDraftService_GetDraftByID_NoAccess_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uint(10)
	draftID := uint(1)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return domain.ErrAccessDenied
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return &domain.Draft{ID: draftID, DatabaseID: 1, Status: domain.DraftStatusApproved}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, userGroupSvc)

	// Act
	_, err := svc.GetDraftByID(ctx, draftID, userID)

	// Assert
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// --- Validation & Repository Error Branch Coverage ---

func TestDraftService_CreateDraft_InvalidInput_ReturnsErrInvalidDraftInput(t *testing.T) {
	// Arrange
	ctx := context.Background()
	svc := draft.NewDraftService(nil, nil, nil)
	input := domain.CreateDraftInput{DatabaseID: 1, UserID: 10, SqlUp: "   "}

	// Act
	_, err := svc.CreateDraft(ctx, input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidDraftInput) {
		t.Fatalf("expected ErrInvalidDraftInput, got %v", err)
	}
}

func TestDraftService_CreateDraft_GroupLookupFails_ReturnsError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repoErr := errors.New("db query failed")
	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return nil, repoErr
		},
	}
	svc := draft.NewDraftService(nil, groupRepo, userGroupSvc)

	// Act
	_, err := svc.CreateDraft(ctx, domain.CreateDraftInput{DatabaseID: 1, UserID: 10, SqlUp: "CREATE TABLE t (id INT);"})

	// Assert
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected %v, got %v", repoErr, err)
	}
}

func TestDraftService_CreateDraft_ExplicitIrreversibleFalse_SavesDraftWithIsReversibleFalse(t *testing.T) {
	// Arrange
	ctx := context.Background()
	isRev := false
	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{}, nil
		},
	}
	draftRepo := &testutil.DraftRepository{
		SaveFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			d.ID = 1
			return d, nil
		},
	}
	svc := draft.NewDraftService(draftRepo, groupRepo, userGroupSvc)

	// Act
	created, err := svc.CreateDraft(ctx, domain.CreateDraftInput{
		DatabaseID:         1,
		UserID:             10,
		SqlUp:              "DROP TABLE legacy;",
		IsReversible:       &isRev,
		IrreversibleReason: "drops table",
	})

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if created.IsReversible {
		t.Fatal("expected IsReversible to be false")
	}
}

func TestDraftService_CreateDraft_SaveDraftFails_ReturnsError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repoErr := errors.New("save draft failed")
	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{}, nil
		},
	}
	draftRepo := &testutil.DraftRepository{
		SaveFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			return nil, repoErr
		},
	}
	svc := draft.NewDraftService(draftRepo, groupRepo, userGroupSvc)

	// Act
	_, err := svc.CreateDraft(ctx, domain.CreateDraftInput{DatabaseID: 1, UserID: 10, SqlUp: "CREATE TABLE t (id INT);"})

	// Assert
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected %v, got %v", repoErr, err)
	}
}

func TestDraftService_CreateDraft_SaveApprovalFails_ReturnsError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repoErr := errors.New("save approval failed")
	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{{ID: 1, LeaderID: 101}}, nil
		},
	}
	draftRepo := &testutil.DraftRepository{
		SaveFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			d.ID = 1
			return d, nil
		},
		SaveApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			return nil, repoErr
		},
	}
	svc := draft.NewDraftService(draftRepo, groupRepo, userGroupSvc)

	// Act
	_, err := svc.CreateDraft(ctx, domain.CreateDraftInput{DatabaseID: 1, UserID: 10, SqlUp: "CREATE TABLE t (id INT);"})

	// Assert
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected %v, got %v", repoErr, err)
	}
}

func TestDraftService_GetDraftsByDatabaseID_InvalidIDs_ReturnsErrInvalidDraftInput(t *testing.T) {
	// Arrange
	ctx := context.Background()
	svc := draft.NewDraftService(nil, nil, nil)

	// Act
	_, _, err := svc.GetDraftsByDatabaseID(ctx, 0, 10, pagination.PaginationParams{Page: 1, Limit: 10})

	// Assert
	if !errors.Is(err, domain.ErrInvalidDraftInput) {
		t.Fatalf("expected ErrInvalidDraftInput, got %v", err)
	}
}

func TestDraftService_GetDraftByID_InvalidIDs_ReturnsErrInvalidDraftInput(t *testing.T) {
	// Arrange
	ctx := context.Background()
	svc := draft.NewDraftService(nil, nil, nil)

	// Act
	_, err := svc.GetDraftByID(ctx, 0, 10)

	// Assert
	if !errors.Is(err, domain.ErrInvalidDraftInput) {
		t.Fatalf("expected ErrInvalidDraftInput, got %v", err)
	}
}

func TestDraftService_GetDraftByID_DraftNotFound_ReturnsErrDraftNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return nil, domain.ErrDraftNotFound
		},
	}
	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	_, err := svc.GetDraftByID(ctx, 99, 10)

	// Assert
	if !errors.Is(err, domain.ErrDraftNotFound) {
		t.Fatalf("expected ErrDraftNotFound, got %v", err)
	}
}

func TestDraftService_SubmitApproval_InvalidInput_ReturnsErrInvalidDraftInput(t *testing.T) {
	// Arrange
	ctx := context.Background()
	svc := draft.NewDraftService(nil, nil, nil)

	// Act
	_, err := svc.SubmitApproval(ctx, domain.SubmitApprovalInput{DraftID: 1, UserID: 10, Status: "maybe"})

	// Assert
	if !errors.Is(err, domain.ErrInvalidDraftInput) {
		t.Fatalf("expected ErrInvalidDraftInput, got %v", err)
	}
}

func TestDraftService_SubmitApproval_DraftNotFound_ReturnsErrDraftNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return nil, domain.ErrDraftNotFound
		},
	}
	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	_, err := svc.SubmitApproval(ctx, domain.SubmitApprovalInput{DraftID: 99, UserID: 10, Status: domain.ApprovalStatusApproved})

	// Assert
	if !errors.Is(err, domain.ErrDraftNotFound) {
		t.Fatalf("expected ErrDraftNotFound, got %v", err)
	}
}

func TestDraftService_SubmitApproval_RepositoryErrors_ReturnsError(t *testing.T) {
	repoErr := errors.New("repository failure")
	leaderID := uint(101)

	testCases := []struct {
		name         string
		status       string
		authorizeErr error
		setupRepo    func() *testutil.DraftRepository
	}{
		{
			name:         "invalidation update fails",
			status:       domain.ApprovalStatusApproved,
			authorizeErr: domain.ErrAccessDenied,
			setupRepo: func() *testutil.DraftRepository {
				return &testutil.DraftRepository{
					FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
						return &domain.Draft{ID: 1, DatabaseID: 1, UserID: 10, Status: domain.DraftStatusPendingApproval}, nil
					},
					UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
						return nil, repoErr
					},
				}
			},
		},
		{
			name:   "find approvals fails",
			status: domain.ApprovalStatusApproved,
			setupRepo: func() *testutil.DraftRepository {
				return &testutil.DraftRepository{
					FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
						return &domain.Draft{ID: 1, DatabaseID: 1, UserID: 10, Status: domain.DraftStatusPendingApproval}, nil
					},
					FindApprovalsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftApproval, error) {
						return nil, repoErr
					},
				}
			},
		},
		{
			name:   "update approval fails",
			status: domain.ApprovalStatusApproved,
			setupRepo: func() *testutil.DraftRepository {
				return &testutil.DraftRepository{
					FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
						return &domain.Draft{ID: 1, DatabaseID: 1, UserID: 10, Status: domain.DraftStatusPendingApproval}, nil
					},
					FindApprovalsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftApproval, error) {
						return []*domain.DraftApproval{{ID: 1, DraftID: 1, LeaderID: &leaderID, Status: domain.ApprovalStatusPending}}, nil
					},
					UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
						return nil, repoErr
					},
				}
			},
		},
		{
			name:   "reject draft update fails",
			status: domain.ApprovalStatusRejected,
			setupRepo: func() *testutil.DraftRepository {
				return &testutil.DraftRepository{
					FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
						return &domain.Draft{ID: 1, DatabaseID: 1, UserID: 10, Status: domain.DraftStatusPendingApproval}, nil
					},
					FindApprovalsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftApproval, error) {
						return []*domain.DraftApproval{{ID: 1, DraftID: 1, LeaderID: &leaderID, Status: domain.ApprovalStatusPending}}, nil
					},
					UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
						return app, nil
					},
					UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
						return nil, repoErr
					},
				}
			},
		},
		{
			name:   "reload approvals after approve fails",
			status: domain.ApprovalStatusApproved,
			setupRepo: func() *testutil.DraftRepository {
				calls := 0
				return &testutil.DraftRepository{
					FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
						return &domain.Draft{ID: 1, DatabaseID: 1, UserID: 10, Status: domain.DraftStatusPendingApproval}, nil
					},
					FindApprovalsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftApproval, error) {
						calls++
						if calls == 1 {
							return []*domain.DraftApproval{{ID: 1, DraftID: 1, LeaderID: &leaderID, Status: domain.ApprovalStatusPending}}, nil
						}
						return nil, repoErr
					},
					UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
						return app, nil
					},
				}
			},
		},
		{
			name:   "approve draft update fails",
			status: domain.ApprovalStatusApproved,
			setupRepo: func() *testutil.DraftRepository {
				return &testutil.DraftRepository{
					FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
						return &domain.Draft{ID: 1, DatabaseID: 1, UserID: 10, Status: domain.DraftStatusPendingApproval}, nil
					},
					FindApprovalsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftApproval, error) {
						return []*domain.DraftApproval{{ID: 1, DraftID: 1, LeaderID: &leaderID, Status: domain.ApprovalStatusApproved}}, nil
					},
					UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
						return app, nil
					},
					UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
						return nil, repoErr
					},
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()
			userGroupSvc := &testutil.UserGroupService{
				AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
					return tc.authorizeErr
				},
			}
			svc := draft.NewDraftService(tc.setupRepo(), nil, userGroupSvc)

			// Act
			_, err := svc.SubmitApproval(ctx, domain.SubmitApprovalInput{
				DraftID: 1,
				UserID:  leaderID,
				Status:  tc.status,
			})

			// Assert
			if !errors.Is(err, repoErr) {
				t.Fatalf("expected %v, got %v", repoErr, err)
			}
		})
	}
}

func TestDraftService_ValidateDraft_InvalidID_ReturnsErrInvalidDraftInput(t *testing.T) {
	// Arrange
	ctx := context.Background()
	svc := draft.NewDraftService(nil, nil, nil)

	// Act
	_, err := svc.ValidateDraft(ctx, 0)

	// Assert
	if !errors.Is(err, domain.ErrInvalidDraftInput) {
		t.Fatalf("expected ErrInvalidDraftInput, got %v", err)
	}
}

func TestDraftService_ValidateDraft_DraftNotFound_ReturnsErrDraftNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return nil, domain.ErrDraftNotFound
		},
	}
	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	_, err := svc.ValidateDraft(ctx, 99)

	// Assert
	if !errors.Is(err, domain.ErrDraftNotFound) {
		t.Fatalf("expected ErrDraftNotFound, got %v", err)
	}
}

func TestDraftService_ExecuteDraft_InvalidIDs_ReturnsErrInvalidDraftInput(t *testing.T) {
	// Arrange
	ctx := context.Background()
	svc := draft.NewDraftService(nil, nil, nil)

	// Act
	err := svc.ExecuteDraft(ctx, 0, 10)

	// Assert
	if !errors.Is(err, domain.ErrInvalidDraftInput) {
		t.Fatalf("expected ErrInvalidDraftInput, got %v", err)
	}
}

func TestDraftService_ExecuteDraft_DraftNotFound_ReturnsErrDraftNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return nil, domain.ErrDraftNotFound
		},
	}
	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	err := svc.ExecuteDraft(ctx, 99, 10)

	// Assert
	if !errors.Is(err, domain.ErrDraftNotFound) {
		t.Fatalf("expected ErrDraftNotFound, got %v", err)
	}
}

func TestDraftService_ExecuteDraft_DatabaseNotFound_ReturnsErrDatabaseNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	dbID := uint(99)

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return &domain.Draft{
				ID:         draftID,
				UserID:     authorID,
				DatabaseID: dbID,
				Status:     domain.DraftStatusApproved,
			}, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		FindByIdFunc: func(ctx context.Context, id uint) (*domain.Database, error) {
			return nil, domain.ErrDatabaseNotFound
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil, dbRepo)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, authorID)

	// Assert
	if !errors.Is(err, domain.ErrDatabaseNotFound) {
		t.Fatalf("expected ErrDatabaseNotFound, got %v", err)
	}
}

func TestDraftService_ExecuteDraft_NotAuthor_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	callerID := uint(99) // Different user

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return &domain.Draft{
				ID:         draftID,
				UserID:     authorID,
				DatabaseID: 1,
				Status:     domain.DraftStatusApproved,
			}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, callerID)

	// Assert
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDraftService_ExecuteDraft_DraftNotApproved_ReturnsErrDraftNotApproved(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return &domain.Draft{
				ID:         draftID,
				UserID:     authorID,
				DatabaseID: 1,
				Status:     domain.DraftStatusPendingApproval, // Not approved!
			}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, authorID)

	// Assert
	if !errors.Is(err, domain.ErrDraftNotApproved) {
		t.Fatalf("expected ErrDraftNotApproved, got %v", err)
	}
}

func TestDraftService_ExecuteDraft_DatabaseExecuting_ReturnsErrMigrationInProgress(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	dbID := uint(1)

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return &domain.Draft{
				ID:         draftID,
				UserID:     authorID,
				DatabaseID: dbID,
				Status:     domain.DraftStatusApproved,
			}, nil
		},
		FindDraftByStatusAndDatabaseIDFunc: func(ctx context.Context, dID uint, status string) (*domain.Draft, error) {
			if status == domain.DraftStatusExecuting {
				return &domain.Draft{ID: 999, DatabaseID: dbID, Status: domain.DraftStatusExecuting}, nil
			}
			return nil, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, authorID)

	// Assert
	if !errors.Is(err, domain.ErrMigrationInProgress) {
		t.Fatalf("expected ErrMigrationInProgress, got %v", err)
	}
}

func TestDraftService_ExecuteDraft_DatabasePartiallyReverted_ReturnsErrDatabaseMustResolvePartiallyReverted(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)
	dbID := uint(1)

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return &domain.Draft{
				ID:         draftID,
				UserID:     authorID,
				DatabaseID: dbID,
				Status:     domain.DraftStatusApproved,
			}, nil
		},
		FindDraftByStatusAndDatabaseIDFunc: func(ctx context.Context, dID uint, status string) (*domain.Draft, error) {
			if status == domain.DraftStatusPartiallyReverted {
				return &domain.Draft{ID: 888, DatabaseID: dbID, Status: domain.DraftStatusPartiallyReverted}, nil
			}
			return nil, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, authorID)

	// Assert
	if !errors.Is(err, domain.ErrDatabaseMustResolvePartiallyReverted) {
		t.Fatalf("expected ErrDatabaseMustResolvePartiallyReverted, got %v", err)
	}
}

func TestDraftService_ExecuteDraft_ConcurrentRequestsSameDatabase_OnlyOneSucceedsAndSecondReturnsErrMigrationInProgress(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(10)
	dbID := uint(1)

	draft1 := &domain.Draft{ID: 1, UserID: authorID, DatabaseID: dbID, Status: domain.DraftStatusApproved}
	draft2 := &domain.Draft{ID: 2, UserID: authorID, DatabaseID: dbID, Status: domain.DraftStatusApproved}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			if id == 1 {
				return draft1, nil
			}
			return draft2, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			return d, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		FindByIdFunc: func(ctx context.Context, id uint) (*domain.Database, error) {
			return &domain.Database{ID: dbID}, nil
		},
		IncrementExecutionIDFunc: func(ctx context.Context, dID uint) (uint, error) {
			return 1, nil
		},
	}
	queue := &testutil.TaskQueue{}

	svc := draft.NewDraftService(draftRepo, nil, nil, dbRepo, queue)

	// Act: First request acquires lock
	err1 := svc.ExecuteDraft(ctx, 1, authorID)
	// Second request for same DB while first is executing
	err2 := svc.ExecuteDraft(ctx, 2, authorID)

	// Assert
	if err1 != nil {
		t.Fatalf("expected first execution to succeed, got %v", err1)
	}
	if !errors.Is(err2, domain.ErrMigrationInProgress) {
		t.Fatalf("expected ErrMigrationInProgress on second concurrent request, got %v", err2)
	}
}

func TestDraftService_ExecuteDraft_DifferentDatabases_AllowsParallelExecution(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(10)
	db1ID := uint(1)
	db2ID := uint(2)

	draft1 := &domain.Draft{ID: 1, UserID: authorID, DatabaseID: db1ID, Status: domain.DraftStatusApproved}
	draft2 := &domain.Draft{ID: 2, UserID: authorID, DatabaseID: db2ID, Status: domain.DraftStatusApproved}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			if id == 1 {
				return draft1, nil
			}
			return draft2, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			return d, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		FindByIdFunc: func(ctx context.Context, id uint) (*domain.Database, error) {
			return &domain.Database{ID: id}, nil
		},
		IncrementExecutionIDFunc: func(ctx context.Context, dID uint) (uint, error) {
			return 1, nil
		},
	}
	queue := &testutil.TaskQueue{}

	svc := draft.NewDraftService(draftRepo, nil, nil, dbRepo, queue)

	// Act: Execute draft on DB 1 and draft on DB 2
	err1 := svc.ExecuteDraft(ctx, 1, authorID)
	err2 := svc.ExecuteDraft(ctx, 2, authorID)

	// Assert: Both succeed because database mutexes are isolated per databaseID
	if err1 != nil {
		t.Fatalf("expected DB1 execution to succeed, got %v", err1)
	}
	if err2 != nil {
		t.Fatalf("expected DB2 execution to succeed, got %v", err2)
	}
}

func TestDraftService_ExecuteDraft_ValidApprovedDraft_IncrementsExecutionIDAndEnqueuesTask(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(42)
	authorID := uint(10)
	dbID := uint(1)
	expectedExecutionID := uint(7)

	var enqueuedTask ports.ExecutionTask
	draftObj := &domain.Draft{
		ID:         draftID,
		UserID:     authorID,
		DatabaseID: dbID,
		Status:     domain.DraftStatusApproved,
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
		FindByIdFunc: func(ctx context.Context, id uint) (*domain.Database, error) {
			return &domain.Database{ID: dbID}, nil
		},
		IncrementExecutionIDFunc: func(ctx context.Context, dID uint) (uint, error) {
			return expectedExecutionID, nil
		},
	}
	queue := &testutil.TaskQueue{
		EnqueueFunc: func(ctx context.Context, task ports.ExecutionTask) error {
			enqueuedTask = task
			return nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil, dbRepo, queue)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, authorID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if draftObj.Status != domain.DraftStatusExecuting {
		t.Fatalf("expected draft status %q, got %q", domain.DraftStatusExecuting, draftObj.Status)
	}
	if enqueuedTask.DraftID != draftID || enqueuedTask.DatabaseID != dbID || enqueuedTask.ExecutionID != expectedExecutionID {
		t.Fatalf("expected enqueued task with execution_id %d, got %+v", expectedExecutionID, enqueuedTask)
	}
}

func TestDraftService_CreateDraft_BatchWithDDL_ReturnsErrBatchContainsDDL(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(10)
	databaseID := uint(1)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{{ID: 1, LeaderID: 101}}, nil
		},
	}
	svc := draft.NewDraftService(nil, groupRepo, userGroupSvc)

	input := domain.CreateDraftInput{
		DatabaseID:  databaseID,
		UserID:      authorID,
		IsBatch:     true,
		BatchColumn: "id",
		BatchSize:   100,
		SqlUp:       "ALTER TABLE users ADD COLUMN age INT;",
	}

	// Act
	_, err := svc.CreateDraft(ctx, input)

	// Assert
	if !errors.Is(err, domain.ErrBatchContainsDDL) {
		t.Fatalf("expected ErrBatchContainsDDL, got %v", err)
	}
}

func TestDraftService_CreateDraft_BatchMissingColumnOrSize_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	ctx := context.Background()
	svc := draft.NewDraftService(nil, nil, nil)

	input1 := domain.CreateDraftInput{
		DatabaseID:  1,
		UserID:      10,
		IsBatch:     true,
		BatchColumn: "",
		BatchSize:   100,
		SqlUp:       "UPDATE users SET active = true;",
	}
	input2 := domain.CreateDraftInput{
		DatabaseID:  1,
		UserID:      10,
		IsBatch:     true,
		BatchColumn: "id",
		BatchSize:   0,
		SqlUp:       "UPDATE users SET active = true;",
	}

	// Act
	_, err1 := svc.CreateDraft(ctx, input1)
	_, err2 := svc.CreateDraft(ctx, input2)

	// Assert
	if !errors.Is(err1, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for missing column, got %v", err1)
	}
	if !errors.Is(err2, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for zero size, got %v", err2)
	}
}

func TestDraftService_CreateDraft_BatchColumnNotFoundInTargetTable_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(10)
	databaseID := uint(1)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: databaseID, Host: "localhost"}, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		GetDatabaseMetadataFunc: func(ctx context.Context, db domain.Database) (*ports.DatabaseMetadata, error) {
			return &ports.DatabaseMetadata{
				Columns: []ports.DatabaseColumnMetadata{
					{Table: "users", Name: "id", DataType: "bigint"},
					{Table: "users", Name: "name", DataType: "varchar"},
				},
			}, nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{{ID: 1, LeaderID: 101}}, nil
		},
	}

	svc := draft.NewDraftService(nil, groupRepo, userGroupSvc, dbRepo, driver)

	input := domain.CreateDraftInput{
		DatabaseID:  databaseID,
		UserID:      authorID,
		IsBatch:     true,
		BatchColumn: "non_existent_column",
		BatchSize:   500,
		SqlUp:       "UPDATE users SET name = 'updated';",
	}

	// Act
	_, err := svc.CreateDraft(ctx, input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput (ErrBatchColumnNotFound), got %v", err)
	}
}

func TestDraftService_CreateDraft_BatchMultipleUnrelatedTables_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{{ID: 1, LeaderID: 101}}, nil
		},
	}
	svc := draft.NewDraftService(nil, groupRepo, userGroupSvc)

	input := domain.CreateDraftInput{
		DatabaseID:  1,
		UserID:      10,
		IsBatch:     true,
		BatchColumn: "id",
		BatchSize:   100,
		SqlUp:       "UPDATE users SET active = true; UPDATE orders SET status = 'complete';",
	}

	// Act
	_, err := svc.CreateDraft(ctx, input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput (ErrBatchMultipleUnrelatedTables), got %v", err)
	}
}

func TestDraftService_CreateDraft_ValidBatch_CreatesDraftSuccessfully(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(10)
	databaseID := uint(1)

	userGroupSvc := &testutil.UserGroupService{
		AuthorizeFunc: func(ctx context.Context, uID uint, dbID uint, reqAccess domain.AccessType) error {
			return nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{{ID: 1, LeaderID: 101}}, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: databaseID, Host: "localhost"}, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		GetDatabaseMetadataFunc: func(ctx context.Context, db domain.Database) (*ports.DatabaseMetadata, error) {
			return &ports.DatabaseMetadata{
				Columns: []ports.DatabaseColumnMetadata{
					{Table: "users", Name: "id", DataType: "bigint"},
					{Table: "users", Name: "active", DataType: "boolean"},
				},
			}, nil
		},
	}

	var savedDraft *domain.Draft
	draftRepo := &testutil.DraftRepository{
		SaveFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			d.ID = 1
			savedDraft = d
			return d, nil
		},
		SaveApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			app.ID = 10
			return app, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, groupRepo, userGroupSvc, dbRepo, driver)

	input := domain.CreateDraftInput{
		DatabaseID:  databaseID,
		UserID:      authorID,
		IsBatch:     true,
		BatchColumn: "id",
		BatchSize:   250,
		SqlUp:       "UPDATE users SET active = true;",
	}

	// Act
	d, err := svc.CreateDraft(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !d.IsBatch || d.BatchColumn != "id" || d.BatchSize != 250 {
		t.Fatalf("unexpected draft batch attributes: %+v", d)
	}
	if savedDraft == nil || savedDraft.CurrentRound != 1 {
		t.Fatalf("expected draft initialized with CurrentRound = 1, got %+v", savedDraft)
	}
}

// --- Batch Execution & Quarantine Checks ---

func TestDraftService_ExecuteBatch_DeterminesTargetScopeAndEnqueues(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(5)
	dbID := uint(1)
	authorID := uint(10)
	expectedExecutionID := uint(99)

	draftObj := &domain.Draft{
		ID:          draftID,
		UserID:      authorID,
		DatabaseID:  dbID,
		Status:      domain.DraftStatusApproved,
		IsBatch:     true,
		BatchColumn: "id",
		BatchSize:   500,
		SqlUp:       "UPDATE users SET active = true;",
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
		FindByIdFunc: func(ctx context.Context, id uint) (*domain.Database, error) {
			return &domain.Database{ID: dbID}, nil
		},
		IncrementExecutionIDFunc: func(ctx context.Context, dID uint) (uint, error) {
			return expectedExecutionID, nil
		},
	}
	var enqueuedTask ports.ExecutionTask
	queue := &testutil.TaskQueue{
		EnqueueFunc: func(ctx context.Context, task ports.ExecutionTask) error {
			enqueuedTask = task
			return nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil, dbRepo, queue)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, authorID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if draftObj.Status != domain.DraftStatusExecuting {
		t.Fatalf("expected draft status executing, got %s", draftObj.Status)
	}
	if enqueuedTask.ExecutionID != expectedExecutionID || enqueuedTask.DraftID != draftID {
		t.Fatalf("unexpected enqueued task: %+v", enqueuedTask)
	}
}

func TestDraftService_ExecuteBatch_DatabaseLocked_ReturnsErrDatabaseLocked(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(5)
	dbID := uint(1)
	authorID := uint(10)

	draftObj := &domain.Draft{
		ID:         draftID,
		UserID:     authorID,
		DatabaseID: dbID,
		Status:     domain.DraftStatusApproved,
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		FindDraftByStatusAndDatabaseIDFunc: func(ctx context.Context, dID uint, status string) (*domain.Draft, error) {
			if status == domain.DraftStatusPartial {
				return &domain.Draft{ID: 99, DatabaseID: dID, Status: domain.DraftStatusPartial}, nil
			}
			return nil, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	err := svc.ExecuteDraft(ctx, draftID, authorID)

	// Assert
	if !errors.Is(err, domain.ErrDatabaseLocked) {
		t.Fatalf("expected ErrDatabaseLocked, got %v", err)
	}
}

// --- Partial Recovery Approval Consensus (Resume vs Revert) ---

func TestDraftService_SubmitApproval_PartialPhase_UserNotLeader_ReturnsErrUnauthorized(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	nonLeaderID := uint(999)

	draftObj := &domain.Draft{
		ID:           draftID,
		Status:       domain.DraftStatusPartial,
		CurrentRound: 2,
	}
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{{ID: 1, LeaderID: 101}}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, groupRepo, nil)

	input := domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  nonLeaderID,
		Status:  domain.VoteResume,
	}

	// Act
	_, err := svc.SubmitApproval(ctx, input)

	// Assert
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for non-leader, got %v", err)
	}
}

func TestDraftService_SubmitApproval_PartialPhase_AllLeadersVoteResume_SetsReadyToResume(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	leader1ID := uint(101)
	leader2ID := uint(102)

	draftObj := &domain.Draft{
		ID:           draftID,
		DatabaseID:   1,
		Status:       domain.DraftStatusPartial,
		CurrentRound: 2,
	}

	approvals := []*domain.DraftApproval{
		{ID: 1, DraftID: draftID, GroupID: 1, LeaderID: &leader1ID, Round: 2, Phase: domain.ApprovalPhasePartialRecovery, Status: domain.ApprovalStatusPending},
		{ID: 2, DraftID: draftID, GroupID: 2, LeaderID: &leader2ID, Round: 2, Phase: domain.ApprovalPhasePartialRecovery, Status: domain.ApprovalStatusPending},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		FindApprovalByDraftGroupAndRoundFunc: func(ctx context.Context, dID, gID, round uint) (*domain.DraftApproval, error) {
			for _, a := range approvals {
				if a.GroupID == gID && a.Round == round {
					return a, nil
				}
			}
			return nil, nil
		},
		FindApprovalsByDraftAndRoundFunc: func(ctx context.Context, dID, round uint) ([]*domain.DraftApproval, error) {
			var res []*domain.DraftApproval
			for _, a := range approvals {
				if a.Round == round {
					res = append(res, a)
				}
			}
			return res, nil
		},
		UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			for i, a := range approvals {
				if a.ID == app.ID {
					approvals[i] = app
					return app, nil
				}
			}
			return app, nil
		},
	}

	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{
				{ID: 1, LeaderID: leader1ID},
				{ID: 2, LeaderID: leader2ID},
			}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, groupRepo, nil)

	// Act & Assert - Leader 1 votes resume
	_, err := svc.SubmitApproval(ctx, domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  leader1ID,
		Status:  domain.VoteResume,
	})
	if err != nil {
		t.Fatalf("unexpected error from leader 1 vote: %v", err)
	}
	if draftObj.RecoveryAction != "" {
		t.Fatalf("expected RecoveryAction to not be finalized yet, got %s", draftObj.RecoveryAction)
	}

	// Act & Assert - Leader 2 votes resume
	updated, err := svc.SubmitApproval(ctx, domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  leader2ID,
		Status:  domain.VoteResume,
	})
	if err != nil {
		t.Fatalf("unexpected error from leader 2 vote: %v", err)
	}
	if updated.RecoveryAction != domain.VoteResume {
		t.Fatalf("expected RecoveryAction == resume, got %s", updated.RecoveryAction)
	}
}

func TestDraftService_SubmitApproval_PartialPhase_SingleLeaderVotesRevert_SetsReadyToRevert(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	leader1ID := uint(101)
	leader2ID := uint(102)

	draftObj := &domain.Draft{
		ID:           draftID,
		DatabaseID:   1,
		Status:       domain.DraftStatusPartial,
		CurrentRound: 2,
	}

	approvals := []*domain.DraftApproval{
		{ID: 1, DraftID: draftID, GroupID: 1, LeaderID: &leader1ID, Round: 2, Phase: domain.ApprovalPhasePartialRecovery, Status: domain.ApprovalStatusPending},
		{ID: 2, DraftID: draftID, GroupID: 2, LeaderID: &leader2ID, Round: 2, Phase: domain.ApprovalPhasePartialRecovery, Status: domain.ApprovalStatusPending},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		FindApprovalByDraftGroupAndRoundFunc: func(ctx context.Context, dID, gID, round uint) (*domain.DraftApproval, error) {
			for _, a := range approvals {
				if a.GroupID == gID && a.Round == round {
					return a, nil
				}
			}
			return nil, nil
		},
		FindApprovalsByDraftAndRoundFunc: func(ctx context.Context, dID, round uint) ([]*domain.DraftApproval, error) {
			var res []*domain.DraftApproval
			for _, a := range approvals {
				if a.Round == round {
					res = append(res, a)
				}
			}
			return res, nil
		},
		UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			for i, a := range approvals {
				if a.ID == app.ID {
					approvals[i] = app
					return app, nil
				}
			}
			return app, nil
		},
	}

	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{
				{ID: 1, LeaderID: leader1ID},
				{ID: 2, LeaderID: leader2ID},
			}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, groupRepo, nil)

	// Act
	svc.SubmitApproval(ctx, domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  leader1ID,
		Status:  domain.VoteResume,
	})

	updated, err := svc.SubmitApproval(ctx, domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  leader2ID,
		Status:  domain.VoteRevert,
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error from leader 2 vote: %v", err)
	}
	if updated.RecoveryAction != domain.VoteRevert {
		t.Fatalf("expected RecoveryAction == revert, got %s", updated.RecoveryAction)
	}
}

// --- Resume Execution & Checkpoint Progression ---

func TestDraftService_ResumeBatch_NotAuthor_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftObj := &domain.Draft{
		ID:             1,
		UserID:         10,
		DatabaseID:     1,
		Status:         domain.DraftStatusPartial,
		RecoveryAction: domain.VoteResume,
	}
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
	}
	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	err := svc.ResumeBatch(ctx, 1, 999) // not author

	// Assert
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDraftService_ResumeBatch_PreservesSameExecutionIDAndResumesFromLastCursor(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(10)
	draftID := uint(1)
	dbID := uint(2)
	sameExecutionID := uint(42)

	draftObj := &domain.Draft{
		ID:             draftID,
		UserID:         authorID,
		DatabaseID:     dbID,
		Status:         domain.DraftStatusPartial,
		RecoveryAction: domain.VoteResume,
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
		FindByIdFunc: func(ctx context.Context, id uint) (*domain.Database, error) {
			return &domain.Database{ID: dbID, LatestExecutionID: sameExecutionID}, nil
		},
	}

	var enqueuedTask ports.ExecutionTask
	queue := &testutil.TaskQueue{
		EnqueueFunc: func(ctx context.Context, task ports.ExecutionTask) error {
			enqueuedTask = task
			return nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil, dbRepo, queue)

	// Act
	err := svc.ResumeBatch(ctx, draftID, authorID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if enqueuedTask.ExecutionID != sameExecutionID {
		t.Fatalf("expected same execution ID %d preserved, got %d", sameExecutionID, enqueuedTask.ExecutionID)
	}
	if enqueuedTask.Operation != "resume" {
		t.Fatalf("expected operation 'resume', got %s", enqueuedTask.Operation)
	}
}

// --- Revert Execution & New Execution ID ---

func TestDraftService_RevertBatch_NotAuthor_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftObj := &domain.Draft{
		ID:             1,
		UserID:         10,
		DatabaseID:     1,
		Status:         domain.DraftStatusPartial,
		RecoveryAction: domain.VoteRevert,
	}
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
	}
	svc := draft.NewDraftService(draftRepo, nil, nil)

	// Act
	err := svc.RevertBatch(ctx, 1, 999) // not author

	// Assert
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDraftService_RevertBatch_GeneratesNewExecutionID(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authorID := uint(10)
	draftID := uint(1)
	dbID := uint(2)
	newExecutionID := uint(43)

	draftObj := &domain.Draft{
		ID:             draftID,
		UserID:         authorID,
		DatabaseID:     dbID,
		Status:         domain.DraftStatusPartial,
		RecoveryAction: domain.VoteRevert,
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
		IncrementExecutionIDFunc: func(ctx context.Context, databaseID uint) (uint, error) {
			return newExecutionID, nil
		},
	}

	var enqueuedTask ports.ExecutionTask
	queue := &testutil.TaskQueue{
		EnqueueFunc: func(ctx context.Context, task ports.ExecutionTask) error {
			enqueuedTask = task
			return nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil, dbRepo, queue)

	// Act
	err := svc.RevertBatch(ctx, draftID, authorID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if enqueuedTask.ExecutionID != newExecutionID {
		t.Fatalf("expected new execution ID %d, got %d", newExecutionID, enqueuedTask.ExecutionID)
	}
	if enqueuedTask.Operation != "revert" {
		t.Fatalf("expected operation 'revert', got %s", enqueuedTask.Operation)
	}
}

// --- Revert Failure & Binary Approval Consensus (Ya / Tidak) ---

func TestDraftService_SubmitApproval_RevertFailurePhase_AllLeadersVoteYes_AllowsRetryRevert(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	leader1ID := uint(101)
	leader2ID := uint(102)

	draftObj := &domain.Draft{
		ID:           draftID,
		DatabaseID:   1,
		Status:       domain.DraftStatusPartiallyReverted,
		CurrentRound: 3,
	}

	approvals := []*domain.DraftApproval{
		{ID: 1, DraftID: draftID, GroupID: 1, LeaderID: &leader1ID, Round: 3, Phase: domain.ApprovalPhaseRevertFailure, Status: domain.ApprovalStatusPending},
		{ID: 2, DraftID: draftID, GroupID: 2, LeaderID: &leader2ID, Round: 3, Phase: domain.ApprovalPhaseRevertFailure, Status: domain.ApprovalStatusPending},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		FindApprovalByDraftGroupAndRoundFunc: func(ctx context.Context, dID, gID, round uint) (*domain.DraftApproval, error) {
			for _, a := range approvals {
				if a.GroupID == gID && a.Round == round {
					return a, nil
				}
			}
			return nil, nil
		},
		FindApprovalsByDraftAndRoundFunc: func(ctx context.Context, dID, round uint) ([]*domain.DraftApproval, error) {
			var res []*domain.DraftApproval
			for _, a := range approvals {
				if a.Round == round {
					res = append(res, a)
				}
			}
			return res, nil
		},
		UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			for i, a := range approvals {
				if a.ID == app.ID {
					approvals[i] = app
					return app, nil
				}
			}
			return app, nil
		},
	}

	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{
				{ID: 1, LeaderID: leader1ID},
				{ID: 2, LeaderID: leader2ID},
			}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, groupRepo, nil)

	// Act
	svc.SubmitApproval(ctx, domain.SubmitApprovalInput{DraftID: draftID, UserID: leader1ID, Status: domain.VoteYes})
	updated, err := svc.SubmitApproval(ctx, domain.SubmitApprovalInput{DraftID: draftID, UserID: leader2ID, Status: domain.VoteYes})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.RecoveryAction != domain.VoteRevert {
		t.Fatalf("expected recovery action to remain 'revert' to allow retry, got %s", updated.RecoveryAction)
	}
}

func TestDraftService_SubmitApproval_RevertFailurePhase_SingleLeaderVotesNo_AbortsRevertLeavesDBStateMarksFailedAndRecordsFailMigration(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	leader1ID := uint(101)
	leader2ID := uint(102)

	draftObj := &domain.Draft{
		ID:           draftID,
		DatabaseID:   1,
		Status:       domain.DraftStatusPartiallyReverted,
		CurrentRound: 3,
	}

	approvals := []*domain.DraftApproval{
		{ID: 1, DraftID: draftID, GroupID: 1, LeaderID: &leader1ID, Round: 3, Phase: domain.ApprovalPhaseRevertFailure, Status: domain.ApprovalStatusPending},
		{ID: 2, DraftID: draftID, GroupID: 2, LeaderID: &leader2ID, Round: 3, Phase: domain.ApprovalPhaseRevertFailure, Status: domain.ApprovalStatusPending},
	}

	var recordedMigration *domain.Migration
	migrationRepo := &testutil.MigrationRepository{
		SaveFunc: func(ctx context.Context, m *domain.Migration) (*domain.Migration, error) {
			recordedMigration = m
			return m, nil
		},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		UpdateFunc: func(ctx context.Context, d *domain.Draft) (*domain.Draft, error) {
			draftObj = d
			return draftObj, nil
		},
		FindApprovalByDraftGroupAndRoundFunc: func(ctx context.Context, dID, gID, round uint) (*domain.DraftApproval, error) {
			for _, a := range approvals {
				if a.GroupID == gID && a.Round == round {
					return a, nil
				}
			}
			return nil, nil
		},
		FindApprovalsByDraftAndRoundFunc: func(ctx context.Context, dID, round uint) ([]*domain.DraftApproval, error) {
			var res []*domain.DraftApproval
			for _, a := range approvals {
				if a.Round == round {
					res = append(res, a)
				}
			}
			return res, nil
		},
		UpdateApprovalFunc: func(ctx context.Context, app *domain.DraftApproval) (*domain.DraftApproval, error) {
			for i, a := range approvals {
				if a.ID == app.ID {
					approvals[i] = app
					return app, nil
				}
			}
			return app, nil
		},
	}

	groupRepo := &testutil.GroupRepository{
		FindAllByDatabaseIDFunc: func(ctx context.Context, dbID uint) ([]domain.Group, error) {
			return []domain.Group{
				{ID: 1, LeaderID: leader1ID},
				{ID: 2, LeaderID: leader2ID},
			}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, groupRepo, nil, migrationRepo)

	// Act
	svc.SubmitApproval(ctx, domain.SubmitApprovalInput{DraftID: draftID, UserID: leader1ID, Status: domain.VoteYes})
	updated, err := svc.SubmitApproval(ctx, domain.SubmitApprovalInput{DraftID: draftID, UserID: leader2ID, Status: domain.VoteNo})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != domain.DraftStatusFailed {
		t.Fatalf("expected draft status failed, got %s", updated.Status)
	}
	if recordedMigration == nil || recordedMigration.Status != domain.MigrationStatusFail {
		t.Fatalf("expected recorded fail migration, got %+v", recordedMigration)
	}
	if !strings.Contains(recordedMigration.ErrorMessage, "[REVERT_ABORTED_BY_LEADER]") {
		t.Fatalf("expected error message to contain [REVERT_ABORTED_BY_LEADER], got %q", recordedMigration.ErrorMessage)
	}
}

// --- Progress Tracking Calculation ---

func TestDraftService_GetProgress_ZeroChunksCompleted_ReturnsZeroPercentage(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)

	draftObj := &domain.Draft{
		ID:          draftID,
		UserID:      authorID,
		DatabaseID:  1,
		IsBatch:     true,
		BatchColumn: "id",
		BatchSize:   100,
		SqlUp:       "UPDATE users SET active = true;",
		Status:      domain.DraftStatusExecuting,
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		FindCheckpointsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftCheckpoint, error) {
			return []*domain.DraftCheckpoint{}, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 1}, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteQueryFunc: func(ctx context.Context, db domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return &ports.DatabaseQueryResult{
				Rows: []map[string]any{{"count": int64(500)}},
			}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil, dbRepo, driver)

	// Act
	progress, err := svc.GetBatchProgress(ctx, draftID, authorID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if progress.Percentage != 0.0 || progress.RowsProcessed != 0 || progress.TotalRows != 500 {
		t.Fatalf("unexpected progress: %+v", progress)
	}
}

func TestDraftService_GetProgress_MidwayProgress_CalculatesPercentageAndRowsAccurately(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uint(1)
	authorID := uint(10)

	draftObj := &domain.Draft{
		ID:          draftID,
		UserID:      authorID,
		DatabaseID:  1,
		IsBatch:     true,
		BatchColumn: "id",
		BatchSize:   100,
		SqlUp:       "UPDATE users SET active = true;",
		Status:      domain.DraftStatusExecuting,
	}

	checkpoints := []*domain.DraftCheckpoint{
		{ID: 1, DraftID: draftID, BatchNumber: 1, Status: "committed", RowsAffected: 100},
		{ID: 2, DraftID: draftID, BatchNumber: 2, Status: "committed", RowsAffected: 100},
		{ID: 3, DraftID: draftID, BatchNumber: 3, Status: "committed", RowsAffected: 100},
	}

	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(ctx context.Context, id uint) (*domain.Draft, error) {
			return draftObj, nil
		},
		FindCheckpointsByDraftIDFunc: func(ctx context.Context, dID uint) ([]*domain.DraftCheckpoint, error) {
			return checkpoints, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(ctx context.Context, id uint) (domain.Database, error) {
			return domain.Database{ID: 1}, nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteQueryFunc: func(ctx context.Context, db domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return &ports.DatabaseQueryResult{
				Rows: []map[string]any{{"count": int64(600)}},
			}, nil
		},
	}

	svc := draft.NewDraftService(draftRepo, nil, nil, dbRepo, driver)

	// Act
	progress, err := svc.GetBatchProgress(ctx, draftID, authorID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if progress.Percentage != 50.0 || progress.RowsProcessed != 300 || progress.TotalRows != 600 || progress.CurrentBatch != 3 {
		t.Fatalf("unexpected progress calculation: %+v", progress)
	}
}
