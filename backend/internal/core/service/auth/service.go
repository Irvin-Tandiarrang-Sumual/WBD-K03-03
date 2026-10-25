package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"golang.org/x/crypto/bcrypt"
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

	if username == "" || password == "" {
		return domain.User{}, domain.Session{}, domain.ErrInvalidCredentials
	}

	user, err := s.users.FindByUsernameWithPassword(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.User{}, domain.Session{}, domain.ErrInvalidCredentials
		}

		return domain.User{}, domain.Session{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return domain.User{}, domain.Session{}, domain.ErrInvalidCredentials
	}

	session, err := s.sessions.Create(ctx, user.ID)
	if err != nil {
		return domain.User{}, domain.Session{}, err
	}

	user.Password = ""

	return user, session, nil
}

func (s *authService) Logout(ctx context.Context, opaqueToken string) error {
	return s.sessions.Revoke(ctx, opaqueToken)
}