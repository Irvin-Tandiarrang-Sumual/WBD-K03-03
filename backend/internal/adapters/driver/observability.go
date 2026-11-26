package driver

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/logger"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func observeSQL[T any](ctx context.Context, db domain.Database, operation, statement string, execute func(context.Context) (T, int64, error), extra ...any) (T, error) {
	ctx, _ = logger.EnsureCorrelationID(ctx)
	event := "sql." + operation
	ctx, span := telemetry.Tracer("db.driver").Start(ctx, "db."+operation,
		trace.WithAttributes(attribute.String("db.system", db.Type), attribute.Int64("db.id", int64(db.ID))),
	)
	defer span.End()

	attrs := append([]any{
		"database_id", db.ID,
		"database_dialect", db.Type,
		"query", redactedSQL(statement),
		"parameters", "[redacted]",
	}, extra...)
	start := time.Now()
	slog.InfoContext(ctx, "SQL started", append(attrs, "event", event+".start", "status", "started")...)
	result, rows, err := execute(ctx)
	attrs = append(attrs, "duration_ms", float64(time.Since(start))/float64(time.Millisecond))
	if err != nil {
		state := extractSQLState(err)
		span.SetAttributes(attribute.String("db.response.status_code", state))
		span.SetStatus(codes.Error, "SQL failed")
		slog.ErrorContext(ctx, "SQL failed", append(attrs, "event", event+".failed", "status", "failed", "sqlstate", state)...)
		return result, err
	}
	span.SetAttributes(attribute.Int64("db.rows", rows))
	span.SetStatus(codes.Ok, "")
	slog.InfoContext(ctx, "SQL completed", append(attrs, "event", event+".complete", "status", "success", "rows", rows)...)
	return result, nil
}

func redactedSQL(statement string) string {
	words := strings.Fields(statement)
	if len(words) == 0 {
		return "[redacted]"
	}
	switch first := strings.ToUpper(words[0]); first {
	case "SELECT", "WITH", "EXPLAIN", "SHOW", "DESCRIBE", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "TRUNCATE", "RENAME", "COMMENT", "GRANT", "REVOKE":
		return first + " [redacted]"
	default:
		return "[redacted]"
	}
}
