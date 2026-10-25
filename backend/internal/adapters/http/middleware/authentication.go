package middleware

import (
	"errors"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

const SessionCookieName = "session"

type authenticationErrorResponse struct {
	Error string `json:"error"`
}

func Authenticate(sessions ports.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		opaqueToken, err := c.Cookie(SessionCookieName)

		if err != nil || opaqueToken == "" {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				authenticationErrorResponse{
					Error: "authentication required",
				},
			)
			return
		}

		session, err := sessions.Resolve(
			c.Request.Context(),
			opaqueToken,
		)

		if err != nil {
			if errors.Is(err, domain.ErrInvalidSession) {
				c.AbortWithStatusJSON(
					http.StatusUnauthorized,
					authenticationErrorResponse{
						Error: "authentication required",
					},
				)
				return
			}

			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				authenticationErrorResponse{
					Error: "internal server error",
				},
			)
			return
		}

		SetAuthenticatedUserID(c, session.UserID)

		c.Next()
	}
}
