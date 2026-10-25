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

func TestGetDatabaseHostsSuccessAndError(t *testing.T) {
	expectedHosts := []domain.DatabaseHost{
		{Host: "localhost", Port: 5432},
		{Host: "db.remote.com", Port: 3306},
	}
	repo := &testutil.DatabaseRepository{
		GetDistinctHostsFunc: func(ctx context.Context) ([]domain.DatabaseHost, error) {
			return expectedHosts, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)
	hosts, err := service.GetDatabaseHosts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(hosts, expectedHosts) {
		t.Fatalf("got=%+v want=%+v", hosts, expectedHosts)
	}

	// Repository returns error
	repoErr := &testutil.DatabaseRepository{
		GetDistinctHostsFunc: func(ctx context.Context) ([]domain.DatabaseHost, error) {
			return nil, errors.New("db query error")
		},
	}
	serviceErr := databaseservice.NewDatabaseService(repoErr, nil, nil, nil, nil)
	_, err = serviceErr.GetDatabaseHosts(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Nil repository returns empty slice
	serviceNil := databaseservice.NewDatabaseService(nil, nil, nil, nil, nil)
	hostsNil, err := serviceNil.GetDatabaseHosts(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(hostsNil) != 0 {
		t.Fatalf("expected empty slice, got %+v", hostsNil)
	}
}

func TestGetDatabasesByUserIDResolvesAccessLevels(t *testing.T) {
	db1 := &domain.Database{ID: 10, Name: "db-edit"}
	db2 := &domain.Database{ID: 20, Name: "db-view"}
	db3 := &domain.Database{ID: 30, Name: "db-both"}
	db4 := &domain.Database{ID: 40, Name: "db-none"}

	repo := &testutil.DatabaseRepository{
		FindAllByUserIDFunc: func(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
			return []*domain.Database{db1, db2, db3, db4}, 4, nil
		},
	}

	groupRepo := &testutil.GroupRepository{
		ListAccessTypesFunc: func(ctx context.Context, userID uint, databaseID uint) ([]domain.AccessType, error) {
			switch databaseID {
			case 10:
				return []domain.AccessType{domain.AccessViewEdit}, nil
			case 20:
				return []domain.AccessType{domain.AccessViewOnly}, nil
			case 30:
				// Multiple groups give both: ViewEdit takes precedence
				return []domain.AccessType{domain.AccessViewOnly, domain.AccessViewEdit}, nil
			case 40:
				// Error from group query
				return nil, errors.New("group query failed")
			default:
				return nil, nil
			}
		},
	}

	service := databaseservice.NewDatabaseService(repo, nil, groupRepo, nil, nil)
	dbs, total, err := service.GetDatabasesByUserID(context.Background(), 5, pagination.NewPaginationParams(1, 10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 4 || len(dbs) != 4 {
		t.Fatalf("total=%d len=%d", total, len(dbs))
	}
	if dbs[0].AccessType != domain.AccessViewEdit {
		t.Fatalf("db1 AccessType = %q, want %q", dbs[0].AccessType, domain.AccessViewEdit)
	}
	if dbs[1].AccessType != domain.AccessViewOnly {
		t.Fatalf("db2 AccessType = %q, want %q", dbs[1].AccessType, domain.AccessViewOnly)
	}
	if dbs[2].AccessType != domain.AccessViewEdit {
		t.Fatalf("db3 AccessType = %q, want %q", dbs[2].AccessType, domain.AccessViewEdit)
	}
	if dbs[3].AccessType != "" {
		t.Fatalf("db4 AccessType = %q, want empty on group error", dbs[3].AccessType)
	}

	// FindAllByUserID error propagation
	repoError := &testutil.DatabaseRepository{
		FindAllByUserIDFunc: func(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
			return nil, 0, errors.New("find all error")
		},
	}
	serviceError := databaseservice.NewDatabaseService(repoError, nil, groupRepo, nil, nil)
	_, _, err = serviceError.GetDatabasesByUserID(context.Background(), 5, pagination.NewPaginationParams(1, 10))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetDatabaseByIDAndUserIDResolvesAccessLevelAndErrors(t *testing.T) {
	db := &domain.Database{ID: 100, Name: "db-inspect"}
	repo := &testutil.DatabaseRepository{
		FindByIDAndUserIDFunc: func(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error) {
			if databaseID == 999 {
				return nil, domain.ErrDatabaseNotFound
			}
			return db, nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		ListAccessTypesFunc: func(ctx context.Context, userID uint, databaseID uint) ([]domain.AccessType, error) {
			return []domain.AccessType{domain.AccessViewOnly, domain.AccessViewEdit}, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, groupRepo, nil, nil)

	// Success case
	res, err := service.GetDatabaseByIDAndUserID(context.Background(), 100, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AccessType != domain.AccessViewEdit {
		t.Fatalf("AccessType=%q want=%q", res.AccessType, domain.AccessViewEdit)
	}

	// Error case (not found)
	_, err = service.GetDatabaseByIDAndUserID(context.Background(), 999, 7)
	if !errors.Is(err, domain.ErrDatabaseNotFound) {
		t.Fatalf("got err=%v, want ErrDatabaseNotFound", err)
	}

	// Nil DB returned without error
	repoNil := &testutil.DatabaseRepository{
		FindByIDAndUserIDFunc: func(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error) {
			return nil, nil
		},
	}
	serviceNil := databaseservice.NewDatabaseService(repoNil, nil, groupRepo, nil, nil)
	nilRes, err := serviceNil.GetDatabaseByIDAndUserID(context.Background(), 100, 7)
	if err != nil || nilRes != nil {
		t.Fatalf("expected nil result and nil err, got res=%v err=%v", nilRes, err)
	}
}

func TestCreateDatabaseReusesExistingCredentialWhenPasswordEmpty(t *testing.T) {
	var testedPassword string
	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(_ context.Context, database domain.Database) error {
			testedPassword = database.Password
			return nil
		},
	}

	repo := &testutil.DatabaseRepository{
		FindCredentialByHostAndPortFunc: func(ctx context.Context, host string, port uint, user string) (string, error) {
			if host == "db.prod.internal" && port == 5432 && user == "appuser" {
				return "reused-secret-password", nil
			}
			return "", nil
		},
		CreateDatabaseFunc: func(ctx context.Context, database *domain.Database) (*domain.Database, error) {
			database.ID = 55
			return database, nil
		},
	}

	service := databaseservice.NewDatabaseService(repo, tester, nil, nil, nil)

	input := &domain.CreateDatabaseInput{
		Host:     "db.prod.internal",
		Port:     5432,
		User:     "appuser",
		Password: "", // empty password to trigger reuse
		Name:     "customer_db",
		Type:     "postgresql",
	}

	created, err := service.CreateDatabase(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if testedPassword != "reused-secret-password" {
		t.Fatalf("connection tester received password=%q, want 'reused-secret-password'", testedPassword)
	}
	if created.Password != "reused-secret-password" {
		t.Fatalf("created database password=%q, want 'reused-secret-password'", created.Password)
	}

	// When credential lookup fails or returns empty, password stays empty
	repoNoCred := &testutil.DatabaseRepository{
		FindCredentialByHostAndPortFunc: func(ctx context.Context, host string, port uint, user string) (string, error) {
			return "", errors.New("lookup error")
		},
		CreateDatabaseFunc: func(ctx context.Context, database *domain.Database) (*domain.Database, error) {
			return database, nil
		},
	}
	testedPassword = "unset"
	serviceNoCred := databaseservice.NewDatabaseService(repoNoCred, tester, nil, nil, nil)
	createdNoCred, err := serviceNoCred.CreateDatabase(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdNoCred.Password != "" {
		t.Fatalf("expected empty password, got %q", createdNoCred.Password)
	}
}
