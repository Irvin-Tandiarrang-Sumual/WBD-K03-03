package group

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

func newGroupTestContext(method string, path string, body []byte, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestListReturnsPagedGroups(t *testing.T) {
	service := &testutil.GroupService{
		ListFunc: func(_ context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error) {
			if params.Page != 2 || params.Limit != 2 {
				t.Fatalf("pagination = %+v", params)
			}
			return []domain.Group{{
				ID:   4,
				Name: "developers",
				DatabaseAccesses: []domain.DatabaseAccess{{
					DatabaseID: 9,
					AccessType: domain.AccessViewEdit,
				}},
			}}, 5, nil
		},
	}

	c, w := newGroupTestContext(http.MethodGet, "/groups?page=2&limit=2", nil, nil)
	NewGroupHandler(service).List(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].([]any)
	group := data[0].(map[string]any)
	accesses := group["databaseAccesses"].([]any)
	access := accesses[0].(map[string]any)
	if group["name"] != "developers" || access["accessType"] != string(domain.AccessViewEdit) {
		t.Fatalf("unexpected group response: %v", group)
	}
	paging := body["paging"].(map[string]any)
	if paging["total_page"] != float64(3) {
		t.Fatalf("unexpected paging: %v", paging)
	}
}

func TestListRejectsInvalidPagination(t *testing.T) {
	called := false
	service := &testutil.GroupService{
		ListFunc: func(context.Context, pagination.PaginationParams) ([]domain.Group, int64, error) {
			called = true
			return nil, 0, nil
		},
	}
	c, w := newGroupTestContext(http.MethodGet, "/groups?limit=bad", nil, nil)
	NewGroupHandler(service).List(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if called {
		t.Fatal("service must not be called for invalid pagination")
	}
}

func TestGetReturnsGroup(t *testing.T) {
	service := &testutil.GroupService{
		GetFunc: func(_ context.Context, id uint) (domain.Group, error) {
			if id != 2 {
				t.Fatalf("id = %d, want 2", id)
			}
			return domain.Group{ID: id, Name: "group"}, nil
		},
	}
	c, w := newGroupTestContext(http.MethodGet, "/groups/2", nil, gin.Params{{Key: "groupId", Value: "2"}})
	NewGroupHandler(service).Get(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestCreateMapsDatabaseAccesses(t *testing.T) {
	service := &testutil.GroupService{
		CreateFunc: func(_ context.Context, input domain.CreateGroupInput) (domain.Group, error) {
			if input.Name != "analytics" || len(input.DatabaseAccesses) != 2 {
				t.Fatalf("unexpected input: %+v", input)
			}
			if input.DatabaseAccesses[0].DatabaseID != 3 || input.DatabaseAccesses[0].AccessType != domain.AccessViewOnly || input.DatabaseAccesses[1].DatabaseID != 7 || input.DatabaseAccesses[1].AccessType != domain.AccessViewEdit {
				t.Fatalf("unexpected accesses: %+v", input.DatabaseAccesses)
			}
			return domain.Group{ID: 11, Name: input.Name, DatabaseAccesses: input.DatabaseAccesses}, nil
		},
	}
	body := []byte(`{"name":"analytics","databaseAccesses":[{"databaseId":3,"accessType":"view_only"},{"databaseId":7,"accessType":"view_edit"}]}`)
	c, w := newGroupTestContext(http.MethodPost, "/groups", body, nil)
	NewGroupHandler(service).Create(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateRejectsInvalidJSON(t *testing.T) {
	called := false
	service := &testutil.GroupService{
		CreateFunc: func(context.Context, domain.CreateGroupInput) (domain.Group, error) {
			called = true
			return domain.Group{}, nil
		},
	}
	c, w := newGroupTestContext(http.MethodPost, "/groups", []byte(`{"name":`), nil)
	NewGroupHandler(service).Create(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if called {
		t.Fatal("service must not be called for invalid JSON")
	}
}

func TestUpdateMapsOptionalFields(t *testing.T) {
	service := &testutil.GroupService{
		UpdateFunc: func(_ context.Context, id uint, input domain.UpdateGroupInput) (domain.Group, error) {
			if id != 5 || input.Name == nil || *input.Name != "renamed" || input.DatabaseAccesses == nil || len(*input.DatabaseAccesses) != 1 {
				t.Fatalf("unexpected update: id=%d input=%+v", id, input)
			}
			access := (*input.DatabaseAccesses)[0]
			if access.DatabaseID != 10 || access.AccessType != domain.AccessViewEdit {
				t.Fatalf("unexpected access: %+v", access)
			}
			return domain.Group{ID: id, Name: *input.Name, DatabaseAccesses: *input.DatabaseAccesses}, nil
		},
	}
	body := []byte(`{"name":"renamed","databaseAccesses":[{"databaseId":10,"accessType":"view_edit"}]}`)
	c, w := newGroupTestContext(http.MethodPatch, "/groups/5", body, gin.Params{{Key: "groupId", Value: "5"}})
	NewGroupHandler(service).Update(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteAddAndRemoveUserDelegate(t *testing.T) {
	var deleted uint
	var added [2]uint
	var removed [2]uint
	service := &testutil.GroupService{
		DeleteFunc: func(_ context.Context, groupID uint) error {
			deleted = groupID
			return nil
		},
		AddUserFunc: func(_ context.Context, groupID uint, userID uint) error {
			added = [2]uint{groupID, userID}
			return nil
		},
		RemoveUserFunc: func(_ context.Context, groupID uint, userID uint) error {
			removed = [2]uint{groupID, userID}
			return nil
		},
	}
	handler := NewGroupHandler(service)

	c, _ := newGroupTestContext(http.MethodDelete, "/groups/4", nil, gin.Params{{Key: "groupId", Value: "4"}})
	handler.Delete(c)
	if c.Writer.Status() != http.StatusNoContent || deleted != 4 {
		t.Fatalf("delete status=%d id=%d", c.Writer.Status(), deleted)
	}

	params := gin.Params{{Key: "groupId", Value: "4"}, {Key: "userId", Value: "8"}}
	c, _ = newGroupTestContext(http.MethodPost, "/groups/4/users/8", nil, params)
	handler.AddUser(c)
	if c.Writer.Status() != http.StatusNoContent || added != [2]uint{4, 8} {
		t.Fatalf("add status=%d call=%v", c.Writer.Status(), added)
	}

	c, _ = newGroupTestContext(http.MethodDelete, "/groups/4/users/8", nil, params)
	handler.RemoveUser(c)
	if c.Writer.Status() != http.StatusNoContent || removed != [2]uint{4, 8} {
		t.Fatalf("remove status=%d call=%v", c.Writer.Status(), removed)
	}
}

func TestGroupHandlersRejectInvalidIDs(t *testing.T) {
	handler := NewGroupHandler(&testutil.GroupService{})

	tests := []struct {
		name   string
		params gin.Params
		call   func(*gin.Context)
	}{
		{name: "get group", params: gin.Params{{Key: "groupId", Value: "x"}}, call: handler.Get},
		{name: "update group", params: gin.Params{{Key: "groupId", Value: "x"}}, call: handler.Update},
		{name: "delete group", params: gin.Params{{Key: "groupId", Value: "x"}}, call: handler.Delete},
		{name: "add invalid group", params: gin.Params{{Key: "groupId", Value: "x"}, {Key: "userId", Value: "2"}}, call: handler.AddUser},
		{name: "add invalid user", params: gin.Params{{Key: "groupId", Value: "1"}, {Key: "userId", Value: "x"}}, call: handler.AddUser},
		{name: "remove invalid group", params: gin.Params{{Key: "groupId", Value: "x"}, {Key: "userId", Value: "2"}}, call: handler.RemoveUser},
		{name: "remove invalid user", params: gin.Params{{Key: "groupId", Value: "1"}, {Key: "userId", Value: "x"}}, call: handler.RemoveUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newGroupTestContext(http.MethodPatch, "/", []byte(`{}`), tt.params)
			tt.call(c)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestGroupErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid input", err: domain.ErrInvalidInput, want: http.StatusBadRequest},
		{name: "invalid pagination", err: domain.ErrInvalidPagination, want: http.StatusBadRequest},
		{name: "invalid access", err: domain.ErrInvalidDatabaseAccess, want: http.StatusBadRequest},
		{name: "invalid groups", err: domain.ErrInvalidGroupsIDs, want: http.StatusBadRequest},
		{name: "group missing", err: domain.ErrGroupNotFound, want: http.StatusNotFound},
		{name: "user missing", err: domain.ErrUserNotFound, want: http.StatusNotFound},
		{name: "database missing", err: domain.ErrDatabaseNotFound, want: http.StatusNotFound},
		{name: "name taken", err: domain.ErrGroupNameTaken, want: http.StatusConflict},
		{name: "member exists", err: domain.ErrMembershipExists, want: http.StatusConflict},
		{name: "member missing", err: domain.ErrMembershipNotFound, want: http.StatusConflict},
		{name: "group needs database", err: domain.ErrGroupMustHaveDatabase, want: http.StatusConflict},
		{name: "user needs group", err: domain.ErrUserMustHaveGroup, want: http.StatusConflict},
		{name: "unknown", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newGroupTestContext(http.MethodGet, "/", nil, nil)
			writeError(c, tt.err)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestListGroupPropagatesServiceError(t *testing.T) {
	service := &testutil.GroupService{
		ListFunc: func(context.Context, pagination.PaginationParams) ([]domain.Group, int64, error) {
			return nil, 0, errors.New("database failure")
		},
	}
	c, w := newGroupTestContext(http.MethodGet, "/groups", nil, nil)
	NewGroupHandler(service).List(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want=500", w.Code)
	}
}

func TestGetGroupPropagatesServiceError(t *testing.T) {
	service := &testutil.GroupService{
		GetFunc: func(context.Context, uint) (domain.Group, error) {
			return domain.Group{}, domain.ErrGroupNotFound
		},
	}
	c, w := newGroupTestContext(http.MethodGet, "/groups/4", nil, gin.Params{{Key: "groupId", Value: "4"}})
	NewGroupHandler(service).Get(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=404", w.Code)
	}
}

func TestCreateGroupPropagatesServiceError(t *testing.T) {
	service := &testutil.GroupService{
		CreateFunc: func(context.Context, domain.CreateGroupInput) (domain.Group, error) {
			return domain.Group{}, domain.ErrGroupNameTaken
		},
	}
	c, w := newGroupTestContext(http.MethodPost, "/groups", []byte(`{"name":"eng","databaseAccesses":[{"databaseId":1,"accessType":"view_only"}]}`), nil)
	NewGroupHandler(service).Create(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d want=409", w.Code)
	}
}

func TestUpdateGroupRejectsInvalidJSONAndPropagatesServiceError(t *testing.T) {
	service := &testutil.GroupService{
		UpdateFunc: func(context.Context, uint, domain.UpdateGroupInput) (domain.Group, error) {
			return domain.Group{}, domain.ErrGroupNotFound
		},
	}
	handler := NewGroupHandler(service)

	// Invalid JSON
	c1, w1 := newGroupTestContext(http.MethodPatch, "/groups/2", []byte(`{"name":`), gin.Params{{Key: "groupId", Value: "2"}})
	handler.Update(c1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=400", w1.Code)
	}

	// Service error (404)
	c2, w2 := newGroupTestContext(http.MethodPatch, "/groups/2", []byte(`{"name":"new"}`), gin.Params{{Key: "groupId", Value: "2"}})
	handler.Update(c2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=404", w2.Code)
	}
}

func TestDeleteGroupPropagatesServiceError(t *testing.T) {
	service := &testutil.GroupService{
		DeleteFunc: func(context.Context, uint) error {
			return domain.ErrGroupNotFound
		},
	}
	c, w := newGroupTestContext(http.MethodDelete, "/groups/9", nil, gin.Params{{Key: "groupId", Value: "9"}})
	NewGroupHandler(service).Delete(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=404", w.Code)
	}
}

func TestAddUserPropagatesServiceError(t *testing.T) {
	service := &testutil.GroupService{
		AddUserFunc: func(context.Context, uint, uint) error {
			return domain.ErrMembershipExists
		},
	}
	c, w := newGroupTestContext(http.MethodPost, "/groups/1/users/2", nil, gin.Params{{Key: "groupId", Value: "1"}, {Key: "userId", Value: "2"}})
	NewGroupHandler(service).AddUser(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d want=409", w.Code)
	}
}

func TestRemoveUserPropagatesServiceError(t *testing.T) {
	service := &testutil.GroupService{
		RemoveUserFunc: func(context.Context, uint, uint) error {
			return domain.ErrUserMustHaveGroup
		},
	}
	c, w := newGroupTestContext(http.MethodDelete, "/groups/1/users/2", nil, gin.Params{{Key: "groupId", Value: "1"}, {Key: "userId", Value: "2"}})
	NewGroupHandler(service).RemoveUser(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d want=409", w.Code)
	}
}
