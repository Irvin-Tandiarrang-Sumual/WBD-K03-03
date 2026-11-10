package database_driver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	databasedriverservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database_driver"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

func newTestDriverService(
	dbService *testutil.DatabaseService,
	dbRepo *testutil.DatabaseRepository,
	executor *testutil.DatabaseExecutorRepository,
	groupService *testutil.UserGroupService,
) ports.DatabaseDriverService {
	if dbService == nil {
		dbService = &testutil.DatabaseService{
			GetDatabaseByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
				return &domain.Database{ID: 5, Name: "appdb"}, nil
			},
		}
	}
	if dbRepo == nil {
		dbRepo = &testutil.DatabaseRepository{
			FindWithCredentialByIDFunc: func(context.Context, uint) (domain.Database, error) {
				return domain.Database{ID: 5, Name: "appdb", Password: "secret"}, nil
			},
		}
	}
	if groupService == nil {
		groupService = &testutil.UserGroupService{
			AuthorizeFunc: func(context.Context, uint, uint, domain.AccessType) error {
				return nil
			},
		}
	}
	if executor == nil {
		executor = &testutil.DatabaseExecutorRepository{
			ExecuteQueryFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
				return &ports.DatabaseQueryResult{}, nil
			},
		}
	}
	return databasedriverservice.NewDatabaseDriverService(dbService, dbRepo, executor, groupService)
}

func TestExecuteDatabaseQuery_ValidReadOnlyQuery_ReturnsQueryResult(t *testing.T) {
	// Arrange
	expectedResult := &ports.DatabaseQueryResult{
		Columns: []string{"id", "username"},
		Rows: []map[string]any{
			{"id": int64(1), "username": "admin"},
		},
	}
	executor := &testutil.DatabaseExecutorRepository{
		ExecuteQueryFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
			return expectedResult, nil
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	result, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT id, username FROM users")

	// Assert
	if err != nil {
		t.Fatalf("unexpected ExecuteDatabaseQuery() error: %v", err)
	}
	if result != expectedResult {
		t.Fatalf("expected result %+v, got: %+v", expectedResult, result)
	}
}

func TestExecuteDatabaseQuery_TrailingSemicolons_StripsSemicolonsBeforeExecution(t *testing.T) {
	// Arrange
	var executedQuery string
	executor := &testutil.DatabaseExecutorRepository{
		ExecuteQueryFunc: func(_ context.Context, _ domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			executedQuery = query
			return &ports.DatabaseQueryResult{}, nil
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	_, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT 1;;;   ")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if executedQuery != "SELECT 1" {
		t.Fatalf("expected trailing semicolons to be stripped, got: %q", executedQuery)
	}
}

func TestExecuteDatabaseQuery_ValidQuery_PassesDecryptedCredentialToExecutor(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(_ context.Context, databaseID uint) (domain.Database, error) {
			return domain.Database{ID: databaseID, Password: "decrypted-pass-123"}, nil
		},
	}
	var executedDatabase domain.Database
	executor := &testutil.DatabaseExecutorRepository{
		ExecuteQueryFunc: func(_ context.Context, db domain.Database, _ string) (*ports.DatabaseQueryResult, error) {
			executedDatabase = db
			return &ports.DatabaseQueryResult{}, nil
		},
	}
	service := newTestDriverService(nil, dbRepo, executor, nil)

	// Act
	_, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if executedDatabase.Password != "decrypted-pass-123" {
		t.Fatalf("expected decrypted password passed to executor, got: %q", executedDatabase.Password)
	}
}

func TestExecuteDatabaseQuery_ValidQuery_VerifiesViewOnlyAccessWithUserGroupService(t *testing.T) {
	// Arrange
	var capturedUserID, capturedDBID uint
	var capturedAccess domain.AccessType
	groupService := &testutil.UserGroupService{
		AuthorizeFunc: func(_ context.Context, userID uint, databaseID uint, access domain.AccessType) error {
			capturedUserID = userID
			capturedDBID = databaseID
			capturedAccess = access
			return nil
		},
	}
	service := newTestDriverService(nil, nil, nil, groupService)

	// Act
	_, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedUserID != 9 || capturedDBID != 5 || capturedAccess != domain.AccessViewOnly {
		t.Fatalf("expected Authorize(userID=9, dbID=5, access=view_only), got: userID=%d dbID=%d access=%v",
			capturedUserID, capturedDBID, capturedAccess)
	}
}

func TestExecuteDatabaseQuery_InvalidDatabaseIDZero_ReturnsErrInvalidDatabaseID(t *testing.T) {
	// Arrange
	service := databasedriverservice.NewDatabaseDriverService(nil, nil, nil, nil)

	// Act
	result, err := service.ExecuteDatabaseQuery(context.Background(), 0, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, domain.ErrInvalidDatabaseID) {
		t.Fatalf("expected ErrInvalidDatabaseID, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExecuteDatabaseQuery_InvalidUserIDZero_ReturnsErrCurrentUserUnavailable(t *testing.T) {
	// Arrange
	service := databasedriverservice.NewDatabaseDriverService(nil, nil, nil, nil)

	// Act
	result, err := service.ExecuteDatabaseQuery(context.Background(), 5, 0, "SELECT 1")

	// Assert
	if !errors.Is(err, domain.ErrCurrentUserUnavailable) {
		t.Fatalf("expected ErrCurrentUserUnavailable, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExecuteDatabaseQuery_DatabaseNotFoundOrNoAccess_PropagatesError(t *testing.T) {
	// Arrange
	notFoundErr := domain.ErrDatabaseNotFound
	databaseService := &testutil.DatabaseService{
		GetDatabaseByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
			return nil, notFoundErr
		},
	}
	service := databasedriverservice.NewDatabaseDriverService(databaseService, nil, nil, nil)

	// Act
	result, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("expected error %v, got: %v", notFoundErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExecuteDatabaseQuery_AuthorizationFails_ReturnsError(t *testing.T) {
	// Arrange
	accessErr := domain.ErrDatabaseAccessDenied
	userGroupService := &testutil.UserGroupService{
		AuthorizeFunc: func(context.Context, uint, uint, domain.AccessType) error {
			return accessErr
		},
	}
	service := newTestDriverService(nil, nil, nil, userGroupService)

	// Act
	result, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, accessErr) {
		t.Fatalf("expected error %v, got: %v", accessErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExecuteDatabaseQuery_EmptyOrWhitespaceQuery_ReturnsErrInvalidDatabaseQuery(t *testing.T) {
	// Arrange
	service := newTestDriverService(nil, nil, nil, nil)

	// Act
	result, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "   ; ;   ")

	// Assert
	if !errors.Is(err, domain.ErrInvalidDatabaseQuery) {
		t.Fatalf("expected ErrInvalidDatabaseQuery for empty/whitespace query, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExecuteDatabaseQuery_WriteOrMutationQuery_ReturnsErrInvalidDatabaseQuery(t *testing.T) {
	// Arrange
	cases := []struct {
		name  string
		query string
	}{
		{name: "insert query", query: "INSERT INTO users (id, name) VALUES (1, 'alice')"},
		{name: "update query", query: "UPDATE users SET name = 'bob' WHERE id = 1"},
		{name: "delete query", query: "DELETE FROM users WHERE id = 1"},
		{name: "drop table query", query: "DROP TABLE users"},
		{name: "alter table query", query: "ALTER TABLE users ADD COLUMN age INT"},
		{name: "truncate table query", query: "TRUNCATE TABLE users"},
		{name: "multi statement injection", query: "SELECT 1; DROP TABLE users;"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			executorCalled := false
			executor := &testutil.DatabaseExecutorRepository{
				ExecuteQueryFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
					executorCalled = true
					return nil, nil
				},
			}
			service := newTestDriverService(nil, nil, executor, nil)

			// Act
			result, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, tc.query)

			// Assert
			if !errors.Is(err, domain.ErrInvalidDatabaseQuery) {
				t.Fatalf("expected ErrInvalidDatabaseQuery for %s, got: %v", tc.name, err)
			}
			if executorCalled {
				t.Fatal("executor must not be called when query is non-read-only")
			}
			if result != nil {
				t.Fatalf("expected nil result, got: %+v", result)
			}
		})
	}
}

func TestExecuteDatabaseQuery_FindWithCredentialFails_PropagatesError(t *testing.T) {
	// Arrange
	credErr := errors.New("cipher decryption failed")
	databaseRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(context.Context, uint) (domain.Database, error) {
			return domain.Database{}, credErr
		},
	}
	service := newTestDriverService(nil, databaseRepo, nil, nil)

	// Act
	result, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, credErr) {
		t.Fatalf("expected credential error %v, got: %v", credErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExecuteDatabaseQuery_ExecutorReturnsError_PropagatesError(t *testing.T) {
	// Arrange
	execErr := errors.New("syntax error in SQL query")
	executor := &testutil.DatabaseExecutorRepository{
		ExecuteQueryFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
			return nil, execErr
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	result, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, execErr) {
		t.Fatalf("expected executor error %v, got: %v", execErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExecuteDatabaseQuery_QueryTimeout_ReturnsErrQueryTimeout(t *testing.T) {
	// Arrange
	executor := &testutil.DatabaseExecutorRepository{
		ExecuteQueryFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
			return nil, context.DeadlineExceeded
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	result, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, domain.ErrQueryTimeout) {
		t.Fatalf("expected ErrQueryTimeout, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExplainDatabaseQuery_ValidQuery_ReturnsExplainResult(t *testing.T) {
	// Arrange
	expectedResult := &ports.DatabaseQueryResult{
		Columns: []string{"QUERY PLAN"},
		Rows:    []map[string]any{{"QUERY PLAN": "Seq Scan on users"}},
	}
	executor := &testutil.DatabaseExecutorRepository{
		ExplainFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
			return expectedResult, nil
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	result, err := service.ExplainDatabaseQuery(context.Background(), 5, 9, "SELECT * FROM users")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expectedResult {
		t.Fatalf("expected result %+v, got: %+v", expectedResult, result)
	}
}

func TestExplainDatabaseQuery_LeadingExplainPrefix_StripsPrefixBeforeExecution(t *testing.T) {
	// Arrange
	var receivedQuery string
	executor := &testutil.DatabaseExecutorRepository{
		ExplainFunc: func(_ context.Context, _ domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			receivedQuery = query
			return &ports.DatabaseQueryResult{}, nil
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	_, err := service.ExplainDatabaseQuery(context.Background(), 5, 9, "EXPLAIN SELECT * FROM users;;")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedQuery != "SELECT * FROM users" {
		t.Fatalf("expected 'EXPLAIN' prefix stripped, got: %q", receivedQuery)
	}
}

func TestExplainDatabaseQuery_LeadingExplainAnalyzePrefix_StripsPrefixBeforeExecution(t *testing.T) {
	// Arrange
	var receivedQuery string
	executor := &testutil.DatabaseExecutorRepository{
		ExplainFunc: func(_ context.Context, _ domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			receivedQuery = query
			return &ports.DatabaseQueryResult{}, nil
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	_, err := service.ExplainDatabaseQuery(context.Background(), 5, 9, "EXPLAIN ANALYZE SELECT * FROM users")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedQuery != "SELECT * FROM users" {
		t.Fatalf("expected 'EXPLAIN ANALYZE' prefix stripped, got: %q", receivedQuery)
	}
}

func TestExplainDatabaseQuery_WriteQuery_ReturnsErrInvalidDatabaseQuery(t *testing.T) {
	// Arrange
	service := newTestDriverService(nil, nil, nil, nil)

	// Act
	result, err := service.ExplainDatabaseQuery(context.Background(), 5, 9, "EXPLAIN DELETE FROM users")

	// Assert
	if !errors.Is(err, domain.ErrInvalidDatabaseQuery) {
		t.Fatalf("expected ErrInvalidDatabaseQuery for explain of write query, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExplainDatabaseQuery_ExecutorReturnsError_PropagatesError(t *testing.T) {
	// Arrange
	wantErr := errors.New("explain failed")
	executor := &testutil.DatabaseExecutorRepository{
		ExplainFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
			return nil, wantErr
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	result, err := service.ExplainDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got: %v", wantErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExplainDatabaseQuery_QueryTimeout_ReturnsErrQueryTimeout(t *testing.T) {
	// Arrange
	executor := &testutil.DatabaseExecutorRepository{
		ExplainFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
			return nil, context.DeadlineExceeded
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	result, err := service.ExplainDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, domain.ErrQueryTimeout) {
		t.Fatalf("expected ErrQueryTimeout, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExplainAnalyzeDatabaseQuery_ValidQuery_ReturnsExplainAnalyzeResult(t *testing.T) {
	// Arrange
	expectedResult := &ports.DatabaseQueryResult{
		Columns: []string{"QUERY PLAN"},
		Rows:    []map[string]any{{"QUERY PLAN": "Index Scan on users (cost=0.15..8.17 rows=1)"}},
	}
	executor := &testutil.DatabaseExecutorRepository{
		ExplainAnalyzeFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
			return expectedResult, nil
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	result, err := service.ExplainAnalyzeDatabaseQuery(context.Background(), 5, 9, "SELECT * FROM users")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expectedResult {
		t.Fatalf("expected result %+v, got: %+v", expectedResult, result)
	}
}

func TestExplainAnalyzeDatabaseQuery_LeadingExplainAnalyzePrefix_StripsPrefixBeforeExecution(t *testing.T) {
	// Arrange
	var receivedQuery string
	executor := &testutil.DatabaseExecutorRepository{
		ExplainAnalyzeFunc: func(_ context.Context, _ domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			receivedQuery = query
			return &ports.DatabaseQueryResult{}, nil
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	_, err := service.ExplainAnalyzeDatabaseQuery(context.Background(), 5, 9, "EXPLAIN ANALYZE SELECT * FROM users;;")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedQuery != "SELECT * FROM users" {
		t.Fatalf("expected 'EXPLAIN ANALYZE' prefix stripped, got: %q", receivedQuery)
	}
}

func TestExplainAnalyzeDatabaseQuery_LeadingExplainPrefix_StripsPrefixBeforeExecution(t *testing.T) {
	// Arrange
	var receivedQuery string
	executor := &testutil.DatabaseExecutorRepository{
		ExplainAnalyzeFunc: func(_ context.Context, _ domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			receivedQuery = query
			return &ports.DatabaseQueryResult{}, nil
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	_, err := service.ExplainAnalyzeDatabaseQuery(context.Background(), 5, 9, "EXPLAIN SELECT * FROM users")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedQuery != "SELECT * FROM users" {
		t.Fatalf("expected 'EXPLAIN' prefix stripped, got: %q", receivedQuery)
	}
}

func TestExplainAnalyzeDatabaseQuery_WriteQuery_ReturnsErrInvalidDatabaseQuery(t *testing.T) {
	// Arrange
	service := newTestDriverService(nil, nil, nil, nil)

	// Act
	result, err := service.ExplainAnalyzeDatabaseQuery(context.Background(), 5, 9, "EXPLAIN ANALYZE UPDATE users SET name = 'foo'")

	// Assert
	if !errors.Is(err, domain.ErrInvalidDatabaseQuery) {
		t.Fatalf("expected ErrInvalidDatabaseQuery, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExplainAnalyzeDatabaseQuery_ExecutorReturnsError_PropagatesError(t *testing.T) {
	// Arrange
	wantErr := errors.New("explain analyze execution error")
	executor := &testutil.DatabaseExecutorRepository{
		ExplainAnalyzeFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
			return nil, wantErr
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	result, err := service.ExplainAnalyzeDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got: %v", wantErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExplainAnalyzeDatabaseQuery_QueryTimeout_ReturnsErrQueryTimeout(t *testing.T) {
	// Arrange
	executor := &testutil.DatabaseExecutorRepository{
		ExplainAnalyzeFunc: func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error) {
			return nil, context.DeadlineExceeded
		},
	}
	service := newTestDriverService(nil, nil, executor, nil)

	// Act
	result, err := service.ExplainAnalyzeDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, domain.ErrQueryTimeout) {
		t.Fatalf("expected ErrQueryTimeout, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

// ----------------------------------------------------------------------------
// PrepareExplain Edge Cases
// ----------------------------------------------------------------------------

func TestExplainDatabaseQuery_DatabaseLookupFails_PropagatesError(t *testing.T) {
	// Arrange
	lookupErr := domain.ErrDatabaseNotFound
	databaseService := &testutil.DatabaseService{
		GetDatabaseByIDAndUserIDFunc: func(context.Context, uint, uint) (*domain.Database, error) {
			return nil, lookupErr
		},
	}
	service := databasedriverservice.NewDatabaseDriverService(databaseService, nil, nil, nil)

	// Act
	result, err := service.ExplainDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, lookupErr) {
		t.Fatalf("expected lookup error %v, got: %v", lookupErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExplainDatabaseQuery_AuthorizationFails_ReturnsError(t *testing.T) {
	// Arrange
	accessErr := domain.ErrDatabaseAccessDenied
	userGroupService := &testutil.UserGroupService{
		AuthorizeFunc: func(context.Context, uint, uint, domain.AccessType) error {
			return accessErr
		},
	}
	service := newTestDriverService(nil, nil, nil, userGroupService)

	// Act
	result, err := service.ExplainDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, accessErr) {
		t.Fatalf("expected access error %v, got: %v", accessErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}

func TestExplainDatabaseQuery_CredentialLookupFails_PropagatesError(t *testing.T) {
	// Arrange
	credErr := errors.New("decryption error")
	databaseRepo := &testutil.DatabaseRepository{
		FindWithCredentialByIDFunc: func(context.Context, uint) (domain.Database, error) {
			return domain.Database{}, credErr
		},
	}
	service := newTestDriverService(nil, databaseRepo, nil, nil)

	// Act
	result, err := service.ExplainDatabaseQuery(context.Background(), 5, 9, "SELECT 1")

	// Assert
	if !errors.Is(err, credErr) {
		t.Fatalf("expected cred error %v, got: %v", credErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}
}
