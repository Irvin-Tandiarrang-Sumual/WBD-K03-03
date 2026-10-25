package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestAuthenticate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		cookie     *http.Cookie
		resolve    func(context.Context, string) (domain.Session, error)
		wantStatus int
		wantUserID uint
	}{
		{
			name:       "missing cookie",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "empty cookie",
			cookie: &http.Cookie{
				Name:  middleware.SessionCookieName,
				Value: "",
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid session",
			cookie: &http.Cookie{
				Name:  middleware.SessionCookieName,
				Value: "expired-token",
			},
			resolve: func(context.Context, string) (domain.Session, error) {
				return domain.Session{}, domain.ErrInvalidSession
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "repository failure",
			cookie: &http.Cookie{
				Name:  middleware.SessionCookieName,
				Value: "token",
			},
			resolve: func(context.Context, string) (domain.Session, error) {
				return domain.Session{}, errors.New("database unavailable")
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "valid session",
			cookie: &http.Cookie{
				Name:  middleware.SessionCookieName,
				Value: "valid-token",
			},
			resolve: func(_ context.Context, token string) (domain.Session, error) {
				if token != "valid-token" {
					t.Fatalf("token = %q, want valid-token", token)
				}
				return domain.Session{UserID: 42, OpaqueToken: "valid-token"}, nil
			},
			wantStatus: http.StatusOK,
			wantUserID: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := &testutil.SessionService{
				ResolveFunc: func(ctx context.Context, token string) (domain.Session, error) {
					if tt.resolve == nil {
						t.Fatalf("Resolve() must not be called")
					}
					return tt.resolve(ctx, token)
				},
			}

			r := gin.New()
			r.GET(
				"/protected",
				middleware.Authenticate(sessions),
				func(c *gin.Context) {
					userID, ok := middleware.AuthenticatedUserID(c)
					if !ok {
						t.Fatal("authenticated user ID missing from context")
					}
					if userID != tt.wantUserID {
						t.Fatalf("userID = %d, want %d", userID, tt.wantUserID)
					}
					c.Status(http.StatusOK)
				},
			)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestAuthenticateUpdatesCookieWhenTokenRotatedOrResolvedViaGracePeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessions := &testutil.SessionService{
		ResolveFunc: func(_ context.Context, token string) (domain.Session, error) {
			if token != "old-token" {
				t.Fatalf("token = %q, want old-token", token)
			}
			return domain.Session{
				UserID:              42,
				OpaqueToken:         "rotated-new-token",
				PreviousOpaqueToken: "old-token",
				GracePeriodUntil:    time.Now().UTC().Add(30 * time.Second),
				AbsoluteExpiration:  time.Now().UTC().Add(10 * time.Hour),
			}, nil
		},
	}

	r := gin.New()
	r.GET("/protected", middleware.Authenticate(sessions, true), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: "old-token",
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	setCookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "session=rotated-new-token") ||
		!strings.Contains(setCookie, "HttpOnly") ||
		!strings.Contains(setCookie, "Secure") ||
		!strings.Contains(setCookie, "SameSite=Strict") {
		t.Fatalf("unexpected Set-Cookie header: %q", setCookie)
	}
}

func TestAuthenticatedUserIDRejectsInvalidContextValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name  string
		value any
		set   bool
	}{
		{name: "missing"},
		{name: "wrong type", value: "42", set: true},
		{name: "zero", value: uint(0), set: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if tt.set {
				c.Set(middleware.AuthenticatedUserIDKey, tt.value)
			}
			if _, ok := middleware.AuthenticatedUserID(c); ok {
				t.Fatal("expected context value to be rejected")
			}
		})
	}
}
