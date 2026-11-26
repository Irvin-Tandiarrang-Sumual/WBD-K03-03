package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	correlationIDKey contextKey = "correlation_id"
	actorKey         contextKey = "actor"
)

type Actor struct {
	UserID    uint   `json:"user_id"`
	Role      string `json:"role,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

func (a Actor) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, 3)
	attrs = append(attrs, slog.Uint64("user_id", uint64(a.UserID)))
	if a.Role != "" {
		attrs = append(attrs, slog.String("role", a.Role))
	}
	if a.SessionID != "" {
		attrs = append(attrs, slog.String("session_id", MaskSession(a.SessionID)))
	}
	return slog.GroupValue(attrs...)
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(correlationIDKey).(string); ok && id != "" {
		return id
	}
	return ""
}

func EnsureCorrelationID(ctx context.Context) (context.Context, string) {
	id := CorrelationIDFromContext(ctx)
	if id == "" {
		id = "req-" + uuid.New().String()
		ctx = WithCorrelationID(ctx, id)
	}
	return ctx, id
}

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey, actor)
}

func ActorFromContext(ctx context.Context) (Actor, bool) {
	if ctx == nil {
		return Actor{}, false
	}
	actor, ok := ctx.Value(actorKey).(Actor)
	return actor, ok
}

type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if id := CorrelationIDFromContext(ctx); id != "" {
			r.Add("correlation_id", id)
		}
		if actor, ok := ActorFromContext(ctx); ok && actor.UserID != 0 {
			r.Add("actor", actor)
		}
		span := trace.SpanFromContext(ctx)
		if sc := span.SpanContext(); sc.IsValid() {
			r.Add("trace_id", sc.TraceID().String())
			r.Add("span_id", sc.SpanID().String())
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, level)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}

func NewJSONLogger(w io.Writer, level slog.Level) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					return slog.String(slog.TimeKey, t.UTC().Format(time.RFC3339Nano))
				}
			}
			return a
		},
	})
	return slog.New(&ContextHandler{Handler: jsonHandler})
}

var (
	urlPasswordRegex = regexp.MustCompile(`(?i)(://[^:]+:)([^@]+)(@)`)
	dsnPasswordRegex = regexp.MustCompile(`(?i)(password=)(\S+)`)
)

func MaskDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	masked := urlPasswordRegex.ReplaceAllString(dsn, "${1}***${3}")
	masked = dsnPasswordRegex.ReplaceAllString(masked, "${1}***")
	return masked
}

func MaskSession(sess string) string {
	trimmed := strings.TrimSpace(sess)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 4 {
		return "sess_***"
	}
	return "sess_***" + trimmed[len(trimmed)-4:]
}
