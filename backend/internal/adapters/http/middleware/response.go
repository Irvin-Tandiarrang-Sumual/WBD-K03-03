package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type authenticationErrorResponse struct {
	Error string `json:"error"`
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
