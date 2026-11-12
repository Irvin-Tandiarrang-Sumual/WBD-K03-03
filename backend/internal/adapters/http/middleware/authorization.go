package middleware

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/gin-gonic/gin"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := CurrentUserID(c); err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		if !isAdmin(c) {
			c.Error(domain.ErrAccessDenied)
			c.Abort()
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
