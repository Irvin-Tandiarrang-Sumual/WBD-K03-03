package auth

import (
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service      ports.AuthService
	secureCookie bool
	userService  ports.UserService
}

func NewAuthHandler(service ports.AuthService, secureCookie bool, userService ports.UserService) *AuthHandler {
	return &AuthHandler{
		service:      service,
		secureCookie: secureCookie,
		userService:  userService,
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
		c.Error(err)
		return
	}

	user, session, err := h.service.Login(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		c.Error(err)
		return
	}

	middleware.SetSessionCookie(c, session, h.secureCookie)

	c.JSON(http.StatusOK, response.Success("login successful", LoginResponse{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CSRFToken: session.CSRFToken,
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
	defer middleware.ClearSessionCookie(c, h.secureCookie)

	opaqueToken, err := c.Cookie(middleware.SessionCookieName)

	if err == nil && opaqueToken != "" {
		if err := h.service.Logout(c.Request.Context(), opaqueToken); err != nil {
			c.Error(err)
			return
		}
	}

	c.Status(http.StatusNoContent)
}

// Me returns the current authenticated user profile.
//
//	@Summary		Get current user
//	@Description	Returns details of the currently authenticated user based on session cookie.
//	@Tags			Auth
//	@Produce		json
//	@Security		CookieAuth
//	@Success		200	{object}	response.BaseResponse[Me]	"Current user profile"
//	@Failure		401	{object}	response.BaseResponse[any]				"Unauthorized"
//	@Failure		500	{object}	response.BaseResponse[any]				"Internal server error"
//	@Router			/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response.Success("current user profile", Me{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CSRFToken: middleware.CurrentCSRFToken(c),
	}))
}
