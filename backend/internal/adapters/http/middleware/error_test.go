package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/gin-gonic/gin"
)

func TestMapError_NilError_ReturnsStatusOK(t *testing.T) {
	// Arrange
	var err error = nil

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if body.Success != true {
		t.Fatalf("expected success true, got %v", body.Success)
	}
}

func TestMapError_InvalidCredentials_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	err := domain.ErrInvalidCredentials

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if body.Message != "invalid username or password" {
		t.Fatalf("message = %q, want %q", body.Message, "invalid username or password")
	}
}

func TestMapError_UserNotFound_ReturnsNotFound(t *testing.T) {
	// Arrange
	err := domain.ErrUserNotFound

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	if body.Message != domain.ErrUserNotFound.Error() {
		t.Fatalf("message = %q, want %q", body.Message, domain.ErrUserNotFound.Error())
	}
}

func TestMapError_AccessDenied_ReturnsForbidden(t *testing.T) {
	// Arrange
	err := domain.ErrDatabaseAccessDenied

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
	}
	if body.Message != domain.ErrDatabaseAccessDenied.Error() {
		t.Fatalf("message = %q, want %q", body.Message, domain.ErrDatabaseAccessDenied.Error())
	}
}

func TestMapError_UsernameTaken_ReturnsConflict(t *testing.T) {
	// Arrange
	err := domain.ErrUsernameTaken

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want %d", status, http.StatusConflict)
	}
	if body.Message != domain.ErrUsernameTaken.Error() {
		t.Fatalf("message = %q, want %q", body.Message, domain.ErrUsernameTaken.Error())
	}
}

func TestMapError_Unauthorized_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	err := domain.ErrUnauthorized

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if body.Message != domain.ErrUnauthorized.Error() {
		t.Fatalf("message = %q, want %q", body.Message, domain.ErrUnauthorized.Error())
	}
}

func TestMapError_Forbidden_ReturnsForbidden(t *testing.T) {
	// Arrange
	err := domain.ErrForbidden

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
	}
	if body.Message != domain.ErrForbidden.Error() {
		t.Fatalf("message = %q, want %q", body.Message, domain.ErrForbidden.Error())
	}
}

func TestMapError_DraftNotFound_ReturnsNotFound(t *testing.T) {
	// Arrange
	err := domain.ErrDraftNotFound

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	if body.Message != domain.ErrDraftNotFound.Error() {
		t.Fatalf("message = %q, want %q", body.Message, domain.ErrDraftNotFound.Error())
	}
}

func TestMapError_InvalidDraftState_ReturnsConflict(t *testing.T) {
	// Arrange
	err := domain.ErrInvalidDraftState

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want %d", status, http.StatusConflict)
	}
	if body.Message != domain.ErrInvalidDraftState.Error() {
		t.Fatalf("message = %q, want %q", body.Message, domain.ErrInvalidDraftState.Error())
	}
}

func TestMapError_InvalidDraftInput_ReturnsBadRequest(t *testing.T) {
	// Arrange
	err := domain.ErrInvalidDraftInput

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if body.Message != domain.ErrInvalidDraftInput.Error() {
		t.Fatalf("message = %q, want %q", body.Message, domain.ErrInvalidDraftInput.Error())
	}
}

func TestMapError_DatabaseConnectionFailure_ReturnsBadRequest(t *testing.T) {
	// Arrange
	err := errors.New("dial tcp 127.0.0.1:5432: connect: connection refused")

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if body.Message != "database connection failed" {
		t.Fatalf("message = %q, want %q", body.Message, "database connection failed")
	}
}

func TestMapError_UnknownInternalError_ReturnsInternalServerErrorAndMasksDetail(t *testing.T) {
	// Arrange
	err := errors.New("fatal postgres crash")

	// Act
	status, body := middleware.MapError(err)

	// Assert
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", status, http.StatusInternalServerError)
	}
	if body.Message != "internal server error" {
		t.Fatalf("message = %q, want %q", body.Message, "internal server error")
	}
}

func TestErrorHandler_HandlerReturnsInternalError_InterceptsAndReturnsMaskedJSON(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())

	rawInternalErr := errors.New("underlying postgres connection pool exhausted")
	engine.GET("/trigger-error", func(c *gin.Context) {
		_ = c.Error(rawInternalErr)
	})

	req := httptest.NewRequest(http.MethodGet, "/trigger-error", nil)
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if body["message"] != "internal server error" || body["success"] != false {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestErrorHandler_NoErrorsOccurred_PassesThroughResponse(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandler())

	engine.GET("/success", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/success", nil)
	rec := httptest.NewRecorder()

	// Act
	engine.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
