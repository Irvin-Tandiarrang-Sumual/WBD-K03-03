package database_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	databaseservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

func TestCreateValidatesTestsConnectionThenPersists(t *testing.T) {
	var tested domain.Database

	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(
			_ context.Context,
			database domain.Database,
		) error {
			tested = database
			return nil
		},
	}

	var persisted domain.Database

	databases := &testutil.DatabaseRepository{
		CreateFunc: func(
			_ context.Context,
			database domain.Database,
		) (domain.Database, error) {
			persisted = database
			database.ID = 11
			database.Password = ""

			return database, nil
		},
	}

	service := databaseservice.NewDatabaseService(
		databases,
		tester,
		&testutil.GroupRepository{},
		&testutil.SessionService{},
		&testutil.DatabasePoolInvalidator{},
	)

	created, err := service.CreateDatabase(
		context.Background(),
		&domain.CreateDatabaseInput{
			Port:     5432,
			Host:     " localhost ",
			User:     " postgres ",
			Password: "secret",
			Name:     " appdb ",
		},
	)

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if tested.Type != "postgresql" ||
		persisted.Type != "postgresql" {
		t.Fatalf(
			"default type not applied: tested=%+v persisted=%+v",
			tested,
			persisted,
		)
	}

	if persisted.Host != "localhost" ||
		persisted.User != "postgres" ||
		persisted.Name != "appdb" {
		t.Fatalf(
			"input was not normalized: %+v",
			persisted,
		)
	}

	if created.ID != 11 {
		t.Fatalf(
			"ID = %d, want 11",
			created.ID,
		)
	}
}

func TestCreateStopsWhenConnectionTestFails(t *testing.T) {
	wantErr := errors.New("cannot connect")

	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(
			context.Context,
			domain.Database,
		) error {
			return wantErr
		},
	}

	createCalled := false

	databases := &testutil.DatabaseRepository{
		CreateFunc: func(
			context.Context,
			domain.Database,
		) (domain.Database, error) {
			createCalled = true
			return domain.Database{}, nil
		},
	}

	service := databaseservice.NewDatabaseService(
		databases,
		tester,
		&testutil.GroupRepository{},
		&testutil.SessionService{},
		&testutil.DatabasePoolInvalidator{},
	)

	_, err := service.CreateDatabase(
		context.Background(),
		&domain.CreateDatabaseInput{
			Port: 5432,
			Host: "localhost",
			User: "postgres",
			Name: "db",
		},
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"error = %v, want %v",
			err,
			wantErr,
		)
	}

	if createCalled {
		t.Fatal(
			"repository Create() must not run after failed connection test",
		)
	}
}

func TestCreateRejectsInvalidInputBeforeConnectionTest(t *testing.T) {
	testCalled := false

	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(
			context.Context,
			domain.Database,
		) error {
			testCalled = true
			return nil
		},
	}

	service := databaseservice.NewDatabaseService(
		&testutil.DatabaseRepository{},
		tester,
		&testutil.GroupRepository{},
		&testutil.SessionService{},
		&testutil.DatabasePoolInvalidator{},
	)

	_, err := service.CreateDatabase(
		context.Background(),
		&domain.CreateDatabaseInput{
			Port: 0,
			Host: "localhost",
			User: "postgres",
			Name: "db",
		},
	)

	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf(
			"error = %v, want ErrInvalidInput",
			err,
		)
	}

	if testCalled {
		t.Fatal(
			"connection tester must not run for invalid input",
		)
	}
}

func TestDeleteRejectsDatabaseThatWouldLeaveGroupEmpty(t *testing.T) {
	databases := &testutil.DatabaseRepository{
		FindByIDFunc: func(
			context.Context,
			uint,
		) (domain.Database, error) {
			return domain.Database{
				ID: 9,
			}, nil
		},
	}

	groups := &testutil.GroupRepository{
		HasGroupWithOnlyDatabaseFunc: func(
			context.Context,
			uint,
		) (bool, error) {
			return true, nil
		},
	}

	service := databaseservice.NewDatabaseService(
		databases,
		&testutil.DatabaseConnectionTester{},
		groups,
		&testutil.SessionService{},
		&testutil.DatabasePoolInvalidator{},
	)

	err := service.DeleteDatabase(
		context.Background(),
		9,
	)

	if !errors.Is(err, domain.ErrDatabaseInUse) {
		t.Fatalf(
			"error = %v, want ErrDatabaseInUse",
			err,
		)
	}
}

func TestDeleteInvalidatesPoolDeletesAndRevokesAffectedUsers(t *testing.T) {
	var steps []string

	databases := &testutil.DatabaseRepository{
		FindByIDFunc: func(
			context.Context,
			uint,
		) (domain.Database, error) {
			return domain.Database{
				ID: 9,
			}, nil
		},

		DeleteFunc: func(
			context.Context,
			uint,
		) error {
			steps = append(
				steps,
				"delete",
			)

			return nil
		},
	}

	groups := &testutil.GroupRepository{
		HasGroupWithOnlyDatabaseFunc: func(
			context.Context,
			uint,
		) (bool, error) {
			return false, nil
		},

		ListUserIDsByDatabaseFunc: func(
			context.Context,
			uint,
		) ([]uint, error) {
			return []uint{2, 3}, nil
		},
	}

	pools := &testutil.DatabasePoolInvalidator{
		InvalidateFunc: func(
			uint,
		) error {
			steps = append(
				steps,
				"invalidate",
			)

			return nil
		},
	}

	sessions := &testutil.SessionService{
		RevokeUserFunc: func(
			_ context.Context,
			userID uint,
		) error {
			steps = append(
				steps,
				"revoke",
			)

			return nil
		},
	}

	service := databaseservice.NewDatabaseService(
		databases,
		&testutil.DatabaseConnectionTester{},
		groups,
		sessions,
		pools,
	)

	if err := service.DeleteDatabase(
		context.Background(),
		9,
	); err != nil {
		t.Fatalf(
			"Delete() error = %v",
			err,
		)
	}

	want := []string{
		"invalidate",
		"delete",
		"revoke",
		"revoke",
	}

	if !reflect.DeepEqual(steps, want) {
		t.Fatalf(
			"steps = %v, want %v",
			steps,
			want,
		)
	}
}
func TestDatabaseReadMethodsDelegateToRepository(t *testing.T) {
	params := pagination.NewPaginationParams(2, 5)
	database := &domain.Database{ID: 7, Name: "target"}
	repository := &testutil.DatabaseRepository{
		FindByIdFunc: func(_ context.Context, id uint) (*domain.Database, error) {
			if id != 7 {
				t.Fatalf("FindById id=%d", id)
			}
			return database, nil
		},
		FindAllFunc: func(_ context.Context, got pagination.PaginationParams) ([]*domain.Database, int64, error) {
			if got != params {
				t.Fatalf("FindAll params=%+v want=%+v", got, params)
			}
			return []*domain.Database{database}, 1, nil
		},
		FindAllByUserIDFunc: func(_ context.Context, userID uint, got pagination.PaginationParams) ([]*domain.Database, int64, error) {
			if userID != 3 || got != params {
				t.Fatalf("FindAllByUserID userID=%d params=%+v", userID, got)
			}
			return []*domain.Database{database}, 1, nil
		},
		FindByIDAndUserIDFunc: func(_ context.Context, databaseID uint, userID uint) (*domain.Database, error) {
			if databaseID != 7 || userID != 3 {
				t.Fatalf("FindByIDAndUserID databaseID=%d userID=%d", databaseID, userID)
			}
			return database, nil
		},
	}
	service := databaseservice.NewDatabaseService(repository, nil, nil, nil, nil)

	if got, err := service.GetDatabaseByID(context.Background(), 7); err != nil || got != database {
		t.Fatalf("GetDatabaseByID got=%v err=%v", got, err)
	}
	if got, total, err := service.GetAllDatabases(context.Background(), params); err != nil || total != 1 || len(got) != 1 {
		t.Fatalf("GetAllDatabases got=%v total=%d err=%v", got, total, err)
	}
	if got, total, err := service.GetDatabasesByUserID(context.Background(), 3, params); err != nil || total != 1 || len(got) != 1 {
		t.Fatalf("GetDatabasesByUserID got=%v total=%d err=%v", got, total, err)
	}
	if got, err := service.GetDatabaseByIDAndUserID(context.Background(), 7, 3); err != nil || got != database {
		t.Fatalf("GetDatabaseByIDAndUserID got=%v err=%v", got, err)
	}
}

func TestCreateRejectsNilInput(t *testing.T) {
	service := databaseservice.NewDatabaseService(&testutil.DatabaseRepository{}, nil, nil, nil, nil)
	_, err := service.CreateDatabase(context.Background(), nil)
	if !errors.Is(err, domain.ErrInvalidDatabaseInput) {
		t.Fatalf("error=%v, want ErrInvalidDatabaseInput", err)
	}
}

func TestCreateNormalizesExplicitDatabaseType(t *testing.T) {
	var saved domain.Database
	repository := &testutil.DatabaseRepository{
		CreateDatabaseFunc: func(_ context.Context, database *domain.Database) (*domain.Database, error) {
			saved = *database
			result := *database
			result.ID = 20
			return &result, nil
		},
	}
	service := databaseservice.NewDatabaseService(repository, nil, nil, nil, nil)

	created, err := service.CreateDatabase(context.Background(), &domain.CreateDatabaseInput{
		Port: 3306, Host: " db.local ", User: " root ", Password: "pw", Type: " MySQL ", Name: " store ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Type != "mysql" || saved.Host != "db.local" || saved.User != "root" || saved.Name != "store" || created.ID != 20 {
		t.Fatalf("saved=%+v created=%+v", saved, created)
	}
}

func TestCreateRejectsInvalidDatabaseFields(t *testing.T) {
	tests := []struct {
		name  string
		input domain.CreateDatabaseInput
	}{
		{name: "port zero", input: domain.CreateDatabaseInput{Host: "h", User: "u", Name: "n"}},
		{name: "port too large", input: domain.CreateDatabaseInput{Port: 65536, Host: "h", User: "u", Name: "n"}},
		{name: "host empty", input: domain.CreateDatabaseInput{Port: 5432, User: "u", Name: "n"}},
		{name: "user empty", input: domain.CreateDatabaseInput{Port: 5432, Host: "h", Name: "n"}},
		{name: "name empty", input: domain.CreateDatabaseInput{Port: 5432, Host: "h", User: "u"}},
		{name: "unsupported type", input: domain.CreateDatabaseInput{Port: 5432, Host: "h", User: "u", Name: "n", Type: "sqlite"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := databaseservice.NewDatabaseService(&testutil.DatabaseRepository{}, nil, nil, nil, nil)
			_, err := service.CreateDatabase(context.Background(), &tt.input)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error=%v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestCreatePropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("insert failed")
	repository := &testutil.DatabaseRepository{
		CreateDatabaseFunc: func(context.Context, *domain.Database) (*domain.Database, error) {
			return nil, wantErr
		},
	}
	service := databaseservice.NewDatabaseService(repository, nil, nil, nil, nil)
	_, err := service.CreateDatabase(context.Background(), &domain.CreateDatabaseInput{Port: 5432, Host: "localhost", User: "postgres", Name: "db"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want=%v", err, wantErr)
	}
}

func TestDeleteStopsAtEachFailedDependency(t *testing.T) {
	tests := []struct {
		name       string
		findErr    error
		onlyErr    error
		listErr    error
		invalidate error
		deleteErr  error
		revokeErr  error
		wantErr    error
	}{
		{name: "database lookup", findErr: domain.ErrDatabaseNotFound, wantErr: domain.ErrDatabaseNotFound},
		{name: "group invariant lookup", onlyErr: errors.New("group lookup"), wantErr: errors.New("group lookup")},
		{name: "affected users lookup", listErr: errors.New("users lookup"), wantErr: errors.New("users lookup")},
		{name: "pool invalidation", invalidate: errors.New("pool failure"), wantErr: errors.New("pool failure")},
		{name: "repository delete", deleteErr: errors.New("delete failure"), wantErr: errors.New("delete failure")},
		{name: "session revoke", revokeErr: errors.New("revoke failure"), wantErr: errors.New("revoke failure")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteCalled := false
			repository := &testutil.DatabaseRepository{
				FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
					if tt.findErr != nil {
						return nil, tt.findErr
					}
					return &domain.Database{ID: 1}, nil
				},
				DeleteFunc: func(context.Context, uint) error {
					deleteCalled = true
					return tt.deleteErr
				},
			}
			groups := &testutil.GroupRepository{
				HasGroupWithOnlyDatabaseFunc: func(context.Context, uint) (bool, error) { return false, tt.onlyErr },
				ListUserIDsByDatabaseFunc: func(context.Context, uint) ([]uint, error) {
					if tt.listErr != nil {
						return nil, tt.listErr
					}
					return []uint{7}, nil
				},
			}
			pools := &testutil.DatabasePoolInvalidator{InvalidateFunc: func(uint) error { return tt.invalidate }}
			sessions := &testutil.SessionService{RevokeUserFunc: func(context.Context, uint) error { return tt.revokeErr }}
			service := databaseservice.NewDatabaseService(repository, nil, groups, sessions, pools)

			err := service.DeleteDatabase(context.Background(), 1)
			if err == nil || err.Error() != tt.wantErr.Error() {
				t.Fatalf("error=%v want=%v", err, tt.wantErr)
			}
			if tt.findErr != nil || tt.onlyErr != nil || tt.listErr != nil || tt.invalidate != nil {
				if deleteCalled {
					t.Fatal("delete must not run after an earlier dependency failure")
				}
			}
		})
	}
}
