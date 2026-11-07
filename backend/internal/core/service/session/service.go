package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

const sessionTokenBytes = 32

type sessionService struct {
	sessions ports.SessionRepository
}

var _ ports.SessionService = (*sessionService)(nil)

func NewSessionService(sessions ports.SessionRepository) ports.SessionService {
	return &sessionService{
		sessions: sessions,
	}
}

func (s *sessionService) CreateSession(ctx context.Context, userID uint) (domain.Session, error) {
	opaqueToken, err := generateOpaqueToken()
	if err != nil {
		return domain.Session{}, err
	}

	session := buildNewSession(userID, opaqueToken, time.Now().UTC())
	return s.sessions.Save(ctx, session)
}

func (s *sessionService) Resolve(ctx context.Context, opaqueToken string) (domain.Session, error) {
	cleanToken := strings.TrimSpace(opaqueToken)
	if cleanToken == "" {
		return domain.Session{}, domain.ErrInvalidSession
	}

	session, err := s.sessions.FindByToken(ctx, cleanToken)
	if err != nil {
		return domain.Session{}, err
	}

	now := time.Now().UTC()
	if session.IsExpired(now) {
		return domain.Session{}, s.purgeExpiredSession(ctx, cleanToken)
	}

	if session.IsPreviousToken(cleanToken) {
		return s.resolveGracePeriodSession(ctx, session, now)
	}

	if session.NeedsRotation(now) {
		return s.rotateSessionToken(ctx, session, now)
	}

	return s.updateSessionActivity(ctx, session, now)
}

func (s *sessionService) Revoke(ctx context.Context, opaqueToken string) error {
	cleanToken := strings.TrimSpace(opaqueToken)
	if cleanToken == "" {
		return nil
	}
	return s.sessions.DeleteByToken(ctx, cleanToken)
}

func (s *sessionService) RevokeUser(ctx context.Context, userID uint) error {
	return s.sessions.DeleteByUserID(ctx, userID)
}

func (s *sessionService) RevokeUsers(ctx context.Context, userIDs []uint) error {
	return s.sessions.DeleteByUserIDs(ctx, userIDs)
}

func (s *sessionService) purgeExpiredSession(ctx context.Context, token string) error {
	if err := s.sessions.DeleteByToken(ctx, token); err != nil {
		return err
	}
	return domain.ErrInvalidSession
}

func (s *sessionService) resolveGracePeriodSession(ctx context.Context, session domain.Session, now time.Time) (domain.Session, error) {
	if !session.IsWithinGracePeriod(now) {
		return domain.Session{}, domain.ErrInvalidSession
	}

	return s.updateSessionActivity(ctx, session, now)
}

func (s *sessionService) rotateSessionToken(ctx context.Context, session domain.Session, now time.Time) (domain.Session, error) {
	newToken, err := generateOpaqueToken()
	if err != nil {
		return domain.Session{}, err
	}

	rotatedSession := session.Rotate(newToken, now)
	didRotate, err := s.sessions.Rotate(ctx, rotatedSession)
	if err != nil {
		return domain.Session{}, err
	}
	if !didRotate {
		return s.sessions.FindByToken(ctx, session.OpaqueToken)
	}

	return rotatedSession, nil
}

func (s *sessionService) updateSessionActivity(ctx context.Context, session domain.Session, now time.Time) (domain.Session, error) {
	if err := s.sessions.Touch(ctx, session.ID, now); err != nil {
		return domain.Session{}, err
	}

	session.LastActive = now
	return session, nil
}

func buildNewSession(userID uint, token string, now time.Time) domain.Session {
	return domain.Session{
		OpaqueToken:        token,
		RotatedAt:          now,
		IdleTimeout:        domain.DefaultSessionIdleTimeout,
		LastActive:         now,
		AbsoluteExpiration: now.Add(domain.DefaultSessionAbsoluteLifetime),
		UserID:             userID,
	}
}

func generateOpaqueToken() (string, error) {
	raw := make([]byte, sessionTokenBytes)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
