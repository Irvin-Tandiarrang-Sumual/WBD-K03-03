package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func setupTestDraftRouter(draftSvc *testutil.DraftService, currentUserID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())

	// Middleware to inject current authenticated user
	engine.Use(func(c *gin.Context) {
		if currentUserID != 0 {
			middleware.SetUserContext(c, currentUserID, domain.RoleUser)
		}
		c.Next()
	})

	handler := handlers.NewDraftHandler(draftSvc)
	engine.POST("/databases/:id/drafts", handler.CreateDraft)
	engine.POST("/me/databases/:database_id/drafts", handler.CreateDraft)
	engine.GET("/databases/:id/drafts", handler.GetDraftsByDatabaseID)
	engine.GET("/me/databases/:database_id/drafts", handler.GetDraftsByDatabaseID)
	engine.GET("/drafts/:id", handler.GetDraftByID)
	engine.POST("/drafts/:id/approvals", handler.SubmitApproval)

	return engine
}

func TestDraftHandler_CreateDraft_ValidInput_ReturnsCreated(t *testing.T) {
	// Arrange
	userID := uint(10)
	databaseID := uint(1)
	leaderID := uint(101)
	expectedDraft := &domain.Draft{
		ID:         1,
		DatabaseID: databaseID,
		UserID:     userID,
		SqlUp:      "ALTER TABLE users ADD COLUMN name TEXT;",
		Status:     domain.DraftStatusPendingApproval,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Approvals: []domain.DraftApproval{
			{ID: 1, DraftID: 1, GroupID: 1, LeaderID: &leaderID, Status: domain.ApprovalStatusPending},
		},
	}

	draftSvc := &testutil.DraftService{
		CreateDraftFunc: func(ctx context.Context, input domain.CreateDraftInput) (*domain.Draft, error) {
			return expectedDraft, nil
		},
	}

	engine := setupTestDraftRouter(draftSvc, userID)

	bodyBytes, _ := json.Marshal(handlers.CreateDraftRequest{
		SqlUp: "ALTER TABLE users ADD COLUMN name TEXT;",
	})
	req := httptest.NewRequest(http.MethodPost, "/me/databases/1/drafts", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDraftHandler_CreateDraft_ErrorScenarios_ReturnsExpectedStatusCode(t *testing.T) {
	testCases := []struct {
		name       string
		path       string
		userID     uint
		body       string
		svcErr     error
		wantStatus int
	}{
		{
			name:       "invalid database id",
			path:       "/databases/abc/drafts",
			userID:     10,
			body:       `{"sqlUp":"SELECT 1;"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthenticated user",
			path:       "/databases/1/drafts",
			userID:     0,
			body:       `{"sqlUp":"SELECT 1;"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid json body",
			path:       "/databases/1/drafts",
			userID:     10,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "forbidden access from service",
			path:       "/databases/1/drafts",
			userID:     10,
			body:       `{"sqlUp":"ALTER TABLE t ADD COLUMN c INT;"}`,
			svcErr:     domain.ErrForbidden,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			draftSvc := &testutil.DraftService{
				CreateDraftFunc: func(ctx context.Context, input domain.CreateDraftInput) (*domain.Draft, error) {
					return nil, tc.svcErr
				},
			}
			engine := setupTestDraftRouter(draftSvc, tc.userID)
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Act
			engine.ServeHTTP(rec, req)

			// Assert
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestDraftHandler_GetDraftsByDatabaseID_ValidRequest_ReturnsOK(t *testing.T) {
	// Arrange
	userID := uint(10)
	drafts := []*domain.Draft{
		{ID: 1, DatabaseID: 1, Status: domain.DraftStatusApproved},
	}

	draftSvc := &testutil.DraftService{
		GetDraftsByDatabaseIDFunc: func(ctx context.Context, dbID uint, uID uint, params pagination.PaginationParams) ([]*domain.Draft, int64, error) {
			return drafts, 1, nil
		},
	}

	engine := setupTestDraftRouter(draftSvc, userID)
	req := httptest.NewRequest(http.MethodGet, "/me/databases/1/drafts", nil)
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestDraftHandler_GetDraftsByDatabaseID_ErrorScenarios_ReturnsExpectedStatusCode(t *testing.T) {
	testCases := []struct {
		name       string
		path       string
		userID     uint
		svcErr     error
		wantStatus int
	}{
		{
			name:       "invalid database id",
			path:       "/databases/abc/drafts",
			userID:     10,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthenticated user",
			path:       "/databases/1/drafts",
			userID:     0,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid pagination query",
			path:       "/databases/1/drafts?page=invalid",
			userID:     10,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "forbidden from service",
			path:       "/databases/1/drafts",
			userID:     10,
			svcErr:     domain.ErrForbidden,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			draftSvc := &testutil.DraftService{
				GetDraftsByDatabaseIDFunc: func(ctx context.Context, dbID uint, uID uint, params pagination.PaginationParams) ([]*domain.Draft, int64, error) {
					return nil, 0, tc.svcErr
				},
			}
			engine := setupTestDraftRouter(draftSvc, tc.userID)
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			// Act
			engine.ServeHTTP(rec, req)

			// Assert
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestDraftHandler_GetDraftByID_ValidID_ReturnsOK(t *testing.T) {
	// Arrange
	userID := uint(10)
	d := &domain.Draft{
		ID:         1,
		DatabaseID: 1,
		Status:     domain.DraftStatusPendingApproval,
	}

	draftSvc := &testutil.DraftService{
		GetDraftByIDFunc: func(ctx context.Context, draftID uint, uID uint) (*domain.Draft, error) {
			return d, nil
		},
	}

	engine := setupTestDraftRouter(draftSvc, userID)
	req := httptest.NewRequest(http.MethodGet, "/drafts/1", nil)
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestDraftHandler_GetDraftByID_ErrorScenarios_ReturnsExpectedStatusCode(t *testing.T) {
	testCases := []struct {
		name       string
		path       string
		userID     uint
		svcErr     error
		wantStatus int
	}{
		{
			name:       "invalid draft id",
			path:       "/drafts/abc",
			userID:     10,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthenticated user",
			path:       "/drafts/1",
			userID:     0,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "draft not found",
			path:       "/drafts/99",
			userID:     10,
			svcErr:     domain.ErrDraftNotFound,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			draftSvc := &testutil.DraftService{
				GetDraftByIDFunc: func(ctx context.Context, draftID uint, uID uint) (*domain.Draft, error) {
					return nil, tc.svcErr
				},
			}
			engine := setupTestDraftRouter(draftSvc, tc.userID)
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			// Act
			engine.ServeHTTP(rec, req)

			// Assert
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestDraftHandler_SubmitApproval_ValidVote_ReturnsOK(t *testing.T) {
	// Arrange
	leaderID := uint(101)
	updatedDraft := &domain.Draft{
		ID:     1,
		Status: domain.DraftStatusApproved,
	}

	draftSvc := &testutil.DraftService{
		SubmitApprovalFunc: func(ctx context.Context, input domain.SubmitApprovalInput) (*domain.Draft, error) {
			return updatedDraft, nil
		},
	}

	engine := setupTestDraftRouter(draftSvc, leaderID)

	bodyBytes, _ := json.Marshal(handlers.SubmitApprovalRequest{
		Status: "approved",
	})
	req := httptest.NewRequest(http.MethodPost, "/drafts/1/approvals", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDraftHandler_SubmitApproval_ErrorScenarios_ReturnsExpectedStatusCode(t *testing.T) {
	testCases := []struct {
		name       string
		path       string
		userID     uint
		body       string
		svcErr     error
		wantStatus int
	}{
		{
			name:       "invalid draft id",
			path:       "/drafts/abc/approvals",
			userID:     101,
			body:       `{"status":"approved"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthenticated user",
			path:       "/drafts/1/approvals",
			userID:     0,
			body:       `{"status":"approved"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid json body",
			path:       "/drafts/1/approvals",
			userID:     101,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid status rejected by service",
			path:       "/drafts/1/approvals",
			userID:     101,
			body:       `{"status":"invalid_status"}`,
			svcErr:     domain.ErrInvalidDraftInput,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "conflict invalid draft state",
			path:       "/drafts/1/approvals",
			userID:     101,
			body:       `{"status":"approved"}`,
			svcErr:     domain.ErrInvalidDraftState,
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			draftSvc := &testutil.DraftService{
				SubmitApprovalFunc: func(ctx context.Context, input domain.SubmitApprovalInput) (*domain.Draft, error) {
					return nil, tc.svcErr
				},
			}
			engine := setupTestDraftRouter(draftSvc, tc.userID)
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Act
			engine.ServeHTTP(rec, req)

			// Assert
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}
