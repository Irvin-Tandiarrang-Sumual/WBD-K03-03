package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	authservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/auth"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginSuccess(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(_ context.Context, username string) (domain.User, error) {
			if username != "admin" {
				t.Fatalf("username = %q", username)
			}

			return domain.User{
				ID:       7,
				Username: "admin",
				Password: string(hash),
				Role:     domain.RoleAdmin,
			}, nil
		},
	}

	created := false

	sessions := &testutil.SessionService{
		CreateFunc: func(_ context.Context, userID uint) (domain.Session, error) {
			created = true

			if userID != 7 {
				t.Fatalf("userID = %d, want 7", userID)
			}

			return domain.Session{
				ID:                 1,
				OpaqueToken:        "token",
				UserID:             userID,
				AbsoluteExpiration: time.Now().Add(time.Hour),
			}, nil
		},
	}

	service := 	authservice.NewAuthService(users, sessions)

	user, session, err := service.Login(
		context.Background(),
		" admin ",
		"secret123",
	)

	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if !created {
		t.Fatal("session was not created")
	}

	if user.Password != "" {
		t.Fatal("password hash leaked from Login()")
	}

	if user.ID != 7 || session.UserID != 7 {
		t.Fatalf(
			"unexpected user/session: %+v %+v",
			user,
			session,
		)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword(
		[]byte("correct-password"),
		bcrypt.MinCost,
	)

	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(
			context.Context,
			string,
		) (domain.User, error) {
			return domain.User{
				ID:       1,
				Password: string(hash),
			}, nil
		},
	}

	sessionCreated := false

	sessions := &testutil.SessionService{
		CreateFunc: func(
			context.Context,
			uint,
		) (domain.Session, error) {
			sessionCreated = true
			return domain.Session{}, nil
		},
	}

	service := authservice.NewAuthService(users, sessions)

	_, _, err := service.Login(
		context.Background(),
		"admin",
		"wrong-password",
	)

	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf(
			"error = %v, want ErrInvalidCredentials",
			err,
		)
	}

	if sessionCreated {
		t.Fatal(
			"session must not be created for invalid password",
		)
	}
}

func TestLoginHidesUnknownUsername(t *testing.T) {
	users := &testutil.UserRepository{
		FindByUsernameWithPasswordFunc: func(
			context.Context,
			string,
		) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}

	sessions := &testutil.SessionService{
		CreateFunc: func(
			context.Context,
			uint,
		) (domain.Session, error) {
			return domain.Session{}, nil
		},
	}

	service := authservice.NewAuthService(users, sessions)

	_, _, err := service.Login(
		context.Background(),
		"missing",
		"anything",
	)

	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf(
			"error = %v, want ErrInvalidCredentials",
			err,
		)
	}
}

func TestLogoutRevokesCurrentToken(t *testing.T) {
	var revoked string

	sessions := &testutil.SessionService{
		RevokeFunc: func(
			_ context.Context,
			token string,
		) error {
			revoked = token
			return nil
		},
	}

	service := authservice.NewAuthService(
		&testutil.UserRepository{},
		sessions,
	)

	if err := service.Logout(
		context.Background(),
		"opaque-token",
	); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if revoked != "opaque-token" {
		t.Fatalf("revoked token = %q", revoked)
	}
}