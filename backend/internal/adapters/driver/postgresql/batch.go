package postgresql

import (
	"context"
	"errors"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/gorm"
)

func (r *ExecutorRepository) PrepareBatch(ctx context.Context, db domain.Database, batch ports.BatchExecution) (int64, error) {
	target, err := r.poolManager.Get(ctx, db)
	if err != nil {
		return 0, err
	}
	var total int64
	err = observedTransaction(ctx, target, db.ID, "batch.prepare", 0, func(tx *gorm.DB) error {
		if err := ensureBatchStorage(tx); err != nil {
			return err
		}
		if batch.Resume {
			var exists bool
			if err := tx.Raw("SELECT EXISTS (SELECT 1 FROM wbd_migration.batch_plans WHERE plan_key = ?)", batch.Key).Scan(&exists).Error; err != nil {
				return err
			}
			if !exists {
				return domain.ErrNoCheckpointToResume
			}
		}
		insert := tx.Exec("INSERT INTO wbd_migration.batch_plans (plan_key) VALUES (?) ON CONFLICT DO NOTHING", batch.Key)
		if insert.Error != nil {
			return insert.Error
		}
		if insert.RowsAffected == 1 {
			if err := saveBatchScope(tx, batch); err != nil {
				return err
			}
		}
		return tx.Raw("SELECT count(*) FROM wbd_migration.batch_keys WHERE plan_key = ?", batch.Key).Scan(&total).Error
	})
	return total, err
}

func ensureBatchStorage(tx *gorm.DB) error {
	return tx.Exec(`
		CREATE SCHEMA IF NOT EXISTS wbd_migration;
		CREATE TABLE IF NOT EXISTS wbd_migration.batch_plans (plan_key text PRIMARY KEY);
		CREATE TABLE IF NOT EXISTS wbd_migration.batch_keys (
			plan_key text NOT NULL, batch_no bigint NOT NULL, row_key bigint NOT NULL,
			affected boolean NOT NULL DEFAULT false, PRIMARY KEY (plan_key, row_key)
		);
		CREATE INDEX IF NOT EXISTS batch_keys_by_batch ON wbd_migration.batch_keys (plan_key, batch_no);
		CREATE TABLE IF NOT EXISTS wbd_migration.batch_receipts (
			plan_key text NOT NULL, batch_no bigint NOT NULL, cursor_key bigint NOT NULL,
			rows_affected bigint NOT NULL, reverted boolean NOT NULL DEFAULT false, PRIMARY KEY (plan_key, batch_no)
		)`).Error
}

func saveBatchScope(tx *gorm.DB, batch ports.BatchExecution) error {
	if batch.SourceKey != "" {
		return copyBatchScope(tx, batch)
	}
	var primaryKey bool
	err := tx.Raw(`SELECT EXISTS (
		SELECT 1 FROM pg_index i JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = i.indkey[0]
		WHERE i.indrelid = to_regclass(?) AND i.indisprimary AND i.indnkeyatts = 1
		  AND a.attname = ? AND a.atttypid IN ('int2'::regtype, 'int4'::regtype, 'int8'::regtype)
	)`, batch.Table, batch.Column).Scan(&primaryKey).Error
	if err != nil {
		return err
	}
	if !primaryKey {
		return domain.ErrBatchColumnNotFound
	}
	return tx.Exec(`INSERT INTO wbd_migration.batch_keys (plan_key, batch_no, row_key)
		SELECT ?, (row_number() OVER (ORDER BY `+batch.Column+`) - 1) / ? + 1, `+batch.Column+
		" FROM "+batch.Table, batch.Key, batch.Size).Error
}

func copyBatchScope(tx *gorm.DB, batch ports.BatchExecution) error {
	var exists bool
	if err := tx.Raw("SELECT EXISTS (SELECT 1 FROM wbd_migration.batch_plans WHERE plan_key = ?)", batch.SourceKey).Scan(&exists).Error; err != nil {
		return err
	}
	if !exists {
		return domain.ErrNoCheckpointToResume
	}
	order := "ASC"
	if batch.Reverse {
		order = "DESC"
	}
	return tx.Exec(`INSERT INTO wbd_migration.batch_keys (plan_key, batch_no, row_key)
		SELECT ?, (row_number() OVER (ORDER BY row_key `+order+`) - 1) / ? + 1, row_key
		FROM wbd_migration.batch_keys WHERE plan_key = ? AND affected`,
		batch.Key, batch.Size, batch.SourceKey).Error
}

type batchReceiptRow struct {
	CursorKey    int64
	RowsAffected int64
	Reverted     bool
}

func (r *ExecutorRepository) ExecuteBatchChunk(ctx context.Context, db domain.Database, batch ports.BatchExecution, number int, revert bool) (*ports.BatchReceipt, error) {
	target, err := r.poolManager.Get(ctx, db)
	if err != nil {
		return nil, err
	}
	var receipt *ports.BatchReceipt
	err = observedTransaction(ctx, target, db.ID, "batch.chunk", number, func(tx *gorm.DB) error {
		var applyErr error
		receipt, applyErr = applyBatchChunk(tx, batch, number, revert)
		return applyErr
	})
	return receipt, err
}

func applyBatchChunk(tx *gorm.DB, batch ports.BatchExecution, number int, revert bool) (*ports.BatchReceipt, error) {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", batch.Key).Error; err != nil {
		return nil, err
	}
	var row batchReceiptRow
	result := tx.Raw("SELECT cursor_key, rows_affected, reverted FROM wbd_migration.batch_receipts WHERE plan_key = ? AND batch_no = ?", batch.Key, number).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 0 && (!revert || row.Reverted) {
		return &ports.BatchReceipt{Number: number, Cursor: strconv.FormatInt(row.CursorKey, 10), Rows: row.RowsAffected}, nil
	}
	if revert && result.RowsAffected == 0 {
		return nil, domain.ErrNoCheckpointToResume
	}
	var keys []int64
	query := tx.Table("wbd_migration.batch_keys").Where("plan_key = ? AND batch_no = ?", batch.Key, number)
	if revert {
		query = query.Where("affected")
	}
	if err := query.Order("row_key").Pluck("row_key", &keys).Error; err != nil {
		return nil, err
	}
	if revert {
		return revertBatchKeys(tx, batch, number, keys, row)
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return commitBatchKeys(tx, batch, number, keys)
}

func commitBatchKeys(tx *gorm.DB, batch ports.BatchExecution, number int, keys []int64) (*ports.BatchReceipt, error) {
	affected, err := updateBatchKeys(tx, batch, keys)
	if err != nil {
		return nil, err
	}
	if batch.SourceKey != "" && len(affected) != len(keys) {
		return nil, errors.New("approved SQL did not change every original affected row")
	}
	if len(affected) > 0 {
		if err := tx.Exec("UPDATE wbd_migration.batch_keys SET affected = true WHERE plan_key = ? AND row_key IN ?", batch.Key, affected).Error; err != nil {
			return nil, err
		}
	}
	cursor := keys[len(keys)-1]
	err = tx.Exec("INSERT INTO wbd_migration.batch_receipts (plan_key, batch_no, cursor_key, rows_affected) VALUES (?, ?, ?, ?)", batch.Key, number, cursor, len(affected)).Error
	return &ports.BatchReceipt{Number: number, Cursor: strconv.FormatInt(cursor, 10), Rows: int64(len(affected))}, err
}

func revertBatchKeys(tx *gorm.DB, batch ports.BatchExecution, number int, keys []int64, row batchReceiptRow) (*ports.BatchReceipt, error) {
	if int64(len(keys)) != row.RowsAffected {
		return nil, domain.ErrNoCheckpointToResume
	}
	affected, err := updateBatchKeys(tx, batch, keys)
	if err != nil {
		return nil, err
	}
	if len(affected) != len(keys) {
		return nil, errors.New("SQL down did not restore every affected row")
	}
	err = tx.Exec("UPDATE wbd_migration.batch_receipts SET reverted = true WHERE plan_key = ? AND batch_no = ?", batch.Key, number).Error
	return &ports.BatchReceipt{Number: number, Cursor: strconv.FormatInt(row.CursorKey, 10), Rows: row.RowsAffected}, err
}

func updateBatchKeys(tx *gorm.DB, batch ports.BatchExecution, keys []int64) ([]int64, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	var affected []int64
	if err := tx.Raw(batch.Statement+" RETURNING "+batch.Column, keys).Scan(&affected).Error; err != nil {
		return nil, err
	}
	allowed := make(map[int64]bool, len(keys))
	for _, key := range keys {
		allowed[key] = true
	}
	for _, key := range affected {
		if !allowed[key] {
			return nil, errors.New("batch SQL changed a row outside its assigned keys")
		}
	}
	return affected, nil
}
