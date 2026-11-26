package middleware_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/logger"
	"github.com/gin-gonic/gin"
)

func TestLogger_IncomingRequestWithoutCorrelationID_GeneratesAndSetsHeader(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	var logBuf bytes.Buffer
	origDefault := slog.Default()
	slog.SetDefault(logger.NewJSONLogger(&logBuf, slog.LevelInfo))
	defer slog.SetDefault(origDefault)

	router := gin.New()
	router.Use(middleware.Logger())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	corrID := w.Header().Get("X-Correlation-ID")
	if corrID == "" {
		t.Fatalf("expected X-Correlation-ID header to be present, got empty")
	}

	output := logBuf.String()
	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("expected valid JSON log, got error %v", err)
	}
	if logEntry["correlation_id"] != corrID {
		t.Fatalf("expected correlation_id in log %q, got %v", corrID, logEntry["correlation_id"])
	}
	if logEntry["status_code"] != float64(200) {
		t.Fatalf("expected status_code 200, got %v", logEntry["status_code"])
	}
}

func TestLogger_IncomingRequestWithCorrelationID_PreservesHeader(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	var logBuf bytes.Buffer
	origDefault := slog.Default()
	slog.SetDefault(logger.NewJSONLogger(&logBuf, slog.LevelInfo))
	defer slog.SetDefault(origDefault)

	router := gin.New()
	router.Use(middleware.Logger())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Correlation-ID", "custom-trace-999")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	corrID := w.Header().Get("X-Correlation-ID")
	if corrID != "custom-trace-999" {
		t.Fatalf("expected X-Correlation-ID 'custom-trace-999', got %q", corrID)
	}

	var logEntry map[string]any
	if err := json.Unmarshal([]byte(logBuf.Bytes()), &logEntry); err != nil {
		t.Fatalf("expected valid JSON log, got error %v", err)
	}
	if logEntry["correlation_id"] != "custom-trace-999" {
		t.Fatalf("expected correlation_id 'custom-trace-999', got %v", logEntry["correlation_id"])
	}
}

func TestLogger_RequestWithInternalError_LogsAtErrorLevelWithRawError(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	var logBuf bytes.Buffer
	origDefault := slog.Default()
	slog.SetDefault(logger.NewJSONLogger(&logBuf, slog.LevelInfo))
	defer slog.SetDefault(origDefault)

	router := gin.New()
	router.Use(middleware.Logger())
	router.Use(middleware.ErrorHandler())
	router.GET("/error", func(c *gin.Context) {
		_ = c.Error(errors.New("unhandled database failure: pq connection timeout"))
	})

	req, _ := http.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	output := logBuf.String()
	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("expected valid JSON log, got %v", err)
	}

	if logEntry["level"] != "ERROR" {
		t.Fatalf("expected log level ERROR, got %v", logEntry["level"])
	}
	if logEntry["status_code"] != float64(500) {
		t.Fatalf("expected status_code 500, got %v", logEntry["status_code"])
	}
	if logEntry["raw_error"] != "unhandled database failure: pq connection timeout" {
		t.Fatalf("expected raw_error in log, got %v", logEntry["raw_error"])
	}
}
