package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestAuthenticate_MissingSessionCookie_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())

	authMiddleware := middleware.NewAuthMiddleware(
		&testutil.SessionService{},
		&testutil.UserService{},
		false,
	)
	engine.Use(authMiddleware.Authenticate())
	engine.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthenticate_InvalidOrExpiredSession_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())

	sessionService := &testutil.SessionService{
		ResolveFunc: func(ctx context.Context, token string) (domain.Session, error) {
			return domain.Session{}, domain.ErrInvalidSession
		},
	}
	authMiddleware := middleware.NewAuthMiddleware(
		sessionService,
		&testutil.UserService{},
		false,
	)
	engine.Use(authMiddleware.Authenticate())
	engine.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: "invalid-token",
	})
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthenticate_UserNotFound_ReturnsNotFound(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())

	sessionService := &testutil.SessionService{
		ResolveFunc: func(ctx context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:          1,
				UserID:      10,
				OpaqueToken: token,
			}, nil
		},
	}
	userService := &testutil.UserService{
		GetUserByIDFunc: func(ctx context.Context, id uint) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}
	authMiddleware := middleware.NewAuthMiddleware(
		sessionService,
		userService,
		false,
	)
	engine.Use(authMiddleware.Authenticate())
	engine.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: "valid-token",
	})
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestAuthenticate_ValidSessionAndUser_SetsContextAndCallsNextHandler(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())

	expectedUser := domain.User{
		ID:       10,
		Username: "john_doe",
		Role:     domain.RoleAdmin,
	}
	expectedSession := domain.Session{
		ID:          1,
		UserID:      10,
		OpaqueToken: "valid-token",
		CSRFToken:   "csrf-secret",
	}

	sessionService := &testutil.SessionService{
		ResolveFunc: func(ctx context.Context, token string) (domain.Session, error) {
			return expectedSession, nil
		},
	}
	userService := &testutil.UserService{
		GetUserByIDFunc: func(ctx context.Context, id uint) (domain.User, error) {
			return expectedUser, nil
		},
	}
	authMiddleware := middleware.NewAuthMiddleware(
		sessionService,
		userService,
		false,
	)
	engine.Use(authMiddleware.Authenticate())

	var capturedUserID uint
	var capturedUserRole string
	var capturedCSRFToken string

	engine.GET("/protected", func(c *gin.Context) {
		capturedUserID, _ = middleware.CurrentUserID(c)
		if role, exists := c.Get(middleware.AuthenticatedUserRoleKey); exists {
			capturedUserRole = role.(string)
		}
		capturedCSRFToken = middleware.CurrentCSRFToken(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: "valid-token",
	})
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if capturedUserID != 10 {
		t.Fatalf("captured user id = %d, want 10", capturedUserID)
	}
	if capturedUserRole != domain.RoleAdmin {
		t.Fatalf("captured role = %q, want %q", capturedUserRole, domain.RoleAdmin)
	}
	if capturedCSRFToken != "csrf-secret" {
		t.Fatalf("captured csrf token = %q, want %q", capturedCSRFToken, "csrf-secret")
	}
}

func TestAuthenticate_TokenRotated_RefreshesSessionCookie(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())

	rotatedSession := domain.Session{
		ID:                 1,
		UserID:             10,
		OpaqueToken:        "new-rotated-token",
		CSRFToken:          "csrf-secret",
		IdleTimeout:        30 * time.Minute,
		AbsoluteExpiration: time.Now().Add(24 * time.Hour),
	}

	sessionService := &testutil.SessionService{
		ResolveFunc: func(ctx context.Context, token string) (domain.Session, error) {
			return rotatedSession, nil
		},
	}
	userService := &testutil.UserService{
		GetUserByIDFunc: func(ctx context.Context, id uint) (domain.User, error) {
			return domain.User{ID: 10, Role: domain.RoleUser}, nil
		},
	}
	authMiddleware := middleware.NewAuthMiddleware(
		sessionService,
		userService,
		false,
	)
	engine.Use(authMiddleware.Authenticate())
	engine.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: "old-token",
	})
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == middleware.SessionCookieName {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected refreshed session cookie, got none")
	}
	if sessionCookie.Value != "new-rotated-token" {
		t.Fatalf("cookie value = %q, want %q", sessionCookie.Value, "new-rotated-token")
	}
}
