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

func TestDatabaseService_CreateDatabase_ValidInput_NormalizesFieldsTestsConnectionAndPersists(t *testing.T) {
	// Arrange
	var testedDatabase domain.Database
	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(_ context.Context, db domain.Database) error {
			testedDatabase = db
			return nil
		},
	}

	var savedDatabase domain.Database
	repo := &testutil.DatabaseRepository{
		SaveFunc: func(_ context.Context, db *domain.Database) (*domain.Database, error) {
			savedDatabase = *db
			persisted := *db
			persisted.ID = 11
			persisted.Password = ""
			return &persisted, nil
		},
	}

	service := databaseservice.NewDatabaseService(
		repo,
		tester,
		&testutil.UserGroupRepository{},
		&testutil.SessionService{},
		&testutil.DatabasePoolInvalidator{},
	)

	input := &domain.CreateDatabaseInput{
		Port:     5432,
		Host:     " localhost ",
		User:     " postgres ",
		Password: "secret",
		Name:     " appdb ",
	}

	// Act
	created, err := service.CreateDatabase(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("unexpected CreateDatabase() error: %v", err)
	}
	if testedDatabase.Type != domain.DatabaseTypePostgreSQL || savedDatabase.Type != domain.DatabaseTypePostgreSQL {
		t.Fatalf("expected default type %q, got tested=%q saved=%q", domain.DatabaseTypePostgreSQL, testedDatabase.Type, savedDatabase.Type)
	}
	if savedDatabase.Host != "localhost" || savedDatabase.User != "postgres" || savedDatabase.Name != "appdb" {
		t.Fatalf("expected fields to be trimmed, got %+v", savedDatabase)
	}
	if created.ID != 11 {
		t.Fatalf("expected created database ID 11, got: %d", created.ID)
	}
	if created.Password != "" {
		t.Fatalf("expected password to be cleared on return value, got: %q", created.Password)
	}
}

func TestDatabaseService_CreateDatabase_EmptyType_DefaultsToPostgresql(t *testing.T) {
	// Arrange
	var savedType string
	repo := &testutil.DatabaseRepository{
		SaveFunc: func(_ context.Context, db *domain.Database) (*domain.Database, error) {
			savedType = db.Type
			return db, nil
		},
	}
	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(context.Context, domain.Database) error { return nil },
	}
	service := databaseservice.NewDatabaseService(repo, tester, nil, nil, nil)

	input := &domain.CreateDatabaseInput{
		Port: 5432,
		Host: "localhost",
		User: "postgres",
		Name: "testdb",
		Type: "",
	}

	// Act
	_, err := service.CreateDatabase(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedType != domain.DatabaseTypePostgreSQL {
		t.Fatalf("expected default type %q, got: %q", domain.DatabaseTypePostgreSQL, savedType)
	}
}

func TestDatabaseService_CreateDatabase_ExplicitTypeUppercase_NormalizesToLowercase(t *testing.T) {
	// Arrange
	var savedType string
	repo := &testutil.DatabaseRepository{
		SaveFunc: func(_ context.Context, db *domain.Database) (*domain.Database, error) {
			savedType = db.Type
			return db, nil
		},
	}
	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(context.Context, domain.Database) error { return nil },
	}
	service := databaseservice.NewDatabaseService(repo, tester, nil, nil, nil)

	input := &domain.CreateDatabaseInput{
		Port: 3306,
		Host: "localhost",
		User: "root",
		Name: "testdb",
		Type: " MySQL ",
	}

	// Act
	_, err := service.CreateDatabase(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedType != domain.DatabaseTypeMySQL {
		t.Fatalf("expected normalized type %q, got: %q", domain.DatabaseTypeMySQL, savedType)
	}
}

func TestDatabaseService_CreateDatabase_NilInput_ReturnsErrInvalidDatabaseInput(t *testing.T) {
	// Arrange
	service := databaseservice.NewDatabaseService(&testutil.DatabaseRepository{}, nil, nil, nil, nil)

	// Act
	created, err := service.CreateDatabase(context.Background(), nil)

	// Assert
	if !errors.Is(err, domain.ErrInvalidDatabaseInput) {
		t.Fatalf("expected ErrInvalidDatabaseInput, got: %v", err)
	}
	if created != nil {
		t.Fatalf("expected nil created database, got: %+v", created)
	}
}

func TestDatabaseService_CreateDatabase_InvalidPortZeroOrTooLarge_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		port uint
	}{
		{name: "port zero", port: 0},
		{name: "port exceeding 65535", port: domain.MaxPortNumber + 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			service := databaseservice.NewDatabaseService(&testutil.DatabaseRepository{}, nil, nil, nil, nil)
			input := &domain.CreateDatabaseInput{
				Port: tc.port,
				Host: "localhost",
				User: "postgres",
				Name: "testdb",
			}

			// Act
			created, err := service.CreateDatabase(context.Background(), input)

			// Assert
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput for %s, got: %v", tc.name, err)
			}
			if created != nil {
				t.Fatalf("expected nil result, got: %+v", created)
			}
		})
	}
}

func TestDatabaseService_CreateDatabase_MissingRequiredFields_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	cases := []struct {
		name  string
		input domain.CreateDatabaseInput
	}{
		{name: "empty host", input: domain.CreateDatabaseInput{Port: 5432, Host: "   ", User: "postgres", Name: "appdb"}},
		{name: "empty user", input: domain.CreateDatabaseInput{Port: 5432, Host: "localhost", User: "   ", Name: "appdb"}},
		{name: "empty name", input: domain.CreateDatabaseInput{Port: 5432, Host: "localhost", User: "postgres", Name: "   "}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			testerCalled := false
			tester := &testutil.DatabaseConnectionTester{
				TestFunc: func(context.Context, domain.Database) error {
					testerCalled = true
					return nil
				},
			}
			service := databaseservice.NewDatabaseService(&testutil.DatabaseRepository{}, tester, nil, nil, nil)

			// Act
			created, err := service.CreateDatabase(context.Background(), &tc.input)

			// Assert
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got: %v", err)
			}
			if testerCalled {
				t.Fatal("connection tester must not be called when validation fails")
			}
			if created != nil {
				t.Fatalf("expected nil result, got: %+v", created)
			}
		})
	}
}

func TestDatabaseService_CreateDatabase_UnsupportedDatabaseType_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	service := databaseservice.NewDatabaseService(&testutil.DatabaseRepository{}, nil, nil, nil, nil)
	input := &domain.CreateDatabaseInput{
		Port: 5432,
		Host: "localhost",
		User: "postgres",
		Name: "appdb",
		Type: "sqlite",
	}

	// Act
	created, err := service.CreateDatabase(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for unsupported database type, got: %v", err)
	}
	if created != nil {
		t.Fatalf("expected nil result, got: %+v", created)
	}
}

func TestDatabaseService_CreateDatabase_FieldExceedsMaxLength_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	cases := []struct {
		name  string
		input domain.CreateDatabaseInput
	}{
		{
			name: "host exceeds 255 chars",
			input: domain.CreateDatabaseInput{
				Port: 5432, Host: string(make([]byte, domain.MaxDatabaseHostLength+1)), User: "postgres", Name: "appdb",
			},
		},
		{
			name: "name exceeds 63 chars",
			input: domain.CreateDatabaseInput{
				Port: 5432, Host: "localhost", User: "postgres", Name: string(make([]byte, domain.MaxDatabaseNameLength+1)),
			},
		},
		{
			name: "user exceeds 63 chars",
			input: domain.CreateDatabaseInput{
				Port: 5432, Host: "localhost", User: string(make([]byte, domain.MaxDatabaseUserLength+1)), Name: "appdb",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := databaseservice.NewDatabaseService(&testutil.DatabaseRepository{}, nil, nil, nil, nil)

			// Act
			created, err := service.CreateDatabase(context.Background(), &tc.input)

			// Assert
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput for %s, got: %v", tc.name, err)
			}
			if created != nil {
				t.Fatalf("expected nil result, got: %+v", created)
			}
		})
	}
}

func TestDatabaseService_CreateDatabase_EmptyPasswordWithExistingCredential_AutoEnrichesPasswordBeforeTesting(t *testing.T) {
	// Arrange
	var testedPassword, savedPassword string
	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(_ context.Context, db domain.Database) error {
			testedPassword = db.Password
			return nil
		},
	}
	repo := &testutil.DatabaseRepository{
		FindCredentialByHostAndPortFunc: func(_ context.Context, host string, port uint, user string) (string, error) {
			if host == "db.internal" && port == 5432 && user == "appuser" {
				return "reused-secret-password", nil
			}
			return "", nil
		},
		SaveFunc: func(_ context.Context, db *domain.Database) (*domain.Database, error) {
			savedPassword = db.Password
			return db, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, tester, nil, nil, nil)

	input := &domain.CreateDatabaseInput{
		Host:     "db.internal",
		Port:     5432,
		User:     "appuser",
		Password: "", // empty to trigger auto-enrichment
		Name:     "customer_db",
	}

	// Act
	_, err := service.CreateDatabase(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("unexpected CreateDatabase() error: %v", err)
	}
	if testedPassword != "reused-secret-password" {
		t.Fatalf("connection tester received %q, want 'reused-secret-password'", testedPassword)
	}
	if savedPassword != "reused-secret-password" {
		t.Fatalf("saved database received %q, want 'reused-secret-password'", savedPassword)
	}
}

func TestDatabaseService_CreateDatabase_EmptyPasswordWithoutExistingCredential_ProceedsWithEmptyPassword(t *testing.T) {
	// Arrange
	var testedPassword string
	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(_ context.Context, db domain.Database) error {
			testedPassword = db.Password
			return nil
		},
	}
	repo := &testutil.DatabaseRepository{
		FindCredentialByHostAndPortFunc: func(context.Context, string, uint, string) (string, error) {
			return "", nil // no existing credentials found
		},
		SaveFunc: func(_ context.Context, db *domain.Database) (*domain.Database, error) {
			return db, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, tester, nil, nil, nil)

	input := &domain.CreateDatabaseInput{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "",
		Name:     "local_db",
	}

	// Act
	_, err := service.CreateDatabase(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if testedPassword != "" {
		t.Fatalf("expected empty password for tester, got: %q", testedPassword)
	}
}

func TestDatabaseService_CreateDatabase_ConnectionTesterFails_PropagatesErrorAndDoesNotPersist(t *testing.T) {
	// Arrange
	connErr := errors.New("connection refused by host")
	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(context.Context, domain.Database) error {
			return connErr
		},
	}
	saveCalled := false
	repo := &testutil.DatabaseRepository{
		SaveFunc: func(context.Context, *domain.Database) (*domain.Database, error) {
			saveCalled = true
			return nil, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, tester, nil, nil, nil)

	input := &domain.CreateDatabaseInput{
		Port: 5432,
		Host: "localhost",
		User: "postgres",
		Name: "appdb",
	}

	// Act
	created, err := service.CreateDatabase(context.Background(), input)

	// Assert
	if !errors.Is(err, connErr) {
		t.Fatalf("expected connection error %v, got: %v", connErr, err)
	}
	if saveCalled {
		t.Fatal("repository Save must not be called when connection test fails")
	}
	if created != nil {
		t.Fatalf("expected nil created database, got: %+v", created)
	}
}

func TestDatabaseService_CreateDatabase_RepositorySaveFails_PropagatesError(t *testing.T) {
	// Arrange
	saveErr := errors.New("database duplicate key error")
	tester := &testutil.DatabaseConnectionTester{
		TestFunc: func(context.Context, domain.Database) error { return nil },
	}
	repo := &testutil.DatabaseRepository{
		SaveFunc: func(context.Context, *domain.Database) (*domain.Database, error) {
			return nil, saveErr
		},
	}
	service := databaseservice.NewDatabaseService(repo, tester, nil, nil, nil)

	input := &domain.CreateDatabaseInput{
		Port: 5432,
		Host: "localhost",
		User: "postgres",
		Name: "appdb",
	}

	// Act
	created, err := service.CreateDatabase(context.Background(), input)

	// Assert
	if !errors.Is(err, saveErr) {
		t.Fatalf("expected save error %v, got: %v", saveErr, err)
	}
	if created != nil {
		t.Fatalf("expected nil result on save failure, got: %+v", created)
	}
}

func TestDatabaseService_DeleteDatabase_ValidIDAndNotInUse_InvalidatesPoolDeletesAndRevokesAffectedUsers(t *testing.T) {
	// Arrange
	var executionOrder []string

	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(_ context.Context, id uint) (*domain.Database, error) {
			if id != 9 {
				t.Fatalf("expected id 9, got: %d", id)
			}
			return &domain.Database{ID: 9, Name: "to-delete"}, nil
		},
		DeleteFunc: func(_ context.Context, id uint) error {
			executionOrder = append(executionOrder, "delete")
			return nil
		},
	}

	userGroups := &testutil.UserGroupRepository{
		HasGroupWithOnlyThisDatabaseFunc: func(context.Context, uint) (bool, error) {
			return false, nil // not in use as sole database
		},
		FindAllUserIDsByDatabaseIDFunc: func(context.Context, uint) ([]uint, error) {
			return []uint{101, 102}, nil
		},
	}

	pools := &testutil.DatabasePoolInvalidator{
		InvalidateFunc: func(id uint) error {
			executionOrder = append(executionOrder, "invalidate")
			return nil
		},
	}

	var revokedUserIDs []uint
	sessions := &testutil.SessionService{
		RevokeUsersFunc: func(_ context.Context, userIDs []uint) error {
			executionOrder = append(executionOrder, "revoke")
			revokedUserIDs = userIDs
			return nil
		},
	}

	service := databaseservice.NewDatabaseService(repo, nil, userGroups, sessions, pools)

	// Act
	err := service.DeleteDatabase(context.Background(), 9)

	// Assert
	if err != nil {
		t.Fatalf("unexpected DeleteDatabase() error: %v", err)
	}
	wantOrder := []string{"invalidate", "delete", "revoke"}
	if !reflect.DeepEqual(executionOrder, wantOrder) {
		t.Fatalf("execution order = %v, want %v", executionOrder, wantOrder)
	}
	if !reflect.DeepEqual(revokedUserIDs, []uint{101, 102}) {
		t.Fatalf("revoked user IDs = %v, want [101 102]", revokedUserIDs)
	}
}

func TestDatabaseService_DeleteDatabase_NoAffectedUsers_DeletesDatabaseWithoutCallingRevokeUsers(t *testing.T) {
	// Arrange
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 5}, nil
		},
		DeleteFunc: func(context.Context, uint) error { return nil },
	}
	userGroups := &testutil.UserGroupRepository{
		HasGroupWithOnlyThisDatabaseFunc: func(context.Context, uint) (bool, error) { return false, nil },
		FindAllUserIDsByDatabaseIDFunc: func(context.Context, uint) ([]uint, error) {
			return []uint{}, nil // no affected users
		},
	}
	pools := &testutil.DatabasePoolInvalidator{
		InvalidateFunc: func(uint) error { return nil },
	}
	revokeCalled := false
	sessions := &testutil.SessionService{
		RevokeUsersFunc: func(context.Context, []uint) error {
			revokeCalled = true
			return nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, userGroups, sessions, pools)

	// Act
	err := service.DeleteDatabase(context.Background(), 5)

	// Assert
	if err != nil {
		t.Fatalf("unexpected DeleteDatabase() error: %v", err)
	}
	if revokeCalled {
		t.Fatal("sessionService.RevokeUsers must not be called when affected user IDs list is empty")
	}
}

func TestDatabaseService_DeleteDatabase_NilUserGroupsRepository_ProceedsWithoutInUseCheck(t *testing.T) {
	// Arrange
	deleted := false
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 5}, nil
		},
		DeleteFunc: func(context.Context, uint) error {
			deleted = true
			return nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	err := service.DeleteDatabase(context.Background(), 5)

	// Assert
	if err != nil {
		t.Fatalf("unexpected DeleteDatabase() error: %v", err)
	}
	if !deleted {
		t.Fatal("expected database to be deleted when user groups repo is nil")
	}
}

func TestDatabaseService_DeleteDatabase_DatabaseNotFound_ReturnsErrorWithoutAttemptingDelete(t *testing.T) {
	// Arrange
	notFoundErr := domain.ErrDatabaseNotFound
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return nil, notFoundErr
		},
		DeleteFunc: func(context.Context, uint) error {
			t.Fatal("repository Delete must not be called when database does not exist")
			return nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	err := service.DeleteDatabase(context.Background(), 999)

	// Assert
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("expected ErrDatabaseNotFound, got: %v", err)
	}
}

func TestDatabaseService_DeleteDatabase_DatabaseInUseByGroup_ReturnsErrDatabaseInUse(t *testing.T) {
	// Arrange
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 3}, nil
		},
		DeleteFunc: func(context.Context, uint) error {
			t.Fatal("repository Delete must not be called when database is in use")
			return nil
		},
	}
	userGroups := &testutil.UserGroupRepository{
		HasGroupWithOnlyThisDatabaseFunc: func(context.Context, uint) (bool, error) {
			return true, nil // leaves a group without any database!
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, userGroups, nil, nil)

	// Act
	err := service.DeleteDatabase(context.Background(), 3)

	// Assert
	if !errors.Is(err, domain.ErrDatabaseInUse) {
		t.Fatalf("expected ErrDatabaseInUse, got: %v", err)
	}
}

func TestDatabaseService_DeleteDatabase_GroupRepositoryFailsInUseCheck_PropagatesError(t *testing.T) {
	// Arrange
	groupErr := errors.New("group query failed")
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 3}, nil
		},
	}
	userGroups := &testutil.UserGroupRepository{
		HasGroupWithOnlyThisDatabaseFunc: func(context.Context, uint) (bool, error) {
			return false, groupErr
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, userGroups, nil, nil)

	// Act
	err := service.DeleteDatabase(context.Background(), 3)

	// Assert
	if !errors.Is(err, groupErr) {
		t.Fatalf("expected group error %v, got: %v", groupErr, err)
	}
}

func TestDatabaseService_DeleteDatabase_UserGroupRepositoryFailsUserLookup_PropagatesError(t *testing.T) {
	// Arrange
	lookupErr := errors.New("user lookup failed")
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 3}, nil
		},
	}
	userGroups := &testutil.UserGroupRepository{
		HasGroupWithOnlyThisDatabaseFunc: func(context.Context, uint) (bool, error) { return false, nil },
		FindAllUserIDsByDatabaseIDFunc: func(context.Context, uint) ([]uint, error) {
			return nil, lookupErr
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, userGroups, nil, nil)

	// Act
	err := service.DeleteDatabase(context.Background(), 3)

	// Assert
	if !errors.Is(err, lookupErr) {
		t.Fatalf("expected lookup error %v, got: %v", lookupErr, err)
	}
}

func TestDatabaseService_DeleteDatabase_PoolInvalidatorFails_PropagatesErrorAndDoesNotDelete(t *testing.T) {
	// Arrange
	poolErr := errors.New("connection pool drain timeout")
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 4}, nil
		},
		DeleteFunc: func(context.Context, uint) error {
			t.Fatal("repository Delete must not be called when pool invalidation fails")
			return nil
		},
	}
	userGroups := &testutil.UserGroupRepository{
		HasGroupWithOnlyThisDatabaseFunc: func(context.Context, uint) (bool, error) { return false, nil },
		FindAllUserIDsByDatabaseIDFunc:   func(context.Context, uint) ([]uint, error) { return []uint{1}, nil },
	}
	pools := &testutil.DatabasePoolInvalidator{
		InvalidateFunc: func(uint) error { return poolErr },
	}
	service := databaseservice.NewDatabaseService(repo, nil, userGroups, nil, pools)

	// Act
	err := service.DeleteDatabase(context.Background(), 4)

	// Assert
	if !errors.Is(err, poolErr) {
		t.Fatalf("expected pool error %v, got: %v", poolErr, err)
	}
}

func TestDatabaseService_DeleteDatabase_RepositoryDeleteFails_PropagatesErrorAndDoesNotRevokeUsers(t *testing.T) {
	// Arrange
	deleteErr := errors.New("foreign key constraint violation on delete")
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 4}, nil
		},
		DeleteFunc: func(context.Context, uint) error { return deleteErr },
	}
	userGroups := &testutil.UserGroupRepository{
		HasGroupWithOnlyThisDatabaseFunc: func(context.Context, uint) (bool, error) { return false, nil },
		FindAllUserIDsByDatabaseIDFunc:   func(context.Context, uint) ([]uint, error) { return []uint{1}, nil },
	}
	pools := &testutil.DatabasePoolInvalidator{
		InvalidateFunc: func(uint) error { return nil },
	}
	revokeCalled := false
	sessions := &testutil.SessionService{
		RevokeUsersFunc: func(context.Context, []uint) error {
			revokeCalled = true
			return nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, userGroups, sessions, pools)

	// Act
	err := service.DeleteDatabase(context.Background(), 4)

	// Assert
	if !errors.Is(err, deleteErr) {
		t.Fatalf("expected delete error %v, got: %v", deleteErr, err)
	}
	if revokeCalled {
		t.Fatal("RevokeUsers must not be called when repository Delete fails")
	}
}

func TestDatabaseService_DeleteDatabase_SessionServiceRevokeUsersFails_PropagatesError(t *testing.T) {
	// Arrange
	revokeErr := errors.New("session revocation store timeout")
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return &domain.Database{ID: 4}, nil
		},
		DeleteFunc: func(context.Context, uint) error { return nil },
	}
	userGroups := &testutil.UserGroupRepository{
		HasGroupWithOnlyThisDatabaseFunc: func(context.Context, uint) (bool, error) { return false, nil },
		FindAllUserIDsByDatabaseIDFunc:   func(context.Context, uint) ([]uint, error) { return []uint{1}, nil },
	}
	pools := &testutil.DatabasePoolInvalidator{
		InvalidateFunc: func(uint) error { return nil },
	}
	sessions := &testutil.SessionService{
		RevokeUsersFunc: func(context.Context, []uint) error { return revokeErr },
	}
	service := databaseservice.NewDatabaseService(repo, nil, userGroups, sessions, pools)

	// Act
	err := service.DeleteDatabase(context.Background(), 4)

	// Assert
	if !errors.Is(err, revokeErr) {
		t.Fatalf("expected revoke error %v, got: %v", revokeErr, err)
	}
}

func TestDatabaseService_GetDatabaseByID_ExistingID_ReturnsDatabase(t *testing.T) {
	// Arrange
	expectedDB := &domain.Database{ID: 7, Name: "prod_db", Port: 5432, Host: "db.prod"}
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(_ context.Context, id uint) (*domain.Database, error) {
			if id != 7 {
				t.Fatalf("expected id 7, got: %d", id)
			}
			return expectedDB, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	db, err := service.GetDatabaseByID(context.Background(), 7)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db != expectedDB {
		t.Fatalf("expected database %+v, got: %+v", expectedDB, db)
	}
}

func TestDatabaseService_GetDatabaseByID_NonExistentID_PropagatesError(t *testing.T) {
	// Arrange
	notFoundErr := domain.ErrDatabaseNotFound
	repo := &testutil.DatabaseRepository{
		FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
			return nil, notFoundErr
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	db, err := service.GetDatabaseByID(context.Background(), 999)

	// Assert
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("expected ErrDatabaseNotFound, got: %v", err)
	}
	if db != nil {
		t.Fatalf("expected nil database on error, got: %+v", db)
	}
}

func TestDatabaseService_GetAllDatabases_ValidParams_ReturnsPaginatedDatabasesAndTotal(t *testing.T) {
	// Arrange
	params := pagination.NewPaginationParams(2, 10)
	dbs := []*domain.Database{
		{ID: 11, Name: "db1"},
		{ID: 12, Name: "db2"},
	}
	repo := &testutil.DatabaseRepository{
		FindAllFunc: func(_ context.Context, p pagination.PaginationParams) ([]*domain.Database, int64, error) {
			if p != params {
				t.Fatalf("expected pagination %+v, got %+v", params, p)
			}
			return dbs, 25, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	result, total, err := service.GetAllDatabases(context.Background(), params)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 25 {
		t.Fatalf("expected total 25, got: %d", total)
	}
	if len(result) != 2 || result[0].ID != 11 || result[1].ID != 12 {
		t.Fatalf("unexpected databases returned: %+v", result)
	}
}

func TestDatabaseService_GetAllDatabases_RepositoryFails_PropagatesError(t *testing.T) {
	// Arrange
	repoErr := errors.New("query failed")
	repo := &testutil.DatabaseRepository{
		FindAllFunc: func(context.Context, pagination.PaginationParams) ([]*domain.Database, int64, error) {
			return nil, 0, repoErr
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	result, total, err := service.GetAllDatabases(context.Background(), pagination.NewPaginationParams(1, 10))

	// Assert
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error %v, got: %v", repoErr, err)
	}
	if total != 0 || result != nil {
		t.Fatalf("expected nil result and 0 total, got: total=%d, result=%+v", total, result)
	}
}

func TestDatabaseService_GetDatabaseHosts_RepositoryAvailable_ReturnsDistinctHosts(t *testing.T) {
	// Arrange
	expectedHosts := []domain.DatabaseHost{
		{Host: "localhost", Port: 5432, User: "postgres", Type: "postgresql"},
		{Host: "remote.db", Port: 3306, User: "root", Type: "mysql"},
	}
	repo := &testutil.DatabaseRepository{
		GetDistinctHostsFunc: func(context.Context) ([]domain.DatabaseHost, error) {
			return expectedHosts, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	hosts, err := service.GetDatabaseHosts(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(hosts, expectedHosts) {
		t.Fatalf("hosts = %+v, want %+v", hosts, expectedHosts)
	}
}

func TestDatabaseService_GetDatabaseHosts_RepositoryFails_PropagatesError(t *testing.T) {
	// Arrange
	queryErr := errors.New("db query error")
	repo := &testutil.DatabaseRepository{
		GetDistinctHostsFunc: func(context.Context) ([]domain.DatabaseHost, error) {
			return nil, queryErr
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	hosts, err := service.GetDatabaseHosts(context.Background())

	// Assert
	if !errors.Is(err, queryErr) {
		t.Fatalf("expected query error %v, got: %v", queryErr, err)
	}
	if hosts != nil {
		t.Fatalf("expected nil hosts on error, got: %+v", hosts)
	}
}

func TestDatabaseService_GetDatabaseHosts_NilRepository_ReturnsEmptySlice(t *testing.T) {
	// Arrange
	service := databaseservice.NewDatabaseService(nil, nil, nil, nil, nil)

	// Act
	hosts, err := service.GetDatabaseHosts(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hosts == nil || len(hosts) != 0 {
		t.Fatalf("expected empty non-nil slice, got: %+v", hosts)
	}
}

func TestDatabaseService_GetDatabasesByUserID_MultipleGroups_ResolvesHighestAccessTypePrecedence(t *testing.T) {
	// Arrange
	dbEdit := &domain.Database{ID: 10, Name: "db-edit"}
	dbView := &domain.Database{ID: 20, Name: "db-view"}
	dbUnion := &domain.Database{ID: 30, Name: "db-union"}
	dbErr := &domain.Database{ID: 40, Name: "db-error"}

	repo := &testutil.DatabaseRepository{
		FindAllByUserIDFunc: func(_ context.Context, userID uint, _ pagination.PaginationParams) ([]*domain.Database, int64, error) {
			if userID != 5 {
				t.Fatalf("expected userID 5, got: %d", userID)
			}
			return []*domain.Database{dbEdit, dbView, dbUnion, dbErr}, 4, nil
		},
	}

	userGroups := &testutil.UserGroupRepository{
		FindAllAccessTypesFunc: func(_ context.Context, userID uint, databaseID uint) ([]domain.AccessType, error) {
			switch databaseID {
			case 10:
				return []domain.AccessType{domain.AccessViewEdit}, nil
			case 20:
				return []domain.AccessType{domain.AccessViewOnly}, nil
			case 30:
				// Multiple groups give both view_only and view_edit -> view_edit takes precedence
				return []domain.AccessType{domain.AccessViewOnly, domain.AccessViewEdit}, nil
			case 40:
				return nil, errors.New("group query failed")
			default:
				return nil, nil
			}
		},
	}

	service := databaseservice.NewDatabaseService(repo, nil, userGroups, nil, nil)

	// Act
	dbs, total, err := service.GetDatabasesByUserID(context.Background(), 5, pagination.NewPaginationParams(1, 10))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 4 || len(dbs) != 4 {
		t.Fatalf("expected 4 databases, got total=%d len=%d", total, len(dbs))
	}
	if dbs[0].AccessType != domain.AccessViewEdit {
		t.Fatalf("db1 AccessType = %q, want %q", dbs[0].AccessType, domain.AccessViewEdit)
	}
	if dbs[1].AccessType != domain.AccessViewOnly {
		t.Fatalf("db2 AccessType = %q, want %q", dbs[1].AccessType, domain.AccessViewOnly)
	}
	if dbs[2].AccessType != domain.AccessViewEdit {
		t.Fatalf("db3 AccessType = %q, want %q (highest privilege precedence)", dbs[2].AccessType, domain.AccessViewEdit)
	}
	if dbs[3].AccessType != "" {
		t.Fatalf("db4 AccessType = %q, want empty when access lookup fails", dbs[3].AccessType)
	}
}

func TestDatabaseService_GetDatabasesByUserID_RepositoryFails_PropagatesError(t *testing.T) {
	// Arrange
	queryErr := errors.New("db find all failed")
	repo := &testutil.DatabaseRepository{
		FindAllByUserIDFunc: func(context.Context, uint, pagination.PaginationParams) ([]*domain.Database, int64, error) {
			return nil, 0, queryErr
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	dbs, total, err := service.GetDatabasesByUserID(context.Background(), 5, pagination.NewPaginationParams(1, 10))

	// Assert
	if !errors.Is(err, queryErr) {
		t.Fatalf("expected query error %v, got: %v", queryErr, err)
	}
	if total != 0 || dbs != nil {
		t.Fatalf("expected nil result and 0 total, got: total=%d, dbs=%+v", total, dbs)
	}
}

func TestDatabaseService_GetDatabasesByUserID_NilUserGroupsRepository_LeavesAccessTypeEmpty(t *testing.T) {
	// Arrange
	db := &domain.Database{ID: 10, Name: "db1"}
	repo := &testutil.DatabaseRepository{
		FindAllByUserIDFunc: func(context.Context, uint, pagination.PaginationParams) ([]*domain.Database, int64, error) {
			return []*domain.Database{db}, 1, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	dbs, total, err := service.GetDatabasesByUserID(context.Background(), 1, pagination.NewPaginationParams(1, 10))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(dbs) != 1 {
		t.Fatalf("expected 1 database, got total=%d, len=%d", total, len(dbs))
	}
	if dbs[0].AccessType != "" {
		t.Fatalf("expected empty access type when user groups repo is nil, got: %q", dbs[0].AccessType)
	}
}

func TestDatabaseService_GetDatabaseByIDAndUserID_UserHasAccess_ResolvesAccessTypeAndReturnsDatabase(t *testing.T) {
	// Arrange
	targetDB := &domain.Database{ID: 100, Name: "db-inspect"}
	repo := &testutil.DatabaseRepository{
		FindByIDAndUserIDFunc: func(_ context.Context, databaseID uint, userID uint) (*domain.Database, error) {
			if databaseID != 100 || userID != 7 {
				t.Fatalf("expected databaseID 100 and userID 7, got: %d, %d", databaseID, userID)
			}
			return targetDB, nil
		},
	}
	userGroups := &testutil.UserGroupRepository{
		FindAllAccessTypesFunc: func(context.Context, uint, uint) ([]domain.AccessType, error) {
			return []domain.AccessType{domain.AccessViewOnly, domain.AccessViewEdit}, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, userGroups, nil, nil)

	// Act
	result, err := service.GetDatabaseByIDAndUserID(context.Background(), 100, 7)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != targetDB {
		t.Fatalf("expected database %+v, got: %+v", targetDB, result)
	}
	if result.AccessType != domain.AccessViewEdit {
		t.Fatalf("expected resolved AccessType %q, got: %q", domain.AccessViewEdit, result.AccessType)
	}
}

func TestDatabaseService_GetDatabaseByIDAndUserID_NotFoundOrNoAccess_PropagatesError(t *testing.T) {
	// Arrange
	notFoundErr := domain.ErrDatabaseNotFound
	repo := &testutil.DatabaseRepository{
		FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
			return nil, notFoundErr
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	result, err := service.GetDatabaseByIDAndUserID(context.Background(), 999, 7)

	// Assert
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("expected error %v, got: %v", notFoundErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result on error, got: %+v", result)
	}
}

func TestDatabaseService_GetDatabaseByIDAndUserID_RepositoryReturnsNil_ReturnsNilWithoutError(t *testing.T) {
	// Arrange
	repo := &testutil.DatabaseRepository{
		FindByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
			return nil, nil
		},
	}
	service := databaseservice.NewDatabaseService(repo, nil, nil, nil, nil)

	// Act
	result, err := service.GetDatabaseByIDAndUserID(context.Background(), 100, 7)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}
