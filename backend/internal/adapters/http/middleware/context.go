package middleware

import (
	"github.com/gin-gonic/gin"
)

const AuthenticatedUserIDKey = "userID"
const AuthenticatedUserRoleKey = "userRole"

func SetUserContext(c *gin.Context, userID uint, role string) {
	c.Set(AuthenticatedUserIDKey, userID)
	c.Set(AuthenticatedUserRoleKey, role)
}

func GetUserID(c *gin.Context) (uint, bool) {
	value, exists := c.Get(AuthenticatedUserIDKey)
	if !exists {
		return 0, false
	}

	userID, ok := value.(uint)
	if !ok || userID == 0 {
		return 0, false
	}

	return userID, true
}
