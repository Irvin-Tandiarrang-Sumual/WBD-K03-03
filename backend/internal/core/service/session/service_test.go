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

	service := sessionservice.NewSessionService(repo)

	session, err := service.CreateSession(
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

	if saved.RotatedAt.IsZero() {
		t.Fatal("rotated_at must be initialized on creation")
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
				RotatedAt:          now.Add(-time.Minute),
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

	service := sessionservice.NewSessionService(repo)

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

func TestResolveRotatesTokenAfterIntervalWithoutExtendingAbsoluteExpiration(t *testing.T) {
	now := time.Now().UTC()
	originalAbsoluteExp := now.Add(12 * time.Hour)
	var rotatedSession domain.Session

	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                 7,
				OpaqueToken:        token,
				UserID:             15,
				RotatedAt:          now.Add(-(sessionservice.DefaultRotationInterval + time.Minute)),
				IdleTimeout:        30 * time.Minute,
				LastActive:         now.Add(-5 * time.Minute),
				AbsoluteExpiration: originalAbsoluteExp,
			}, nil
		},
		RotateFunc: func(_ context.Context, session domain.Session) error {
			rotatedSession = session
			return nil
		},
	}

	service := sessionservice.NewSessionService(repo)
	got, err := service.Resolve(context.Background(), "old-token-123")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got.OpaqueToken == "" || got.OpaqueToken == "old-token-123" {
		t.Fatalf("expected new rotated token, got %q", got.OpaqueToken)
	}
	if got.PreviousOpaqueToken != "old-token-123" || rotatedSession.PreviousOpaqueToken != "old-token-123" {
		t.Fatalf("previous token = %q, want old-token-123", got.PreviousOpaqueToken)
	}
	if !got.AbsoluteExpiration.Equal(originalAbsoluteExp) {
		t.Fatalf("AbsoluteExpiration = %v, want unchanged %v", got.AbsoluteExpiration, originalAbsoluteExp)
	}
	if !got.GracePeriodUntil.After(now) {
		t.Fatalf("expected GracePeriodUntil after now, got %v", got.GracePeriodUntil)
	}
}

func TestResolveAllowsPreviousTokenWithinGracePeriod(t *testing.T) {
	now := time.Now().UTC()
	touched := false

	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, _ string) (domain.Session, error) {
			return domain.Session{
				ID:                  11,
				OpaqueToken:         "new-rotated-token",
				PreviousOpaqueToken: "old-grace-token",
				GracePeriodUntil:    now.Add(20 * time.Second),
				RotatedAt:           now.Add(-10 * time.Second),
				IdleTimeout:         30 * time.Minute,
				LastActive:          now.Add(-10 * time.Second),
				AbsoluteExpiration:  now.Add(time.Hour),
				UserID:              21,
			}, nil
		},
		TouchFunc: func(_ context.Context, id uint, _ time.Time) error {
			touched = true
			if id != 11 {
				t.Fatalf("id = %d, want 11", id)
			}
			return nil
		},
	}

	service := sessionservice.NewSessionService(repo)
	got, err := service.Resolve(context.Background(), "old-grace-token")
	if err != nil {
		t.Fatalf("Resolve() error = %v, want valid during grace period", err)
	}
	if !touched {
		t.Fatal("expected Touch() to be called when resolving via grace period")
	}
	if got.OpaqueToken != "new-rotated-token" || got.UserID != 21 {
		t.Fatalf("got = %+v, want new-rotated-token for user 21", got)
	}
}

func TestResolveRejectsPreviousTokenAfterGracePeriodExpires(t *testing.T) {
	now := time.Now().UTC()
	deleted := false

	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, _ string) (domain.Session, error) {
			return domain.Session{
				ID:                  12,
				OpaqueToken:         "new-rotated-token",
				PreviousOpaqueToken: "expired-grace-token",
				GracePeriodUntil:    now.Add(-5 * time.Second),
				RotatedAt:           now.Add(-35 * time.Second),
				IdleTimeout:         30 * time.Minute,
				LastActive:          now.Add(-5 * time.Second),
				AbsoluteExpiration:  now.Add(time.Hour),
				UserID:              21,
			}, nil
		},
		DeleteByTokenFunc: func(context.Context, string) error {
			deleted = true
			return nil
		},
	}

	service := sessionservice.NewSessionService(repo)
	_, err := service.Resolve(context.Background(), "expired-grace-token")
	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("error = %v, want ErrInvalidSession after grace period expires", err)
	}
	if deleted {
		t.Fatal("active session must not be deleted when only previous token grace period expired")
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

	service := sessionservice.NewSessionService(repo)

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

	service := sessionservice.NewSessionService(repo)

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
	service := sessionservice.NewSessionService(
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

	service := sessionservice.NewSessionService(repo)

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

func TestCreatePropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("create session failed")
	repo := &testutil.SessionRepository{
		CreateFunc: func(context.Context, domain.Session) (domain.Session, error) {
			return domain.Session{}, wantErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	_, err := service.CreateSession(context.Background(), 5)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
}

func TestResolvePropagatesFindError(t *testing.T) {
	wantErr := errors.New("lookup failed")
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(context.Context, string) (domain.Session, error) {
			return domain.Session{}, wantErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	_, err := service.Resolve(context.Background(), "token")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Resolve() error = %v, want %v", err, wantErr)
	}
}

func TestResolvePropagatesExpiredSessionDeleteError(t *testing.T) {
	now := time.Now().UTC()
	wantErr := errors.New("delete failed")
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(context.Context, string) (domain.Session, error) {
			return domain.Session{
				ID:                 1,
				IdleTimeout:        time.Minute,
				LastActive:         now.Add(-2 * time.Minute),
				AbsoluteExpiration: now.Add(time.Hour),
			}, nil
		},
		DeleteByTokenFunc: func(context.Context, string) error {
			return wantErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	_, err := service.Resolve(context.Background(), "token")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Resolve() error = %v, want %v", err, wantErr)
	}
}

func TestResolvePropagatesTouchError(t *testing.T) {
	now := time.Now().UTC()
	wantErr := errors.New("touch failed")
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(context.Context, string) (domain.Session, error) {
			return domain.Session{
				ID:                 1,
				IdleTimeout:        time.Hour,
				LastActive:         now,
				AbsoluteExpiration: now.Add(time.Hour),
			}, nil
		},
		TouchFunc: func(context.Context, uint, time.Time) error {
			return wantErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	_, err := service.Resolve(context.Background(), "token")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Resolve() error = %v, want %v", err, wantErr)
	}
}

func TestRevokeBlankTokenDoesNotCallRepository(t *testing.T) {
	called := false
	repo := &testutil.SessionRepository{
		DeleteByTokenFunc: func(context.Context, string) error {
			called = true
			return nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	if err := service.Revoke(context.Background(), "   "); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if called {
		t.Fatal("repository must not be called for blank token")
	}
}

func TestRevokeAndRevokeUserPropagateRepositoryErrors(t *testing.T) {
	tokenErr := errors.New("token revoke failed")
	userErr := errors.New("user revoke failed")
	repo := &testutil.SessionRepository{
		DeleteByTokenFunc:  func(context.Context, string) error { return tokenErr },
		DeleteByUserIDFunc: func(context.Context, uint) error { return userErr },
	}
	service := sessionservice.NewSessionService(repo)

	if err := service.Revoke(context.Background(), "token"); !errors.Is(err, tokenErr) {
		t.Fatalf("Revoke() error = %v, want %v", err, tokenErr)
	}
	if err := service.RevokeUser(context.Background(), 9); !errors.Is(err, userErr) {
		t.Fatalf("RevokeUser() error = %v, want %v", err, userErr)
	}
}
