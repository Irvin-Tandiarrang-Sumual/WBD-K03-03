package migration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	migrationservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

type migrationRepositoryFake struct {
	ports.MigrationRepository
	latest        *domain.Migration
	byID          map[uint]*domain.Migration
	byVersion     map[string]*domain.Migration
	saved         *domain.Migration
	findLatestErr error
}

func (f *migrationRepositoryFake) FindLatestByDatabaseID(context.Context, uint) (*domain.Migration, error) {
	return f.latest, f.findLatestErr
}

func (f *migrationRepositoryFake) FindByIDAndDatabaseID(_ context.Context, id, _ uint) (*domain.Migration, error) {
	migration, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrMigrationNotFound
	}
	return migration, nil
}

func (f *migrationRepositoryFake) FindByVersionAndDatabaseID(_ context.Context, version string, _ uint) (*domain.Migration, error) {
	migration, ok := f.byVersion[version]
	if !ok {
		return nil, domain.ErrMigrationNotFound
	}
	return migration, nil
}

func (f *migrationRepositoryFake) Save(_ context.Context, migration *domain.Migration) (*domain.Migration, error) {
	f.saved = migration
	return migration, nil
}

type migrationExecutorFake struct {
	ports.DatabaseExecutor
	commands []string
}

func (f *migrationExecutorFake) ExecuteQuery(context.Context, domain.Database, string, ...any) (*ports.DatabaseQueryResult, error) {
	return &ports.DatabaseQueryResult{}, nil
}

func (f *migrationExecutorFake) ExecuteCommand(_ context.Context, _ domain.Database, command string, _ ...any) (*ports.DatabaseCommandResult, error) {
	f.commands = append(f.commands, command)
	return &ports.DatabaseCommandResult{}, nil
}

func (f *migrationExecutorFake) ExplainQuery(context.Context, domain.Database, string, bool, ...any) (*ports.DatabaseQueryResult, error) {
	return &ports.DatabaseQueryResult{}, nil
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

func TestRollbackCreatesLinearRollbackMigration(t *testing.T) {
	executor := &migrationExecutorFake{}
	repository := &migrationRepositoryFake{
		latest: &domain.Migration{Version: "3"},
		byID:   map[uint]*domain.Migration{1: {ID: 1, Version: "1"}},
		byVersion: map[string]*domain.Migration{
			"3": {Version: "3", SqlUp: "up3", SqlDown: "down3"},
			"2": {Version: "2", SqlUp: "up2", SqlDown: "down2"},
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
