package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

func TestMaskDSN_URLFormatWithPassword_MasksPassword(t *testing.T) {
	// Arrange
	dsn := "postgres://admin:SuperSecretPass123@db.prod.internal:5432/app"

	// Act
	masked := logger.MaskDSN(dsn)

	// Assert
	if strings.Contains(masked, "SuperSecretPass123") {
		t.Fatalf("expected password to be masked, got %s", masked)
	}
	expected := "postgres://admin:***@db.prod.internal:5432/app"
	if masked != expected {
		t.Fatalf("expected %q, got %q", expected, masked)
	}
}

func TestMaskDSN_KeyValueFormatWithPassword_MasksPassword(t *testing.T) {
	// Arrange
	dsn := "host=localhost user=dbadmin password=MySecretPassword dbname=testdb"

	// Act
	masked := logger.MaskDSN(dsn)

	// Assert
	if strings.Contains(masked, "MySecretPassword") {
		t.Fatalf("expected password to be masked, got %s", masked)
	}
	if !strings.Contains(masked, "password=***") {
		t.Fatalf("expected password=***, got %s", masked)
	}
}

func TestMaskSession_StandardSessionID_RetainsOnlyLastFourChars(t *testing.T) {
	// Arrange
	sessionID := "sess_9b1deb4d3b7d4bad9bdd2b0d7b3dcb6d"

	// Act
	masked := logger.MaskSession(sessionID)

	// Assert
	expected := "sess_***cb6d"
	if masked != expected {
		t.Fatalf("expected %q, got %q", expected, masked)
	}
}

func TestMaskSession_ShortSessionID_ReturnsDefaultMask(t *testing.T) {
	// Arrange
	sessionID := "abc"

	// Act
	masked := logger.MaskSession(sessionID)

	// Assert
	expected := "sess_***"
	if masked != expected {
		t.Fatalf("expected %q, got %q", expected, masked)
	}
}

func TestNewJSONLogger_ContextWithCorrelationAndActor_OutputsStructuredJSON(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	log := logger.NewJSONLogger(&buf, slog.LevelInfo)

	ctx := context.Background()
	ctx = logger.WithCorrelationID(ctx, "req-test-123")
	ctx = logger.WithActor(ctx, logger.Actor{
		UserID:    42,
		Role:      "engineer",
		SessionID: "sess_secret_token_1234",
	})

	// Act
	log.InfoContext(ctx, "database migration executed", "duration_ms", 12.5)

	// Assert
	output := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v, raw: %s", err, output)
	}

	if parsed["correlation_id"] != "req-test-123" {
		t.Fatalf("expected correlation_id 'req-test-123', got %v", parsed["correlation_id"])
	}
	if parsed["msg"] != "database migration executed" {
		t.Fatalf("expected msg 'database migration executed', got %v", parsed["msg"])
	}
	if parsed["level"] != "INFO" {
		t.Fatalf("expected level 'INFO', got %v", parsed["level"])
	}

	actor, ok := parsed["actor"].(map[string]any)
	if !ok {
		t.Fatalf("expected actor object in log, got %v", parsed["actor"])
	}
	if actor["user_id"] != float64(42) {
		t.Fatalf("expected actor user_id 42, got %v", actor["user_id"])
	}
	if actor["role"] != "engineer" {
		t.Fatalf("expected actor role 'engineer', got %v", actor["role"])
	}
	if actor["session_id"] != "sess_***1234" {
		t.Fatalf("expected masked session_id 'sess_***1234', got %v", actor["session_id"])
	}
}

func TestNewJSONLogger_ContextWithActiveSpan_IncludesTraceAndSpanID(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	log := logger.NewJSONLogger(&buf, slog.LevelInfo)

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("failed to parse trace ID: %v", err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatalf("failed to parse span ID: %v", err)
	}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	// Act
	log.InfoContext(ctx, "worker processing task")

	// Assert
	output := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v, raw: %s", err, output)
	}

	if parsed["trace_id"] != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("expected trace_id 4bf92f3577b34da6a3ce929d0e0e4736, got %v", parsed["trace_id"])
	}
	if parsed["span_id"] != "00f067aa0ba902b7" {
		t.Errorf("expected span_id 00f067aa0ba902b7, got %v", parsed["span_id"])
	}
}
