package database_driver

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
)

const defaultQueryTimeout = 30 * time.Second

type databaseDriverService struct {
	databaseService    ports.DatabaseService
	databaseRepository ports.DatabaseRepository
	databaseExecutor   ports.DatabaseExecutorRepository
	userGroupsService  ports.UserGroupService
}

var _ ports.DatabaseDriverService = (*databaseDriverService)(nil)

func NewDatabaseDriverService(
	databaseService ports.DatabaseService,
	databaseRepository ports.DatabaseRepository,
	databaseExecutor ports.DatabaseExecutorRepository,
	userGroupsService ports.UserGroupService,
) ports.DatabaseDriverService {
	return &databaseDriverService{
		databaseService:    databaseService,
		databaseRepository: databaseRepository,
		databaseExecutor:   databaseExecutor,
		userGroupsService:  userGroupsService,
	}
}

func (s *databaseDriverService) ExecuteDatabaseQuery(
	ctx context.Context,
	databaseID uint,
	userID uint,
	query string,
) (*ports.DatabaseQueryResult, error) {
	database, err := s.getAccessibleDatabase(ctx, databaseID, userID)
	if err != nil {
		return nil, err
	}
	if err := s.userGroupsService.Authorize(ctx, userID, databaseID, domain.AccessViewOnly); err != nil {
		return nil, err
	}
	if err := validateSelectQuery(query); err != nil {
		return nil, err
	}
	databaseWithCredential, err := s.databaseRepository.FindWithCredentialByID(ctx, database.ID)
	if err != nil {
		return nil, err
	}
	cleanQuery := stripTrailingSemicolons(query)
	return executeWithTimeout(ctx, func(qCtx context.Context) (*ports.DatabaseQueryResult, error) {
		return s.databaseExecutor.ExecuteQuery(qCtx, databaseWithCredential, cleanQuery)
	})
}

func (s *databaseDriverService) ExplainDatabaseQuery(
	ctx context.Context,
	databaseID uint,
	userID uint,
	query string,
) (*ports.DatabaseQueryResult, error) {
	databaseWithCredential, cleanedQuery, err := s.prepareExplain(ctx, databaseID, userID, query)
	if err != nil {
		return nil, err
	}
	return executeWithTimeout(ctx, func(qCtx context.Context) (*ports.DatabaseQueryResult, error) {
		return s.databaseExecutor.Explain(qCtx, databaseWithCredential, cleanedQuery)
	})
}

func (s *databaseDriverService) ExplainAnalyzeDatabaseQuery(
	ctx context.Context,
	databaseID uint,
	userID uint,
	query string,
) (*ports.DatabaseQueryResult, error) {
	databaseWithCredential, cleanedQuery, err := s.prepareExplain(ctx, databaseID, userID, query)
	if err != nil {
		return nil, err
	}
	return executeWithTimeout(ctx, func(qCtx context.Context) (*ports.DatabaseQueryResult, error) {
		return s.databaseExecutor.ExplainAnalyze(qCtx, databaseWithCredential, cleanedQuery)
	})
}

func executeWithTimeout[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	result, err := fn(queryCtx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || queryCtx.Err() == context.DeadlineExceeded {
			var zero T
			return zero, domain.ErrQueryTimeout
		}
		return result, err
	}
	return result, nil
}

func (s *databaseDriverService) prepareExplain(
	ctx context.Context,
	databaseID uint,
	userID uint,
	query string,
) (domain.Database, string, error) {
	database, err := s.getAccessibleDatabase(ctx, databaseID, userID)
	if err != nil {
		return domain.Database{}, "", err
	}
	if err := s.userGroupsService.Authorize(ctx, userID, databaseID, domain.AccessViewOnly); err != nil {
		return domain.Database{}, "", err
	}
	cleanedQuery := stripLeadingExplain(stripTrailingSemicolons(query))
	if err := validateSelectQuery(cleanedQuery); err != nil {
		return domain.Database{}, "", err
	}
	databaseWithCredential, err := s.databaseRepository.FindWithCredentialByID(ctx, database.ID)
	if err != nil {
		return domain.Database{}, "", err
	}
	return databaseWithCredential, cleanedQuery, nil
}

func (s *databaseDriverService) getAccessibleDatabase(
	ctx context.Context,
	databaseID uint,
	userID uint,
) (*domain.Database, error) {
	if databaseID == 0 {
		return nil, domain.ErrInvalidDatabaseID
	}
	if userID == 0 {
		return nil, domain.ErrCurrentUserUnavailable
	}

	database, err := s.databaseService.GetDatabaseByIDAndUserID(ctx, databaseID, userID)
	if err != nil {
		return nil, err
	}
	return database, nil
}

func validateSelectQuery(query string) error {
	trimmedQuery := stripTrailingSemicolons(query)
	if strings.TrimSpace(trimmedQuery) == "" {
		return domain.ErrInvalidDatabaseQuery
	}

	isReadOnly, err := sqlutil.IsReadOnlyQuery(trimmedQuery)
	if err != nil || !isReadOnly {
		return domain.ErrInvalidDatabaseQuery
	}

	return nil
}

func stripTrailingSemicolons(query string) string {
	trimmed := strings.TrimSpace(query)
	for strings.HasSuffix(trimmed, ";") {
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	}
	return trimmed
}

func stripLeadingExplain(query string) string {
	trimmed := strings.TrimSpace(query)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "explain analyze ") {
		return strings.TrimSpace(trimmed[len("explain analyze "):])
	}
	if strings.HasPrefix(lower, "explain ") {
		return strings.TrimSpace(trimmed[len("explain "):])
	}
	return trimmed
}
