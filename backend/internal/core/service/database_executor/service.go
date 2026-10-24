package database_executor

import (
	"context"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type databaseExecutorService struct {
	databaseService       ports.DatabaseService
	databaseRepository    ports.DatabaseRepository
	databaseExecutor     ports.DatabaseExecutor
}

var _ ports.DatabaseExecutorService = (*databaseExecutorService)(nil)

func NewDatabaseExecutorService(
	databaseService ports.DatabaseService,
	databaseRepository ports.DatabaseRepository,
	databaseExecutor ports.DatabaseExecutor,
) ports.DatabaseExecutorService {
	return &databaseExecutorService{
		databaseService:    databaseService,
		databaseRepository: databaseRepository,
		databaseExecutor:  databaseExecutor,
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
	if err := validateSelectQuery(query); err != nil {
		return nil, err
	}
	databaseWithCredential, err := s.databaseRepository.FindWithCredentialByID(ctx, database.ID)
	if err != nil {
		return nil, err
	}
	return s.databaseExecutor.ExecuteQuery(ctx, databaseWithCredential, query)
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
	if err := validateSelectQuery(query); err != nil {
		return nil, err
	}
	databaseWithCredential, err := s.databaseRepository.FindWithCredentialByID(ctx, database.ID)
	if err != nil {
		return nil, err
	}
	return s.databaseExecutor.ExplainQuery(ctx, databaseWithCredential, query, analyze)
}

func (s *databaseExecutorService) getAccessibleDatabase(
	ctx context.Context,
	databaseID uint,
	userID uint,
) (*domain.Database, error) {
	return s.databaseService.GetDatabaseByIDAndUserID(ctx, databaseID, userID)
}

func validateSelectQuery(query string) error {
	normalizedQuery := strings.TrimSpace(strings.ToLower(query))
	if normalizedQuery == "" {
		return domain.ErrInvalidDatabaseQuery
	}
	if !strings.HasPrefix(normalizedQuery, "select ") &&
		!strings.HasPrefix(normalizedQuery, "select\n") &&
		!strings.HasPrefix(normalizedQuery, "with ") &&
		!strings.HasPrefix(normalizedQuery, "with\n") {
		return domain.ErrInvalidDatabaseQuery
	}
	if strings.Contains(normalizedQuery, ";") {
		return domain.ErrInvalidDatabaseQuery
	}
	return nil
}
