package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	authservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/auth"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Login_ValidCredentials_ReturnsUserWithClearedPasswordAndCreatedSession(t *testing.T) {
	// Arrange
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password for test: %v", err)
	}

	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(_ context.Context, username string) (domain.User, error) {
			if username != "admin" {
				t.Fatalf("expected username 'admin', got: %q", username)
			}
			return domain.User{
				ID:       7,
				Username: "admin",
				Password: string(hash),
				Role:     domain.RoleAdmin,
			}, nil
		},
	}

	sessionCreated := false
	sessions := &testutil.SessionService{
		CreateSessionFunc: func(_ context.Context, userID uint) (domain.Session, error) {
			sessionCreated = true
			if userID != 7 {
				t.Fatalf("expected userID 7, got: %d", userID)
			}
			return domain.Session{
				ID:                 1,
				OpaqueToken:        "valid-opaque-token",
				UserID:             userID,
				AbsoluteExpiration: time.Now().Add(time.Hour),
			}, nil
		},
	}

	service := authservice.NewAuthService(users, sessions)

	// Act
	user, session, err := service.Login(context.Background(), "admin", "secret123")

	// Assert
	if err != nil {
		t.Fatalf("unexpected Login() error: %v", err)
	}
	if !sessionCreated {
		t.Fatal("expected session to be created, but CreateSession was not called")
	}
	if user.Password != "" {
		t.Fatalf("expected password hash to be cleared from return user, got: %q", user.Password)
	}
	if user.ID != 7 || user.Username != "admin" || user.Role != domain.RoleAdmin {
		t.Fatalf("unexpected user returned: %+v", user)
	}
	if session.UserID != 7 || session.OpaqueToken != "valid-opaque-token" {
		t.Fatalf("unexpected session returned: %+v", session)
	}
}

func TestAuthService_Login_WhitespaceSurroundingUsername_TrimsUsernameAndAuthenticatesSuccessfully(t *testing.T) {
	// Arrange
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password for test: %v", err)
	}

	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(_ context.Context, username string) (domain.User, error) {
			if username != "admin" {
				t.Fatalf("expected trimmed username 'admin', got: %q", username)
			}
			return domain.User{
				ID:       7,
				Username: "admin",
				Password: string(hash),
				Role:     domain.RoleUser,
			}, nil
		},
	}

	sessions := &testutil.SessionService{
		CreateSessionFunc: func(_ context.Context, userID uint) (domain.Session, error) {
			return domain.Session{
				ID:          1,
				OpaqueToken: "token",
				UserID:      userID,
			}, nil
		},
	}

	service := authservice.NewAuthService(users, sessions)

	// Act
	user, session, err := service.Login(context.Background(), "   admin   ", "secret123")

	// Assert
	if err != nil {
		t.Fatalf("unexpected Login() error: %v", err)
	}
	if user.Username != "admin" {
		t.Fatalf("expected trimmed username 'admin', got: %q", user.Username)
	}
	if session.UserID != 7 {
		t.Fatalf("expected session userID 7, got: %d", session.UserID)
	}
}

func TestAuthService_Login_BlankUsername_ReturnsErrInvalidCredentialsWithoutQueryingRepository(t *testing.T) {
	// Arrange
	repoCalled := false
	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(context.Context, string) (domain.User, error) {
			repoCalled = true
			return domain.User{}, nil
		},
	}
	sessions := &testutil.SessionService{}
	service := authservice.NewAuthService(users, sessions)

	// Act
	user, session, err := service.Login(context.Background(), "   ", "secret123")

	// Assert
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
	if repoCalled {
		t.Fatal("user repository must not be queried when username is blank")
	}
	if user != (domain.User{}) || session != (domain.Session{}) {
		t.Fatalf("expected empty user and session, got: user=%+v, session=%+v", user, session)
	}
}

func TestAuthService_Login_BlankPassword_ReturnsErrInvalidCredentialsWithoutQueryingRepository(t *testing.T) {
	// Arrange
	repoCalled := false
	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(context.Context, string) (domain.User, error) {
			repoCalled = true
			return domain.User{}, nil
		},
	}
	sessions := &testutil.SessionService{}
	service := authservice.NewAuthService(users, sessions)

	// Act
	user, session, err := service.Login(context.Background(), "admin", "")

	// Assert
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
	if repoCalled {
		t.Fatal("user repository must not be queried when password is blank")
	}
	if user != (domain.User{}) || session != (domain.Session{}) {
		t.Fatalf("expected empty user and session, got: user=%+v, session=%+v", user, session)
	}
}

func TestAuthService_Login_UsernameExceedsMaxLength_ReturnsErrInvalidCredentialsWithoutQueryingRepository(t *testing.T) {
	// Arrange
	repoCalled := false
	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(context.Context, string) (domain.User, error) {
			repoCalled = true
			return domain.User{}, nil
		},
	}
	sessions := &testutil.SessionService{}
	service := authservice.NewAuthService(users, sessions)
	longUsername := strings.Repeat("u", 256)

	// Act
	user, session, err := service.Login(context.Background(), longUsername, "secret123")

	// Assert
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for username > 255 chars, got: %v", err)
	}
	if repoCalled {
		t.Fatal("user repository must not be queried when username exceeds max length")
	}
	if user != (domain.User{}) || session != (domain.Session{}) {
		t.Fatalf("expected empty user and session, got: user=%+v, session=%+v", user, session)
	}
}

func TestAuthService_Login_PasswordExceedsMaxBytes_ReturnsErrInvalidCredentialsWithoutQueryingRepository(t *testing.T) {
	// Arrange
	repoCalled := false
	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(context.Context, string) (domain.User, error) {
			repoCalled = true
			return domain.User{}, nil
		},
	}
	sessions := &testutil.SessionService{}
	service := authservice.NewAuthService(users, sessions)
	longPassword := strings.Repeat("p", domain.MaxPasswordBytes+1)

	// Act
	user, session, err := service.Login(context.Background(), "admin", longPassword)

	// Assert
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for password > %d bytes, got: %v", domain.MaxPasswordBytes, err)
	}
	if repoCalled {
		t.Fatal("user repository must not be queried when password exceeds max bytes")
	}
	if user != (domain.User{}) || session != (domain.Session{}) {
		t.Fatalf("expected empty user and session, got: user=%+v, session=%+v", user, session)
	}
}

func TestAuthService_Login_UserNotFoundInRepository_ReturnsErrInvalidCredentialsToPreventUserEnumeration(t *testing.T) {
	// Arrange
	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(context.Context, string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}
	sessionCreated := false
	sessions := &testutil.SessionService{
		CreateSessionFunc: func(context.Context, uint) (domain.Session, error) {
			sessionCreated = true
			return domain.Session{}, nil
		},
	}
	service := authservice.NewAuthService(users, sessions)

	// Act
	user, session, err := service.Login(context.Background(), "nonexistent", "secret123")

	// Assert
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for unknown user to prevent enumeration, got: %v", err)
	}
	if sessionCreated {
		t.Fatal("session must not be created when user is not found")
	}
	if user != (domain.User{}) || session != (domain.Session{}) {
		t.Fatalf("expected empty user and session, got: user=%+v, session=%+v", user, session)
	}
}

func TestAuthService_Login_IncorrectPassword_ReturnsErrInvalidCredentialsAndDoesNotCreateSession(t *testing.T) {
	// Arrange
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password for test: %v", err)
	}

	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(context.Context, string) (domain.User, error) {
			return domain.User{
				ID:       1,
				Username: "admin",
				Password: string(hash),
			}, nil
		},
	}
	sessionCreated := false
	sessions := &testutil.SessionService{
		CreateSessionFunc: func(context.Context, uint) (domain.Session, error) {
			sessionCreated = true
			return domain.Session{}, nil
		},
	}
	service := authservice.NewAuthService(users, sessions)

	// Act
	user, session, err := service.Login(context.Background(), "admin", "wrong-password")

	// Assert
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for incorrect password, got: %v", err)
	}
	if sessionCreated {
		t.Fatal("session must not be created for incorrect password")
	}
	if user != (domain.User{}) || session != (domain.Session{}) {
		t.Fatalf("expected empty user and session, got: user=%+v, session=%+v", user, session)
	}
}

func TestAuthService_Login_UserRepositoryReturnsDatabaseError_PropagatesError(t *testing.T) {
	// Arrange
	wantErr := errors.New("database connection refused")
	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(context.Context, string) (domain.User, error) {
			return domain.User{}, wantErr
		},
	}
	sessions := &testutil.SessionService{}
	service := authservice.NewAuthService(users, sessions)

	// Act
	user, session, err := service.Login(context.Background(), "admin", "secret123")

	// Assert
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got: %v", wantErr, err)
	}
	if user != (domain.User{}) || session != (domain.Session{}) {
		t.Fatalf("expected empty user and session on repository error, got: user=%+v, session=%+v", user, session)
	}
}

func TestAuthService_Login_SessionServiceFailsToCreateSession_PropagatesErrorAndDoesNotReturnUser(t *testing.T) {
	// Arrange
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password for test: %v", err)
	}

	wantErr := errors.New("session storage unavailable")
	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(context.Context, string) (domain.User, error) {
			return domain.User{
				ID:       8,
				Username: "admin",
				Password: string(hash),
			}, nil
		},
	}
	sessions := &testutil.SessionService{
		CreateSessionFunc: func(context.Context, uint) (domain.Session, error) {
			return domain.Session{}, wantErr
		},
	}
	service := authservice.NewAuthService(users, sessions)

	// Act
	user, session, err := service.Login(context.Background(), "admin", "secret123")

	// Assert
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got: %v", wantErr, err)
	}
	if user != (domain.User{}) || session != (domain.Session{}) {
		t.Fatalf("expected empty user and session on session failure, got: user=%+v, session=%+v", user, session)
	}
}

func TestAuthService_Login_ContextProvided_PropagatesContextToDependencies(t *testing.T) {
	// Arrange
	type ctxKey string
	key := ctxKey("trace-id")
	expectedCtx := context.WithValue(context.Background(), key, "trace-xyz-999")

	var receivedRepoCtx, receivedSessionCtx context.Context

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password for test: %v", err)
	}

	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(ctx context.Context, _ string) (domain.User, error) {
			receivedRepoCtx = ctx
			return domain.User{
				ID:       10,
				Username: "admin",
				Password: string(hash),
			}, nil
		},
	}
	sessions := &testutil.SessionService{
		CreateSessionFunc: func(ctx context.Context, _ uint) (domain.Session, error) {
			receivedSessionCtx = ctx
			return domain.Session{
				ID:          1,
				OpaqueToken: "token",
				UserID:      10,
			}, nil
		},
	}
	service := authservice.NewAuthService(users, sessions)

	// Act
	_, _, err = service.Login(expectedCtx, "admin", "secret123")

	// Assert
	if err != nil {
		t.Fatalf("unexpected Login() error: %v", err)
	}
	if receivedRepoCtx == nil || receivedRepoCtx.Value(key) != "trace-xyz-999" {
		t.Fatalf("expected repo to receive propagated context with trace-id, got: %v", receivedRepoCtx)
	}
	if receivedSessionCtx == nil || receivedSessionCtx.Value(key) != "trace-xyz-999" {
		t.Fatalf("expected session service to receive propagated context with trace-id, got: %v", receivedSessionCtx)
	}
}

func TestAuthService_Logout_ValidToken_CallsSessionServiceRevokeSuccessfully(t *testing.T) {
	// Arrange
	var revokedToken string
	sessions := &testutil.SessionService{
		RevokeFunc: func(_ context.Context, token string) error {
			revokedToken = token
			return nil
		},
	}
	service := authservice.NewAuthService(&testutil.UserRepository{}, sessions)

	// Act
	err := service.Logout(context.Background(), "   opaque-token-12345   ")

	// Assert
	if err != nil {
		t.Fatalf("unexpected Logout() error: %v", err)
	}
	if revokedToken != "opaque-token-12345" {
		t.Fatalf("expected trimmed revoked token 'opaque-token-12345', got: %q", revokedToken)
	}
}

func TestAuthService_Logout_EmptyOrWhitespaceToken_ReturnsNilWithoutCallingSessionService(t *testing.T) {
	// Arrange
	sessionRevoked := false
	sessions := &testutil.SessionService{
		RevokeFunc: func(context.Context, string) error {
			sessionRevoked = true
			return nil
		},
	}
	service := authservice.NewAuthService(&testutil.UserRepository{}, sessions)

	// Act
	errEmpty := service.Logout(context.Background(), "")
	errWhitespace := service.Logout(context.Background(), "     ")

	// Assert
	if errEmpty != nil {
		t.Fatalf("expected nil for empty token logout, got: %v", errEmpty)
	}
	if errWhitespace != nil {
		t.Fatalf("expected nil for whitespace token logout, got: %v", errWhitespace)
	}
	if sessionRevoked {
		t.Fatal("session service Revoke must not be called when token is empty or whitespace")
	}
}

func TestAuthService_Logout_SessionServiceRevokeFails_PropagatesError(t *testing.T) {
	// Arrange
	wantErr := errors.New("revoke failed in database")
	sessions := &testutil.SessionService{
		RevokeFunc: func(context.Context, string) error {
			return wantErr
		},
	}
	service := authservice.NewAuthService(&testutil.UserRepository{}, sessions)

	// Act
	err := service.Logout(context.Background(), "opaque-token-12345")

	// Assert
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got: %v", wantErr, err)
	}
}
