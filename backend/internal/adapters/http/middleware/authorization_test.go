package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
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

func TestRequireDatabaseViewAndEditWrappers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		middleware func(ports.UserGroupService) gin.HandlerFunc
		effective  domain.AccessType
		wantStatus int
	}{
		{
			name:       "view wrapper allows view_only",
			middleware: middleware.RequireDatabaseView,
			effective:  domain.AccessViewOnly,
			wantStatus: http.StatusOK,
		},
		{
			name:       "view wrapper allows view_edit",
			middleware: middleware.RequireDatabaseView,
			effective:  domain.AccessViewEdit,
			wantStatus: http.StatusOK,
		},
		{
			name:       "edit wrapper rejects view_only (access denied)",
			middleware: middleware.RequireDatabaseEdit,
			effective:  domain.AccessViewOnly,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "edit wrapper allows view_edit",
			middleware: middleware.RequireDatabaseEdit,
			effective:  domain.AccessViewEdit,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := &testutil.GroupService{
				EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
					return tt.effective, nil
				},
			}

			r := gin.New()
			r.GET(
				"/databases/:database_id",
				func(c *gin.Context) {
					middleware.SetAuthenticatedUserID(c, 10)
					c.Next()
				},
				tt.middleware(groups),
				func(c *gin.Context) {
					c.Status(http.StatusOK)
				},
			)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/databases/1", nil))
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireDatabaseAccessErrorCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Case 1: unauthenticated
	groups := &testutil.GroupService{}
	r1 := gin.New()
	r1.GET("/databases/:database_id", middleware.RequireDatabaseView(groups), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w1 := httptest.NewRecorder()
	r1.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/databases/1", nil))
	if w1.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for unauthenticated", w1.Code)
	}

	// Case 2: database not found
	groupsNotFound := &testutil.GroupService{
		EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
			return "", domain.ErrDatabaseNotFound
		},
	}
	r2 := gin.New()
	r2.GET("/databases/:database_id", func(c *gin.Context) {
		middleware.SetAuthenticatedUserID(c, 10)
		c.Next()
	}, middleware.RequireDatabaseView(groupsNotFound), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/databases/999", nil))
	if w2.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for not found database", w2.Code)
	}

	// Case 3: internal service error
	groupsInternalErr := &testutil.GroupService{
		EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
			return "", errors.New("db query error")
		},
	}
	r3 := gin.New()
	r3.GET("/databases/:database_id", func(c *gin.Context) {
		middleware.SetAuthenticatedUserID(c, 10)
		c.Next()
	}, middleware.RequireDatabaseView(groupsInternalErr), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w3 := httptest.NewRecorder()
	r3.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/databases/1", nil))
	if w3.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 for internal error", w3.Code)
	}
}

func TestRequireAdminPropagatesInternalError(t *testing.T) {
	users := &testutil.UserService{
		GetFunc: func(context.Context, uint) (domain.User, error) {
			return domain.User{}, errors.New("db connection timeout")
		},
	}
	r := gin.New()
	r.GET("/admin", func(c *gin.Context) {
		middleware.SetAuthenticatedUserID(c, 1)
		c.Next()
	}, middleware.RequireAdmin(users), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
