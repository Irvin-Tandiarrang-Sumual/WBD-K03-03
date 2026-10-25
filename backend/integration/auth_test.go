//go:build integration

package integration_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	authhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/auth"
	userhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	grouprepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/group"
	sessionrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/session"
	userrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/user"
	authservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/auth"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func authRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)

	userRepo := userrepository.NewUserRepository(db)

	groupRepo := grouprepository.NewGroupRepository(db)

	sessionRepo := sessionrepository.NewSessionRepository(db)

	sessionService := sessionservice.NewSessionService(
		sessionRepo,
	)

	authService := authservice.NewAuthService(
		userRepo,
		sessionService,
	)

	userService := userservice.NewUserService(
		userRepo,
		groupRepo,
		sessionService,
	)

	authHandler := authhandler.NewAuthHandler(
		authService,
		false,
		userService,
	)

	userHandler := userhandler.NewUserHandler(
		userService,
	)

	r := gin.New()

	r.POST(
		"/auth/login",
		authHandler.Login,
	)

	r.POST(
		"/auth/logout",
		authHandler.Logout,
	)

	r.GET(
		"/auth/me",
		middleware.Authenticate(
			sessionService,
		),
		authHandler.Me,
	)

	admin := r.Group("/admin")

	admin.Use(
		middleware.Authenticate(
			sessionService,
		),
		middleware.RequireAdmin(
			userService,
		),
	)

	admin.GET(
		"/users",
		userHandler.List,
	)

	return r
}

func login(
	t *testing.T,
	r http.Handler,
	username string,
	password string,
) *http.Cookie {
	t.Helper()

	body := []byte(
		`{"username":"` +
			username +
			`","password":"` +
			password +
			`"}`,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		bytes.NewReader(body),
	)

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(
			"login status = %d, body = %s",
			w.Code,
			w.Body.String(),
		)
	}

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name ==
			middleware.SessionCookieName {
			if !cookie.HttpOnly {
				t.Fatal(
					"session cookie must be HttpOnly",
				)
			}

			return cookie
		}
	}

	t.Fatal(
		"login response did not set session cookie",
	)

	return nil
}

func TestIntegrationLoginAdminRouteAndLogout(t *testing.T) {
	db := openTestDB(t)

	seedTestDB(t, db)

	r := authRouter(db)

	cookie := login(
		t,
		r,
		"admin",
		"admin123",
	)

	adminReq := httptest.NewRequest(
		http.MethodGet,
		"/admin/users",
		nil,
	)

	adminReq.AddCookie(cookie)

	adminRes := httptest.NewRecorder()

	r.ServeHTTP(
		adminRes,
		adminReq,
	)

	if adminRes.Code != http.StatusOK {
		t.Fatalf(
			"admin route status = %d, body = %s",
			adminRes.Code,
			adminRes.Body.String(),
		)
	}

	logoutReq := httptest.NewRequest(
		http.MethodPost,
		"/auth/logout",
		nil,
	)

	logoutReq.AddCookie(cookie)

	logoutRes := httptest.NewRecorder()

	r.ServeHTTP(
		logoutRes,
		logoutReq,
	)

	if logoutRes.Code != http.StatusNoContent {
		t.Fatalf(
			"logout status = %d, body = %s",
			logoutRes.Code,
			logoutRes.Body.String(),
		)
	}

	afterLogoutReq := httptest.NewRequest(
		http.MethodGet,
		"/admin/users",
		nil,
	)

	afterLogoutReq.AddCookie(cookie)

	afterLogoutRes := httptest.NewRecorder()

	r.ServeHTTP(
		afterLogoutRes,
		afterLogoutReq,
	)

	if afterLogoutRes.Code !=
		http.StatusUnauthorized {
		t.Fatalf(
			"status after logout = %d, want 401",
			afterLogoutRes.Code,
		)
	}
}

func TestIntegrationRegularUserCannotAccessAdminRoute(t *testing.T) {
	db := openTestDB(t)

	seedTestDB(t, db)

	r := authRouter(db)

	cookie := login(
		t,
		r,
		"user",
		"user123",
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/users",
		nil,
	)

	req.AddCookie(cookie)

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf(
			"status = %d, want 403; body = %s",
			w.Code,
			w.Body.String(),
		)
	}
}

func TestIntegrationAuthMeReturnsProfileAndDeniesRevokedSession(t *testing.T) {
	db := openTestDB(t)
	seedTestDB(t, db)
	r := authRouter(db)

	// 1. Unauthenticated request to /auth/me returns 401
	unauthReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	unauthRec := httptest.NewRecorder()
	r.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /auth/me status = %d, want 401", unauthRec.Code)
	}

	// 2. Login as regular user
	cookie := login(t, r, "user", "user123")

	// 3. Authenticated request to /auth/me returns 200 with profile
	meReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	meReq.AddCookie(cookie)
	meRec := httptest.NewRecorder()
	r.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("authenticated /auth/me status = %d, want 200; body = %s", meRec.Code, meRec.Body.String())
	}
	body := meRec.Body.String()
	if !bytes.Contains(meRec.Body.Bytes(), []byte(`"username":"user"`)) {
		t.Fatalf("profile missing username in body: %s", body)
	}
	if bytes.Contains(meRec.Body.Bytes(), []byte("user123")) || bytes.Contains(meRec.Body.Bytes(), []byte("$2a$")) {
		t.Fatalf("password or hash leaked in /auth/me response: %s", body)
	}

	// 4. Logout to revoke session in PostgreSQL
	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutRec := httptest.NewRecorder()
	r.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", logoutRec.Code)
	}

	// 5. Subsequent request to /auth/me with revoked cookie returns 401
	revokedReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	revokedReq.AddCookie(cookie)
	revokedRec := httptest.NewRecorder()
	r.ServeHTTP(revokedRec, revokedReq)
	if revokedRec.Code != http.StatusUnauthorized {
		t.Fatalf("revoked cookie /auth/me status = %d, want 401; body = %s", revokedRec.Code, revokedRec.Body.String())
	}
}
