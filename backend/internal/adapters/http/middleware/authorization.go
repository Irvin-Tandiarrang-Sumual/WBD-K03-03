package middleware

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/gin-gonic/gin"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := GetUserID(c)
		if !ok {
			abortUnauthorized(c)
			return
		}

		if !isAdmin(c) {
			abortForbidden(c)
			return
		}

		c.Next()
	}
}

func isAdmin(c *gin.Context) bool {
	userRole, exists := c.Get(AuthenticatedUserRoleKey)
	if !exists {
		return false
	}

	return userRole == domain.RoleAdmin
}
