//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func TestAuthJourney_GivenValidCredentials_WhenLogin_ThenSetOpaqueTokenCookieAndCreateSessionInPlatformDB(t *testing.T) {
	// Arrange
	stack := setupTestStack(t)
	plainPassword := "ValidPass123!"
	user := createRegularUser(t, stack, "auth_journey_user", plainPassword, nil)

	// Act - Call POST /api/v1/auth/login
	loginReq := handlers.LoginRequest{
		Username: user.Username,
		Password: plainPassword,
	}
	rec := executeRequest(stack.engine, http.MethodPost, "/api/v1/auth/login", loginReq, "", "")

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on login, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify Cookie is set
	var sessionCookie string
	for _, c := range rec.Result().Cookies() {
		if c.Name == middleware.SessionCookieName {
			sessionCookie = c.Value
			break
		}
	}
	if sessionCookie == "" {
		t.Fatalf("expected session cookie to be set in login response")
	}

	// Verify session record physically exists in PostgreSQL platform DB
	storedSession, err := stack.sessionRepo.FindByToken(context.Background(), sessionCookie)
	if err != nil {
		t.Fatalf("failed to query session from platform DB: %v", err)
	}
	if storedSession.UserID != user.ID {
		t.Fatalf("expected session UserID %d, got %d", user.ID, storedSession.UserID)
	}
}

func TestAuthJourney_GivenActiveSessionCookie_WhenCallMeEndpoint_ThenReturnCurrentProfile(t *testing.T) {
	// Arrange
	stack := setupTestStack(t)
	plainPassword := "SecretPass123!"
	user := createRegularUser(t, stack, "auth_me_user", plainPassword, nil)
	sessionCookie, _ := loginUser(t, stack.engine, user.Username, plainPassword)

	// Act - Call GET /api/v1/auth/me with session cookie
	rec := executeRequest(stack.engine, http.MethodGet, "/api/v1/auth/me", nil, sessionCookie, "")

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on /auth/me, got %d: %s", rec.Code, rec.Body.String())
	}

	res := parseResponseBody[domain.User](t, rec)
	if res.Data.Username != user.Username {
		t.Fatalf("expected username %q, got %q", user.Username, res.Data.Username)
	}
}

func TestAuthJourney_GivenActiveSessionCookie_WhenLogout_ThenClearCookieAndRevokeSessionInPlatformDB(t *testing.T) {
	// Arrange
	stack := setupTestStack(t)
	plainPassword := "SecretPass123!"
	user := createRegularUser(t, stack, "auth_logout_user", plainPassword, nil)
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, plainPassword)

	// Act - Call POST /api/v1/auth/logout with session cookie and CSRF token
	rec := executeRequest(stack.engine, http.MethodPost, "/api/v1/auth/logout", nil, sessionCookie, csrfToken)

	// Assert 1 - Logout succeeds and clears cookie
	if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
		t.Fatalf("expected 204 No Content or 200 OK on logout, got %d: %s", rec.Code, rec.Body.String())
	}

	// Assert 2 - Session is removed / invalid in platform DB
	_, err := stack.sessionRepo.FindByToken(context.Background(), sessionCookie)
	if err == nil {
		t.Fatalf("expected session to be revoked in platform DB, but was still found")
	}

	// Assert 3 - Re-accessing /auth/me with the same cookie returns 401 Unauthorized
	recAfter := executeRequest(stack.engine, http.MethodGet, "/api/v1/auth/me", nil, sessionCookie, "")
	if recAfter.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized on revoked session, got %d: %s", recAfter.Code, recAfter.Body.String())
	}
}

func TestAuthJourney_GivenRevokedOrInvalidSessionCookie_WhenAccessProtectedEndpoint_ThenRejectWithUnauthorized(t *testing.T) {
	// Arrange
	stack := setupTestStack(t)

	// Act - Call protected endpoint with fake/bogus session token
	rec := executeRequest(stack.engine, http.MethodGet, "/api/v1/auth/me", nil, "invalid-random-token-xyz", "")

	// Assert
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid token, got %d: %s", rec.Code, rec.Body.String())
	}
}
