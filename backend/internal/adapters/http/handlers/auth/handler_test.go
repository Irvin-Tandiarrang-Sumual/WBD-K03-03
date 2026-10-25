package auth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type fakeUserService struct {
	ports.UserService
	getFunc func(context.Context, uint) (domain.User, error)
}

func (f *fakeUserService) Get(ctx context.Context, id uint) (domain.User, error) {
	if f.getFunc != nil {
		return f.getFunc(ctx, id)
	}
	return domain.User{}, domain.ErrUserNotFound
}

type fakeAuthService struct {
	ports.AuthService
	loginUser    domain.User
	loginSession domain.Session
	loginErr     error
	logoutErr    error
	logoutToken  string
}

func (f *fakeAuthService) Login(
	context.Context,
	string,
	string,
) (domain.User, domain.Session, error) {
	return f.loginUser, f.loginSession, f.loginErr
}

func (f *fakeAuthService) Logout(_ context.Context, token string) error {
	f.logoutToken = token
	return f.logoutErr
}

func TestLoginReturnsSessionUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeAuthService{
		loginUser: domain.User{ID: 4, Username: "admin", Role: domain.RoleAdmin},
		loginSession: domain.Session{
			OpaqueToken:        "opaque-token",
			AbsoluteExpiration: time.Now().Add(time.Hour),
		},
	}
	handler := NewAuthHandler(service, false)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/login",
		bytes.NewBufferString(`{"username":"admin","password":"secret"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	handler.Login(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Header().Get("Set-Cookie") == "" {
		t.Fatal("expected session cookie")
	}
}

func TestLoginRejectsInvalidJSON(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{}, false)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/login",
		bytes.NewBufferString(`{"username":`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	handler.Login(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestLoginMapsAuthenticationErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid credentials", err: domain.ErrInvalidCredentials, want: http.StatusUnauthorized},
		{name: "internal error", err: errors.New("repository down"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeAuthService{loginErr: tt.err}
			handler := NewAuthHandler(service, false)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"user","password":"pw"}`))
			context.Request.Header.Set("Content-Type", "application/json")

			handler.Login(context)
			if recorder.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, tt.want, recorder.Body.String())
			}
		})
	}
}

func TestLoginHonorsSecureCookieSetting(t *testing.T) {
	service := &fakeAuthService{
		loginUser: domain.User{ID: 1, Username: "user", Role: domain.RoleUser},
		loginSession: domain.Session{
			OpaqueToken:        "secure-token",
			AbsoluteExpiration: time.Now().Add(time.Hour),
		},
	}
	handler := NewAuthHandler(service, true)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"user","password":"pw"}`))
	context.Request.Header.Set("Content-Type", "application/json")

	handler.Login(context)
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies=%v", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != middleware.SessionCookieName || cookie.Value != "secure-token" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("unexpected cookie=%+v", cookie)
	}
}

func TestLogoutRevokesCookieTokenAndClearsCookie(t *testing.T) {
	service := &fakeAuthService{}
	handler := NewAuthHandler(service, false)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/logout", nil)
	context.Request.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "opaque"})

	handler.Logout(context)
	if context.Writer.Status() != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", context.Writer.Status(), recorder.Body.String())
	}
	if service.logoutToken != "opaque" {
		t.Fatalf("logout token=%q", service.logoutToken)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != middleware.SessionCookieName || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected clearing cookie, got=%v", cookies)
	}
}

func TestLogoutWithoutCookieStillSucceeds(t *testing.T) {
	service := &fakeAuthService{}
	handler := NewAuthHandler(service, false)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/logout", nil)

	handler.Logout(context)
	if context.Writer.Status() != http.StatusNoContent || service.logoutToken != "" {
		t.Fatalf("status=%d token=%q", context.Writer.Status(), service.logoutToken)
	}
}

func TestLogoutReturnsInternalErrorWhenRevokeFails(t *testing.T) {
	service := &fakeAuthService{logoutErr: errors.New("delete failed")}
	handler := NewAuthHandler(service, false)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/logout", nil)
	context.Request.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "opaque"})

	handler.Logout(context)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestLoginNormalizesNegativeMaxAge(t *testing.T) {
	service := &fakeAuthService{
		loginUser: domain.User{ID: 1, Username: "user", Role: domain.RoleUser},
		loginSession: domain.Session{
			OpaqueToken:        "tok-past",
			AbsoluteExpiration: time.Now().Add(-1 * time.Hour), // expired in past
		},
	}
	handler := NewAuthHandler(service, false)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"user","password":"pw"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.Login(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", recorder.Code, http.StatusOK)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected cookie, got %v", cookies)
	}
	if cookies[0].MaxAge != 0 {
		t.Fatalf("expected MaxAge=0 for expired/negative session lifetime, got %d", cookies[0].MaxAge)
	}
}

func TestLoginResponseDoesNotLeakPasswordOrSessionToken(t *testing.T) {
	secretPasswordHash := "$2a$10$somethingrandomandsecret"
	secretOpaqueToken := "secret-opaque-session-token-12345"
	service := &fakeAuthService{
		loginUser: domain.User{
			ID:       10,
			Username: "alice",
			Password: secretPasswordHash,
			Role:     domain.RoleUser,
		},
		loginSession: domain.Session{
			OpaqueToken:        secretOpaqueToken,
			AbsoluteExpiration: time.Now().Add(time.Hour),
		},
	}
	handler := NewAuthHandler(service, true)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"alice","password":"mysecretpassword"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.Login(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()
	// Security requirement: Password and session identifier must not appear in response body
	if strings.Contains(body, secretPasswordHash) {
		t.Fatalf("password hash leaked in response body: %s", body)
	}
	if strings.Contains(body, secretOpaqueToken) {
		t.Fatalf("opaque session token leaked in response body: %s", body)
	}
	if strings.Contains(body, "mysecretpassword") {
		t.Fatalf("plain password leaked in response body: %s", body)
	}
}

func TestMeReturnsAuthenticatedUserProfile(t *testing.T) {
	fakeUser := &fakeUserService{
		getFunc: func(ctx context.Context, id uint) (domain.User, error) {
			if id != 42 {
				t.Fatalf("expected user ID 42, got %d", id)
			}
			return domain.User{
				ID:       42,
				Username: "bob",
				Role:     domain.RoleAdmin,
				Password: "hashed-password-do-not-leak",
			}, nil
		},
	}
	handler := NewAuthHandler(&fakeAuthService{}, false, fakeUser)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/me", nil)
	middleware.SetAuthenticatedUserID(ctx, 42)

	handler.Me(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"username":"bob"`) || !strings.Contains(body, `"role":"admin"`) {
		t.Fatalf("body missing user info: %s", body)
	}
	if strings.Contains(body, "hashed-password-do-not-leak") {
		t.Fatalf("password hash leaked in /me response: %s", body)
	}
}

func TestMeReturnsFallbackWhenUserServiceNil(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{}, false)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/me", nil)
	middleware.SetAuthenticatedUserID(ctx, 15)

	handler.Me(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"id":15`) {
		t.Fatalf("expected id:15 in body: %s", body)
	}
}

func TestMeRejectsUnauthenticatedRequest(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{}, false)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/me", nil)
	// Notice: no middleware.SetAuthenticatedUserID called

	handler.Me(ctx)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "authentication required") {
		t.Fatalf("expected 'authentication required' error: %s", recorder.Body.String())
	}
}

func TestMeReturnsUnauthorizedWhenUserNotFound(t *testing.T) {
	fakeUser := &fakeUserService{
		getFunc: func(ctx context.Context, id uint) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}
	handler := NewAuthHandler(&fakeAuthService{}, false, fakeUser)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/me", nil)
	middleware.SetAuthenticatedUserID(ctx, 99)

	handler.Me(ctx)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "user not found") {
		t.Fatalf("expected 'user not found' error: %s", recorder.Body.String())
	}
}

func TestMeReturnsInternalServerErrorOnServiceError(t *testing.T) {
	fakeUser := &fakeUserService{
		getFunc: func(ctx context.Context, id uint) (domain.User, error) {
			return domain.User{}, errors.New("db connection failure")
		},
	}
	handler := NewAuthHandler(&fakeAuthService{}, false, fakeUser)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/me", nil)
	middleware.SetAuthenticatedUserID(ctx, 99)

	handler.Me(ctx)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "internal server error") {
		t.Fatalf("expected 'internal server error': %s", recorder.Body.String())
	}
}

func TestNewAuthHandlerInitializesUserService(t *testing.T) {
	fakeUser := &fakeUserService{}
	h1 := NewAuthHandler(&fakeAuthService{}, true, fakeUser)
	if h1.userService != fakeUser || !h1.secureCookie {
		t.Fatalf("unexpected handler state: %+v", h1)
	}

	h2 := NewAuthHandler(&fakeAuthService{}, false)
	if h2.userService != nil || h2.secureCookie {
		t.Fatalf("unexpected handler state: %+v", h2)
	}
}
