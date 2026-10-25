package database

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

func newDatabaseTestContext(method string, path string, body []byte, params gin.Params, userID *uint) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestGetDatabaseByIDReturnsDatabaseWithoutPassword(t *testing.T) {
	service := &testutil.DatabaseService{
		GetDatabaseByIDFunc: func(_ context.Context, id uint) (*domain.Database, error) {
			if id != 6 {
				t.Fatalf("id=%d, want 6", id)
			}
			return &domain.Database{ID: 6, Host: "db", Port: 5432, User: "postgres", Password: "secret", Type: "postgresql", Name: "app"}, nil
		},
	}
	c, w := newDatabaseTestContext(http.MethodGet, "/databases/6", nil, gin.Params{{Key: "id", Value: "6"}}, nil)
	NewDatabaseHandler(service).GetDatabaseByID(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte("secret")) || bytes.Contains(w.Body.Bytes(), []byte("password")) {
		t.Fatalf("password leaked: %s", w.Body.String())
	}
}

func TestGetDatabaseByIDRejectsInvalidID(t *testing.T) {
	c, w := newDatabaseTestContext(http.MethodGet, "/databases/nope", nil, gin.Params{{Key: "id", Value: "nope"}}, nil)
	NewDatabaseHandler(&testutil.DatabaseService{}).GetDatabaseByID(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", w.Code)
	}
}

func TestGetAllDatabasesReturnsPagedResult(t *testing.T) {
	service := &testutil.DatabaseService{
		GetAllDatabasesFunc: func(_ context.Context, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
			if params.Page != 2 || params.Limit != 2 {
				t.Fatalf("pagination=%+v", params)
			}
			return []*domain.Database{{ID: 3, Host: "db", Port: 5432, User: "postgres", Type: "postgresql", Name: "one"}}, 5, nil
		},
	}
	c, w := newDatabaseTestContext(http.MethodGet, "/databases?page=2&limit=2", nil, nil, nil)
	NewDatabaseHandler(service).GetAllDatabases(c)
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

func TestGetAllDatabasesRejectsInvalidPagination(t *testing.T) {
	called := false
	service := &testutil.DatabaseService{
		GetAllDatabasesFunc: func(context.Context, pagination.PaginationParams) ([]*domain.Database, int64, error) {
			called = true
			return nil, 0, nil
		},
	}
	c, w := newDatabaseTestContext(http.MethodGet, "/databases?page=x", nil, nil, nil)
	NewDatabaseHandler(service).GetAllDatabases(c)
	if w.Code != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%v body=%s", w.Code, called, w.Body.String())
	}
}

func TestGetDatabasesByUserIDUsesAuthenticatedUser(t *testing.T) {
	userID := uint(9)
	service := &testutil.DatabaseService{
		GetDatabasesByUserIDFunc: func(_ context.Context, gotUserID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
			if gotUserID != 9 || params.Page != 1 || params.Limit != 10 {
				t.Fatalf("userID=%d params=%+v", gotUserID, params)
			}
			return []*domain.Database{{ID: 2, Name: "allowed"}}, 1, nil
		},
	}
	c, w := newDatabaseTestContext(http.MethodGet, "/me/databases", nil, nil, &userID)
	NewDatabaseHandler(service).GetDatabasesByUserID(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestGetDatabasesByUserIDRejectsMissingUser(t *testing.T) {
	c, w := newDatabaseTestContext(http.MethodGet, "/me/databases", nil, nil, nil)
	NewDatabaseHandler(&testutil.DatabaseService{}).GetDatabasesByUserID(c)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d, want 501", w.Code)
	}
}

func TestGetDatabasesByUserIDRejectsInvalidPagination(t *testing.T) {
	userID := uint(9)
	c, w := newDatabaseTestContext(http.MethodGet, "/me/databases?page=invalid", nil, nil, &userID)
	NewDatabaseHandler(&testutil.DatabaseService{}).GetDatabasesByUserID(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400 for invalid pagination", w.Code)
	}
}

func TestGetDatabaseByIDAndUserIDDelegatesBothIDs(t *testing.T) {
	userID := uint(12)
	service := &testutil.DatabaseService{
		GetDatabaseByIDAndUserIDFunc: func(_ context.Context, databaseID uint, gotUserID uint) (*domain.Database, error) {
			if databaseID != 4 || gotUserID != 12 {
				t.Fatalf("databaseID=%d userID=%d", databaseID, gotUserID)
			}
			return &domain.Database{ID: 4, Name: "target"}, nil
		},
	}
	c, w := newDatabaseTestContext(http.MethodGet, "/me/databases/4", nil, gin.Params{{Key: "database_id", Value: "4"}}, &userID)
	NewDatabaseHandler(service).GetDatabaseByIDAndUserID(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateDatabaseMapsRequest(t *testing.T) {
	service := &testutil.DatabaseService{
		CreateDatabaseFunc: func(_ context.Context, input *ports.DatabaseInput) (*domain.Database, error) {
			if input.Host != "localhost" || input.Port != 5432 || input.User != "postgres" || input.Password != "secret" || input.Type != "postgresql" || input.Name != "app" {
				t.Fatalf("unexpected input: %+v", input)
			}
			return &domain.Database{ID: 14, Host: input.Host, Port: input.Port, User: input.User, Type: input.Type, Name: input.Name}, nil
		},
	}
	body := []byte(`{"host":"localhost","port":5432,"user":"postgres","password":"secret","type":"postgresql","name":"app"}`)
	c, w := newDatabaseTestContext(http.MethodPost, "/databases", body, nil, nil)
	NewDatabaseHandler(service).CreateDatabase(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte("secret")) {
		t.Fatalf("password leaked: %s", w.Body.String())
	}
}

func TestCreateDatabaseRejectsBindingErrors(t *testing.T) {
	called := false
	service := &testutil.DatabaseService{
		CreateDatabaseFunc: func(context.Context, *ports.DatabaseInput) (*domain.Database, error) {
			called = true
			return nil, nil
		},
	}
	body := []byte(`{"host":"localhost","port":70000,"user":"postgres","password":"secret","type":"postgresql","name":"app"}`)
	c, w := newDatabaseTestContext(http.MethodPost, "/databases", body, nil, nil)
	NewDatabaseHandler(service).CreateDatabase(c)
	if w.Code != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%v body=%s", w.Code, called, w.Body.String())
	}
}

func TestDeleteDatabaseDelegates(t *testing.T) {
	deleted := uint(0)
	service := &testutil.DatabaseService{
		DeleteDatabaseFunc: func(_ context.Context, id uint) error {
			deleted = id
			return nil
		},
	}
	c, w := newDatabaseTestContext(http.MethodDelete, "/databases/8", nil, gin.Params{{Key: "id", Value: "8"}}, nil)
	NewDatabaseHandler(service).DeleteDatabase(c)
	if w.Code != http.StatusOK || deleted != 8 {
		t.Fatalf("status=%d deleted=%d", w.Code, deleted)
	}
}

func TestDatabaseErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: domain.ErrDatabaseNotFound, want: http.StatusNotFound},
		{name: "invalid id", err: domain.ErrInvalidDatabaseID, want: http.StatusBadRequest},
		{name: "invalid user", err: domain.ErrInvalidUserID, want: http.StatusBadRequest},
		{name: "invalid pagination", err: domain.ErrInvalidPagination, want: http.StatusBadRequest},
		{name: "invalid database", err: domain.ErrInvalidDatabaseInput, want: http.StatusBadRequest},
		{name: "invalid input", err: domain.ErrInvalidInput, want: http.StatusBadRequest},
		{name: "duplicate", err: domain.ErrDatabaseAlreadyExists, want: http.StatusConflict},
		{name: "in use", err: domain.ErrDatabaseInUse, want: http.StatusConflict},
		{name: "missing user context", err: domain.ErrCurrentUserUnavailable, want: http.StatusNotImplemented},
		{name: "unknown", err: errors.New("boom"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newDatabaseTestContext(http.MethodGet, "/", nil, nil, nil)
			writeDatabaseErrorResponse(c, tt.err)
			if w.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestGetDatabaseByIDPropagatesServiceError(t *testing.T) {
	service := &testutil.DatabaseService{
		GetDatabaseByIDFunc: func(context.Context, uint) (*domain.Database, error) {
			return nil, domain.ErrDatabaseNotFound
		},
	}
	c, w := newDatabaseTestContext(http.MethodGet, "/databases/9", nil, gin.Params{{Key: "id", Value: "9"}}, nil)
	NewDatabaseHandler(service).GetDatabaseByID(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", w.Code)
	}
}

func TestGetAllDatabasesPropagatesServiceError(t *testing.T) {
	service := &testutil.DatabaseService{
		GetAllDatabasesFunc: func(context.Context, pagination.PaginationParams) ([]*domain.Database, int64, error) {
			return nil, 0, errors.New("db query failed")
		},
	}
	c, w := newDatabaseTestContext(http.MethodGet, "/databases", nil, nil, nil)
	NewDatabaseHandler(service).GetAllDatabases(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", w.Code)
	}
}

func TestGetDatabasesByUserIDPropagatesServiceError(t *testing.T) {
	userID := uint(5)
	service := &testutil.DatabaseService{
		GetDatabasesByUserIDFunc: func(context.Context, uint, pagination.PaginationParams) ([]*domain.Database, int64, error) {
			return nil, 0, errors.New("service failure")
		},
	}
	c, w := newDatabaseTestContext(http.MethodGet, "/me/databases", nil, nil, &userID)
	NewDatabaseHandler(service).GetDatabasesByUserID(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", w.Code)
	}
}

func TestGetDatabaseByIDAndUserIDRejectsInvalidIDAndMissingUser(t *testing.T) {
	service := &testutil.DatabaseService{}
	userID := uint(5)

	// Invalid ID
	c1, w1 := newDatabaseTestContext(http.MethodGet, "/me/databases/bad", nil, gin.Params{{Key: "database_id", Value: "bad"}}, &userID)
	NewDatabaseHandler(service).GetDatabaseByIDAndUserID(c1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400 for bad id", w1.Code)
	}

	// Missing user context
	c2, w2 := newDatabaseTestContext(http.MethodGet, "/me/databases/1", nil, gin.Params{{Key: "database_id", Value: "1"}}, nil)
	NewDatabaseHandler(service).GetDatabaseByIDAndUserID(c2)
	if w2.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d, want 501 for missing user context", w2.Code)
	}
}

func TestGetDatabaseByIDAndUserIDPropagatesServiceError(t *testing.T) {
	userID := uint(10)
	service := &testutil.DatabaseService{
		GetDatabaseByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
			return nil, domain.ErrDatabaseNotFound
		},
	}
	c, w := newDatabaseTestContext(http.MethodGet, "/me/databases/10", nil, gin.Params{{Key: "database_id", Value: "10"}}, &userID)
	NewDatabaseHandler(service).GetDatabaseByIDAndUserID(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", w.Code)
	}
}

func TestCreateDatabasePropagatesServiceError(t *testing.T) {
	service := &testutil.DatabaseService{
		CreateDatabaseFunc: func(context.Context, *ports.DatabaseInput) (*domain.Database, error) {
			return nil, domain.ErrDatabaseAlreadyExists
		},
	}
	body := []byte(`{"host":"localhost","port":5432,"user":"postgres","password":"secret","type":"postgresql","name":"app"}`)
	c, w := newDatabaseTestContext(http.MethodPost, "/databases", body, nil, nil)
	NewDatabaseHandler(service).CreateDatabase(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d, want 409", w.Code)
	}
}

func TestDeleteDatabaseRejectsInvalidIDAndPropagatesServiceError(t *testing.T) {
	service := &testutil.DatabaseService{
		DeleteDatabaseFunc: func(context.Context, uint) error {
			return domain.ErrDatabaseInUse
		},
	}

	// Invalid ID
	c1, w1 := newDatabaseTestContext(http.MethodDelete, "/databases/invalid", nil, gin.Params{{Key: "id", Value: "invalid"}}, nil)
	NewDatabaseHandler(service).DeleteDatabase(c1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", w1.Code)
	}

	// In use error (409)
	c2, w2 := newDatabaseTestContext(http.MethodDelete, "/databases/8", nil, gin.Params{{Key: "id", Value: "8"}}, nil)
	NewDatabaseHandler(service).DeleteDatabase(c2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("status=%d, want 409", w2.Code)
	}
}

func TestDatabaseErrorMappingConnectionErrors(t *testing.T) {
	c1, w1 := newDatabaseTestContext(http.MethodGet, "/", nil, nil, nil)
	writeDatabaseErrorResponse(c1, errors.New("dial tcp 127.0.0.1:5432: connect: connection refused"))
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400 for connection refused", w1.Code)
	}

	c2, w2 := newDatabaseTestContext(http.MethodGet, "/", nil, nil, nil)
	writeDatabaseErrorResponse(c2, nil)
	if w2.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500 for nil error", w2.Code)
	}
}
