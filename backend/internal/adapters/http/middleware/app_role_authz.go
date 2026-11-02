package middleware

import (
	"errors"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

const AuthenticatedUserIDKey = "userID"

func RequireAdmin(users ports.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := AuthenticatedUserID(c)
		if !ok {
			abortUnauthorized(c)
			return
		}

		user, err := users.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				abortUnauthorized(c)
				return
			}

			c.AbortWithStatusJSON(http.StatusInternalServerError, authorizationErrorResponse{
				Error: "internal server error",
			})
			return
		}

		if user.Role != domain.RoleAdmin {
			abortForbidden(c)
			return
		}

		c.Next()
	}
}

func AuthenticatedUserID(c *gin.Context) (uint, bool) {
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
