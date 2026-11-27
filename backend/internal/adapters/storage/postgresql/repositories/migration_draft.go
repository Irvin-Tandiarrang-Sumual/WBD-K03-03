package repositories

import (
	"context"
	"errors"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *migrationRepository) WithPosition(ctx context.Context, id uint, fn func(ports.MigrationRepository, domain.MigrationPosition) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var db model.Database
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "current_migration_id", "latest_execution_id").First(&db, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrDatabaseNotFound
			}
			return err
		}
		var executing int64
		if err := tx.Model(&model.Draft{}).Where("database_id=? AND status='executing'", id).Count(&executing).Error; err != nil {
			return err
		}
		if executing > 0 {
			return domain.ErrMigrationInProgress
		}
		return fn(&migrationRepository{tx}, domain.MigrationPosition{HeadID: db.CurrentMigrationID, ExecutionID: db.LatestExecutionID})
	})
}

func (r *migrationRepository) UpdatePosition(ctx context.Context, dbID uint, head, parent, forward *uint, executionID uint) error {
	tx := r.db.WithContext(ctx)
	if parent != nil {
		if err := tx.Model(&model.Migration{}).Where("database_id=? AND id=?", dbID, *parent).Update("forward_migration_id", forward).Error; err != nil {
			return err
		}
	}
	return tx.Model(&model.Database{}).Where("id=?", dbID).Updates(map[string]any{"current_migration_id": head, "latest_execution_id": executionID}).Error
}

func (r *migrationRepository) SaveDraft(ctx context.Context, d *domain.Draft) error {
	row := model.Draft{
		DatabaseID:      d.DatabaseID,
		UserID:          d.UserID,
		Operation:       d.Operation,
		TargetID:        d.TargetID,
		BaseExecutionID: d.BaseExecutionID,
		SqlUp:           d.SqlUp,
		SqlDown:         d.SqlDown,
		Status:          d.Status,
		IsReversible:    d.IsReversible,
		IsBatch:         d.IsBatch,
		BatchColumn:     d.BatchColumn,
		BatchSize:       d.BatchSize,
	}
	tx := r.db.WithContext(ctx)
	if err := tx.Create(&row).Error; err != nil {
		return err
	}
	d.ID = row.ID
	result := tx.Exec(`INSERT INTO draft_approvals (draft_id,group_id,leader_id,status)
 SELECT ?,g.id,g.leader_id,'pending' FROM groups g JOIN group_databases gd ON gd.group_id=g.id WHERE gd.database_id=?`, d.ID, d.DatabaseID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrDraftConflict
	}
	return nil
}

func (r *migrationRepository) FindDraft(ctx context.Context, dbID, id uint) (*domain.Draft, error) {
	var row model.Draft
	err := r.db.WithContext(ctx).Where("database_id=? AND id=? AND operation IN ?", dbID, id, []string{domain.DraftOperationRollback, domain.DraftOperationForward}).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrMigrationNotFound
	}
	if err != nil {
		return nil, err
	}
	return toDomainDraft(&row), nil
}

func (r *migrationRepository) UpdateDraftStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&model.Draft{}).Where("id=?", id).Update("status", status).Error
}

func (r *migrationRepository) Vote(ctx context.Context, id, userID uint, approve bool) error {
	status := "rejected"
	if approve {
		status = "approved"
	}
	result := r.db.WithContext(ctx).Exec(`UPDATE draft_approvals a SET status=?,responded_at=CURRENT_TIMESTAMP
 FROM groups g WHERE a.group_id=g.id AND g.leader_id=a.leader_id AND a.leader_id=? AND a.draft_id=? AND a.status='pending'`, status, userID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrAccessDenied
	}
	return nil
}

func (r *migrationRepository) HasConsensus(ctx context.Context, dbID, id uint) (bool, error) {
	var approved bool
	err := r.db.WithContext(ctx).Raw(`SELECT count(*)>0 AND bool_and(COALESCE(a.status='approved' AND a.leader_id=g.leader_id,false))
 FROM group_databases gd JOIN groups g ON g.id=gd.group_id
 JOIN drafts d ON d.id=? AND d.database_id=gd.database_id
 LEFT JOIN draft_approvals a ON a.group_id=g.id AND a.draft_id=d.id AND a.round=d.current_round
 WHERE gd.database_id=?`, id, dbID).Scan(&approved).Error
	return approved, err
}

func (r *migrationRepository) FindActiveExecution(ctx context.Context, nodeID uint) (uint, error) {
	var row model.Migration
	err := r.db.WithContext(ctx).Select("id").Where("(id=? OR source_migration_id=?) AND reverts_migration_id IS NULL AND status='success'", nodeID, nodeID).Order("id DESC").Take(&row).Error
	return row.ID, err
}

func (r *migrationRepository) FinishDraft(ctx context.Context, id uint, audit *domain.Migration, head, parent, forward *uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := &migrationRepository{tx}
		saved, err := repo.Save(ctx, audit)
		if err != nil {
			return err
		}
		if head == nil {
			head, forward = &saved.ID, &saved.ID
		}
		*audit = *saved
		if err = repo.UpdatePosition(ctx, audit.DatabaseID, head, parent, forward, audit.ExecutionID); err != nil {
			return err
		}
		return repo.UpdateDraftStatus(ctx, id, "applied")
	})
}
