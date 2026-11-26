package middleware

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/logger"
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
			c.Error(domain.ErrCurrentUserUnavailable)
			c.Abort()
			return
		}

		session, err := am.sessionService.Resolve(
			c.Request.Context(),
			opaqueToken,
		)

		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		if shouldRefreshSessionCookie(opaqueToken, session.OpaqueToken) {
			am.refreshSessionCookie(c, session)
		}

		currentUser, err := am.userService.GetUserByID(c.Request.Context(), session.UserID)

		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		SetUserContext(c, session.UserID, currentUser.Role)
		SetSessionContext(c, session)
		c.Request = c.Request.WithContext(logger.WithActor(c.Request.Context(), logger.Actor{
			UserID: session.UserID, Role: currentUser.Role,
		}))

		c.Next()
	}
}

func shouldRefreshSessionCookie(currentCookieToken string, newSessionToken string) bool {
	return newSessionToken != "" && newSessionToken != currentCookieToken
}

func (am *AuthMiddleware) refreshSessionCookie(c *gin.Context, session domain.Session) {
	SetSessionCookie(c, session, am.isSecureCookie)
}
