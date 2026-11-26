package draft

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/telemetry"
)

type dbLock struct {
	mu     sync.Mutex
	locked atomic.Bool
}

func (l *dbLock) TryLock() bool {
	if l.mu.TryLock() {
		l.locked.Store(true)
		return true
	}
	return false
}

func (l *dbLock) Unlock() {
	if l.locked.CompareAndSwap(true, false) {
		l.mu.Unlock()
	}
}

type draftService struct {
	draftRepo        ports.DraftRepository
	groupRepo        ports.GroupRepository
	databaseRepo     ports.DatabaseRepository
	userGroupService ports.UserGroupService
	taskQueue        ports.TaskQueue
	databaseDriver   ports.DatabaseDriver
	migrationRepo    ports.MigrationRepository
	dbLocks          sync.Map
}

var _ ports.DraftService = (*draftService)(nil)

func NewDraftService(
	draftRepo ports.DraftRepository,
	groupRepo ports.GroupRepository,
	userGroupService ports.UserGroupService,
	extra ...any,
) ports.DraftService {
	s := &draftService{
		draftRepo:        draftRepo,
		groupRepo:        groupRepo,
		userGroupService: userGroupService,
	}
	for _, dep := range extra {
		switch d := dep.(type) {
		case ports.DatabaseRepository:
			s.databaseRepo = d
		case ports.TaskQueue:
			s.taskQueue = d
		case ports.DatabaseDriver:
			s.databaseDriver = d
		case ports.MigrationRepository:
			s.migrationRepo = d
		}
	}
	return s
}

func (s *draftService) CreateDraft(ctx context.Context, input domain.CreateDraftInput) (*domain.Draft, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.IsBatch {
		tbl, err := sqlutil.ValidateBatchDML(input.SqlUp)
		if err != nil {
			if errors.Is(err, sqlutil.ErrBatchContainsDDL) {
				return nil, domain.ErrBatchContainsDDL
			}
			if errors.Is(err, sqlutil.ErrBatchMultipleUnrelatedTables) {
				return nil, domain.ErrBatchMultipleUnrelatedTables
			}
			return nil, domain.ErrInvalidDraftInput
		}
		if s.databaseDriver != nil && s.databaseRepo != nil {
			db, err := s.databaseRepo.FindWithCredentialByID(ctx, input.DatabaseID)
			if err == nil {
				meta, err := s.databaseDriver.GetDatabaseMetadata(ctx, db)
				if err == nil && meta != nil && len(meta.Columns) > 0 {
					foundTable := false
					foundCol := false
					for _, col := range meta.Columns {
						if strings.EqualFold(col.Table, tbl) {
							foundTable = true
							if strings.EqualFold(col.Name, input.BatchColumn) {
								foundCol = true
								break
							}
						}
					}
					if foundTable && !foundCol {
						return nil, domain.ErrBatchColumnNotFound
					}
				}
			}
		}
	}

	if err := s.userGroupService.Authorize(ctx, input.UserID, input.DatabaseID, domain.AccessViewEdit); err != nil {
		return nil, domain.ErrForbidden
	}

	groups, err := s.groupRepo.FindAllByDatabaseID(ctx, input.DatabaseID)
	if err != nil {
		return nil, err
	}

	isReversible := true
	if input.IsReversible != nil {
		isReversible = *input.IsReversible
	}
	now := time.Now()

	draft := &domain.Draft{
		DatabaseID:         input.DatabaseID,
		UserID:             input.UserID,
		SqlUp:              input.SqlUp,
		SqlDown:            input.SqlDown,
		IsReversible:       isReversible,
		IrreversibleReason: input.IrreversibleReason,
		IsBatch:            input.IsBatch,
		BatchColumn:        input.BatchColumn,
		BatchSize:          input.BatchSize,
		CurrentRound:       1,
		Status:             domain.DraftStatusPendingApproval,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	var savedDraft *domain.Draft
	err = s.draftRepo.WithTransaction(ctx, func(txRepo ports.DraftRepository) error {
		var saveErr error
		savedDraft, saveErr = txRepo.Save(ctx, draft)
		if saveErr != nil {
			return saveErr
		}

		approvals := make([]domain.DraftApproval, 0, len(groups))
		for _, grp := range groups {
			var leaderID *uint
			if grp.LeaderID != 0 {
				lid := grp.LeaderID
				leaderID = &lid
			}
			approval := &domain.DraftApproval{
				DraftID:  savedDraft.ID,
				GroupID:  grp.ID,
				LeaderID: leaderID,
				Status:   domain.ApprovalStatusPending,
			}
			savedApproval, err := txRepo.SaveApproval(ctx, approval)
			if err != nil {
				return err
			}
			approvals = append(approvals, *savedApproval)
		}

		savedDraft.Approvals = approvals
		return nil
	})
	if err != nil {
		return nil, err
	}

	return savedDraft, nil
}

func (s *draftService) GetDraftsByDatabaseID(
	ctx context.Context,
	databaseID uint,
	userID uint,
	params pagination.PaginationParams,
) ([]*domain.Draft, int64, error) {
	if databaseID == 0 || userID == 0 {
		return nil, 0, domain.ErrInvalidDraftInput
	}

	if err := s.userGroupService.Authorize(ctx, userID, databaseID, domain.AccessViewOnly); err != nil {
		return nil, 0, domain.ErrForbidden
	}

	return s.draftRepo.FindAllByDatabaseID(ctx, databaseID, params)
}

func (s *draftService) GetDraftByID(ctx context.Context, draftID uint, userID uint) (*domain.Draft, error) {
	if draftID == 0 || userID == 0 {
		return nil, domain.ErrInvalidDraftInput
	}

	draft, err := s.draftRepo.FindByID(ctx, draftID)
	if err != nil {
		return nil, err
	}

	if err := s.userGroupService.Authorize(ctx, userID, draft.DatabaseID, domain.AccessViewOnly); err != nil {
		return nil, domain.ErrForbidden
	}

	return draft, nil
}

func (s *draftService) SubmitApproval(ctx context.Context, input domain.SubmitApprovalInput) (*domain.Draft, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	var resultDraft *domain.Draft
	var invalidated bool

	err := s.draftRepo.WithTransaction(ctx, func(txRepo ports.DraftRepository) error {
		draft, err := txRepo.FindByID(ctx, input.DraftID)
		if err != nil {
			return err
		}

		if draft.Status != domain.DraftStatusPendingApproval &&
			draft.Status != domain.DraftStatusPartial &&
			draft.Status != domain.DraftStatusPartiallyReverted {
			return domain.ErrInvalidDraftState
		}

		if s.userGroupService != nil {
			if authErr := s.userGroupService.Authorize(ctx, draft.UserID, draft.DatabaseID, domain.AccessViewEdit); authErr != nil {
				draft.Status = domain.DraftStatusInvalidated
				draft.UpdatedAt = time.Now()
				if _, updErr := txRepo.Update(ctx, draft); updErr != nil {
					return updErr
				}
				invalidated = true
				return nil
			}
		}

		round := draft.CurrentRound
		if round == 0 {
			round = 1
		}

		approvals, err := txRepo.FindApprovalsByDraftAndRound(ctx, draft.ID, round)
		if err != nil || len(approvals) == 0 {
			approvals, err = txRepo.FindApprovalsByDraftID(ctx, draft.ID)
			if err != nil {
				return err
			}
		}

		var matchedApprovals []*domain.DraftApproval
		for _, app := range approvals {
			if app.LeaderID != nil && *app.LeaderID == input.UserID {
				matchedApprovals = append(matchedApprovals, app)
			}
		}

		if len(matchedApprovals) == 0 {
			return domain.ErrUnauthorized
		}

		vote := strings.ToLower(strings.TrimSpace(input.Vote))
		if vote == "" {
			vote = strings.ToLower(strings.TrimSpace(input.Status))
		}
		now := time.Now()

		for _, app := range matchedApprovals {
			app.Vote = vote
			app.Status = vote
			app.RespondedAt = &now
			if _, err := txRepo.UpdateApproval(ctx, app); err != nil {
				return err
			}
		}

		updatedApprovals, err := txRepo.FindApprovalsByDraftAndRound(ctx, draft.ID, round)
		if err != nil || len(updatedApprovals) == 0 {
			updatedApprovals, err = txRepo.FindApprovalsByDraftID(ctx, draft.ID)
			if err != nil {
				return err
			}
		}

		switch draft.Status {
		case domain.DraftStatusPendingApproval:
			if vote == domain.ApprovalStatusRejected {
				draft.Status = domain.DraftStatusRejected
				draft.UpdatedAt = now
				updatedDraft, err := txRepo.Update(ctx, draft)
				if err != nil {
					return err
				}
				resultDraft = updatedDraft
				return nil
			}

			allApproved := len(updatedApprovals) > 0
			for _, app := range updatedApprovals {
				v := strings.ToLower(strings.TrimSpace(app.Vote))
				if v != domain.ApprovalStatusApproved && app.Status != domain.ApprovalStatusApproved {
					allApproved = false
					break
				}
			}

			if allApproved {
				draft.Status = domain.DraftStatusApproved
				draft.UpdatedAt = now
				updatedDraft, err := txRepo.Update(ctx, draft)
				if err != nil {
					return err
				}
				resultDraft = updatedDraft
				return nil
			}

		case domain.DraftStatusPartial:
			hasRevert := false
			allResume := len(updatedApprovals) > 0
			for _, app := range updatedApprovals {
				v := strings.ToLower(strings.TrimSpace(app.Vote))
				if v == domain.VoteRevert {
					hasRevert = true
				}
				if v != domain.VoteResume {
					allResume = false
				}
			}

			if hasRevert {
				draft.RecoveryAction = domain.VoteRevert
				draft.UpdatedAt = now
				updatedDraft, err := txRepo.Update(ctx, draft)
				if err != nil {
					return err
				}
				resultDraft = updatedDraft
				return nil
			} else if allResume {
				draft.RecoveryAction = domain.VoteResume
				draft.UpdatedAt = now
				updatedDraft, err := txRepo.Update(ctx, draft)
				if err != nil {
					return err
				}
				resultDraft = updatedDraft
				return nil
			}

		case domain.DraftStatusPartiallyReverted:
			hasNo := false
			allYes := len(updatedApprovals) > 0
			for _, app := range updatedApprovals {
				v := strings.ToLower(strings.TrimSpace(app.Vote))
				if v == domain.VoteNo {
					hasNo = true
				}
				if v != domain.VoteYes {
					allYes = false
				}
			}

			if hasNo {
				draft.Status = domain.DraftStatusFailed
				draft.RecoveryAction = "aborted"
				draft.UpdatedAt = now
				updatedDraft, err := txRepo.Update(ctx, draft)
				if err != nil {
					return err
				}

				if s.migrationRepo != nil {
					mig := &domain.Migration{
						DatabaseID:   draft.DatabaseID,
						UserID:       input.UserID,
						DraftID:      &draft.ID,
						Status:       domain.MigrationStatusFail,
						ErrorMessage: "[REVERT_ABORTED_BY_LEADER] Group leader voted to abort revert. Database left in intermediate state.",
						MigratedAt:   now,
					}
					_, _ = s.migrationRepo.Save(ctx, mig)
				}
				resultDraft = updatedDraft
				return nil
			} else if allYes {
				draft.RecoveryAction = domain.VoteRevert
				draft.UpdatedAt = now
				updatedDraft, err := txRepo.Update(ctx, draft)
				if err != nil {
					return err
				}
				resultDraft = updatedDraft
				return nil
			}
		}

		draft.Approvals = make([]domain.DraftApproval, 0, len(updatedApprovals))
		for _, app := range updatedApprovals {
			draft.Approvals = append(draft.Approvals, *app)
		}
		resultDraft = draft
		return nil
	})
	if err != nil {
		return nil, err
	}
	if invalidated {
		return nil, domain.ErrInvalidDraftState
	}

	return resultDraft, nil
}

func (s *draftService) ValidateDraft(ctx context.Context, draftID uint) (*domain.Draft, error) {
	if draftID == 0 {
		return nil, domain.ErrInvalidDraftInput
	}

	draft, err := s.draftRepo.FindByID(ctx, draftID)
	if err != nil {
		return nil, err
	}

	if draft.IsTerminal() {
		return draft, nil
	}

	if err := s.userGroupService.Authorize(ctx, draft.UserID, draft.DatabaseID, domain.AccessViewEdit); err != nil {
		draft.Status = domain.DraftStatusInvalidated
		draft.UpdatedAt = time.Now()
		return s.draftRepo.Update(ctx, draft)
	}

	return draft, nil
}

func (s *draftService) ExecuteDraft(ctx context.Context, draftID uint, userID uint) error {
	if draftID == 0 || userID == 0 {
		return domain.ErrInvalidDraftInput
	}

	draft, err := s.draftRepo.FindByID(ctx, draftID)
	if err != nil {
		return err
	}

	if s.databaseRepo != nil {
		db, err := s.databaseRepo.FindById(ctx, draft.DatabaseID)
		if err != nil || db == nil {
			return domain.ErrDatabaseNotFound
		}
	}

	if draft.UserID != userID {
		return domain.ErrForbidden
	}

	if s.userGroupService != nil {
		if authErr := s.userGroupService.Authorize(ctx, draft.UserID, draft.DatabaseID, domain.AccessViewEdit); authErr != nil {
			draft.Status = domain.DraftStatusInvalidated
			draft.UpdatedAt = time.Now()
			_, _ = s.draftRepo.Update(ctx, draft)
			return domain.ErrDraftNotApproved
		}
	}

	if draft.Status != domain.DraftStatusApproved {
		return domain.ErrDraftNotApproved
	}

	if s.draftRepo != nil {
		partiallyReverted, err := s.draftRepo.FindDraftByStatusAndDatabaseID(ctx, draft.DatabaseID, domain.DraftStatusPartiallyReverted)
		if err == nil && partiallyReverted != nil && partiallyReverted.ID != 0 {
			return domain.ErrDatabaseMustResolvePartiallyReverted
		}

		partialDraft, err := s.draftRepo.FindDraftByStatusAndDatabaseID(ctx, draft.DatabaseID, domain.DraftStatusPartial)
		if err == nil && partialDraft != nil && partialDraft.ID != 0 {
			return domain.ErrDatabaseMustResolvePartiallyReverted
		}

		executingDraft, err := s.draftRepo.FindDraftByStatusAndDatabaseID(ctx, draft.DatabaseID, domain.DraftStatusExecuting)
		if err == nil && executingDraft != nil && executingDraft.ID != 0 {
			return domain.ErrMigrationInProgress
		}
	}

	val, _ := s.dbLocks.LoadOrStore(draft.DatabaseID, &dbLock{})
	lock := val.(*dbLock)
	if !lock.TryLock() {
		return domain.ErrMigrationInProgress
	}

	var execID uint
	if s.databaseRepo != nil {
		incID, err := s.databaseRepo.IncrementExecutionID(ctx, draft.DatabaseID)
		if err != nil {
			lock.Unlock()
			return err
		}
		execID = incID
	}

	draft.Status = domain.DraftStatusExecuting
	draft.UpdatedAt = time.Now()
	if _, err := s.draftRepo.Update(ctx, draft); err != nil {
		lock.Unlock()
		return err
	}

	if s.taskQueue != nil {
		carrier := make(map[string]string)
		telemetry.Inject(ctx, carrier)
		task := ports.ExecutionTask{
			DraftID:      draft.ID,
			DatabaseID:   draft.DatabaseID,
			ExecutionID:  execID,
			UserID:       userID,
			Operation:    "execute",
			EnqueuedAt:   time.Now(),
			TraceCarrier: carrier,
		}
		if err := s.taskQueue.Enqueue(ctx, task); err != nil {
			lock.Unlock()
			return err
		}
	}

	return nil
}

func (s *draftService) ResumeBatch(ctx context.Context, draftID uint, userID uint) error {
	if draftID == 0 || userID == 0 {
		return domain.ErrInvalidDraftInput
	}

	draft, err := s.draftRepo.FindByID(ctx, draftID)
	if err != nil {
		return err
	}

	if draft.UserID != userID {
		return domain.ErrForbidden
	}

	if s.userGroupService != nil {
		if err := s.userGroupService.Authorize(ctx, userID, draft.DatabaseID, domain.AccessViewEdit); err != nil {
			return domain.ErrForbidden
		}
	}

	if draft.Status != domain.DraftStatusPartial || draft.RecoveryAction != domain.VoteResume {
		return domain.ErrInvalidDraftState
	}

	val, _ := s.dbLocks.LoadOrStore(draft.DatabaseID, &dbLock{})
	lock := val.(*dbLock)
	if !lock.TryLock() {
		return domain.ErrMigrationInProgress
	}

	var execID uint
	if s.databaseRepo != nil {
		db, err := s.databaseRepo.FindById(ctx, draft.DatabaseID)
		if err != nil {
			lock.Unlock()
			return err
		}
		if db != nil {
			execID = db.LatestExecutionID
		}
	}

	draft.Status = domain.DraftStatusExecuting
	draft.UpdatedAt = time.Now()
	if _, err := s.draftRepo.Update(ctx, draft); err != nil {
		lock.Unlock()
		return err
	}

	if s.taskQueue != nil {
		carrier := make(map[string]string)
		telemetry.Inject(ctx, carrier)
		task := ports.ExecutionTask{
			DraftID:      draft.ID,
			DatabaseID:   draft.DatabaseID,
			ExecutionID:  execID,
			UserID:       userID,
			Operation:    "resume",
			EnqueuedAt:   time.Now(),
			TraceCarrier: carrier,
		}
		if err := s.taskQueue.Enqueue(ctx, task); err != nil {
			lock.Unlock()
			return err
		}
	}

	return nil
}

func (s *draftService) RevertBatch(ctx context.Context, draftID uint, userID uint) error {
	if draftID == 0 || userID == 0 {
		return domain.ErrInvalidDraftInput
	}

	draft, err := s.draftRepo.FindByID(ctx, draftID)
	if err != nil {
		return err
	}

	if draft.UserID != userID {
		return domain.ErrForbidden
	}

	if s.userGroupService != nil {
		if err := s.userGroupService.Authorize(ctx, userID, draft.DatabaseID, domain.AccessViewEdit); err != nil {
			return domain.ErrForbidden
		}
	}

	if draft.RecoveryAction != domain.VoteRevert {
		return domain.ErrInvalidDraftState
	}

	val, _ := s.dbLocks.LoadOrStore(draft.DatabaseID, &dbLock{})
	lock := val.(*dbLock)
	if !lock.TryLock() {
		return domain.ErrMigrationInProgress
	}

	var newExecID uint
	if s.databaseRepo != nil {
		incID, err := s.databaseRepo.IncrementExecutionID(ctx, draft.DatabaseID)
		if err != nil {
			lock.Unlock()
			return err
		}
		newExecID = incID
	}

	draft.Status = domain.DraftStatusExecuting
	draft.UpdatedAt = time.Now()
	if _, err := s.draftRepo.Update(ctx, draft); err != nil {
		lock.Unlock()
		return err
	}

	if s.taskQueue != nil {
		carrier := make(map[string]string)
		telemetry.Inject(ctx, carrier)
		task := ports.ExecutionTask{
			DraftID:      draft.ID,
			DatabaseID:   draft.DatabaseID,
			ExecutionID:  newExecID,
			UserID:       userID,
			Operation:    "revert",
			EnqueuedAt:   time.Now(),
			TraceCarrier: carrier,
		}
		if err := s.taskQueue.Enqueue(ctx, task); err != nil {
			lock.Unlock()
			return err
		}
	}

	return nil
}

func (s *draftService) GetBatchProgress(ctx context.Context, draftID uint, userID uint) (*domain.BatchProgress, error) {
	if draftID == 0 || userID == 0 {
		return nil, domain.ErrInvalidDraftInput
	}

	draft, err := s.draftRepo.FindByID(ctx, draftID)
	if err != nil {
		return nil, err
	}

	if s.userGroupService != nil {
		if err := s.userGroupService.Authorize(ctx, userID, draft.DatabaseID, domain.AccessViewOnly); err != nil {
			return nil, domain.ErrForbidden
		}
	}

	checkpoints, _ := s.draftRepo.FindCheckpointsByDraftID(ctx, draft.ID)
	var committedCount int
	var rowsProcessed int64
	for _, cp := range checkpoints {
		if cp.Status == "committed" {
			committedCount++
			rowsProcessed += cp.RowsAffected
		}
	}

	var totalRows int64
	if s.databaseDriver != nil && s.databaseRepo != nil && draft.IsBatch {
		db, err := s.databaseRepo.FindWithCredentialByID(ctx, draft.DatabaseID)
		if err == nil {
			tbl, vErr := sqlutil.ValidateBatchDML(draft.SqlUp)
			if vErr == nil && tbl != "" {
				countRes, qErr := s.databaseDriver.ExecuteQuery(ctx, db, fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tbl))
				if qErr == nil && countRes != nil && len(countRes.Rows) > 0 {
					if cVal, ok := countRes.Rows[0]["count"]; ok {
						switch v := cVal.(type) {
						case int64:
							totalRows = v
						case float64:
							totalRows = int64(v)
						case int:
							totalRows = int64(v)
						}
					}
				}
			}
		}
	}

	batchSize := int64(draft.BatchSize)
	if batchSize <= 0 {
		batchSize = 1000
	}

	totalBatches := committedCount
	if totalRows > 0 {
		totalBatches = int((totalRows + batchSize - 1) / batchSize)
	} else if totalBatches == 0 {
		totalBatches = 1
	}

	var percentage float64 = 0.0
	if draft.Status == domain.DraftStatusApplied {
		percentage = 100.0
		if totalRows == 0 {
			totalRows = rowsProcessed
		}
	} else if totalRows > 0 {
		percentage = math.Round((float64(rowsProcessed)/float64(totalRows))*10000) / 100
		if percentage > 100.0 {
			percentage = 100.0
		}
	}

	return &domain.BatchProgress{
		CurrentBatch:  committedCount,
		TotalBatches:  totalBatches,
		RowsProcessed: rowsProcessed,
		TotalRows:     totalRows,
		Percentage:    percentage,
		Status:        draft.Status,
	}, nil
}

func (s *draftService) UnlockDatabase(databaseID uint) {
	if val, ok := s.dbLocks.Load(databaseID); ok {
		lock := val.(*dbLock)
		lock.Unlock()
	}
}
