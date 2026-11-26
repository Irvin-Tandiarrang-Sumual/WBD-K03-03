package postgresql

import (
	"context"
	"log/slog"

	"gorm.io/gorm"
)

func observedTransaction(ctx context.Context, db *gorm.DB, databaseID uint, operation string, batchNumber int, run func(*gorm.DB) error) error {
	tx := db.WithContext(ctx).Begin()
	attrs := []any{"database_id", databaseID, "phase", operation, "batch_number", batchNumber}
	if tx.Error != nil {
		slog.ErrorContext(ctx, "transaction could not start", append(attrs, "event", "sql.transaction.begin_failed", "status", "failed")...)
		return tx.Error
	}
	slog.InfoContext(ctx, "transaction started", append(attrs, "event", "sql.transaction.start", "status", "started")...)
	committed := false
	defer func() {
		if !committed && tx.Rollback().Error == nil {
			slog.InfoContext(ctx, "transaction rolled back", append(attrs, "event", "sql.transaction.rollback", "status", "rolled_back")...)
		}
	}()
	if err := run(tx); err != nil {
		return err
	}
	if err := tx.Commit().Error; err != nil {
		slog.ErrorContext(ctx, "transaction commit failed", append(attrs, "event", "sql.transaction.commit_failed", "status", "failed")...)
		return err
	}
	committed = true
	slog.InfoContext(ctx, "transaction committed", append(attrs, "event", "sql.transaction.commit", "status", "committed")...)
	return nil
}
