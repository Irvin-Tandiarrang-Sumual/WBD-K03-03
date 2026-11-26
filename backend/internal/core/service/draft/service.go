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

	if err := s.validateBatchMetadata(ctx, input); err != nil {
		return nil, err
	}

	if err := s.userGroupService.Authorize(ctx, input.UserID, input.DatabaseID, domain.AccessViewEdit); err != nil {
		return nil, domain.ErrForbidden
	}

	groups, err := s.groupRepo.FindAllByDatabaseID(ctx, input.DatabaseID)
	if err != nil {
		return nil, err
	}

	draft := buildNewDraftEntity(input)

	var savedDraft *domain.Draft
	err = s.draftRepo.WithTransaction(ctx, func(txRepo ports.DraftRepository) error {
		var saveErr error
		savedDraft, saveErr = txRepo.Save(ctx, draft)
		if saveErr != nil {
			return saveErr
		}

		approvals, err := createInitialGroupApprovals(ctx, txRepo, savedDraft.ID, groups)
		if err != nil {
			return err
		}
		savedDraft.Approvals = approvals
		return nil
	})
	if err != nil {
		return nil, err
	}

	return savedDraft, nil
}

func (s *draftService) validateBatchMetadata(ctx context.Context, input domain.CreateDraftInput) error {
	if !input.IsBatch {
		return nil
	}

	tbl, err := sqlutil.ValidateBatchDML(input.SqlUp)
	if err != nil {
		if errors.Is(err, sqlutil.ErrBatchContainsDDL) {
			return domain.ErrBatchContainsDDL
		}
		if errors.Is(err, sqlutil.ErrBatchMultipleUnrelatedTables) {
			return domain.ErrBatchMultipleUnrelatedTables
		}
		return domain.ErrInvalidDraftInput
	}

	if s.databaseDriver == nil || s.databaseRepo == nil {
		return nil
	}

	db, err := s.databaseRepo.FindWithCredentialByID(ctx, input.DatabaseID)
	if err != nil {
		return nil
	}

	meta, err := s.databaseDriver.GetDatabaseMetadata(ctx, db)
	if err != nil || meta == nil || len(meta.Columns) == 0 {
		return nil
	}

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
		return domain.ErrBatchColumnNotFound
	}

	return nil
}

func buildNewDraftEntity(input domain.CreateDraftInput) *domain.Draft {
	isReversible := true
	if input.IsReversible != nil {
		isReversible = *input.IsReversible
	}
	now := time.Now()
	return &domain.Draft{
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
}

func createInitialGroupApprovals(
	ctx context.Context,
	txRepo ports.DraftRepository,
	draftID uint,
	groups []domain.Group,
) ([]domain.DraftApproval, error) {
	approvals := make([]domain.DraftApproval, 0, len(groups))
	for _, grp := range groups {
		var leaderID *uint
		if grp.LeaderID != 0 {
			lid := grp.LeaderID
			leaderID = &lid
		}
		approval := &domain.DraftApproval{
			DraftID:  draftID,
			GroupID:  grp.ID,
			LeaderID: leaderID,
			Status:   domain.ApprovalStatusPending,
		}
		savedApproval, err := txRepo.SaveApproval(ctx, approval)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, *savedApproval)
	}
	return approvals, nil
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

		if !isVotableDraftStatus(draft.Status) {
			return domain.ErrInvalidDraftState
		}

		if inv, err := s.invalidateIfAuthorUnauthorized(ctx, txRepo, draft); err != nil {
			return err
		} else if inv {
			invalidated = true
			return nil
		}

		round := draft.CurrentRound
		if round == 0 {
			round = 1
		}

		approvals, err := fetchRoundApprovals(ctx, txRepo, draft.ID, round)
		if err != nil {
			return err
		}

		vote := normalizeVote(input.Vote, input.Status)
		now := time.Now()

		if err := applyLeaderVote(ctx, txRepo, approvals, input.UserID, vote, now); err != nil {
			return err
		}

		updatedApprovals, err := fetchRoundApprovals(ctx, txRepo, draft.ID, round)
		if err != nil {
			return err
		}

		resolvedDraft, err := s.resolveConsensus(ctx, txRepo, draft, updatedApprovals, vote, input.UserID, now)
		if err != nil {
			return err
		}
		if resolvedDraft != nil {
			resultDraft = resolvedDraft
			return nil
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

func isVotableDraftStatus(status string) bool {
	return status == domain.DraftStatusPendingApproval ||
		status == domain.DraftStatusPartial ||
		status == domain.DraftStatusPartiallyReverted
}

func normalizeVote(vote, fallback string) string {
	v := strings.ToLower(strings.TrimSpace(vote))
	if v == "" {
		v = strings.ToLower(strings.TrimSpace(fallback))
	}
	return v
}

func (s *draftService) invalidateIfAuthorUnauthorized(
	ctx context.Context,
	repo ports.DraftRepository,
	draft *domain.Draft,
) (bool, error) {
	if s.userGroupService == nil {
		return false, nil
	}
	if authErr := s.userGroupService.Authorize(ctx, draft.UserID, draft.DatabaseID, domain.AccessViewEdit); authErr != nil {
		draft.Status = domain.DraftStatusInvalidated
		draft.UpdatedAt = time.Now()
		if _, updErr := repo.Update(ctx, draft); updErr != nil {
			return true, updErr
		}
		return true, nil
	}
	return false, nil
}

func fetchRoundApprovals(ctx context.Context, repo ports.DraftRepository, draftID uint, round uint) ([]*domain.DraftApproval, error) {
	approvals, err := repo.FindApprovalsByDraftAndRound(ctx, draftID, round)
	if err != nil || len(approvals) == 0 {
		approvals, err = repo.FindApprovalsByDraftID(ctx, draftID)
		if err != nil {
			return nil, err
		}
	}
	return approvals, nil
}

func applyLeaderVote(
	ctx context.Context,
	repo ports.DraftRepository,
	approvals []*domain.DraftApproval,
	userID uint,
	vote string,
	now time.Time,
) error {
	var matched []*domain.DraftApproval
	for _, app := range approvals {
		if app.LeaderID != nil && *app.LeaderID == userID {
			matched = append(matched, app)
		}
	}

	if len(matched) == 0 {
		return domain.ErrUnauthorized
	}

	for _, app := range matched {
		app.Vote = vote
		app.Status = vote
		app.RespondedAt = &now
		if _, err := repo.UpdateApproval(ctx, app); err != nil {
			return err
		}
	}
	return nil
}

func (s *draftService) resolveConsensus(
	ctx context.Context,
	repo ports.DraftRepository,
	draft *domain.Draft,
	approvals []*domain.DraftApproval,
	vote string,
	userID uint,
	now time.Time,
) (*domain.Draft, error) {
	switch draft.Status {
	case domain.DraftStatusPendingApproval:
		return resolvePendingApprovalConsensus(ctx, repo, draft, approvals, vote, now)
	case domain.DraftStatusPartial:
		return resolvePartialRecoveryConsensus(ctx, repo, draft, approvals, now)
	case domain.DraftStatusPartiallyReverted:
		return s.resolvePartiallyRevertedConsensus(ctx, repo, draft, approvals, userID, now)
	default:
		return nil, nil
	}
}

func resolvePendingApprovalConsensus(
	ctx context.Context,
	repo ports.DraftRepository,
	draft *domain.Draft,
	approvals []*domain.DraftApproval,
	vote string,
	now time.Time,
) (*domain.Draft, error) {
	if vote == domain.ApprovalStatusRejected {
		draft.Status = domain.DraftStatusRejected
		draft.UpdatedAt = now
		return repo.Update(ctx, draft)
	}

	allApproved := len(approvals) > 0
	for _, app := range approvals {
		v := strings.ToLower(strings.TrimSpace(app.Vote))
		if v != domain.ApprovalStatusApproved && app.Status != domain.ApprovalStatusApproved {
			allApproved = false
			break
		}
	}

	if allApproved {
		draft.Status = domain.DraftStatusApproved
		draft.UpdatedAt = now
		return repo.Update(ctx, draft)
	}

	return nil, nil
}

func resolvePartialRecoveryConsensus(
	ctx context.Context,
	repo ports.DraftRepository,
	draft *domain.Draft,
	approvals []*domain.DraftApproval,
	now time.Time,
) (*domain.Draft, error) {
	hasRevert := false
	allResume := len(approvals) > 0
	for _, app := range approvals {
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
		return repo.Update(ctx, draft)
	} else if allResume {
		draft.RecoveryAction = domain.VoteResume
		draft.UpdatedAt = now
		return repo.Update(ctx, draft)
	}

	return nil, nil
}

func (s *draftService) resolvePartiallyRevertedConsensus(
	ctx context.Context,
	repo ports.DraftRepository,
	draft *domain.Draft,
	approvals []*domain.DraftApproval,
	userID uint,
	now time.Time,
) (*domain.Draft, error) {
	hasNo := false
	allYes := len(approvals) > 0
	for _, app := range approvals {
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
		updatedDraft, err := repo.Update(ctx, draft)
		if err != nil {
			return nil, err
		}

		if s.migrationRepo != nil {
			mig := &domain.Migration{
				DatabaseID:   draft.DatabaseID,
				UserID:       userID,
				DraftID:      &draft.ID,
				Status:       domain.MigrationStatusFail,
				ErrorMessage: "[REVERT_ABORTED_BY_LEADER] Group leader voted to abort revert. Database left in intermediate state.",
				MigratedAt:   now,
			}
			_, _ = s.migrationRepo.Save(ctx, mig)
		}
		return updatedDraft, nil
	} else if allYes {
		draft.RecoveryAction = domain.VoteRevert
		draft.UpdatedAt = now
		return repo.Update(ctx, draft)
	}

	return nil, nil
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

	if inv, err := s.invalidateIfAuthorUnauthorized(ctx, s.draftRepo, draft); err != nil {
		return nil, err
	} else if inv {
		return draft, nil
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

	if inv, _ := s.invalidateIfAuthorUnauthorized(ctx, s.draftRepo, draft); inv {
		return domain.ErrDraftNotApproved
	}

	if draft.Status != domain.DraftStatusApproved {
		return domain.ErrDraftNotApproved
	}

	if err := s.checkQuarantineConflicts(ctx, draft.DatabaseID); err != nil {
		return err
	}

	lock, err := s.acquireDatabaseLock(draft.DatabaseID)
	if err != nil {
		return err
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

	return s.dispatchExecutionTask(ctx, draft, "execute", execID, userID, lock)
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

	lock, err := s.acquireDatabaseLock(draft.DatabaseID)
	if err != nil {
		return err
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

	return s.dispatchExecutionTask(ctx, draft, "resume", execID, userID, lock)
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

	lock, err := s.acquireDatabaseLock(draft.DatabaseID)
	if err != nil {
		return err
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

	return s.dispatchExecutionTask(ctx, draft, "revert", newExecID, userID, lock)
}

func (s *draftService) acquireDatabaseLock(databaseID uint) (*dbLock, error) {
	val, _ := s.dbLocks.LoadOrStore(databaseID, &dbLock{})
	lock := val.(*dbLock)
	if !lock.TryLock() {
		return nil, domain.ErrMigrationInProgress
	}
	return lock, nil
}

func (s *draftService) checkQuarantineConflicts(ctx context.Context, databaseID uint) error {
	if s.draftRepo == nil {
		return nil
	}
	for _, status := range []string{domain.DraftStatusPartiallyReverted, domain.DraftStatusPartial} {
		d, err := s.draftRepo.FindDraftByStatusAndDatabaseID(ctx, databaseID, status)
		if err == nil && d != nil && d.ID != 0 {
			return domain.ErrDatabaseMustResolvePartiallyReverted
		}
	}
	executing, err := s.draftRepo.FindDraftByStatusAndDatabaseID(ctx, databaseID, domain.DraftStatusExecuting)
	if err == nil && executing != nil && executing.ID != 0 {
		return domain.ErrMigrationInProgress
	}
	return nil
}

func (s *draftService) dispatchExecutionTask(
	ctx context.Context,
	draft *domain.Draft,
	operation string,
	execID uint,
	userID uint,
	lock *dbLock,
) error {
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
			Operation:    operation,
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
