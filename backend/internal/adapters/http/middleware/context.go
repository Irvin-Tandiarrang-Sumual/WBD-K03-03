package middleware

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/gin-gonic/gin"
)

const (
	AuthenticatedUserIDKey   = "userID"
	AuthenticatedUserRoleKey = "userRole"
)

func SetUserContext(c *gin.Context, userID uint, role string) {
	c.Set(AuthenticatedUserIDKey, userID)
	c.Set(AuthenticatedUserRoleKey, role)
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
