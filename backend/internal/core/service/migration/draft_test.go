package migration_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type draftStore struct {
	testutil.MigrationRepository
	pos           domain.MigrationPosition
	nodes         map[uint]*domain.Migration
	draft         *domain.MigrationDraft
	votes         map[uint]bool
	lastExecution uint
}

func (r *draftStore) WithPosition(_ context.Context, _ uint, fn func(ports.MigrationRepository, domain.MigrationPosition) error) error {
	return fn(r, r.pos)
}

func (r *draftStore) SaveDraft(_ context.Context, d *domain.MigrationDraft) error {
	d.ID = 5
	r.draft = d
	r.votes = map[uint]bool{7: false, 8: false}
	return nil
}

func (r *draftStore) FindDraft(context.Context, uint, uint) (*domain.MigrationDraft, error) {
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
	if m.RevertsMigrationID == nil {
		r.lastExecution = m.ID
	}
	r.draft.Status = "applied"
	return r.UpdatePosition(ctx, 1, head, parent, forward, m.ExecutionID)
}

type journey struct {
	repo     *draftStore
	svc      ports.MigrationService
	commands []string
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
	j.svc = migration.NewMigrationService(r, &testutil.DatabaseRepository{}, driver, auth)
	return j
}

func (j *journey) apply(t *testing.T, op string) *domain.Migration {
	t.Helper()
	d, err := j.svc.ProposeMigrationDraft(j.ctx, 1, 1, op)
	if err != nil {
		t.Fatal(err)
	}
	for _, leader := range []uint{7, 8} {
		if err := j.svc.VoteMigrationDraft(j.ctx, 1, d.ID, leader, true); err != nil {
			t.Fatal(err)
		}
	}
	m, err := j.svc.ExecuteMigrationDraft(j.ctx, 1, d.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMigrationService_ProposeRollback_ViewOnlyUser_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	j := newJourney()
	// Act
	d, err := j.svc.ProposeMigrationDraft(j.ctx, 1, 6, domain.OperationRollback)
	// Assert
	if d != nil || !errors.Is(err, domain.ErrAccessDenied) || j.repo.draft != nil {
		t.Fatalf("unexpected draft/error: %v %v", d, err)
	}
}

func TestMigrationService_ProposeForward_ViewOnlyUser_ReturnsErrForbidden(t *testing.T) {
	// Arrange
	j := newJourney()
	// Act
	d, err := j.svc.ProposeMigrationDraft(j.ctx, 1, 6, domain.OperationForward)
	// Assert
	if d != nil || !errors.Is(err, domain.ErrAccessDenied) || j.repo.draft != nil {
		t.Fatalf("unexpected draft/error: %v %v", d, err)
	}
}

func TestMigrationService_ProposeRollback_NoActiveMigration_ReturnsErrNoActiveMigration(t *testing.T) {
	// Arrange
	j := newJourney()
	j.repo.pos.HeadID = nil
	// Act
	d, err := j.svc.ProposeMigrationDraft(j.ctx, 1, 1, domain.OperationRollback)
	// Assert
	if d != nil || !errors.Is(err, domain.ErrNoActiveMigration) || j.repo.draft != nil {
		t.Fatalf("unexpected draft/error: %v %v", d, err)
	}
}

func TestMigrationService_ProposeRollback_Irreversible_ReturnsErrIrreversible(t *testing.T) {
	// Arrange
	j := newJourney()
	j.repo.nodes[3].IsReversible = false
	j.repo.nodes[3].IrreversibleReason = "DROP TABLE"
	// Act
	d, err := j.svc.ProposeMigrationDraft(j.ctx, 1, 1, domain.OperationRollback)
	// Assert
	if d != nil || !errors.Is(err, domain.ErrMigrationIrreversible) || j.repo.draft != nil {
		t.Fatalf("unexpected draft/error: %v %v", d, err)
	}
}

func TestMigrationService_ProposeForward_NotFromRollbackPosition_ReturnsErrNoForwardAvailable(t *testing.T) {
	// Arrange
	j := newJourney()
	// Act
	d, err := j.svc.ProposeMigrationDraft(j.ctx, 1, 1, domain.OperationForward)
	// Assert
	if d != nil || !errors.Is(err, domain.ErrNoForwardAvailable) || j.repo.draft != nil {
		t.Fatalf("unexpected draft/error: %v %v", d, err)
	}
}

func TestMigrationService_ProposeRollback_Reversible_CreatesDraftAndApprovals(t *testing.T) {
	// Arrange
	j := newJourney()
	// Act
	d, err := j.svc.ProposeMigrationDraft(j.ctx, 1, 1, domain.OperationRollback)
	// Assert
	if err != nil || d.Status != "pending_approval" || d.SqlUp != "down C" || d.TargetID != 3 || len(j.repo.votes) != 2 {
		t.Fatalf("proposal: %+v %v", d, err)
	}
}

func TestMigrationService_ProposeForward_ValidFromRollback_CreatesDraftAndApprovals(t *testing.T) {
	// Arrange
	j := newJourney()
	j.apply(t, domain.OperationRollback)
	// Act
	d, err := j.svc.ProposeMigrationDraft(j.ctx, 1, 1, domain.OperationForward)
	// Assert
	if err != nil || d.Status != "pending_approval" || d.SqlUp != "up C" || d.TargetID != 3 || len(j.repo.votes) != 2 {
		t.Fatalf("proposal: %+v %v", d, err)
	}
}

func TestMigrationService_ExecuteRollback_RewindsHeadOneStepAndSetsRevertsID(t *testing.T) {
	// Arrange
	j := newJourney()
	// Act
	audit := j.apply(t, domain.OperationRollback)
	// Assert
	if *j.repo.pos.HeadID != 2 || *audit.RevertsMigrationID != 3 || len(audit.Snapshot) == 0 || len(j.commands) != 1 || j.commands[0] != "down C" {
		t.Fatalf("rollback: %+v %v", audit, j.commands)
	}
}

func TestMigrationService_ExecuteRollback_ConsecutiveRollbacks_RewindsHeadStepByStep(t *testing.T) {
	// Arrange
	j := newJourney()
	j.apply(t, domain.OperationRollback)
	// Act
	audit := j.apply(t, domain.OperationRollback)
	// Assert
	if *j.repo.pos.HeadID != 1 || *audit.RevertsMigrationID != 2 || j.commands[1] != "down B" {
		t.Fatalf("second rollback: %+v", audit)
	}
}

func TestMigrationService_ExecuteForward_AdvancesHeadOneStep(t *testing.T) {
	// Arrange
	j := newJourney()
	j.apply(t, domain.OperationRollback)
	j.apply(t, domain.OperationRollback)
	// Act
	audit := j.apply(t, domain.OperationForward)
	// Assert
	if *j.repo.pos.HeadID != 2 || audit.RevertsMigrationID != nil || audit.Version != "2" || j.commands[2] != "up B" {
		t.Fatalf("forward: %+v", audit)
	}
}

func TestMigrationService_ApplyNewMigrationAfterRollback_OverwritesForwardAndPreservesDeadNode(t *testing.T) {
	// Arrange
	j := newJourney()
	j.apply(t, domain.OperationRollback)
	// Act
	m, err := j.svc.CreateMigration(j.ctx, domain.CreateMigrationInput{DatabaseID: 1, UserID: 1, SqlUp: "CREATE TABLE d (id INT)"})
	// Assert
	if err != nil || *j.repo.nodes[2].ForwardMigrationID != m.ID || *m.ParentMigrationID != 2 || j.repo.nodes[3].SqlUp != "up C" {
		t.Fatalf("branch: %+v %v", m, err)
	}
}

func TestMigrationService_ExecuteDraft_PartialApproval_ReturnsConflict(t *testing.T) {
	// Arrange
	j := newJourney()
	d, err := j.svc.ProposeMigrationDraft(j.ctx, 1, 1, domain.OperationRollback)
	if err != nil {
		t.Fatal(err)
	}
	if err = j.svc.VoteMigrationDraft(j.ctx, 1, d.ID, 7, true); err != nil {
		t.Fatal(err)
	}
	// Act
	_, err = j.svc.ExecuteMigrationDraft(j.ctx, 1, d.ID, 1)
	// Assert
	if !errors.Is(err, domain.ErrDraftConflict) || len(j.commands) != 0 {
		t.Fatalf("partial vote: %v", err)
	}
}

func TestMigrationService_ExecuteRollback_AfterForward_RevertsLatestExecution(t *testing.T) {
	// Arrange
	j := newJourney()
	j.apply(t, domain.OperationRollback)
	forward := j.apply(t, domain.OperationForward)
	// Act
	audit := j.apply(t, domain.OperationRollback)
	// Assert
	if *audit.RevertsMigrationID != forward.ID {
		t.Fatalf("expected execution %d, got %+v", forward.ID, audit)
	}
}

func TestMigrationService_Classify_Irreversible_ReturnsFalseWithReason(t *testing.T) {
	// Arrange
	queries := []string{"DROP TABLE t", "alter table t drop column a", "drop database test", "TRUNCATE t", "CREATE TABLE t(id int); DELETE FROM t"}
	// Act
	for _, sql := range queries {
		reversible, reason := domain.ClassifyMigration(sql)
		// Assert
		if reversible != false || (strings.TrimSpace(reason) == "") != reversible {
			t.Errorf("%s: %t %q", sql, reversible, reason)
		}
	}
}

func TestMigrationService_Classify_Reversible_ReturnsTrue(t *testing.T) {
	// Arrange
	queries := []string{"CREATE TABLE t(id int)", "ALTER TABLE t ADD COLUMN foo int", "CREATE INDEX i ON t(id)", "ALTER TABLE t RENAME TO b", "CREATE TABLE t(s text DEFAULT 'DROP TABLE')"}
	// Act
	for _, sql := range queries {
		reversible, reason := domain.ClassifyMigration(sql)
		// Assert
		if reversible != true || (strings.TrimSpace(reason) == "") != reversible {
			t.Errorf("%s: %t %q", sql, reversible, reason)
		}
	}
}

func TestMigrationService_GetMigrations_HideRollbackTrue_FiltersRevertedPairs(t *testing.T) {
	// Arrange
	j := newJourney()
	var received bool
	j.repo.FindAllByDatabaseIDFunc = func(_ context.Context, _ uint, _ pagination.PaginationParams, hide bool) ([]*domain.Migration, int64, error) {
		received = true
		if hide != true {
			t.Fatalf("filter: %v", hide)
		}
		return nil, 0, nil
	}
	// Act
	_, _, err := j.svc.GetMigrationsByDatabaseID(j.ctx, 1, 1, pagination.PaginationParams{}, true)
	// Assert
	if err != nil || !received {
		t.Fatalf("history: %v", err)
	}
}

func TestMigrationService_GetMigrations_HideRollbackFalse_ReturnsAll(t *testing.T) {
	// Arrange
	j := newJourney()
	var received bool
	j.repo.FindAllByDatabaseIDFunc = func(_ context.Context, _ uint, _ pagination.PaginationParams, hide bool) ([]*domain.Migration, int64, error) {
		received = true
		if hide != false {
			t.Fatalf("filter: %v", hide)
		}
		return nil, 0, nil
	}
	// Act
	_, _, err := j.svc.GetMigrationsByDatabaseID(j.ctx, 1, 1, pagination.PaginationParams{}, false)
	// Assert
	if err != nil || !received {
		t.Fatalf("history: %v", err)
	}
}
