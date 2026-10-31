package database_executor

import (
	"context"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type databaseExecutorService struct {
	databaseService    ports.DatabaseService
	databaseRepository ports.DatabaseRepository
	databaseExecutor   ports.DatabaseExecutor
	userGroupsService  ports.UserGroupService
}

var _ ports.DatabaseExecutorService = (*databaseExecutorService)(nil)

func NewDatabaseExecutorService(
	databaseService ports.DatabaseService,
	databaseRepository ports.DatabaseRepository,
	databaseExecutor ports.DatabaseExecutor,
	userGroupsService ports.UserGroupService,
) ports.DatabaseExecutorService {
	return &databaseExecutorService{
		databaseService:    databaseService,
		databaseRepository: databaseRepository,
		databaseExecutor:   databaseExecutor,
		userGroupsService:  userGroupsService,
	}
}

func (s *databaseExecutorService) ExecuteDatabaseQuery(
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

func (s *databaseExecutorService) ExplainDatabaseQuery(
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
	return s.databaseExecutor.ExplainQuery(ctx, databaseWithCredential, cleanedQuery, analyze)
}

func (s *databaseExecutorService) requireDatabaseAccess(
	ctx context.Context,
	userID uint,
	databaseID uint,
	required domain.AccessType,
) error {
	access, err := s.userGroupsService.GetEffectiveDatabaseAccess(ctx, userID, databaseID)
	if err != nil {
		return err
	}
	if !access.Allows(required) {
		return domain.ErrDatabaseAccessDenied
	}
	return nil
}

func (s *databaseExecutorService) getAccessibleDatabase(
	ctx context.Context,
	databaseID uint,
	userID uint,
) (*domain.Database, error) {
	return s.databaseService.GetDatabaseByIDAndUserID(ctx, databaseID, userID)
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

func hasKeywordPrefix(normalizedQuery, keyword string) bool {
	if !strings.HasPrefix(normalizedQuery, keyword) {
		return false
	}
	rest := normalizedQuery[len(keyword):]
	if rest == "" {
		return false
	}
	first := rest[0]
	return first == ' ' || first == '\n' || first == '\t' || first == '\r' || first == '*' || first == '('
}

func validateSelectQuery(query string) error {
	normalizedQuery := strings.ToLower(stripTrailingSemicolons(query))
	if normalizedQuery == "" {
		return domain.ErrInvalidDatabaseQuery
	}
	if !hasKeywordPrefix(normalizedQuery, "select") &&
		!hasKeywordPrefix(normalizedQuery, "with") {
		return domain.ErrInvalidDatabaseQuery
	}
	if strings.Contains(normalizedQuery, ";") {
		return domain.ErrInvalidDatabaseQuery
	}
	return nil
}
