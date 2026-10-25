package migration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	migrationservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type migrationRepositoryFake struct {
	ports.MigrationRepository
	list             []*domain.Migration
	total            int64
	latest           *domain.Migration
	latestSuccessful *domain.Migration
	byID             map[uint]*domain.Migration
	byVersion        map[string]*domain.Migration
	saved            *domain.Migration
	findAllErr       error
	findLatestErr    error
	findByIDErr      error
	findByVersionErr error
	saveErr          error
}

func (f *migrationRepositoryFake) FindAllByDatabaseID(context.Context, uint, pagination.PaginationParams) ([]*domain.Migration, int64, error) {
	return f.list, f.total, f.findAllErr
}

func (f *migrationRepositoryFake) FindLatestByDatabaseID(context.Context, uint) (*domain.Migration, error) {
	return f.latest, f.findLatestErr
}

func (f *migrationRepositoryFake) FindLatestSuccessfulByDatabaseID(context.Context, uint) (*domain.Migration, error) {
	if f.latestSuccessful != nil {
		return f.latestSuccessful, nil
	}
	return f.latest, f.findLatestErr
}

func (f *migrationRepositoryFake) FindByIDAndDatabaseID(_ context.Context, id, _ uint) (*domain.Migration, error) {
	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}
	migration, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrMigrationNotFound
	}
	return migration, nil
}

func (f *migrationRepositoryFake) FindByVersionAndDatabaseID(_ context.Context, version string, _ uint) (*domain.Migration, error) {
	if f.findByVersionErr != nil {
		return nil, f.findByVersionErr
	}
	migration, ok := f.byVersion[version]
	if !ok {
		return nil, domain.ErrMigrationNotFound
	}
	return migration, nil
}

func (f *migrationRepositoryFake) Save(_ context.Context, migration *domain.Migration) (*domain.Migration, error) {
	if f.saveErr != nil {
		return nil, f.saveErr
	}
	if migration.ID == 0 {
		migration.ID = 1
	}
	f.saved = migration
	return migration, nil
}

func (f *migrationRepositoryFake) Update(_ context.Context, migration *domain.Migration) (*domain.Migration, error) {
	f.saved = migration
	return migration, nil
}

type migrationExecutorFake struct {
	ports.DatabaseExecutor
	commands      []string
	commandErr    error
	failOnCommand string
}

func (f *migrationExecutorFake) ExecuteQuery(context.Context, domain.Database, string, ...any) (*ports.DatabaseQueryResult, error) {
	return &ports.DatabaseQueryResult{}, nil
}

func (f *migrationExecutorFake) ExecuteCommand(_ context.Context, _ domain.Database, command string, _ ...any) (*ports.DatabaseCommandResult, error) {
	f.commands = append(f.commands, command)
	if f.failOnCommand != "" && command == f.failOnCommand {
		return nil, errors.New("execution failed on " + command)
	}
	if f.commandErr != nil {
		return nil, f.commandErr
	}
	return &ports.DatabaseCommandResult{}, nil
}

func (f *migrationExecutorFake) ExplainQuery(context.Context, domain.Database, string, bool, ...any) (*ports.DatabaseQueryResult, error) {
	return &ports.DatabaseQueryResult{}, nil
}

func (f *migrationExecutorFake) GetDatabaseMetadata(context.Context, domain.Database) (*ports.DatabaseMetadata, error) {
	return &ports.DatabaseMetadata{}, nil
}

func TestCreateMigrationPersistsNextVersionAfterExecutingSQL(t *testing.T) {
	executor := &migrationExecutorFake{}
	repository := &migrationRepositoryFake{latest: &domain.Migration{Version: "4"}}
	databases := &testutil.DatabaseRepository{
		FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
			return &domain.Database{ID: 8}, nil
		},
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 8, Type: "postgresql"}, nil
		},
	}
	groups := &testutil.GroupService{
		EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
			return domain.AccessViewEdit, nil
		},
	}

	service := migrationservice.NewMigrationService(repository, databases, executor, groups)
	created, err := service.CreateMigration(context.Background(), 8, 3, "CREATE TABLE users (id INT)")
	if err != nil {
		t.Fatalf("CreateMigration() error = %v", err)
	}
	if created.Version != "5" || repository.saved.Version != "5" {
		t.Fatalf("version = %q, want 5", created.Version)
	}
	if len(executor.commands) != 1 || executor.commands[0] != "CREATE TABLE users (id INT)" {
		t.Fatalf("commands = %v", executor.commands)
	}
}

func TestCreateMigrationRequiresEditAccess(t *testing.T) {
	service := migrationservice.NewMigrationService(
		&migrationRepositoryFake{},
		&testutil.DatabaseRepository{},
		&migrationExecutorFake{},
		&testutil.GroupService{
			EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return domain.AccessViewOnly, nil
			},
		},
	)

	_, err := service.CreateMigration(context.Background(), 1, 2, "CREATE TABLE t (id INT)")
	if !errors.Is(err, domain.ErrDatabaseAccessDenied) {
		t.Fatalf("error = %v, want access denied", err)
	}
}

func TestCreateMigrationPersistsFailedExecutionAsNextVersion(t *testing.T) {
	executor := &migrationExecutorFake{commandErr: errors.New("syntax error near INVALID")}
	repository := &migrationRepositoryFake{latest: &domain.Migration{Version: "4", Status: "fail"}}
	service := migrationservice.NewMigrationService(
		repository,
		&testutil.DatabaseRepository{
			FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
				return &domain.Database{ID: 8}, nil
			},
			FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
				return &domain.Database{ID: 8, Type: "postgresql"}, nil
			},
		},
		executor,
		&testutil.GroupService{
			EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return domain.AccessViewEdit, nil
			},
		},
	)

	created, err := service.CreateMigration(context.Background(), 8, 3, "CREATE TABLE invalid (")
	if err != nil {
		t.Fatalf("CreateMigration() error = %v", err)
	}
	if created.Version != "5" || created.Status != "fail" {
		t.Fatalf("created = %+v, want failed version 5", created)
	}
	if created.ErrorMessage == "" || repository.saved != created {
		t.Fatalf("failed migration was not saved with its error: %+v", created)
	}
}

func TestCreateMigrationSupportsMultiRowInsert(t *testing.T) {
	executor := &migrationExecutorFake{}
	repository := &migrationRepositoryFake{findLatestErr: domain.ErrMigrationNotFound}
	service := migrationservice.NewMigrationService(
		repository,
		&testutil.DatabaseRepository{
			FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
				return &domain.Database{ID: 8}, nil
			},
			FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
				return &domain.Database{ID: 8, Type: "postgresql"}, nil
			},
		},
		executor,
		&testutil.GroupService{
			EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return domain.AccessViewEdit, nil
			},
		},
	)

	sql := "INSERT INTO customers (name, email) VALUES ('Rina', 'rina@example.com'), ('Budi', 'budi@example.com')"
	created, err := service.CreateMigration(context.Background(), 8, 3, sql)
	if err != nil {
		t.Fatalf("CreateMigration() error = %v", err)
	}
	if created.Status != "success" {
		t.Fatalf("status = %q, want success; error=%q", created.Status, created.ErrorMessage)
	}
	wantDown := "DELETE FROM customers WHERE (name = 'Rina' AND email = 'rina@example.com') OR (name = 'Budi' AND email = 'budi@example.com')"
	if created.SqlDown != wantDown {
		t.Fatalf("sql_down = %q, want %q", created.SqlDown, wantDown)
	}
}

func TestCreateMigrationMultiStatementSucceedsAsSingleMigration(t *testing.T) {
	executor := &migrationExecutorFake{}
	repository := &migrationRepositoryFake{latest: &domain.Migration{Version: "1"}}
	service := migrationservice.NewMigrationService(
		repository,
		&testutil.DatabaseRepository{
			FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
				return &domain.Database{ID: 8}, nil
			},
			FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
				return &domain.Database{ID: 8, Type: "postgresql"}, nil
			},
		},
		executor,
		&testutil.GroupService{
			EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return domain.AccessViewEdit, nil
			},
		},
	)

	sql := "CREATE TABLE orders (id INT PRIMARY KEY);\nALTER TABLE orders ADD COLUMN total INT;\nINSERT INTO orders (id, total) VALUES (1, 50000);"
	created, err := service.CreateMigration(context.Background(), 8, 3, sql)
	if err != nil {
		t.Fatalf("CreateMigration() error = %v", err)
	}
	if created.Status != "success" || created.Version != "2" {
		t.Fatalf("created = %+v, want success version 2", created)
	}
	wantDown := "DELETE FROM orders WHERE id = 1 AND total = 50000;\nALTER TABLE orders DROP COLUMN total;\nDROP TABLE orders"
	if created.SqlDown != wantDown {
		t.Fatalf("sql_down = %q, want %q", created.SqlDown, wantDown)
	}
	if len(executor.commands) != 3 {
		t.Fatalf("commands = %v, want 3 executed statements", executor.commands)
	}
}

func TestCreateMigrationMultiStatementAtomicallyRollsBackOnFailure(t *testing.T) {
	executor := &migrationExecutorFake{
		failOnCommand: "INSERT INTO orders (id, total) VALUES (1, 50000)",
	}
	repository := &migrationRepositoryFake{latest: &domain.Migration{Version: "1"}}
	service := migrationservice.NewMigrationService(
		repository,
		&testutil.DatabaseRepository{
			FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
				return &domain.Database{ID: 8}, nil
			},
			FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
				return &domain.Database{ID: 8, Type: "postgresql"}, nil
			},
		},
		executor,
		&testutil.GroupService{
			EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return domain.AccessViewEdit, nil
			},
		},
	)

	sql := "CREATE TABLE orders (id INT PRIMARY KEY);\nALTER TABLE orders ADD COLUMN total INT;\nINSERT INTO orders (id, total) VALUES (1, 50000);"
	created, err := service.CreateMigration(context.Background(), 8, 3, sql)
	if err != nil {
		t.Fatalf("CreateMigration() error = %v", err)
	}
	if created.Status != "fail" || created.ErrorMessage == "" {
		t.Fatalf("created = %+v, want failed migration with error message", created)
	}

	wantCommands := []string{
		"CREATE TABLE orders (id INT PRIMARY KEY)",
		"ALTER TABLE orders ADD COLUMN total INT",
		"INSERT INTO orders (id, total) VALUES (1, 50000)",
		"ALTER TABLE orders DROP COLUMN total",
		"DROP TABLE orders",
	}
	if len(executor.commands) != len(wantCommands) {
		t.Fatalf("commands = %v, want %v", executor.commands, wantCommands)
	}
	for index, wantCmd := range wantCommands {
		if executor.commands[index] != wantCmd {
			t.Fatalf("command[%d] = %q, want %q", index, executor.commands[index], wantCmd)
		}
	}
}

func TestCreateMigrationDoesNotExecuteWhenHistoryCannotBeCreated(t *testing.T) {
	executor := &migrationExecutorFake{}
	repositoryError := errors.New("cannot create migration history")
	repository := &migrationRepositoryFake{
		latest:  &domain.Migration{Version: "1", Status: "fail"},
		saveErr: repositoryError,
	}
	service := migrationservice.NewMigrationService(
		repository,
		&testutil.DatabaseRepository{
			FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
				return &domain.Database{ID: 8}, nil
			},
			FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
				return &domain.Database{ID: 8, Type: "postgresql"}, nil
			},
		},
		executor,
		&testutil.GroupService{
			EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return domain.AccessViewEdit, nil
			},
		},
	)

	_, err := service.CreateMigration(context.Background(), 8, 3, "CREATE TABLE users (id INT)")
	if !errors.Is(err, repositoryError) {
		t.Fatalf("error = %v, want repository error", err)
	}
	if len(executor.commands) != 0 {
		t.Fatalf("commands = %v, want no target database execution", executor.commands)
	}
}

func TestRollbackCreatesLinearRollbackMigration(t *testing.T) {
	executor := &migrationExecutorFake{}
	repository := &migrationRepositoryFake{
		latest: &domain.Migration{Version: "3", Status: "success"},
		byID:   map[uint]*domain.Migration{1: {ID: 1, Version: "1", Status: "success"}},
		byVersion: map[string]*domain.Migration{
			"3": {Version: "3", Status: "success", SqlUp: "up3", SqlDown: "down3"},
			"2": {Version: "2", Status: "success", SqlUp: "up2", SqlDown: "down2"},
		},
	}
	databases := &testutil.DatabaseRepository{
		FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
			return &domain.Database{ID: 9}, nil
		},
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 9}, nil
		},
	}
	service := migrationservice.NewMigrationService(
		repository,
		databases,
		executor,
		&testutil.GroupService{
			EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return domain.AccessViewEdit, nil
			},
		},
	)

	rolledBack, err := service.RollbackMigrations(context.Background(), 9, 4, 1)
	if err != nil {
		t.Fatalf("RollbackMigrations() error = %v", err)
	}
	if len(rolledBack) != 3 || repository.saved.Version != "4" {
		t.Fatalf("rollback result=%+v saved=%+v", rolledBack, repository.saved)
	}
	if repository.saved.SqlUp != "down3;\ndown2" || repository.saved.SqlDown != "up2;\nup3" {
		t.Fatalf("rollback SQL up=%q down=%q", repository.saved.SqlUp, repository.saved.SqlDown)
	}
	if len(executor.commands) != 2 || executor.commands[0] != "down3" || executor.commands[1] != "down2" {
		t.Fatalf("commands = %v", executor.commands)
	}
}
