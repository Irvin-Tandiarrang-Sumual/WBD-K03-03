package auth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

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
