package postgresql

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (q *Queue) Enqueue(ctx context.Context, task ports.ExecutionTask) error {
	_, err := q.Submit(ctx, task)
	return err
}

func (q *Queue) Submit(ctx context.Context, task ports.ExecutionTask) (string, error) {
	if err := q.ctx.Err(); err != nil {
		return "", err
	}
	carrier, err := json.Marshal(task.TraceCarrier)
	if err != nil {
		return "", err
	}
	job := model.MigrationJob{
		DatabaseID: task.DatabaseID, DraftID: task.DraftID, UserID: task.UserID,
		Operation: task.Operation, Status: jobPending, TraceCarrier: carrier,
	}
	err = q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return q.saveSubmission(tx, &job)
	})
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(uint64(job.ID), 10), nil
}

func (q *Queue) saveSubmission(tx *gorm.DB, job *model.MigrationJob) error {
	var db model.Database
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(&db, job.DatabaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrDatabaseNotFound
		}
		return err
	}
	var draft model.Draft
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("database_id = ?", db.ID).Take(&draft, job.DraftID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrDraftNotFound
		}
		return err
	}
	if err := checkSubmission(tx, db, draft, job.Operation); err != nil {
		return err
	}
	if err := q.checkCapacity(tx); err != nil {
		return err
	}

	job.ExecutionID = db.LatestExecutionID
	if job.Operation != ports.MetricOpResume {
		job.ExecutionID++
		if err := tx.Model(&db).Update("latest_execution_id", job.ExecutionID).Error; err != nil {
			return err
		}
	}
	if err := tx.Create(job).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrMigrationInProgress
		}
		return err
	}
	return tx.Model(&draft).Update("status", domain.DraftStatusExecuting).Error
}

func (q *Queue) checkCapacity(tx *gorm.DB) error {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(148122, 0)").Error; err != nil {
		return err
	}
	var waiting int64
	if err := tx.Model(&model.MigrationJob{}).Where("status = ?", jobPending).Count(&waiting).Error; err != nil {
		return err
	}
	if waiting >= int64(q.capacity) {
		return domain.ErrQueueFull
	}
	return nil
}

func checkSubmission(tx *gorm.DB, db model.Database, draft model.Draft, operation string) error {
	var active int64
	if err := tx.Model(&model.MigrationJob{}).
		Where("database_id = ? AND status IN ?", db.ID, []string{jobPending, jobRunning, jobUncertain}).
		Count(&active).Error; err != nil {
		return err
	}
	if active > 0 {
		return domain.ErrMigrationInProgress
	}
	switch operation {
	case ports.MetricOpExecute:
		if draft.Status != domain.DraftStatusApproved {
			return domain.ErrDraftNotApproved
		}
		if (draft.Operation == domain.DraftOperationRollback || draft.Operation == domain.DraftOperationForward) &&
			draft.BaseExecutionID != db.LatestExecutionID {
			return domain.ErrDraftConflict
		}
	case ports.MetricOpResume:
		if draft.Status != domain.DraftStatusPartial || draft.RecoveryAction != domain.VoteResume {
			return domain.ErrInvalidDraftState
		}
	case ports.MetricOpRevert:
		if (draft.Status != domain.DraftStatusPartial && draft.Status != domain.DraftStatusPartiallyReverted) ||
			draft.RecoveryAction != domain.VoteRevert {
			return domain.ErrInvalidDraftState
		}
	default:
		return domain.ErrInvalidDraftInput
	}
	return checkOtherDrafts(tx, db.ID, draft.ID)
}

func checkOtherDrafts(tx *gorm.DB, databaseID, draftID uint) error {
	var active model.Draft
	err := tx.Select("status").Where("database_id = ? AND id <> ? AND status IN ?", databaseID, draftID,
		[]string{domain.DraftStatusPartial, domain.DraftStatusPartiallyReverted, domain.DraftStatusExecuting}).
		Take(&active).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil
	case err != nil:
		return err
	case active.Status == domain.DraftStatusExecuting:
		return domain.ErrMigrationInProgress
	default:
		return domain.ErrDatabaseMustResolvePartiallyReverted
	}
}
