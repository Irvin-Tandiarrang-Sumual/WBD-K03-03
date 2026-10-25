package migration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func newMigrationTestContext(method string, path string, body []byte, params gin.Params, userID *uint) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Params = params
	if body != nil {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	if userID != nil {
		middleware.SetAuthenticatedUserID(c, *userID)
	}
	return c, w
}

func TestGetMigrationsByDatabaseIDReturnsPagedData(t *testing.T) {
	service := &testutil.MigrationService{
		GetMigrationsByDatabaseIDFunc: func(_ context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error) {
			if databaseID != 5 || params.Page != 2 || params.Limit != 1 {
				t.Fatalf("databaseID=%d params=%+v", databaseID, params)
			}
			return []*domain.Migration{{ID: 8, DatabaseID: 5, Version: "2", Status: "success", SqlUp: "CREATE TABLE x(id int)", SqlDown: "DROP TABLE x", UserID: 3}}, 3, nil
		},
	}
	c, w := newMigrationTestContext(http.MethodGet, "/databases/5/migrations?page=2&limit=1", nil, gin.Params{{Key: "database_id", Value: "5"}}, nil)
	NewMigrationHandler(service).GetMigrationsByDatabaseID(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["paging"].(map[string]any)["total_page"] != float64(3) {
		t.Fatalf("unexpected paging=%v", body["paging"])
	}
}

func TestGetMigrationsRejectsInvalidParameters(t *testing.T) {
	handler := NewMigrationHandler(&testutil.MigrationService{})

	c, w := newMigrationTestContext(http.MethodGet, "/databases/x/migrations", nil, gin.Params{{Key: "database_id", Value: "x"}}, nil)
	handler.GetMigrationsByDatabaseID(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid database status=%d", w.Code)
	}

	c, w = newMigrationTestContext(http.MethodGet, "/databases/1/migrations?page=x", nil, gin.Params{{Key: "database_id", Value: "1"}}, nil)
	handler.GetMigrationsByDatabaseID(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid pagination status=%d", w.Code)
	}
}

func TestCreateMigrationUsesAuthenticatedUser(t *testing.T) {
	userID := uint(7)
	service := &testutil.MigrationService{
		CreateMigrationFunc: func(_ context.Context, databaseID uint, gotUserID uint, sqlUp string) (*domain.Migration, error) {
			if databaseID != 4 || gotUserID != 7 || sqlUp != "CREATE TABLE audit(id int)" {
				t.Fatalf("databaseID=%d userID=%d sql=%q", databaseID, gotUserID, sqlUp)
			}
			return &domain.Migration{ID: 1, DatabaseID: 4, Version: "1", Status: "success", SqlUp: sqlUp, SqlDown: "DROP TABLE audit", UserID: 7}, nil
		},
	}
	c, w := newMigrationTestContext(http.MethodPost, "/databases/4/migrations", []byte(`{"sql_up":"CREATE TABLE audit(id int)"}`), gin.Params{{Key: "database_id", Value: "4"}}, &userID)
	NewMigrationHandler(service).CreateMigration(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateMigrationRejectsMissingUserAndInvalidJSON(t *testing.T) {
	handler := NewMigrationHandler(&testutil.MigrationService{})

	c, w := newMigrationTestContext(http.MethodPost, "/databases/4/migrations", []byte(`{"sql_up":"CREATE TABLE x(id int)"}`), gin.Params{{Key: "database_id", Value: "4"}}, nil)
	handler.CreateMigration(c)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("missing user status=%d, want 501", w.Code)
	}

	userID := uint(2)
	c, w = newMigrationTestContext(http.MethodPost, "/databases/4/migrations", []byte(`{"sql_up":`), gin.Params{{Key: "database_id", Value: "4"}}, &userID)
	handler.CreateMigration(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON status=%d", w.Code)
	}
}

func TestGetMigrationByIDAndDatabaseID(t *testing.T) {
	service := &testutil.MigrationService{
		GetMigrationByIDAndDatabaseIDFunc: func(_ context.Context, migrationID uint, databaseID uint) (*domain.Migration, int64, error) {
			if migrationID != 3 || databaseID != 8 {
				t.Fatalf("migrationID=%d databaseID=%d", migrationID, databaseID)
			}
			return &domain.Migration{ID: 3, DatabaseID: 8, Version: "2"}, 1, nil
		},
	}
	params := gin.Params{{Key: "database_id", Value: "8"}, {Key: "migration_id", Value: "3"}}
	c, w := newMigrationTestContext(http.MethodGet, "/databases/8/migrations/3", nil, params, nil)
	NewMigrationHandler(service).GetMigrationByIDAndDatabaseID(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestGetMigrationRejectsInvalidIDs(t *testing.T) {
	handler := NewMigrationHandler(&testutil.MigrationService{})

	params := gin.Params{{Key: "database_id", Value: "8"}, {Key: "migration_id", Value: "x"}}
	c, w := newMigrationTestContext(http.MethodGet, "/", nil, params, nil)
	handler.GetMigrationByIDAndDatabaseID(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid migration status=%d", w.Code)
	}

	params = gin.Params{{Key: "database_id", Value: "x"}, {Key: "migration_id", Value: "1"}}
	c, w = newMigrationTestContext(http.MethodGet, "/", nil, params, nil)
	handler.GetMigrationByIDAndDatabaseID(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid database status=%d", w.Code)
	}
}

func TestCompareMigrationsParsesQueryAndUser(t *testing.T) {
	userID := uint(11)
	service := &testutil.MigrationService{
		CompareMigrationsFunc: func(_ context.Context, databaseID uint, gotUserID uint, first uint, second uint) (*ports.MigrationComparison, error) {
			if databaseID != 6 || gotUserID != 11 || first != 2 || second != 4 {
				t.Fatalf("databaseID=%d userID=%d first=%d second=%d", databaseID, gotUserID, first, second)
			}
			return &ports.MigrationComparison{
				FirstMigration:  &domain.Migration{ID: 2, DatabaseID: 6, Version: "1"},
				SecondMigration: &domain.Migration{ID: 4, DatabaseID: 6, Version: "2"},
				SQLUpEqual:      false,
				SQLDownEqual:    true,
			}, nil
		},
	}
	c, w := newMigrationTestContext(http.MethodGet, "/databases/6/migrations/compare?migration_id_1=2&migration_id_2=4", nil, gin.Params{{Key: "database_id", Value: "6"}}, &userID)
	NewMigrationHandler(service).CompareMigrations(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"sql_down_equal":true`)) {
		t.Fatalf("unexpected body=%s", w.Body.String())
	}
}

func TestCompareMigrationsRejectsInvalidQueryAndMissingUser(t *testing.T) {
	handler := NewMigrationHandler(&testutil.MigrationService{})
	userID := uint(1)

	c, w := newMigrationTestContext(http.MethodGet, "/databases/6/migrations/compare?migration_id_1=x&migration_id_2=4", nil, gin.Params{{Key: "database_id", Value: "6"}}, &userID)
	handler.CompareMigrations(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid comparison status=%d", w.Code)
	}

	c, w = newMigrationTestContext(http.MethodGet, "/databases/6/migrations/compare?migration_id_1=2&migration_id_2=4", nil, gin.Params{{Key: "database_id", Value: "6"}}, nil)
	handler.CompareMigrations(c)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("missing user status=%d", w.Code)
	}
}

func TestRollbackMigrationsMapsRequest(t *testing.T) {
	userID := uint(5)
	service := &testutil.MigrationService{
		RollbackMigrationsFunc: func(_ context.Context, databaseID uint, gotUserID uint, targetID uint) ([]*domain.Migration, error) {
			if databaseID != 9 || gotUserID != 5 || targetID != 2 {
				t.Fatalf("databaseID=%d userID=%d target=%d", databaseID, gotUserID, targetID)
			}
			return []*domain.Migration{{ID: 3, DatabaseID: 9, Version: "3"}, {ID: 4, DatabaseID: 9, Version: "4"}}, nil
		},
	}
	body := []byte(`{"database_id":9,"target_migration_id":2}`)
	c, w := newMigrationTestContext(http.MethodPost, "/migrations/rollback", body, nil, &userID)
	NewMigrationHandler(service).RollbackMigrations(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestRollbackRejectsInvalidJSONAndMissingUser(t *testing.T) {
	handler := NewMigrationHandler(&testutil.MigrationService{})

	c, w := newMigrationTestContext(http.MethodPost, "/migrations/rollback", []byte(`{"database_id":1,"target_migration_id":1}`), nil, nil)
	handler.RollbackMigrations(c)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("missing user status=%d", w.Code)
	}

	userID := uint(1)
	c, w = newMigrationTestContext(http.MethodPost, "/migrations/rollback", []byte(`{"database_id":`), nil, &userID)
	handler.RollbackMigrations(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON status=%d", w.Code)
	}
}

func TestMigrationErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: domain.ErrMigrationNotFound, want: http.StatusNotFound},
		{name: "invalid database", err: domain.ErrInvalidDatabaseID, want: http.StatusBadRequest},
		{name: "invalid migration", err: domain.ErrInvalidMigrationInput, want: http.StatusBadRequest},
		{name: "invalid pagination", err: domain.ErrInvalidPagination, want: http.StatusBadRequest},
		{name: "invalid comparison", err: domain.ErrInvalidMigrationComparison, want: http.StatusBadRequest},
		{name: "rollback unavailable", err: domain.ErrMigrationRollbackUnavailable, want: http.StatusConflict},
		{name: "version unavailable", err: domain.ErrMigrationVersionUnavailable, want: http.StatusConflict},
		{name: "access denied", err: domain.ErrDatabaseAccessDenied, want: http.StatusForbidden},
		{name: "missing user", err: domain.ErrCurrentUserUnavailable, want: http.StatusNotImplemented},
		{name: "unknown", err: errors.New("boom"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newMigrationTestContext(http.MethodGet, "/", nil, nil, nil)
			writeErrorResponse(c, tt.err)
			if w.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}
