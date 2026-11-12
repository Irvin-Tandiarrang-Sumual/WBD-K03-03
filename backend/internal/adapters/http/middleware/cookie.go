package middleware

import (
	"net/http"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/gin-gonic/gin"
)

const SessionCookieName = "session"

func SetSessionCookie(c *gin.Context, session domain.Session, secure bool) {
	c.SetSameSite(http.SameSiteStrictMode)
	maxAge := calculateMaxAge(session)
	c.SetCookie(SessionCookieName, session.OpaqueToken, maxAge, "/", "", secure, true)
}

func ClearSessionCookie(c *gin.Context, secure bool) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(SessionCookieName, "", -1, "/", "", secure, true)
}

func calculateMaxAge(session domain.Session) int {
	if session.AbsoluteExpiration.IsZero() {
		return int(domain.DefaultSessionAbsoluteLifetime.Seconds())
	}
	maxAge := int(time.Until(session.AbsoluteExpiration).Seconds())
	if maxAge < 0 {
		return 0
	}
	return maxAge
}
