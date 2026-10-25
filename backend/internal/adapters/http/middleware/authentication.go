package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

const SessionCookieName = "session"

type authenticationErrorResponse struct {
	Error string `json:"error"`
}

func Authenticate(sessions ports.SessionService, secureCookie ...bool) gin.HandlerFunc {
	isSecure := true
	if len(secureCookie) > 0 {
		isSecure = secureCookie[0]
	}

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

		if session.OpaqueToken != "" && session.OpaqueToken != opaqueToken {
			maxAge := 86400
			if !session.AbsoluteExpiration.IsZero() {
				maxAge = int(time.Until(session.AbsoluteExpiration).Seconds())
				if maxAge < 0 {
					maxAge = 0
				}
			}

			c.SetSameSite(http.SameSiteStrictMode)
			c.SetCookie(
				SessionCookieName,
				session.OpaqueToken,
				maxAge,
				"/",
				"",
				isSecure,
				true,
			)
		}

		SetAuthenticatedUserID(c, session.UserID)

		c.Next()
	}
}
