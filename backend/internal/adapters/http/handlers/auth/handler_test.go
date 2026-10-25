package auth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type fakeAuthService struct {
	ports.AuthService
	loginUser    domain.User
	loginSession domain.Session
	loginErr     error
}

func (f *fakeAuthService) Login(
	context.Context,
	string,
	string,
) (domain.User, domain.Session, error) {
	return f.loginUser, f.loginSession, f.loginErr
}

func (f *fakeAuthService) Logout(context.Context, string) error {
	return nil
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
