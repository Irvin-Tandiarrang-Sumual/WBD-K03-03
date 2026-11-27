package telemetry

import (
	"context"
	"io"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultServiceName = "migration-service"
	defaultExporter    = "console"
	defaultOTLPEndpoint = "localhost:4317"
)

type Config struct {
	ServiceName string
	Exporter    string
	Endpoint    string
	Writer      io.Writer
}

func InitTracer(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = defaultServiceName
	}

	exporterType := strings.ToLower(strings.TrimSpace(cfg.Exporter))
	if exporterType == "" {
		exporterType = strings.ToLower(os.Getenv("OTEL_TRACES_EXPORTER"))
		if exporterType == "" {
			exporterType = defaultExporter
		}
	}

	res, err := sdkresource.New(ctx,
		sdkresource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	var spanExporter sdktrace.SpanExporter
	switch exporterType {
	case "none":
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return tp.Shutdown, nil

	case "otlp":
		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			if endpoint == "" {
				endpoint = defaultOTLPEndpoint
			}
		}
		otlpExp, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(endpoint),
		)
		if err != nil {
			return nil, err
		}
		spanExporter = otlpExp

	case "console":
		fallthrough
	default:
		var opts []stdouttrace.Option
		opts = append(opts, stdouttrace.WithPrettyPrint())
		if cfg.Writer != nil {
			opts = append(opts, stdouttrace.WithWriter(cfg.Writer))
		}
		consoleExp, err := stdouttrace.New(opts...)
		if err != nil {
			return nil, err
		}
		spanExporter = consoleExp
	}

	bsp := sdktrace.NewBatchSpanProcessor(spanExporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func Tracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

func Inject(ctx context.Context, carrier map[string]string) {
	if carrier == nil {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(carrier))
}

func Extract(ctx context.Context, carrier map[string]string) context.Context {
	if carrier == nil {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(carrier))
}
