package middleware

import (
	"errors"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	sessionService ports.SessionService
	userService    ports.UserService
	isSecureCookie bool
}

func NewAuthMiddleware(sessionService ports.SessionService,
	userService ports.UserService, isSecureCookie bool) *AuthMiddleware {
	return &AuthMiddleware{
		sessionService: sessionService,
		userService:    userService,
		isSecureCookie: isSecureCookie,
	}
}

func (am *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		opaqueToken, err := c.Cookie(SessionCookieName)

		if err != nil || opaqueToken == "" {
			abortUnauthorized(c)
			return
		}

		session, err := am.sessionService.Resolve(
			c.Request.Context(),
			opaqueToken,
		)

		if err != nil {
			handleSessionError(c, err)
			return
		}

		if shouldRefreshSessionCookie(opaqueToken, session.OpaqueToken) {
			am.refreshSessionCookie(c, session)
		}

		currentUser, err := am.userService.GetUserByID(c.Request.Context(), session.UserID)

		if err != nil {
			handleUserFetchError(c, err)
			return
		}

		SetUserContext(c, session.UserID, currentUser.Role)

		c.Next()
	}
}

func shouldRefreshSessionCookie(currentCookieToken string, newSessionToken string) bool {
	return newSessionToken != "" && newSessionToken != currentCookieToken
}

func (am *AuthMiddleware) refreshSessionCookie(c *gin.Context, session domain.Session) {
	SetSessionCookie(c, session, am.isSecureCookie)
}

func handleUserFetchError(c *gin.Context, err error) {
	if errors.Is(err, domain.ErrUserNotFound) {
		abortUnauthorized(c)
		return
	}

	c.AbortWithStatusJSON(http.StatusInternalServerError, authorizationErrorResponse{
		Error: "internal server error",
	})
}

func handleSessionError(c *gin.Context, err error) {
	if errors.Is(err, domain.ErrInvalidSession) {
		abortUnauthorized(c)
		return
	}

	c.AbortWithStatusJSON(http.StatusInternalServerError, authenticationErrorResponse{
		Error: "internal server error",
	})
}
