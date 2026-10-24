package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

const AuthenticatedUserIDKey = "userID"

func SetAuthenticatedUserID(c *gin.Context, userID uint) {
	c.Set(AuthenticatedUserIDKey, userID)
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

type authorizationErrorResponse struct {
	Error string `json:"error"`
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, authorizationErrorResponse{
		Error: "authentication required",
	})
}

func abortForbidden(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, authorizationErrorResponse{
		Error: "access denied",
	})
}

func RequireAdmin(users ports.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := AuthenticatedUserID(c)
		if !ok {
			abortUnauthorized(c)
			return
		}

		user, err := users.Get(c.Request.Context(), userID)
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

func RequireDatabaseAccess(groups ports.GroupService, required domain.AccessType) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := AuthenticatedUserID(c)
		if !ok {
			abortUnauthorized(c)
			return
		}

		databaseID, err := parseDatabaseID(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, authorizationErrorResponse{
				Error: domain.ErrInvalidInput.Error(),
			})
			return
		}

		effectiveAccess, err := groups.EffectiveDatabaseAccess(c.Request.Context(), userID, databaseID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrDatabaseAccessDenied):
				abortForbidden(c)

			case errors.Is(err, domain.ErrDatabaseNotFound):
				c.AbortWithStatusJSON(http.StatusNotFound, authorizationErrorResponse{
					Error: err.Error(),
				})

			default:
				c.AbortWithStatusJSON(http.StatusInternalServerError, authorizationErrorResponse{
					Error: "internal server error",
				})
			}

			return
		}

		if !effectiveAccess.Allows(required) {
			abortForbidden(c)
			return
		}

		c.Next()
	}
}

func parseDatabaseID(c *gin.Context) (uint, error) {
	rawID := c.Param("databaseId")

	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || id == 0 {
		return 0, domain.ErrInvalidInput
	}

	return uint(id), nil
}

func RequireDatabaseView(groups ports.GroupService) gin.HandlerFunc {
	return RequireDatabaseAccess(groups, domain.AccessViewOnly)
}

func RequireDatabaseEdit(groups ports.GroupService) gin.HandlerFunc {
	return RequireDatabaseAccess(groups, domain.AccessViewEdit)
}

