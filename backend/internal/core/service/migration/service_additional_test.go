package migration_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	migrationservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

func migrationDatabaseRepository() *testutil.DatabaseRepository {
	return &testutil.DatabaseRepository{
		FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
			return &domain.Database{ID: 8}, nil
		},
		FindWithCredentialByIDFunc: func(context.Context, uint) (domain.Database, error) {
			return domain.Database{ID: 8, Type: "postgresql", Password: "secret"}, nil
		},
	}
}

func editGroupService() *testutil.GroupService {
	return &testutil.GroupService{
		EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
			return domain.AccessViewEdit, nil
		},
	}
}

func TestGetMigrationsByDatabaseIDDelegates(t *testing.T) {
	params := pagination.NewPaginationParams(2, 5)
	repository := &migrationRepositoryFake{list: []*domain.Migration{{ID: 1}}, total: 7}
	service := migrationservice.NewMigrationService(repository, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService())

	migrations, total, err := service.GetMigrationsByDatabaseID(context.Background(), 8, params)
	if err != nil || total != 7 || len(migrations) != 1 || migrations[0].ID != 1 {
		t.Fatalf("migrations=%v total=%d err=%v", migrations, total, err)
	}
}

func TestCreateMigrationRejectsInvalidInput(t *testing.T) {
	service := migrationservice.NewMigrationService(&migrationRepositoryFake{}, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService())
	tests := []struct {
		name       string
		databaseID uint
		userID     uint
		sql        string
	}{
		{name: "database zero", userID: 1, sql: "CREATE TABLE x(id int)"},
		{name: "user zero", databaseID: 1, sql: "CREATE TABLE x(id int)"},
		{name: "sql empty", databaseID: 1, userID: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateMigration(context.Background(), tt.databaseID, tt.userID, tt.sql)
			if !errors.Is(err, domain.ErrInvalidMigrationInput) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestCreateMigrationDependencyFailures(t *testing.T) {
	wantAccessErr := errors.New("access lookup")
	wantDatabaseErr := errors.New("membership lookup")
	wantCredentialErr := errors.New("credential lookup")
	wantExecuteErr := errors.New("execute failed")
	wantLatestErr := errors.New("latest failed")
	wantSaveErr := errors.New("save failed")

	tests := []struct {
		name       string
		groups     *testutil.GroupService
		databases  *testutil.DatabaseRepository
		executor   *migrationExecutorFake
		repository *migrationRepositoryFake
		sql        string
		want       error
		wantFailed bool
	}{
		{
			name: "access lookup",
			groups: &testutil.GroupService{EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return "", wantAccessErr
			}},
			databases: migrationDatabaseRepository(), executor: &migrationExecutorFake{}, repository: &migrationRepositoryFake{},
			sql: "CREATE TABLE x(id int)", want: wantAccessErr,
		},
		{
			name: "database membership lookup", groups: editGroupService(),
			databases: &testutil.DatabaseRepository{FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) { return nil, wantDatabaseErr }},
			executor:  &migrationExecutorFake{}, repository: &migrationRepositoryFake{}, sql: "CREATE TABLE x(id int)", want: wantDatabaseErr,
		},
		{
			name: "credential lookup", groups: editGroupService(),
			databases: &testutil.DatabaseRepository{
				FindByIDAndUserIDFunc:      func(context.Context, uint, uint) (*domain.Database, error) { return &domain.Database{}, nil },
				FindWithCredentialByIDFunc: func(context.Context, uint) (domain.Database, error) { return domain.Database{}, wantCredentialErr },
			},
			executor: &migrationExecutorFake{}, repository: &migrationRepositoryFake{}, sql: "CREATE TABLE x(id int)", want: wantCredentialErr,
		},
		{
			name: "unsupported inverse SQL", groups: editGroupService(), databases: migrationDatabaseRepository(),
			executor: &migrationExecutorFake{}, repository: &migrationRepositoryFake{}, sql: "TRUNCATE TABLE x", want: domain.ErrSQLDownGenerationUnsupported, wantFailed: true,
		},
		{
			name: "execute command", groups: editGroupService(), databases: migrationDatabaseRepository(),
			executor: &migrationExecutorFake{commandErr: wantExecuteErr}, repository: &migrationRepositoryFake{}, sql: "CREATE TABLE x(id int)", want: wantExecuteErr, wantFailed: true,
		},
		{
			name: "latest lookup", groups: editGroupService(), databases: migrationDatabaseRepository(),
			executor: &migrationExecutorFake{}, repository: &migrationRepositoryFake{findLatestErr: wantLatestErr}, sql: "CREATE TABLE x(id int)", want: wantLatestErr,
		},
		{
			name: "save", groups: editGroupService(), databases: migrationDatabaseRepository(),
			executor: &migrationExecutorFake{}, repository: &migrationRepositoryFake{findLatestErr: domain.ErrMigrationNotFound, saveErr: wantSaveErr}, sql: "CREATE TABLE x(id int)", want: wantSaveErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := migrationservice.NewMigrationService(tt.repository, tt.databases, tt.executor, tt.groups)
			created, err := service.CreateMigration(context.Background(), 8, 3, tt.sql)
			if tt.wantFailed {
				if err != nil || created == nil || created.Status != "fail" || !strings.Contains(created.ErrorMessage, tt.want.Error()) {
					t.Fatalf("created=%+v error=%v want persisted failed migration containing %q", created, err, tt.want.Error())
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("error=%v want=%v", err, tt.want)
			}
		})
	}
}

func TestCreateMigrationStartsAtVersionOne(t *testing.T) {
	repository := &migrationRepositoryFake{findLatestErr: domain.ErrMigrationNotFound}
	service := migrationservice.NewMigrationService(repository, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService())
	migration, err := service.CreateMigration(context.Background(), 8, 3, "CREATE TABLE first_table(id int)")
	if err != nil {
		t.Fatal(err)
	}
	if migration.Version != "1" || migration.SqlDown != "DROP TABLE first_table" {
		t.Fatalf("migration=%+v", migration)
	}
}

func TestCreateMigrationRejectsInvalidLatestVersion(t *testing.T) {
	repository := &migrationRepositoryFake{latest: &domain.Migration{Version: "not-a-number"}}
	service := migrationservice.NewMigrationService(repository, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService())
	_, err := service.CreateMigration(context.Background(), 8, 3, "CREATE TABLE x(id int)")
	if !errors.Is(err, domain.ErrMigrationVersionUnavailable) {
		t.Fatalf("error=%v", err)
	}
}

func TestGetMigrationByIDAndDatabaseIDValidationAndLookup(t *testing.T) {
	migration := &domain.Migration{ID: 4, DatabaseID: 8}
	repository := &migrationRepositoryFake{byID: map[uint]*domain.Migration{4: migration}}
	service := migrationservice.NewMigrationService(repository, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService())

	if _, _, err := service.GetMigrationByIDAndDatabaseID(context.Background(), 0, 8); !errors.Is(err, domain.ErrInvalidMigrationInput) {
		t.Fatalf("zero migration error=%v", err)
	}
	if _, _, err := service.GetMigrationByIDAndDatabaseID(context.Background(), 4, 0); !errors.Is(err, domain.ErrInvalidMigrationInput) {
		t.Fatalf("zero database error=%v", err)
	}
	got, total, err := service.GetMigrationByIDAndDatabaseID(context.Background(), 4, 8)
	if err != nil || got != migration || total != 1 {
		t.Fatalf("got=%v total=%d err=%v", got, total, err)
	}
}

func TestCompareMigrations(t *testing.T) {
	first := &domain.Migration{ID: 1, SqlUp: "up", SqlDown: "down"}
	second := &domain.Migration{ID: 2, SqlUp: "up", SqlDown: "different"}
	repository := &migrationRepositoryFake{byID: map[uint]*domain.Migration{1: first, 2: second}}
	service := migrationservice.NewMigrationService(repository, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService())

	comparison, err := service.CompareMigrations(context.Background(), 8, 3, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !comparison.SQLUpEqual || comparison.SQLDownEqual || comparison.FirstMigration != first || comparison.SecondMigration != second {
		t.Fatalf("comparison=%+v", comparison)
	}
}

func TestCompareMigrationsRejectsInvalidInputs(t *testing.T) {
	service := migrationservice.NewMigrationService(&migrationRepositoryFake{}, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService())
	cases := [][4]uint{{0, 1, 1, 2}, {1, 0, 1, 2}, {1, 1, 0, 2}, {1, 1, 1, 0}, {1, 1, 2, 2}}
	for _, values := range cases {
		_, err := service.CompareMigrations(context.Background(), values[0], values[1], values[2], values[3])
		if !errors.Is(err, domain.ErrInvalidMigrationComparison) {
			t.Fatalf("values=%v error=%v", values, err)
		}
	}
}

func TestCompareMigrationsPropagatesLookupErrors(t *testing.T) {
	wantErr := errors.New("lookup failed")
	service := migrationservice.NewMigrationService(
		&migrationRepositoryFake{},
		&testutil.DatabaseRepository{FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) { return nil, wantErr }},
		&migrationExecutorFake{}, editGroupService(),
	)
	_, err := service.CompareMigrations(context.Background(), 8, 3, 1, 2)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v", err)
	}

	service = migrationservice.NewMigrationService(
		&migrationRepositoryFake{findByIDErr: wantErr}, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService(),
	)
	_, err = service.CompareMigrations(context.Background(), 8, 3, 1, 2)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v", err)
	}
}

func TestRollbackValidationAndAccess(t *testing.T) {
	service := migrationservice.NewMigrationService(&migrationRepositoryFake{}, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService())
	cases := [][3]uint{{0, 1, 1}, {1, 0, 1}, {1, 1, 0}}
	for _, values := range cases {
		_, err := service.RollbackMigrations(context.Background(), values[0], values[1], values[2])
		if !errors.Is(err, domain.ErrInvalidMigrationInput) {
			t.Fatalf("values=%v error=%v", values, err)
		}
	}

	service = migrationservice.NewMigrationService(&migrationRepositoryFake{}, migrationDatabaseRepository(), &migrationExecutorFake{}, &testutil.GroupService{
		EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) { return domain.AccessViewOnly, nil },
	})
	_, err := service.RollbackMigrations(context.Background(), 8, 3, 1)
	if !errors.Is(err, domain.ErrDatabaseAccessDenied) {
		t.Fatalf("error=%v", err)
	}
}

func TestRollbackRejectsUnavailableTargets(t *testing.T) {
	databaseRepo := migrationDatabaseRepository()
	executor := &migrationExecutorFake{}

	tests := []struct {
		name       string
		repository *migrationRepositoryFake
		want       error
	}{
		{
			name:       "invalid target version",
			repository: &migrationRepositoryFake{latest: &domain.Migration{Version: "2", Status: "success"}, byID: map[uint]*domain.Migration{1: {ID: 1, Version: "bad", Status: "success"}}},
			want:       domain.ErrMigrationVersionUnavailable,
		},
		{
			name:       "target newer than latest",
			repository: &migrationRepositoryFake{latest: &domain.Migration{Version: "2", Status: "success"}, byID: map[uint]*domain.Migration{1: {ID: 1, Version: "3", Status: "success"}}},
			want:       domain.ErrMigrationRollbackUnavailable,
		},
		{
			name:       "latest invalid version",
			repository: &migrationRepositoryFake{latest: &domain.Migration{Version: "bad", Status: "success"}, byID: map[uint]*domain.Migration{1: {ID: 1, Version: "1", Status: "success"}}},
			want:       domain.ErrMigrationRollbackUnavailable,
		},
		{
			name: "missing sql down",
			repository: &migrationRepositoryFake{
				latest:    &domain.Migration{Version: "2", Status: "success"},
				byID:      map[uint]*domain.Migration{1: {ID: 1, Version: "1", Status: "success"}},
				byVersion: map[string]*domain.Migration{"2": {Version: "2", Status: "success", SqlDown: ""}},
			},
			want: domain.ErrMigrationRollbackUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := migrationservice.NewMigrationService(tt.repository, databaseRepo, executor, editGroupService())
			_, err := service.RollbackMigrations(context.Background(), 8, 3, 1)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error=%v want=%v", err, tt.want)
			}
		})
	}
}

func TestRollbackNoopWhenTargetIsLatest(t *testing.T) {
	repository := &migrationRepositoryFake{
		latest: &domain.Migration{Version: "2", Status: "success"},
		byID:   map[uint]*domain.Migration{2: {ID: 2, Version: "2", Status: "success"}},
	}
	service := migrationservice.NewMigrationService(repository, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService())
	rolledBack, err := service.RollbackMigrations(context.Background(), 8, 3, 2)
	if err != nil || len(rolledBack) != 0 || repository.saved != nil {
		t.Fatalf("rolledBack=%v saved=%v err=%v", rolledBack, repository.saved, err)
	}
}

func TestRollbackPropagatesCommandAndSaveFailures(t *testing.T) {
	commandErr := errors.New("down failed")
	repository := &migrationRepositoryFake{
		latest:    &domain.Migration{Version: "2", Status: "success"},
		byID:      map[uint]*domain.Migration{1: {ID: 1, Version: "1", Status: "success"}},
		byVersion: map[string]*domain.Migration{"2": {Version: "2", Status: "success", SqlUp: "up2", SqlDown: "down2"}},
	}
	service := migrationservice.NewMigrationService(repository, migrationDatabaseRepository(), &migrationExecutorFake{commandErr: commandErr}, editGroupService())
	_, err := service.RollbackMigrations(context.Background(), 8, 3, 1)
	if !errors.Is(err, commandErr) {
		t.Fatalf("error=%v", err)
	}

	saveErr := errors.New("save rollback failed")
	repository.saveErr = saveErr
	service = migrationservice.NewMigrationService(repository, migrationDatabaseRepository(), &migrationExecutorFake{}, editGroupService())
	_, err = service.RollbackMigrations(context.Background(), 8, 3, 1)
	if !errors.Is(err, saveErr) {
		t.Fatalf("error=%v", err)
	}
}
