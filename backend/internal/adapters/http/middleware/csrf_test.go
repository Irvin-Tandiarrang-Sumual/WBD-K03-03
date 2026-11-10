package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/gin-gonic/gin"
)

func TestCSRFValidate_SafeMethodGet_CallsNextHandler(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())
	csrfMiddleware := middleware.NewCSRFMiddleware("http://localhost:3000")
	engine.Use(csrfMiddleware.Validate())
	engine.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCSRFValidate_OriginMismatch_ReturnsForbidden(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())
	csrfMiddleware := middleware.NewCSRFMiddleware("http://localhost:3000")
	engine.Use(csrfMiddleware.Validate())
	engine.POST("/resource", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/resource", nil)
	req.Header.Set("Origin", "http://malicious-site.com")
	req.Header.Set("X-CSRF-Token", "valid-csrf-token")
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestCSRFValidate_MissingOrMismatchToken_ReturnsForbidden(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())
	engine.Use(func(c *gin.Context) {
		middleware.SetSessionContext(c, domain.Session{CSRFToken: "correct-csrf-token"})
		c.Next()
	})
	csrfMiddleware := middleware.NewCSRFMiddleware("http://localhost:3000")
	engine.Use(csrfMiddleware.Validate())
	engine.POST("/resource", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/resource", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("X-CSRF-Token", "wrong-csrf-token")
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestCSRFValidate_ValidOriginAndToken_CallsNextHandler(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())
	engine.Use(func(c *gin.Context) {
		middleware.SetSessionContext(c, domain.Session{CSRFToken: "valid-csrf-token"})
		c.Next()
	})
	csrfMiddleware := middleware.NewCSRFMiddleware("http://localhost:3000")
	engine.Use(csrfMiddleware.Validate())
	engine.POST("/resource", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/resource", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("X-CSRF-Token", "valid-csrf-token")
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
