package draft_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/draft"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

type draftStore struct {
	testutil.MigrationRepository
	pos           domain.MigrationPosition
	nodes         map[uint]*domain.Migration
	draft         *domain.Draft
	votes         map[uint]bool
	lastExecution uint
	lastAudit     *domain.Migration
}

func (r *draftStore) WithPosition(_ context.Context, _ uint, fn func(ports.MigrationRepository, domain.MigrationPosition) error) error {
	return fn(r, r.pos)
}

func (r *draftStore) SaveDraft(_ context.Context, d *domain.Draft) error {
	d.ID = 5
	r.draft = d
	r.votes = map[uint]bool{7: false, 8: false}
	return nil
}

func (r *draftStore) FindDraft(context.Context, uint, uint) (*domain.Draft, error) {
	return r.draft, nil
}

func (r *draftStore) UpdateDraftStatus(_ context.Context, _ uint, status string) error {
	r.draft.Status = status
	return nil
}

func (r *draftStore) HasConsensus(context.Context, uint, uint) (bool, error) {
	return r.votes[7] && r.votes[8], nil
}

func (r *draftStore) Vote(_ context.Context, _ uint, user uint, approve bool) error {
	if _, ok := r.votes[user]; !ok {
		return domain.ErrAccessDenied
	}
	r.votes[user] = approve
	return nil
}

func (r *draftStore) FindActiveExecution(_ context.Context, id uint) (uint, error) {
	if r.lastExecution != 0 {
		return r.lastExecution, nil
	}
	return id, nil
}

func (r *draftStore) UpdatePosition(_ context.Context, _ uint, head, parent, forward *uint, exec uint) error {
	r.pos = domain.MigrationPosition{HeadID: head, ExecutionID: exec}
	if parent != nil {
		r.nodes[*parent].ForwardMigrationID = forward
	}
	return nil
}

func (r *draftStore) FinishDraft(ctx context.Context, _ uint, m *domain.Migration, head, parent, forward *uint) error {
	m.ID = m.ExecutionID + 10
	r.lastAudit = m
	if m.RevertsMigrationID == nil {
		r.lastExecution = m.ID
	}
	r.draft.Status = "applied"
	return r.UpdatePosition(ctx, 1, head, parent, forward, m.ExecutionID)
}

type journey struct {
	repo     *draftStore
	svc      ports.DraftService
	commands []string
	worker   *draft.ExecutionWorker
	task     *ports.ExecutionTask
	ctx      context.Context
}

func newJourney() *journey {
	a, b, c := uint(1), uint(2), uint(3)
	r := &draftStore{pos: domain.MigrationPosition{HeadID: &c, ExecutionID: 3}, nodes: map[uint]*domain.Migration{
		a: {ID: a, Version: "1", ForwardMigrationID: &b},
		b: {ID: b, Version: "2", ParentMigrationID: &a, ForwardMigrationID: &c, SqlUp: "up B", SqlDown: "down B", IsReversible: true},
		c: {ID: c, Version: "3", ParentMigrationID: &b, SqlUp: "up C", SqlDown: "down C", IsReversible: true},
	}}
	r.FindByIDAndDatabaseIDFunc = func(_ context.Context, id, _ uint) (*domain.Migration, error) { return r.nodes[id], nil }
	r.FindLatestByDatabaseIDFunc = func(context.Context, uint) (*domain.Migration, error) { return r.nodes[c], nil }
	r.SaveFunc = func(_ context.Context, m *domain.Migration) (*domain.Migration, error) {
		m.ID = 9
		r.nodes[9] = m
		return m, nil
	}
	j := &journey{repo: r, ctx: context.Background()}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(_ context.Context, _ domain.Database, sql string) (*ports.DatabaseCommandResult, error) {
			j.commands = append(j.commands, sql)
			return nil, nil
		},
		GetDatabaseMetadataFunc: func(context.Context, domain.Database) (*ports.DatabaseMetadata, error) {
			return &ports.DatabaseMetadata{}, nil
		},
		GenerateSQLDownFunc: func(context.Context, domain.Database, string) (string, error) { return "DROP TABLE d", nil },
	}
	auth := &testutil.UserGroupService{AuthorizeFunc: func(_ context.Context, user, _ uint, access domain.AccessType) error {
		if user == 6 && access == domain.AccessViewEdit {
			return domain.ErrAccessDenied
		}
		return nil
	}}
	draftRepo := &testutil.DraftRepository{
		FindByIDFunc: func(context.Context, uint) (*domain.Draft, error) { return r.draft, nil },
	}
	dbRepo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 1, CurrentMigrationID: r.pos.HeadID, LatestExecutionID: r.pos.ExecutionID}, nil
		},
		FindWithCredentialByIDFunc: func(context.Context, uint) (domain.Database, error) {
			return domain.Database{ID: 1, CurrentMigrationID: r.pos.HeadID, LatestExecutionID: r.pos.ExecutionID}, nil
		},
		IncrementExecutionIDFunc: func(context.Context, uint) (uint, error) {
			r.pos.ExecutionID++
			return r.pos.ExecutionID, nil
		},
	}
	queue := &testutil.TaskQueue{
		EnqueueFunc: func(_ context.Context, task ports.ExecutionTask) error {
			j.task = &task
			return nil
		},
	}
	j.svc = draft.NewDraftService(draftRepo, nil, auth, dbRepo, driver, r, queue)
	j.worker = draft.NewExecutionWorker(draftRepo, dbRepo, r, driver, nil, j.svc.UnlockDatabase, nil, auth)
	return j
}

func (j *journey) apply(t *testing.T, op string) *domain.Migration {
	t.Helper()
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, op)
	if err != nil {
		t.Fatal(err)
	}
	for _, leader := range []uint{7, 8} {
		if err := j.svc.VoteDraft(j.ctx, 1, d.ID, leader, true); err != nil {
			t.Fatal(err)
		}
	}
	commandsBeforeEnqueue := len(j.commands)
	if _, err := j.svc.ExecuteMigrationDraft(j.ctx, 1, d.ID, 1); err != nil {
		t.Fatal(err)
	}
	if len(j.commands) != commandsBeforeEnqueue || j.task == nil {
		t.Fatal("expected SQL to wait for the worker")
	}
	if err := j.worker.Execute(j.ctx, *j.task); err != nil {
		t.Fatal(err)
	}
	j.task = nil
	return j.repo.lastAudit
}

func TestDraftService_ProposeRollback_ViewOnlyUser_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	j := newJourney()
	// Act
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 6, domain.DraftOperationRollback)
	// Assert
	if d != nil || !errors.Is(err, domain.ErrAccessDenied) || j.repo.draft != nil {
		t.Fatalf("unexpected draft/error: %v %v", d, err)
	}
}

func TestDraftService_ProposeForward_ViewOnlyUser_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	j := newJourney()
	// Act
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 6, domain.DraftOperationForward)
	// Assert
	if d != nil || !errors.Is(err, domain.ErrAccessDenied) || j.repo.draft != nil {
		t.Fatalf("unexpected draft/error: %v %v", d, err)
	}
}

func TestDraftService_ProposeRollback_NoActiveMigration_ReturnsErrNoActiveMigration(t *testing.T) {
	// Arrange
	j := newJourney()
	j.repo.pos.HeadID = nil
	// Act
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, domain.DraftOperationRollback)
	// Assert
	if d != nil || !errors.Is(err, domain.ErrNoActiveMigration) || j.repo.draft != nil {
		t.Fatalf("unexpected draft/error: %v %v", d, err)
	}
}

func TestDraftService_ProposeRollback_Irreversible_ReturnsErrIrreversible(t *testing.T) {
	// Arrange
	j := newJourney()
	j.repo.nodes[3].IsReversible = false
	j.repo.nodes[3].IrreversibleReason = "DROP TABLE"
	// Act
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, domain.DraftOperationRollback)
	// Assert
	if d != nil || !errors.Is(err, domain.ErrMigrationIrreversible) || j.repo.draft != nil {
		t.Fatalf("unexpected draft/error: %v %v", d, err)
	}
}

func TestDraftService_ProposeForward_NotFromRollbackPosition_ReturnsErrNoForwardAvailable(t *testing.T) {
	// Arrange
	j := newJourney()
	// Act
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, domain.DraftOperationForward)
	// Assert
	if d != nil || !errors.Is(err, domain.ErrNoForwardAvailable) || j.repo.draft != nil {
		t.Fatalf("unexpected draft/error: %v %v", d, err)
	}
}

func TestDraftService_ProposeRollback_Reversible_CreatesDraftAndApprovals(t *testing.T) {
	// Arrange
	j := newJourney()
	// Act
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, domain.DraftOperationRollback)
	// Assert
	if err != nil || d.Status != "pending_approval" || d.SqlUp != "down C" || d.TargetID != 3 || len(j.repo.votes) != 2 {
		t.Fatalf("proposal: %+v %v", d, err)
	}
}

func TestDraftService_ProposeForward_ValidFromRollback_CreatesDraftAndApprovals(t *testing.T) {
	// Arrange
	j := newJourney()
	j.apply(t, domain.DraftOperationRollback)
	// Act
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, domain.DraftOperationForward)
	// Assert
	if err != nil || d.Status != "pending_approval" || d.SqlUp != "up C" || d.TargetID != 3 || len(j.repo.votes) != 2 {
		t.Fatalf("proposal: %+v %v", d, err)
	}
}

func TestDraftService_ExecuteRollback_RewindsHeadOneStepAndSetsRevertsID(t *testing.T) {
	// Arrange
	j := newJourney()
	// Act
	audit := j.apply(t, domain.DraftOperationRollback)
	// Assert
	if *j.repo.pos.HeadID != 2 || *audit.RevertsMigrationID != 3 || len(audit.Snapshot) == 0 || len(j.commands) != 1 || j.commands[0] != "down C" {
		t.Fatalf("rollback: %+v %v", audit, j.commands)
	}
}

func TestDraftService_ExecuteRollback_ConsecutiveRollbacks_RewindsHeadStepByStep(t *testing.T) {
	// Arrange
	j := newJourney()
	j.apply(t, domain.DraftOperationRollback)
	// Act
	audit := j.apply(t, domain.DraftOperationRollback)
	// Assert
	if *j.repo.pos.HeadID != 1 || *audit.RevertsMigrationID != 2 || j.commands[1] != "down B" {
		t.Fatalf("second rollback: %+v", audit)
	}
}

func TestDraftService_ExecuteForward_AdvancesHeadOneStep(t *testing.T) {
	// Arrange
	j := newJourney()
	j.apply(t, domain.DraftOperationRollback)
	j.apply(t, domain.DraftOperationRollback)
	// Act
	audit := j.apply(t, domain.DraftOperationForward)
	// Assert
	if *j.repo.pos.HeadID != 2 || audit.RevertsMigrationID != nil || audit.Version != "2" || j.commands[2] != "up B" {
		t.Fatalf("forward: %+v", audit)
	}
}

func TestDraftService_ExecuteDraft_PartialApproval_ReturnsConflict(t *testing.T) {
	// Arrange
	j := newJourney()
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, domain.DraftOperationRollback)
	if err != nil {
		t.Fatal(err)
	}
	if err = j.svc.VoteDraft(j.ctx, 1, d.ID, 7, true); err != nil {
		t.Fatal(err)
	}
	// Act
	_, err = j.svc.ExecuteMigrationDraft(j.ctx, 1, d.ID, 1)
	// Assert
	if !errors.Is(err, domain.ErrDraftConflict) || len(j.commands) != 0 {
		t.Fatalf("partial vote: %v", err)
	}
}

func TestDraftService_ExecuteRollback_AfterForward_RevertsLatestExecution(t *testing.T) {
	// Arrange
	j := newJourney()
	j.apply(t, domain.DraftOperationRollback)
	forward := j.apply(t, domain.DraftOperationForward)
	// Act
	audit := j.apply(t, domain.DraftOperationRollback)
	// Assert
	if *audit.RevertsMigrationID != forward.ID {
		t.Fatalf("expected execution %d, got %+v", forward.ID, audit)
	}
}

func TestDraftService_VoteDraft_LeaderRejects_TransitionsToRejected(t *testing.T) {
	// Arrange
	j := newJourney()
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, domain.DraftOperationRollback)
	if err != nil {
		t.Fatalf("unexpected error proposing draft: %v", err)
	}

	// Act: Leader 7 casts a rejection vote
	err = j.svc.VoteDraft(j.ctx, 1, d.ID, 7, false)

	// Assert: Draft status flips to rejected
	if err != nil {
		t.Fatalf("unexpected error voting reject: %v", err)
	}
	if j.repo.draft.Status != "rejected" {
		t.Fatalf("expected draft status 'rejected', got %q", j.repo.draft.Status)
	}

	// Assert: Subsequent execution attempt is rejected with ErrDraftConflict
	_, execErr := j.svc.ExecuteMigrationDraft(j.ctx, 1, d.ID, 1)
	if !errors.Is(execErr, domain.ErrDraftConflict) {
		t.Fatalf("expected ErrDraftConflict when executing rejected draft, got %v", execErr)
	}
}

func TestDraftService_VoteDraft_BaseExecutionConflict_ReturnsErrDraftConflict(t *testing.T) {
	// Arrange
	j := newJourney()
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, domain.DraftOperationRollback)
	if err != nil {
		t.Fatalf("unexpected error proposing draft: %v", err)
	}

	// Another migration ran in background, advancing ExecutionID from 3 to 4
	j.repo.pos.ExecutionID = 4

	// Act
	err = j.svc.VoteDraft(j.ctx, 1, d.ID, 7, true)

	// Assert
	if !errors.Is(err, domain.ErrDraftConflict) {
		t.Fatalf("expected ErrDraftConflict, got %v", err)
	}
}

func TestDraftService_ExecuteMigrationDraft_BaseExecutionConflict_ReturnsErrDraftConflict(t *testing.T) {
	// Arrange
	j := newJourney()
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, domain.DraftOperationRollback)
	if err != nil {
		t.Fatalf("unexpected error proposing draft: %v", err)
	}
	for _, leader := range []uint{7, 8} {
		if err := j.svc.VoteDraft(j.ctx, 1, d.ID, leader, true); err != nil {
			t.Fatalf("unexpected error voting approve: %v", err)
		}
	}

	// Another migration completed right before execute call
	j.repo.pos.ExecutionID = 99

	// Act
	_, execErr := j.svc.ExecuteMigrationDraft(j.ctx, 1, d.ID, 1)

	// Assert
	if !errors.Is(execErr, domain.ErrDraftConflict) {
		t.Fatalf("expected ErrDraftConflict, got %v", execErr)
	}
}

func TestDraftService_ExecuteMigrationDraft_AuthorLostEditAccess_ReturnsErrAccessDenied(t *testing.T) {
	// Arrange: Propose by author user 1, approved by leaders, but then author 1 loses edit access
	j := newJourney()
	d, err := j.svc.CreateAutoDraft(j.ctx, 1, 1, domain.DraftOperationRollback)
	if err != nil {
		t.Fatalf("unexpected error proposing draft: %v", err)
	}
	for _, leader := range []uint{7, 8} {
		if err := j.svc.VoteDraft(j.ctx, 1, d.ID, leader, true); err != nil {
			t.Fatalf("unexpected error voting approve: %v", err)
		}
	}

	// Author (d.UserID = 1) has edit permission revoked right before execution
	authorEditRevoked := false
	auth := &testutil.UserGroupService{
		AuthorizeFunc: func(_ context.Context, user, _ uint, access domain.AccessType) error {
			if authorEditRevoked && user == 1 && access == domain.AccessViewEdit {
				return domain.ErrAccessDenied
			}
			return nil
		},
	}
	driver := &testutil.DatabaseDriver{
		ExecuteCommandFunc: func(_ context.Context, _ domain.Database, _ string) (*ports.DatabaseCommandResult, error) {
			return nil, nil
		},
	}
	j.svc = draft.NewDraftService(nil, nil, auth, &testutil.DatabaseRepository{}, driver, j.repo)
	authorEditRevoked = true

	// Act: Caller (admin user 2) tries to execute the draft whose author has lost edit access
	_, execErr := j.svc.ExecuteMigrationDraft(j.ctx, 1, d.ID, 2)

	// Assert
	if !errors.Is(execErr, domain.ErrAccessDenied) {
		t.Fatalf("expected ErrAccessDenied for unauthorized draft author, got %v", execErr)
	}
}
