//go:build integration

package integration_test

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	queuepostgres "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/queue/postgresql"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/gorm"
)

func TestMigrationQueue_GivenPersistedJobs_WhenWorkersRestart_ThenPendingRunsOnceAndInterruptedJobIsHeld(t *testing.T) {
	// Arrange
	db := requirePlatformDB(t)
	pending := seedQueueJob(t, db, "pending_db", "pending")
	running := seedQueueJob(t, db, "running_db", "running")
	var executions atomic.Int32
	handler := func(ctx context.Context, task ports.ExecutionTask) error {
		executions.Add(1)
		return db.WithContext(ctx).Model(&model.Draft{}).Where("id = ?", task.DraftID).
			Update("status", domain.DraftStatusApplied).Error
	}

	// Act
	first := queuepostgres.NewQueue(db, handler, 1, 4, 5*time.Second)
	defer first.Close()
	second := queuepostgres.NewQueue(db, handler, 1, 4, 5*time.Second)
	defer second.Close()
	waitForJobStatus(t, db, pending.ID, "done")
	waitForJobStatus(t, db, running.ID, "uncertain")

	// Assert
	if executions.Load() != 1 {
		t.Fatalf("expected only the pending job to execute once, got %d executions", executions.Load())
	}
	_, err := second.Submit(context.Background(), ports.ExecutionTask{
		DraftID: running.DraftID, DatabaseID: running.DatabaseID, Operation: ports.MetricOpExecute,
	})
	if !errors.Is(err, domain.ErrMigrationInProgress) {
		t.Fatalf("expected uncertain job to block a new execution, got %v", err)
	}
}

func TestMigrationQueue_GivenApprovedDraft_WhenSubmittedTwice_ThenOnlyOneJobIsStored(t *testing.T) {
	// Arrange
	db := requirePlatformDB(t)
	draft := createQueueDraft(t, db, "exclusive_db", domain.DraftStatusApproved)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	queue := queuepostgres.NewQueue(db, func(ctx context.Context, _ ports.ExecutionTask) error {
		<-ctx.Done()
		return domain.ErrMigrationOutcomeUnknown
	}, 1, 4, time.Second)
	defer queue.Close()
	task := ports.ExecutionTask{
		DraftID: draft.ID, DatabaseID: draft.DatabaseID, UserID: draft.UserID, Operation: ports.MetricOpExecute,
	}

	// Act
	jobID, err := queue.Submit(ctx, task)
	if err != nil {
		t.Fatal(err)
	}
	_, duplicateErr := queue.Submit(ctx, task)

	// Assert
	if !errors.Is(duplicateErr, domain.ErrMigrationInProgress) {
		t.Fatalf("expected duplicate submission to be rejected, got %v", duplicateErr)
	}
	var jobs []model.MigrationJob
	if err := db.Where("draft_id = ?", draft.ID).Find(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || strconv.FormatUint(uint64(jobs[0].ID), 10) != jobID || jobs[0].ExecutionID != 1 {
		t.Fatalf("expected one persisted execution for submitted job %q, got %+v", jobID, jobs)
	}
}

func createQueueDraft(t *testing.T, db *gorm.DB, name, status string) model.Draft {
	t.Helper()
	user := insertTestUser(t, &testStack{db: db}, name+"_owner", "password123", domain.RoleUser)
	target := model.Database{Name: name, Host: "localhost", Port: 5432, User: user.Username, Type: "postgresql"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	draft := model.Draft{DatabaseID: target.ID, UserID: user.ID, SqlUp: "SELECT 1", Status: status}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	return draft
}

func seedQueueJob(t *testing.T, db *gorm.DB, name, status string) model.MigrationJob {
	t.Helper()
	draft := createQueueDraft(t, db, name, domain.DraftStatusExecuting)
	job := model.MigrationJob{
		DraftID: draft.ID, DatabaseID: draft.DatabaseID, UserID: draft.UserID, ExecutionID: 1,
		Operation: ports.MetricOpExecute, Status: status, TraceCarrier: []byte("{}"),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	return job
}

func waitForJobStatus(t *testing.T, db *gorm.DB, id uint, status string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var job model.MigrationJob
		if err := db.Take(&job, id).Error; err != nil {
			t.Fatal(err)
		}
		if job.Status == status {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %d did not reach status %s", id, status)
}
