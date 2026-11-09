package middleware

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/gin-gonic/gin"
)

const (
	AuthenticatedUserIDKey   = "userID"
	AuthenticatedUserRoleKey = "userRole"
	AuthenticatedSessionKey  = "session"
)

func SetUserContext(c *gin.Context, userID uint, role string) {
	c.Set(AuthenticatedUserIDKey, userID)
	c.Set(AuthenticatedUserRoleKey, role)
}

func SetSessionContext(c *gin.Context, session domain.Session) {
	c.Set(AuthenticatedSessionKey, session)
}

func CurrentSession(c *gin.Context) (domain.Session, error) {
	if val, ok := c.Get(AuthenticatedSessionKey); ok {
		if session, ok := val.(domain.Session); ok {
			return session, nil
		}
	}
	return domain.Session{}, domain.ErrInvalidSession
}

func CurrentCSRFToken(c *gin.Context) string {
	if val, ok := c.Get(AuthenticatedSessionKey); ok {
		if session, ok := val.(domain.Session); ok {
			return session.CSRFToken
		}
	}
	return ""
}

func SetAuthenticatedUserID(c *gin.Context, userID uint) {
	c.Set(AuthenticatedUserIDKey, userID)
}

func CurrentUserID(c *gin.Context) (uint, error) {
	value, exists := c.Get(AuthenticatedUserIDKey)
	if !exists {
		return 0, domain.ErrCurrentUserUnavailable
	}

	userID, ok := value.(uint)
	if !ok || userID == 0 {
		return 0, domain.ErrCurrentUserUnavailable
	}

	return userID, nil
}
