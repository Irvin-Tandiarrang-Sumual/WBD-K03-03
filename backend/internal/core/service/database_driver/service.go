package database_driver

import (
	"context"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

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
	if err := s.requireDatabaseAccess(ctx, userID, databaseID, domain.AccessViewOnly); err != nil {
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
	return s.databaseExecutor.ExecuteQuery(ctx, databaseWithCredential, cleanQuery)
}

func (s *databaseDriverService) ExplainDatabaseQuery(
	ctx context.Context,
	databaseID uint,
	userID uint,
	query string,
	analyze bool,
) (*ports.DatabaseQueryResult, error) {
	database, err := s.getAccessibleDatabase(ctx, databaseID, userID)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabaseAccess(ctx, userID, databaseID, domain.AccessViewOnly); err != nil {
		return nil, err
	}
	cleanedQuery := stripLeadingExplain(stripTrailingSemicolons(query))
	if err := validateSelectQuery(cleanedQuery); err != nil {
		return nil, err
	}
	databaseWithCredential, err := s.databaseRepository.FindWithCredentialByID(ctx, database.ID)
	if err != nil {
		return nil, err
	}
	if analyze {
		return s.databaseExecutor.ExplainAnalyze(ctx, databaseWithCredential, cleanedQuery)
	}
	return s.databaseExecutor.Explain(ctx, databaseWithCredential, cleanedQuery)
}

func (s *databaseDriverService) requireDatabaseAccess(
	ctx context.Context,
	userID uint,
	databaseID uint,
	required domain.AccessType,
) error {
	if err := s.userGroupsService.Authorize(ctx, userID, databaseID, required); err != nil {
		return domain.ErrDatabaseAccessDenied
	}

	return nil
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
	normalized := strings.ToUpper(strings.TrimSpace(query))
	if !strings.HasPrefix(normalized, "SELECT") {
		return domain.ErrInvalidDatabaseQuery
	}
	return nil
}

func stripTrailingSemicolons(query string) string {
	return strings.TrimRight(strings.TrimSpace(query), ";")
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
