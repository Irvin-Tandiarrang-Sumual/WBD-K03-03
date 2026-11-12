package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func TestCORS_ConfiguredOriginPreflight_ReturnsNoContentAndAllowHeaders(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.CORS("http://localhost:3000"))
	engine.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	response := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("allowed origin = %q, want %q", got, "http://localhost:3000")
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow credentials = %q, want %q", got, "true")
	}
}

func TestCORS_UnknownOrigin_DoesNotSetAllowOriginHeader(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.CORS("http://localhost:3000"))
	engine.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "http://example.com")
	response := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(response, request)

	// Assert
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected allowed origin %q", got)
	}
}
