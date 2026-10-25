package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

func TestHealthHandlerHealthyWithoutPinger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.Check(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var res response.BaseResponse[HealthResponse]
	if err := json.Unmarshal(recorder.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !res.Success || res.Data.Status != "ok" {
		t.Fatalf("expected status ok, got %+v", res)
	}
}

func TestHealthHandlerHealthyWithPinger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandlerWithPinger(func(ctx context.Context) error {
		return nil
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.Check(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var res response.BaseResponse[HealthResponse]
	if err := json.Unmarshal(recorder.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !res.Success || res.Data.Status != "ok" || res.Data.Services["database"] != "up" {
		t.Fatalf("expected database up, got %+v", res)
	}
}

func TestHealthHandlerDegradedWhenPingerFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandlerWithPinger(func(ctx context.Context) error {
		return errors.New("connection refused")
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.Check(c)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", recorder.Code)
	}

	var res response.BaseResponse[HealthResponse]
	if err := json.Unmarshal(recorder.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Success || res.Data.Status != "degraded" || res.Data.Services["database"] != "down" {
		t.Fatalf("expected database down and status degraded, got %+v", res)
	}
}
