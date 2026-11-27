package middleware

import (
	"log/slog"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/logger"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = c.GetHeader("X-Request-ID")
		}
		if correlationID == "" {
			correlationID = "req-" + uuid.New().String()
		}

		c.Header("X-Correlation-ID", correlationID)
		c.Set("correlation_id", correlationID)

		ctx := logger.WithCorrelationID(c.Request.Context(), correlationID)
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(c.Request.Header))
		ctx, span := telemetry.Tracer("http").Start(ctx, c.Request.Method+" "+c.FullPath(),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attribute.String("http.route", c.FullPath())),
		)
		defer span.End()
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		var actor logger.Actor
		if uid, err := CurrentUserID(c); err == nil && uid != 0 {
			actor.UserID = uid
			if roleVal, exists := c.Get(AuthenticatedUserRoleKey); exists {
				if rStr, ok := roleVal.(string); ok {
					actor.Role = rStr
				}
			}
			if sess, err := CurrentSession(c); err == nil {
				actor.SessionID = sess.OpaqueToken
			}
		}

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		span.SetAttributes(attribute.Int("http.response.status_code", statusCode))
		if statusCode >= 500 {
			span.SetStatus(codes.Error, "HTTP request failed")
		}

		attrs := []slog.Attr{
			slog.String("event", "http.request"),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status_code", statusCode),
			slog.Float64("latency_ms", float64(latency.Nanoseconds())/1e6),
			slog.String("client_ip", c.ClientIP()),
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("raw_error", c.Errors.Last().Err.Error()))
		}

		logCtx := c.Request.Context()
		if actor.UserID != 0 {
			logCtx = logger.WithActor(logCtx, actor)
		}

		level := slog.LevelInfo
		msg := "HTTP request completed"
		if statusCode >= 500 {
			level = slog.LevelError
			msg = "HTTP request completed with server error"
		} else if statusCode >= 400 {
			level = slog.LevelWarn
			msg = "HTTP request completed with client error"
		}

		slog.LogAttrs(logCtx, level, msg, attrs...)
	}
}
