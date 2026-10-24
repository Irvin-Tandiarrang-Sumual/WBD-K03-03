package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service      ports.AuthService
	secureCookie bool
}

func NewHandler(service ports.AuthService, secureCookie bool) *Handler {
	return &Handler{
		service:      service,
		secureCookie: secureCookie,
	}
}

func (h *Handler) Login(c *gin.Context) {
	var request LoginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": domain.ErrInvalidInput.Error(),
		})
		return
	}

	user, session, err := h.service.Login(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	maxAge := int(time.Until(session.AbsoluteExpiration).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	c.SetSameSite(http.SameSiteStrictMode)

	c.SetCookie(
		middleware.SessionCookieName,
		session.OpaqueToken,
		maxAge,
		"/",
		"",
		h.secureCookie,
		true,
	)

	c.JSON(http.StatusOK, LoginResponse{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	opaqueToken, err := c.Cookie(middleware.SessionCookieName)

	if err == nil && opaqueToken != "" {
		if err := h.service.Logout(c.Request.Context(), opaqueToken); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}
	}

	c.SetSameSite(http.SameSiteStrictMode)

	c.SetCookie(
		middleware.SessionCookieName,
		"",
		-1,
		"/",
		"",
		h.secureCookie,
		true,
	)

	c.Status(http.StatusNoContent)
}