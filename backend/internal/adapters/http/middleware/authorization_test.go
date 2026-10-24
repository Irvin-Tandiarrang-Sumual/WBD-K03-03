package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestRequireAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		setUserID  bool
		user       domain.User
		userErr    error
		wantStatus int
	}{
		{
			name:       "missing authentication",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:      "regular user",
			setUserID: true,
			user: domain.User{
				ID:   1,
				Role: domain.RoleUser,
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:      "admin",
			setUserID: true,
			user: domain.User{
				ID:   1,
				Role: domain.RoleAdmin,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "deleted user",
			setUserID:  true,
			userErr:    domain.ErrUserNotFound,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := &testutil.UserService{
				GetFunc: func(
					context.Context,
					uint,
				) (domain.User, error) {
					return tt.user, tt.userErr
				},
			}

			r := gin.New()

			r.GET(
				"/protected",
				func(c *gin.Context) {
					if tt.setUserID {
						middleware.SetAuthenticatedUserID(
							c,
							1,
						)
					}

					c.Next()
				},
				middleware.RequireAdmin(users),
				func(c *gin.Context) {
					c.Status(http.StatusOK)
				},
			)

			w := httptest.NewRecorder()

			r.ServeHTTP(
				w,
				httptest.NewRequest(
					http.MethodGet,
					"/protected",
					nil,
				),
			)

			if w.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					w.Code,
					tt.wantStatus,
				)
			}
		})
	}
}

func TestRequireDatabaseAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		required   domain.AccessType
		effective  domain.AccessType
		serviceErr error
		path       string
		wantStatus int
	}{
		{
			name:       "view only may view",
			required:   domain.AccessViewOnly,
			effective:  domain.AccessViewOnly,
			path:       "/databases/5",
			wantStatus: http.StatusOK,
		},
		{
			name:       "edit may view",
			required:   domain.AccessViewOnly,
			effective:  domain.AccessViewEdit,
			path:       "/databases/5",
			wantStatus: http.StatusOK,
		},
		{
			name:       "view only may not edit",
			required:   domain.AccessViewEdit,
			effective:  domain.AccessViewOnly,
			path:       "/databases/5",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "edit may edit",
			required:   domain.AccessViewEdit,
			effective:  domain.AccessViewEdit,
			path:       "/databases/5",
			wantStatus: http.StatusOK,
		},
		{
			name:       "no grant",
			required:   domain.AccessViewOnly,
			serviceErr: domain.ErrDatabaseAccessDenied,
			path:       "/databases/5",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid database id",
			required:   domain.AccessViewOnly,
			path:       "/databases/not-a-number",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := &testutil.GroupService{
				EffectiveDatabaseAccessFunc: func(
					_ context.Context,
					userID uint,
					databaseID uint,
				) (domain.AccessType, error) {
					if databaseID != 5 {
						return "",
							errors.New(
								"unexpected database id",
							)
					}

					return tt.effective,
						tt.serviceErr
				},
			}

			r := gin.New()

			r.GET(
				"/databases/:databaseId",
				func(c *gin.Context) {
					middleware.SetAuthenticatedUserID(
						c,
						1,
					)

					c.Next()
				},
				middleware.RequireDatabaseAccess(
					groups,
					tt.required,
				),
				func(c *gin.Context) {
					c.Status(http.StatusOK)
				},
			)

			w := httptest.NewRecorder()

			r.ServeHTTP(
				w,
				httptest.NewRequest(
					http.MethodGet,
					tt.path,
					nil,
				),
			)

			if w.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					w.Code,
					tt.wantStatus,
				)
			}
		})
	}
}