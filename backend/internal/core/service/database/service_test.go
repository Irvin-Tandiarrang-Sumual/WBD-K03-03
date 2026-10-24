package database_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	databaseservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
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

	service := 	databaseservice.NewDatabaseService(
		databases,
		tester,
		&testutil.GroupRepository{},
		&testutil.SessionService{},
		&testutil.DatabasePoolInvalidator{},
	)

	created, err := 	service.CreateDatabase(
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

	_, err := 	service.CreateDatabase(
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

	_, err := 	service.CreateDatabase(
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

	err := 	service.DeleteDatabase(
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

	if err := 	service.DeleteDatabase(
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