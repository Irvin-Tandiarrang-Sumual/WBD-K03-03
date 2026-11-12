package session_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

func TestCreateSession_ValidUserID_GeneratesNonEmptyOpaqueToken(t *testing.T) {
	// Arrange
	var savedSession domain.Session
	repo := &testutil.SessionRepository{
		SaveFunc: func(_ context.Context, s domain.Session) (domain.Session, error) {
			savedSession = s
			return s, nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.CreateSession(context.Background(), 10)

	// Assert
	if err != nil || savedSession.OpaqueToken == "" {
		t.Fatalf("expected non-empty OpaqueToken, got '%s', err: %v", savedSession.OpaqueToken, err)
	}
}

func TestCreateSession_ValidUserID_GeneratesNonEmptyCSRFToken(t *testing.T) {
	// Arrange
	var savedSession domain.Session
	repo := &testutil.SessionRepository{
		SaveFunc: func(_ context.Context, s domain.Session) (domain.Session, error) {
			savedSession = s
			return s, nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.CreateSession(context.Background(), 10)

	// Assert
	if err != nil || savedSession.CSRFToken == "" {
		t.Fatalf("expected non-empty CSRFToken, got '%s', err: %v", savedSession.CSRFToken, err)
	}
}

func TestCreateSession_ValidUserID_SetsUserID(t *testing.T) {
	// Arrange
	var savedSession domain.Session
	repo := &testutil.SessionRepository{
		SaveFunc: func(_ context.Context, s domain.Session) (domain.Session, error) {
			savedSession = s
			return s, nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.CreateSession(context.Background(), 42)

	// Assert
	if err != nil || savedSession.UserID != 42 {
		t.Fatalf("expected UserID 42, got %d, err: %v", savedSession.UserID, err)
	}
}

func TestCreateSession_ValidUserID_SetsDefaultIdleTimeout(t *testing.T) {
	// Arrange
	var savedSession domain.Session
	repo := &testutil.SessionRepository{
		SaveFunc: func(_ context.Context, s domain.Session) (domain.Session, error) {
			savedSession = s
			return s, nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.CreateSession(context.Background(), 1)

	// Assert
	if err != nil || savedSession.IdleTimeout != domain.DefaultSessionIdleTimeout {
		t.Fatalf("expected IdleTimeout %v, got %v, err: %v", domain.DefaultSessionIdleTimeout, savedSession.IdleTimeout, err)
	}
}

func TestCreateSession_ValidUserID_SetsFutureAbsoluteExpiration(t *testing.T) {
	// Arrange
	var savedSession domain.Session
	repo := &testutil.SessionRepository{
		SaveFunc: func(_ context.Context, s domain.Session) (domain.Session, error) {
			savedSession = s
			return s, nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	before := time.Now().UTC()
	_, err := service.CreateSession(context.Background(), 1)

	// Assert
	expectedMinExpiry := before.Add(domain.DefaultSessionAbsoluteLifetime - time.Minute)
	if err != nil || !savedSession.AbsoluteExpiration.After(expectedMinExpiry) {
		t.Fatalf("expected AbsoluteExpiration to be in the future (around 24h), got %v, err: %v", savedSession.AbsoluteExpiration, err)
	}
}

func TestCreateSession_ValidUserID_SetsRotatedAt(t *testing.T) {
	// Arrange
	var savedSession domain.Session
	repo := &testutil.SessionRepository{
		SaveFunc: func(_ context.Context, s domain.Session) (domain.Session, error) {
			savedSession = s
			return s, nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.CreateSession(context.Background(), 1)

	// Assert
	if err != nil || savedSession.RotatedAt.IsZero() {
		t.Fatalf("expected non-zero RotatedAt timestamp, got %v, err: %v", savedSession.RotatedAt, err)
	}
}

func TestCreateSession_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &testutil.SessionRepository{
		SaveFunc: func(_ context.Context, _ domain.Session) (domain.Session, error) {
			return domain.Session{}, expectedErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.CreateSession(context.Background(), 1)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestCreateSession_ValidInput_ReturnsSavedSession(t *testing.T) {
	// Arrange
	expectedSession := domain.Session{
		ID:          99,
		OpaqueToken: "dummy-token",
		UserID:      1,
	}
	repo := &testutil.SessionRepository{
		SaveFunc: func(_ context.Context, _ domain.Session) (domain.Session, error) {
			return expectedSession, nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	session, err := service.CreateSession(context.Background(), 1)

	// Assert
	if err != nil || !reflect.DeepEqual(session, expectedSession) {
		t.Fatalf("expected session %+v, got %+v, err: %v", expectedSession, session, err)
	}
}

func TestResolve_EmptyToken_ReturnsErrInvalidSession(t *testing.T) {
	// Arrange
	service := sessionservice.NewSessionService(&testutil.SessionRepository{})

	// Act
	_, err := service.Resolve(context.Background(), "")

	// Assert
	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("expected ErrInvalidSession, got %v", err)
	}
}

func TestResolve_WhitespaceOnlyToken_ReturnsErrInvalidSession(t *testing.T) {
	// Arrange
	service := sessionservice.NewSessionService(&testutil.SessionRepository{})

	// Act
	_, err := service.Resolve(context.Background(), "   \t  \n  ")

	// Assert
	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("expected ErrInvalidSession, got %v", err)
	}
}

func TestResolve_DataFetchFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("repository query error")
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, _ string) (domain.Session, error) {
			return domain.Session{}, expectedErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.Resolve(context.Background(), "some-token")

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestResolve_AbsoluteExpiredSession_PurgesSessionAndReturnsErrInvalidSession(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	purgedToken := ""
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                 1,
				OpaqueToken:        token,
				AbsoluteExpiration: now.Add(-1 * time.Hour), // Expired 1 hour ago
				LastActive:         now.Add(-10 * time.Minute),
				IdleTimeout:        30 * time.Minute,
			}, nil
		},
		DeleteByTokenFunc: func(_ context.Context, token string) error {
			purgedToken = token
			return nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.Resolve(context.Background(), "expired-token")

	// Assert
	if !errors.Is(err, domain.ErrInvalidSession) || purgedToken != "expired-token" {
		t.Fatalf("expected ErrInvalidSession and purged token 'expired-token', got err: %v, purged: '%s'", err, purgedToken)
	}
}

func TestResolve_IdleExpiredSession_PurgesSessionAndReturnsErrInvalidSession(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	purgedToken := ""
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                 1,
				OpaqueToken:        token,
				AbsoluteExpiration: now.Add(10 * time.Hour),
				LastActive:         now.Add(-45 * time.Minute), // Inactive for 45 min (> 30 min idle timeout)
				IdleTimeout:        30 * time.Minute,
			}, nil
		},
		DeleteByTokenFunc: func(_ context.Context, token string) error {
			purgedToken = token
			return nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.Resolve(context.Background(), "idle-expired-token")

	// Assert
	if !errors.Is(err, domain.ErrInvalidSession) || purgedToken != "idle-expired-token" {
		t.Fatalf("expected ErrInvalidSession and purged token 'idle-expired-token', got err: %v, purged: '%s'", err, purgedToken)
	}
}

func TestResolve_SessionPurgeFailure_ReturnsError(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	expectedErr := errors.New("delete token database error")
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                 1,
				OpaqueToken:        token,
				AbsoluteExpiration: now.Add(-1 * time.Hour), // Expired
			}, nil
		},
		DeleteByTokenFunc: func(_ context.Context, _ string) error {
			return expectedErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.Resolve(context.Background(), "expired-token")

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestResolve_PreviousTokenWithinGracePeriod_UpdatesLastActiveAndReturnsSession(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	touchedSessionID := uint(0)
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                  10,
				OpaqueToken:         "new-token",
				PreviousOpaqueToken: token,
				GracePeriodUntil:    now.Add(15 * time.Second), // Within grace period
				RotatedAt:           now.Add(-1 * time.Minute),
				LastActive:          now.Add(-1 * time.Minute),
				IdleTimeout:         30 * time.Minute,
				AbsoluteExpiration:  now.Add(12 * time.Hour),
			}, nil
		},
		TouchFunc: func(_ context.Context, id uint, _ time.Time) error {
			touchedSessionID = id
			return nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	session, err := service.Resolve(context.Background(), "old-token")

	// Assert
	if err != nil || touchedSessionID != 10 || session.ID != 10 {
		t.Fatalf("expected session 10 to be returned with activity touched, got id: %d, touched: %d, err: %v", session.ID, touchedSessionID, err)
	}
}

func TestResolve_PreviousTokenPastGracePeriod_ReturnsErrInvalidSession(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                  10,
				OpaqueToken:         "new-token",
				PreviousOpaqueToken: token,
				GracePeriodUntil:    now.Add(-5 * time.Second), // Grace period passed
				RotatedAt:           now.Add(-2 * time.Minute),
				LastActive:          now.Add(-2 * time.Minute),
				IdleTimeout:         30 * time.Minute,
				AbsoluteExpiration:  now.Add(12 * time.Hour),
			}, nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.Resolve(context.Background(), "old-token")

	// Assert
	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("expected ErrInvalidSession, got %v", err)
	}
}

func TestResolve_PreviousTokenActivityUpdateFailure_ReturnsError(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	expectedErr := errors.New("touch failure")
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                  10,
				OpaqueToken:         "new-token",
				PreviousOpaqueToken: token,
				GracePeriodUntil:    now.Add(15 * time.Second),
				RotatedAt:           now.Add(-1 * time.Minute),
				LastActive:          now.Add(-1 * time.Minute),
				IdleTimeout:         30 * time.Minute,
				AbsoluteExpiration:  now.Add(12 * time.Hour),
			}, nil
		},
		TouchFunc: func(_ context.Context, _ uint, _ time.Time) error {
			return expectedErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.Resolve(context.Background(), "old-token")

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestResolve_SessionNeedsRotation_RotatesSessionToken(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	var capturedRotatedSession domain.Session
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                 5,
				OpaqueToken:        token,
				RotatedAt:          now.Add(-20 * time.Minute), // > 15 minutes rotation interval
				LastActive:         now.Add(-1 * time.Minute),
				IdleTimeout:        30 * time.Minute,
				AbsoluteExpiration: now.Add(12 * time.Hour),
			}, nil
		},
		RotateFunc: func(_ context.Context, s domain.Session) (bool, error) {
			capturedRotatedSession = s
			return true, nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	resolved, err := service.Resolve(context.Background(), "current-token")

	// Assert
	if err != nil || resolved.OpaqueToken == "current-token" || capturedRotatedSession.OpaqueToken == "" {
		t.Fatalf("expected session token to be rotated, got new token: '%s', err: %v", resolved.OpaqueToken, err)
	}
}

func TestResolve_SessionNeedsRotation_SetsPreviousTokenAndGracePeriod(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	var capturedRotatedSession domain.Session
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                 5,
				OpaqueToken:        token,
				RotatedAt:          now.Add(-20 * time.Minute),
				LastActive:         now.Add(-1 * time.Minute),
				IdleTimeout:        30 * time.Minute,
				AbsoluteExpiration: now.Add(12 * time.Hour),
			}, nil
		},
		RotateFunc: func(_ context.Context, s domain.Session) (bool, error) {
			capturedRotatedSession = s
			return true, nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.Resolve(context.Background(), "current-token")

	// Assert
	if err != nil || capturedRotatedSession.PreviousOpaqueToken != "current-token" || capturedRotatedSession.GracePeriodUntil.IsZero() {
		t.Fatalf("expected previous token 'current-token' and valid grace period, got prev: '%s', grace: %v, err: %v",
			capturedRotatedSession.PreviousOpaqueToken, capturedRotatedSession.GracePeriodUntil, err)
	}
}

func TestResolve_SessionNeedsRotation_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	expectedErr := errors.New("rotate database error")
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                 5,
				OpaqueToken:        token,
				RotatedAt:          now.Add(-20 * time.Minute),
				LastActive:         now.Add(-1 * time.Minute),
				IdleTimeout:        30 * time.Minute,
				AbsoluteExpiration: now.Add(12 * time.Hour),
			}, nil
		},
		RotateFunc: func(_ context.Context, _ domain.Session) (bool, error) {
			return false, expectedErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.Resolve(context.Background(), "current-token")

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestResolve_RotationConflict_RefetchesLatestSession(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	latestSession := domain.Session{
		ID:          5,
		OpaqueToken: "newest-parallel-token",
	}
	callCount := 0
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			callCount++
			if callCount == 1 {
				return domain.Session{
					ID:                 5,
					OpaqueToken:        token,
					RotatedAt:          now.Add(-20 * time.Minute),
					LastActive:         now.Add(-1 * time.Minute),
					IdleTimeout:        30 * time.Minute,
					AbsoluteExpiration: now.Add(12 * time.Hour),
				}, nil
			}
			return latestSession, nil
		},
		RotateFunc: func(_ context.Context, _ domain.Session) (bool, error) {
			return false, nil // Rotation conflict: another request already rotated it
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	resolved, err := service.Resolve(context.Background(), "current-token")

	// Assert
	if err != nil || resolved.OpaqueToken != "newest-parallel-token" {
		t.Fatalf("expected refetched session with token 'newest-parallel-token', got '%s', err: %v", resolved.OpaqueToken, err)
	}
}

func TestResolve_ValidActiveSession_TouchesActivityAndUpdatesLastActive(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	touchedSessionID := uint(0)
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                 10,
				OpaqueToken:        token,
				RotatedAt:          now.Add(-5 * time.Minute), // < 15 min, no rotation needed
				LastActive:         now.Add(-2 * time.Minute), // < 30 min, not idle
				IdleTimeout:        30 * time.Minute,
				AbsoluteExpiration: now.Add(10 * time.Hour), // Not expired
			}, nil
		},
		TouchFunc: func(_ context.Context, id uint, _ time.Time) error {
			touchedSessionID = id
			return nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	resolved, err := service.Resolve(context.Background(), "active-token")

	// Assert
	if err != nil || touchedSessionID != 10 || !resolved.LastActive.After(now.Add(-1*time.Minute)) {
		t.Fatalf("expected active session 10 to touch activity and update LastActive, touchedID: %d, err: %v", touchedSessionID, err)
	}
}

func TestResolve_ActivityUpdateFailure_ReturnsError(t *testing.T) {
	// Arrange
	now := time.Now().UTC()
	expectedErr := errors.New("touch failure")
	repo := &testutil.SessionRepository{
		FindByTokenFunc: func(_ context.Context, token string) (domain.Session, error) {
			return domain.Session{
				ID:                 10,
				OpaqueToken:        token,
				RotatedAt:          now.Add(-5 * time.Minute),
				LastActive:         now.Add(-2 * time.Minute),
				IdleTimeout:        30 * time.Minute,
				AbsoluteExpiration: now.Add(10 * time.Hour),
			}, nil
		},
		TouchFunc: func(_ context.Context, _ uint, _ time.Time) error {
			return expectedErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	_, err := service.Resolve(context.Background(), "active-token")

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRevoke_EmptyToken_ReturnsNilWithoutCallingRepository(t *testing.T) {
	// Arrange
	deleteCalled := false
	repo := &testutil.SessionRepository{
		DeleteByTokenFunc: func(_ context.Context, _ string) error {
			deleteCalled = true
			return nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	err := service.Revoke(context.Background(), "")

	// Assert
	if err != nil || deleteCalled {
		t.Fatalf("expected nil and no repository call for empty token, got deleteCalled: %v, err: %v", deleteCalled, err)
	}
}

func TestRevoke_WhitespaceOnlyToken_ReturnsNilWithoutCallingRepository(t *testing.T) {
	// Arrange
	deleteCalled := false
	repo := &testutil.SessionRepository{
		DeleteByTokenFunc: func(_ context.Context, _ string) error {
			deleteCalled = true
			return nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	err := service.Revoke(context.Background(), "   \t  \n  ")

	// Assert
	if err != nil || deleteCalled {
		t.Fatalf("expected nil and no repository call for whitespace token, got deleteCalled: %v, err: %v", deleteCalled, err)
	}
}

func TestRevoke_ValidToken_CallsDeleteByToken(t *testing.T) {
	// Arrange
	deletedToken := ""
	repo := &testutil.SessionRepository{
		DeleteByTokenFunc: func(_ context.Context, token string) error {
			deletedToken = token
			return nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	err := service.Revoke(context.Background(), "  token-to-revoke  ")

	// Assert
	if err != nil || deletedToken != "token-to-revoke" {
		t.Fatalf("expected token 'token-to-revoke' to be deleted, got '%s', err: %v", deletedToken, err)
	}
}

func TestRevoke_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("delete token database error")
	repo := &testutil.SessionRepository{
		DeleteByTokenFunc: func(_ context.Context, _ string) error {
			return expectedErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	err := service.Revoke(context.Background(), "some-token")

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRevokeUser_ValidUserID_CallsDeleteByUserID(t *testing.T) {
	// Arrange
	deletedUserID := uint(0)
	repo := &testutil.SessionRepository{
		DeleteByUserIDFunc: func(_ context.Context, userID uint) error {
			deletedUserID = userID
			return nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	err := service.RevokeUser(context.Background(), 77)

	// Assert
	if err != nil || deletedUserID != 77 {
		t.Fatalf("expected user ID 77 to be deleted, got %d, err: %v", deletedUserID, err)
	}
}

func TestRevokeUser_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("delete by user ID error")
	repo := &testutil.SessionRepository{
		DeleteByUserIDFunc: func(_ context.Context, _ uint) error {
			return expectedErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	err := service.RevokeUser(context.Background(), 77)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRevokeUsers_ValidUserIDs_CallsDeleteByUserIDs(t *testing.T) {
	// Arrange
	var deletedUserIDs []uint
	repo := &testutil.SessionRepository{
		DeleteByUserIDsFunc: func(_ context.Context, userIDs []uint) error {
			deletedUserIDs = userIDs
			return nil
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	err := service.RevokeUsers(context.Background(), []uint{1, 2, 3})

	// Assert
	expectedIDs := []uint{1, 2, 3}
	if err != nil || !reflect.DeepEqual(deletedUserIDs, expectedIDs) {
		t.Fatalf("expected user IDs %v to be deleted, got %v, err: %v", expectedIDs, deletedUserIDs, err)
	}
}

func TestRevokeUsers_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("delete by user IDs error")
	repo := &testutil.SessionRepository{
		DeleteByUserIDsFunc: func(_ context.Context, _ []uint) error {
			return expectedErr
		},
	}
	service := sessionservice.NewSessionService(repo)

	// Act
	err := service.RevokeUsers(context.Background(), []uint{1, 2, 3})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
