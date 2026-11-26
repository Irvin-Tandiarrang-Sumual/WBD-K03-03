package telemetry_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/telemetry"
	"go.opentelemetry.io/otel/trace"
)

func TestInitTracer_ConsoleExporter_Succeeds(t *testing.T) {
	// Arrange
	ctx := context.Background()
	var buf bytes.Buffer
	cfg := telemetry.Config{
		ServiceName: "test-service",
		Exporter:    "console",
		Writer:      &buf,
	}

	// Act
	shutdown, err := telemetry.InitTracer(ctx, cfg)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}

	tr := telemetry.Tracer("test-tracer")
	_, span := tr.Start(ctx, "test-span")
	span.End()

	if err := shutdown(ctx); err != nil {
		t.Errorf("expected clean shutdown, got %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected console exporter to write span JSON to buffer")
	}
}

func TestContextPropagation_InjectAndExtract_PreservesTraceContext(t *testing.T) {
	// Arrange
	ctx := context.Background()
	_, _ = telemetry.InitTracer(ctx, telemetry.Config{Exporter: "none"})

	tr := telemetry.Tracer("propagation-test")
	parentCtx, parentSpan := tr.Start(ctx, "parent-operation")
	defer parentSpan.End()

	carrier := make(map[string]string)

	// Act
	telemetry.Inject(parentCtx, carrier)
	childCtx := telemetry.Extract(context.Background(), carrier)
	childSpanContext := trace.SpanContextFromContext(childCtx)

	// Assert
	parentSpanContext := parentSpan.SpanContext()
	if !childSpanContext.IsValid() {
		t.Fatal("expected valid extracted span context")
	}
	if childSpanContext.TraceID() != parentSpanContext.TraceID() {
		t.Errorf("trace ID mismatch: got %s, want %s",
			childSpanContext.TraceID().String(),
			parentSpanContext.TraceID().String(),
		)
	}
}
