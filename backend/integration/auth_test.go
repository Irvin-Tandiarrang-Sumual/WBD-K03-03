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

	userRepo := userrepository.NewRepository(db)

	groupRepo := grouprepository.NewRepository(db)

	sessionRepo := sessionrepository.NewRepository(db)

	sessionService := sessionservice.NewService(
		sessionRepo,
	)

	authService := authservice.NewService(
		userRepo,
		sessionService,
	)

	userService := userservice.NewService(
		userRepo,
		groupRepo,
		sessionService,
	)

	authHandler := authhandler.NewHandler(
		authService,
		false,
	)

	userHandler := userhandler.NewHandler(
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