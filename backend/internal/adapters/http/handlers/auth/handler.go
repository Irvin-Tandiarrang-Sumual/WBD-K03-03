package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service      ports.AuthService
	secureCookie bool
}

func NewAuthHandler(service ports.AuthService, secureCookie bool) *AuthHandler {
	return &AuthHandler{
		service:      service,
		secureCookie: secureCookie,
	}
}

// Login authenticates a user and sets a session cookie.
//
//	@Summary		Login
//	@Description	Authenticate user with username and password. On success, a session cookie is set.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginRequest							true	"Login credentials"
//	@Success		200		{object}	response.BaseResponse[LoginResponse]	"Login successful"
//	@Failure		400		{object}	response.BaseResponse[any]				"Invalid input"
//	@Failure		401		{object}	response.BaseResponse[any]				"Invalid credentials"
//	@Failure		500		{object}	response.BaseResponse[any]				"Internal server error"
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var request LoginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(domain.ErrInvalidInput.Error()))
		return
	}

	user, session, err := h.service.Login(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, response.Error(err.Error()))
			return
		}

		c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
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

	c.JSON(http.StatusOK, response.Success("login successful", LoginResponse{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}))
}

// Logout invalidates the current session and clears the session cookie.
//
//	@Summary		Logout
//	@Description	Invalidate the current session. Clears the session cookie.
//	@Tags			Auth
//	@Produce		json
//	@Security		CookieAuth
//	@Success		204	"Logged out successfully"
//	@Failure		500	{object}	response.BaseResponse[any]	"Internal server error"
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	opaqueToken, err := c.Cookie(middleware.SessionCookieName)

	if err == nil && opaqueToken != "" {
		if err := h.service.Logout(c.Request.Context(), opaqueToken); err != nil {
			c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
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
