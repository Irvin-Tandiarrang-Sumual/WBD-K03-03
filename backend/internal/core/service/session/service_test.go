package session_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

func TestCreateBuildsOpaqueSession(t *testing.T) {
	var saved domain.Session

	repo := &testutil.SessionRepository{
		CreateFunc: func(
			_ context.Context,
			session domain.Session,
		) (domain.Session, error) {
			saved = session
			session.ID = 10

			return session, nil
		},
	}

	service := sessionservice.NewService(repo)

	session, err := service.Create(
		context.Background(),
		5,
	)

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if session.ID != 10 || saved.UserID != 5 {
		t.Fatalf(
			"unexpected session: %+v",
			session,
		)
	}

	if saved.OpaqueToken == "" {
		t.Fatal("opaque token must not be empty")
	}

	if saved.IdleTimeout != sessionservice.DefaultIdleTimeout {
		t.Fatalf(
			"idle timeout = %v",
			saved.IdleTimeout,
		)
	}

	if !saved.AbsoluteExpiration.After(saved.LastActive) {
		t.Fatal(
			"absolute expiration must be after last active",
		)
	}
}

func TestResolveValidSessionTouchesLastActive(t *testing.T) {
	now := time.Now().UTC()
	touched := false

	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(
			_ context.Context,
			token string,
		) (domain.Session, error) {
			return domain.Session{
				ID:                 3,
				OpaqueToken:        token,
				UserID:             9,
				IdleTimeout:        30 * time.Minute,
				LastActive:         now.Add(-time.Minute),
				AbsoluteExpiration: now.Add(time.Hour),
			}, nil
		},

		TouchFunc: func(
			_ context.Context,
			id uint,
			lastActive time.Time,
		) error {
			touched = true

			if id != 3 {
				t.Fatalf(
					"id = %d, want 3",
					id,
				)
			}

			if !lastActive.After(now.Add(-time.Second)) {
				t.Fatalf(
					"lastActive not refreshed: %v",
					lastActive,
				)
			}

			return nil
		},
	}

	service := sessionservice.NewService(repo)

	got, err := service.Resolve(
		context.Background(),
		"token",
	)

	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if !touched {
		t.Fatal("Touch() was not called")
	}

	if got.UserID != 9 {
		t.Fatalf(
			"UserID = %d, want 9",
			got.UserID,
		)
	}
}

func TestResolveRejectsIdleExpiredSession(t *testing.T) {
	now := time.Now().UTC()
	deleted := false

	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(
			context.Context,
			string,
		) (domain.Session, error) {
			return domain.Session{
				ID:                 4,
				IdleTimeout:        30 * time.Minute,
				LastActive:         now.Add(-31 * time.Minute),
				AbsoluteExpiration: now.Add(time.Hour),
			}, nil
		},

		DeleteByTokenFunc: func(
			_ context.Context,
			token string,
		) error {
			if token != "token" {
				t.Fatalf("deleted token = %q, want %q", token, "token")
			}

			deleted = true
			return nil
		},
	}

	service := sessionservice.NewService(repo)

	_, err := service.Resolve(
		context.Background(),
		"token",
	)

	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf(
			"error = %v, want ErrInvalidSession",
			err,
		)
	}

	if !deleted {
		t.Fatal("expired session was not deleted")
	}
}

func TestResolveRejectsAbsoluteExpiredSession(t *testing.T) {
	now := time.Now().UTC()

	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(
			context.Context,
			string,
		) (domain.Session, error) {
			return domain.Session{
				ID:                 4,
				IdleTimeout:        30 * time.Minute,
				LastActive:         now,
				AbsoluteExpiration: now.Add(-time.Second),
			}, nil
		},

		// Absolute-expired session juga dibuang berdasarkan
		// opaque token yang diterima Resolve().
		DeleteByTokenFunc: func(
			_ context.Context,
			token string,
		) error {
			if token != "token" {
				t.Fatalf(
					"deleted token = %q, want %q",
					token,
					"token",
				)
			}

			return nil
		},
	}

	service := sessionservice.NewService(repo)

	_, err := service.Resolve(
		context.Background(),
		"token",
	)

	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf(
			"error = %v, want ErrInvalidSession",
			err,
		)
	}
}

func TestResolveRejectsEmptyToken(t *testing.T) {
	service := sessionservice.NewService(
		&testutil.SessionRepository{},
	)

	_, err := service.Resolve(
		context.Background(),
		"   ",
	)

	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf(
			"error = %v, want ErrInvalidSession",
			err,
		)
	}
}

func TestRevokeAndRevokeUser(t *testing.T) {
	var token string
	var userID uint

	repo := &testutil.SessionRepository{
		DeleteByTokenFunc: func(
			_ context.Context,
			value string,
		) error {
			token = value
			return nil
		},

		DeleteByUserIDFunc: func(
			_ context.Context,
			value uint,
		) error {
			userID = value
			return nil
		},
	}

	service := sessionservice.NewService(repo)

	if err := service.Revoke(
		context.Background(),
		"abc",
	); err != nil {
		t.Fatal(err)
	}

	if err := service.RevokeUser(
		context.Background(),
		22,
	); err != nil {
		t.Fatal(err)
	}

	if token != "abc" || userID != 22 {
		t.Fatalf(
			"token=%q userID=%d",
			token,
			userID,
		)
	}
}