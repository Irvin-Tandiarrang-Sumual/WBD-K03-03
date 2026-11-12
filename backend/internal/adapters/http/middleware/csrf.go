package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/gin-gonic/gin"
)

const CSRFHeaderName = "X-CSRF-Token"

type CSRFMiddleware struct {
	frontendOrigin string
}

func NewCSRFMiddleware(frontendOrigin string) *CSRFMiddleware {
	return &CSRFMiddleware{
		frontendOrigin: strings.TrimRight(frontendOrigin, "/"),
	}
}

func (m *CSRFMiddleware) Validate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		if !m.isOriginAllowed(c.GetHeader("Origin")) {
			c.Error(domain.ErrInvalidCSRFToken)
			c.Abort()
			return
		}

		if !isValidToken(c.GetHeader(CSRFHeaderName), CurrentCSRFToken(c)) {
			c.Error(domain.ErrInvalidCSRFToken)
			c.Abort()
			return
		}

		c.Next()
	}
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

func (m *CSRFMiddleware) isOriginAllowed(origin string) bool {
	if origin == "" || m.frontendOrigin == "" {
		return true
	}
	return strings.EqualFold(strings.TrimRight(origin, "/"), m.frontendOrigin)
}

func isValidToken(headerToken, expectedToken string) bool {
	if headerToken == "" || expectedToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(headerToken), []byte(expectedToken)) == 1
}
