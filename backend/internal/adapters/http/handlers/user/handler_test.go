package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func newUserTestContext(method string, path string, body []byte, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Params = params
	if body != nil {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	return c, w
}

func TestListReturnsPagedUsersWithoutPassword(t *testing.T) {
	service := &testutil.UserService{
		ListFunc: func(_ context.Context, params pagination.PaginationParams) ([]domain.User, int64, error) {
			if params.Page != 2 || params.Limit != 1 {
				t.Fatalf("pagination = %+v, want page=2 limit=1", params)
			}
			return []domain.User{{ID: 2, Username: "alice", Password: "must-not-leak", Role: domain.RoleUser}}, 3, nil
		},
	}

	c, w := newUserTestContext(http.MethodGet, "/users?page=2&limit=1", nil, nil)
	NewUserHandler(service).List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].([]any)
	user := data[0].(map[string]any)
	if user["username"] != "alice" {
		t.Fatalf("unexpected response user: %v", user)
	}
	if _, exists := user["password"]; exists {
		t.Fatal("password must not be exposed in response")
	}
	paging := body["paging"].(map[string]any)
	if paging["current_page"] != float64(2) || paging["total_page"] != float64(3) {
		t.Fatalf("unexpected paging: %v", paging)
	}
}

func TestListRejectsInvalidPagination(t *testing.T) {
	called := false
	service := &testutil.UserService{
		ListFunc: func(context.Context, pagination.PaginationParams) ([]domain.User, int64, error) {
			called = true
			return nil, 0, nil
		},
	}

	c, w := newUserTestContext(http.MethodGet, "/users?page=invalid", nil, nil)
	NewUserHandler(service).List(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if called {
		t.Fatal("service must not be called for invalid pagination")
	}
}

func TestGetReturnsUser(t *testing.T) {
	service := &testutil.UserService{
		GetFunc: func(_ context.Context, id uint) (domain.User, error) {
			if id != 7 {
				t.Fatalf("id = %d, want 7", id)
			}
			return domain.User{ID: 7, Username: "bob", Password: "hidden", Role: domain.RoleAdmin}, nil
		},
	}
	c, w := newUserTestContext(http.MethodGet, "/users/7", nil, gin.Params{{Key: "userId", Value: "7"}})
	NewUserHandler(service).Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if bytes.Contains(w.Body.Bytes(), []byte("hidden")) || bytes.Contains(w.Body.Bytes(), []byte("password")) {
		t.Fatalf("password leaked: %s", w.Body.String())
	}
}

func TestUserHandlersRejectInvalidUserID(t *testing.T) {
	handler := NewUserHandler(&testutil.UserService{})

	tests := []struct {
		name string
		call func(*gin.Context)
	}{
		{name: "get", call: handler.Get},
		{name: "update", call: handler.Update},
		{name: "delete", call: handler.Delete},
		{name: "set role", call: handler.SetRole},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newUserTestContext(http.MethodGet, "/users/nope", []byte(`{}`), gin.Params{{Key: "userId", Value: "nope"}})
			tt.call(c)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateMapsRequestToService(t *testing.T) {
	service := &testutil.UserService{
		CreateFunc: func(_ context.Context, input domain.CreateUserInput) (domain.User, error) {
			if input.Username != "charlie" || input.Password != "secret123" || len(input.GroupIDs) != 2 || input.GroupIDs[0] != 3 || input.GroupIDs[1] != 5 {
				t.Fatalf("unexpected input: %+v", input)
			}
			return domain.User{ID: 10, Username: input.Username, Role: domain.RoleUser}, nil
		},
	}
	c, w := newUserTestContext(http.MethodPost, "/users", []byte(`{"username":"charlie","password":"secret123","groupIds":[3,5]}`), nil)
	NewUserHandler(service).Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateRejectsInvalidJSON(t *testing.T) {
	called := false
	service := &testutil.UserService{
		CreateFunc: func(context.Context, domain.CreateUserInput) (domain.User, error) {
			called = true
			return domain.User{}, nil
		},
	}
	c, w := newUserTestContext(http.MethodPost, "/users", []byte(`{"username":`), nil)
	NewUserHandler(service).Create(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if called {
		t.Fatal("service must not be called for invalid JSON")
	}
}

func TestUpdateMapsPatchFields(t *testing.T) {
	service := &testutil.UserService{
		UpdateFunc: func(_ context.Context, id uint, input domain.UpdateUserInput) (domain.User, error) {
			if id != 8 || input.Username == nil || *input.Username != "renamed" || input.Password != nil {
				t.Fatalf("unexpected update: id=%d input=%+v", id, input)
			}
			return domain.User{ID: id, Username: *input.Username, Role: domain.RoleUser}, nil
		},
	}
	c, w := newUserTestContext(http.MethodPatch, "/users/8", []byte(`{"username":"renamed"}`), gin.Params{{Key: "userId", Value: "8"}})
	NewUserHandler(service).Update(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteAndSetRoleDelegateToService(t *testing.T) {
	deletedID := uint(0)
	roleID := uint(0)
	role := ""
	service := &testutil.UserService{
		DeleteFunc: func(_ context.Context, id uint) error {
			deletedID = id
			return nil
		},
		SetRoleFunc: func(_ context.Context, id uint, value string) (domain.User, error) {
			roleID, role = id, value
			return domain.User{ID: id, Username: "admin2", Role: value}, nil
		},
	}
	handler := NewUserHandler(service)

	c, w := newUserTestContext(http.MethodDelete, "/users/12", nil, gin.Params{{Key: "userId", Value: "12"}})
	handler.Delete(c)
	if c.Writer.Status() != http.StatusNoContent || deletedID != 12 {
		t.Fatalf("delete status=%d id=%d", c.Writer.Status(), deletedID)
	}

	c, w = newUserTestContext(http.MethodPatch, "/users/12/role", []byte(`{"role":"admin"}`), gin.Params{{Key: "userId", Value: "12"}})
	handler.SetRole(c)
	if w.Code != http.StatusOK || roleID != 12 || role != domain.RoleAdmin {
		t.Fatalf("set role status=%d id=%d role=%q body=%s", w.Code, roleID, role, w.Body.String())
	}
}

func TestUserErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid input", err: domain.ErrInvalidInput, want: http.StatusBadRequest},
		{name: "invalid pagination", err: domain.ErrInvalidPagination, want: http.StatusBadRequest},
		{name: "invalid role", err: domain.ErrInvalidRole, want: http.StatusBadRequest},
		{name: "invalid groups", err: domain.ErrInvalidGroupsIDs, want: http.StatusBadRequest},
		{name: "user missing", err: domain.ErrUserNotFound, want: http.StatusNotFound},
		{name: "group missing", err: domain.ErrGroupNotFound, want: http.StatusNotFound},
		{name: "username taken", err: domain.ErrUsernameTaken, want: http.StatusConflict},
		{name: "last group", err: domain.ErrUserMustHaveGroup, want: http.StatusConflict},
		{name: "last admin", err: domain.ErrLastAdmin, want: http.StatusConflict},
		{name: "unknown", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newUserTestContext(http.MethodGet, "/", nil, nil)
			writeError(c, tt.err)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestListPropagatesServiceError(t *testing.T) {
	service := &testutil.UserService{
		ListFunc: func(context.Context, pagination.PaginationParams) ([]domain.User, int64, error) {
			return nil, 0, errors.New("database unreachable")
		},
	}
	c, w := newUserTestContext(http.MethodGet, "/users", nil, nil)
	NewUserHandler(service).List(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestGetPropagatesServiceError(t *testing.T) {
	service := &testutil.UserService{
		GetFunc: func(context.Context, uint) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}
	c, w := newUserTestContext(http.MethodGet, "/users/99", nil, gin.Params{{Key: "userId", Value: "99"}})
	NewUserHandler(service).Get(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestCreatePropagatesServiceError(t *testing.T) {
	service := &testutil.UserService{
		CreateFunc: func(context.Context, domain.CreateUserInput) (domain.User, error) {
			return domain.User{}, domain.ErrUsernameTaken
		},
	}
	c, w := newUserTestContext(http.MethodPost, "/users", []byte(`{"username":"existing","password":"password123","groupIds":[1]}`), nil)
	NewUserHandler(service).Create(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestUpdateRejectsInvalidJSONAndPropagatesError(t *testing.T) {
	service := &testutil.UserService{
		UpdateFunc: func(context.Context, uint, domain.UpdateUserInput) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}
	handler := NewUserHandler(service)

	// Invalid JSON
	c1, w1 := newUserTestContext(http.MethodPatch, "/users/5", []byte(`{"username":`), gin.Params{{Key: "userId", Value: "5"}})
	handler.Update(c1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for bad json", w1.Code)
	}

	// Service error (404)
	c2, w2 := newUserTestContext(http.MethodPatch, "/users/5", []byte(`{"username":"new"}`), gin.Params{{Key: "userId", Value: "5"}})
	handler.Update(c2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w2.Code)
	}
}

func TestDeletePropagatesServiceError(t *testing.T) {
	service := &testutil.UserService{
		DeleteFunc: func(context.Context, uint) error {
			return domain.ErrLastAdmin
		},
	}
	c, w := newUserTestContext(http.MethodDelete, "/users/1", nil, gin.Params{{Key: "userId", Value: "1"}})
	NewUserHandler(service).Delete(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 for deleting last admin", w.Code)
	}
}

func TestSetRoleRejectsInvalidJSONAndPropagatesError(t *testing.T) {
	service := &testutil.UserService{
		SetRoleFunc: func(context.Context, uint, string) (domain.User, error) {
			return domain.User{}, domain.ErrInvalidRole
		},
	}
	handler := NewUserHandler(service)

	// Invalid JSON
	c1, w1 := newUserTestContext(http.MethodPatch, "/users/3/role", []byte(`{"role":`), gin.Params{{Key: "userId", Value: "3"}})
	handler.SetRole(c1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for bad json", w1.Code)
	}

	// Service error (ErrInvalidRole -> 400)
	c2, w2 := newUserTestContext(http.MethodPatch, "/users/3/role", []byte(`{"role":"superadmin"}`), gin.Params{{Key: "userId", Value: "3"}})
	handler.SetRole(c2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for invalid role", w2.Code)
	}
}
