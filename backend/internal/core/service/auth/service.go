package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/hasher"
)

const (
	maxUsernameLength = 255
	dummyBcryptHash   = "$2a$10$EjZ3EaWdv8XR4yUBVaphXO12lsl8jZk2jRF3BHOmYG2qFntsH0x3O"
)

type authService struct {
	users    ports.UserRepository
	sessions ports.SessionService
}

var _ ports.AuthService = (*authService)(nil)

func NewAuthService(users ports.UserRepository, sessions ports.SessionService) ports.AuthService {
	return &authService{
		users:    users,
		sessions: sessions,
	}
}

func (s *authService) Login(ctx context.Context, username string, password string) (domain.User, domain.Session, error) {
	username = strings.TrimSpace(username)

	if username == "" || password == "" || len(username) > maxUsernameLength || len([]byte(password)) > domain.MaxPasswordBytes {
		return domain.User{}, domain.Session{}, domain.ErrInvalidCredentials
	}

	user, err := s.users.FindByUsernameWithPassword(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			_ = hasher.ComparePassword(dummyBcryptHash, password)
			return domain.User{}, domain.Session{}, domain.ErrInvalidCredentials
		}

		return domain.User{}, domain.Session{}, err
	}

	if err := hasher.ComparePassword(user.Password, password); err != nil {
		return domain.User{}, domain.Session{}, domain.ErrInvalidCredentials
	}

	session, err := s.sessions.CreateSession(ctx, user.ID)
	if err != nil {
		return domain.User{}, domain.Session{}, err
	}

	user.Password = ""

	return user, session, nil
}

func (s *authService) Logout(ctx context.Context, opaqueToken string) error {
	token := strings.TrimSpace(opaqueToken)
	if token == "" {
		return nil
	}
	return s.sessions.Revoke(ctx, token)
}
