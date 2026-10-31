package usergroup

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/gin-gonic/gin"
)

func newUserGroupTestContext(method string, path string, body []byte, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestAddAndRemoveUserDelegate(t *testing.T) {
	var added [2]uint
	var removed [2]uint
	service := &testutil.UserGroupService{
		AddUserFunc: func(_ context.Context, groupID uint, userID uint) error {
			added = [2]uint{groupID, userID}
			return nil
		},
		RemoveUserFunc: func(_ context.Context, groupID uint, userID uint) error {
			removed = [2]uint{groupID, userID}
			return nil
		},
	}
	handler := NewUserGroupHandler(service)

	params := gin.Params{{Key: "groupId", Value: "4"}, {Key: "userId", Value: "8"}}
	c, _ := newUserGroupTestContext(http.MethodPost, "/groups/4/users/8", nil, params)
	handler.AddUser(c)
	if c.Writer.Status() != http.StatusNoContent || added != [2]uint{4, 8} {
		t.Fatalf("add status=%d call=%v", c.Writer.Status(), added)
	}

	c, _ = newUserGroupTestContext(http.MethodDelete, "/groups/4/users/8", nil, params)
	handler.RemoveUser(c)
	if c.Writer.Status() != http.StatusNoContent || removed != [2]uint{4, 8} {
		t.Fatalf("remove status=%d call=%v", c.Writer.Status(), removed)
	}
}

func TestUserGroupHandlersRejectInvalidIDs(t *testing.T) {
	handler := NewUserGroupHandler(&testutil.UserGroupService{})

	tests := []struct {
		name   string
		params gin.Params
		call   func(*gin.Context)
	}{
		{name: "add invalid group", params: gin.Params{{Key: "groupId", Value: "x"}, {Key: "userId", Value: "2"}}, call: handler.AddUser},
		{name: "add invalid user", params: gin.Params{{Key: "groupId", Value: "1"}, {Key: "userId", Value: "x"}}, call: handler.AddUser},
		{name: "remove invalid group", params: gin.Params{{Key: "groupId", Value: "x"}, {Key: "userId", Value: "2"}}, call: handler.RemoveUser},
		{name: "remove invalid user", params: gin.Params{{Key: "groupId", Value: "1"}, {Key: "userId", Value: "x"}}, call: handler.RemoveUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newUserGroupTestContext(http.MethodPost, "/", nil, tt.params)
			tt.call(c)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAddUserPropagatesServiceError(t *testing.T) {
	service := &testutil.UserGroupService{
		AddUserFunc: func(context.Context, uint, uint) error {
			return domain.ErrMembershipExists
		},
	}
	c, w := newUserGroupTestContext(http.MethodPost, "/groups/1/users/2", nil, gin.Params{{Key: "groupId", Value: "1"}, {Key: "userId", Value: "2"}})
	NewUserGroupHandler(service).AddUser(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d want=409", w.Code)
	}
}

func TestRemoveUserPropagatesServiceError(t *testing.T) {
	service := &testutil.UserGroupService{
		RemoveUserFunc: func(context.Context, uint, uint) error {
			return domain.ErrUserMustHaveGroup
		},
	}
	c, w := newUserGroupTestContext(http.MethodDelete, "/groups/1/users/2", nil, gin.Params{{Key: "groupId", Value: "1"}, {Key: "userId", Value: "2"}})
	NewUserGroupHandler(service).RemoveUser(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d want=409", w.Code)
	}
}

func TestUserGroupErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid input", err: domain.ErrInvalidInput, want: http.StatusBadRequest},
		{name: "group missing", err: domain.ErrGroupNotFound, want: http.StatusNotFound},
		{name: "user missing", err: domain.ErrUserNotFound, want: http.StatusNotFound},
		{name: "membership exists", err: domain.ErrMembershipExists, want: http.StatusConflict},
		{name: "membership missing", err: domain.ErrMembershipNotFound, want: http.StatusConflict},
		{name: "user needs group", err: domain.ErrUserMustHaveGroup, want: http.StatusConflict},
		{name: "unknown", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newUserGroupTestContext(http.MethodGet, "/", nil, nil)
			writeError(c, tt.err)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}
