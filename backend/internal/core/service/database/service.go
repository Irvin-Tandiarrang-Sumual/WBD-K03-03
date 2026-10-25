package database

import (
	"context"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type databaseService struct {
	databaseRepository       ports.DatabaseRepository
	databaseConnectionTester ports.DatabaseConnectionTester
	groupRepository          ports.GroupRepository
	sessionService           ports.SessionService
	databasePoolInvalidator  ports.DatabasePoolInvalidator
}

var _ ports.DatabaseService = (*databaseService)(nil)

func NewDatabaseService(
	databaseRepository ports.DatabaseRepository,
	databaseConnectionTester ports.DatabaseConnectionTester,
	groupRepository ports.GroupRepository,
	sessionService ports.SessionService,
	databasePoolInvalidator ports.DatabasePoolInvalidator,
) ports.DatabaseService {
	return &databaseService{
		databaseRepository:       databaseRepository,
		databaseConnectionTester: databaseConnectionTester,
		groupRepository:          groupRepository,
		sessionService:           sessionService,
		databasePoolInvalidator:  databasePoolInvalidator,
	}
}

func (s *databaseService) GetDatabaseByID(ctx context.Context, databaseID uint) (*domain.Database, error) {
	return s.databaseRepository.FindById(ctx, databaseID)
}

func (s *databaseService) GetAllDatabases(ctx context.Context, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	return s.databaseRepository.FindAll(ctx, params)
}

func (s *databaseService) GetDatabaseHosts(ctx context.Context) ([]domain.DatabaseHost, error) {
	if s.databaseRepository == nil {
		return []domain.DatabaseHost{}, nil
	}
	return s.databaseRepository.GetDistinctHosts(ctx)
}

func (s *databaseService) CreateDatabase(ctx context.Context, databaseInput *ports.DatabaseInput) (*domain.Database, error) {
	if databaseInput == nil {
		return nil, domain.ErrInvalidDatabaseInput
	}

	database := domain.Database{
		Port:     databaseInput.Port,
		Host:     strings.TrimSpace(databaseInput.Host),
		User:     strings.TrimSpace(databaseInput.User),
		Password: databaseInput.Password,
		Type:     strings.ToLower(strings.TrimSpace(databaseInput.Type)),
		Name:     strings.TrimSpace(databaseInput.Name),
	}
	if database.Type == "" {
		database.Type = "postgresql"
	}
	if database.Port == 0 || database.Port > 65535 || database.Host == "" ||
		database.User == "" || database.Name == "" || database.Type != "postgresql" && database.Type != "mysql" {
		return nil, domain.ErrInvalidInput
	}

	// If password is not provided, attempt to reuse credential from existing database on that host & port
	if database.Password == "" && s.databaseRepository != nil {
		if pwd, err := s.databaseRepository.FindCredentialByHostAndPort(ctx, database.Host, database.Port, database.User); err == nil && pwd != "" {
			database.Password = pwd
		}
	}

	if s.databaseConnectionTester != nil {
		if err := s.databaseConnectionTester.Test(ctx, database); err != nil {
			return nil, err
		}
	}

	return s.databaseRepository.CreateDatabase(ctx, &database)
}

func (s *databaseService) DeleteDatabase(ctx context.Context, databaseID uint) error {
	if _, err := s.databaseRepository.FindById(ctx, databaseID); err != nil {
		return err
	}

	if s.groupRepository != nil {
		leavesEmptyGroup, err := s.groupRepository.HasGroupWithOnlyDatabase(ctx, databaseID)
		if err != nil {
			return err
		}
		if leavesEmptyGroup {
			return domain.ErrDatabaseInUse
		}
	}

	var userIDs []uint
	if s.groupRepository != nil {
		var err error
		userIDs, err = s.groupRepository.ListUserIDsByDatabase(ctx, databaseID)
		if err != nil {
			return err
		}
	}
	if s.databasePoolInvalidator != nil {
		if err := s.databasePoolInvalidator.Invalidate(databaseID); err != nil {
			return err
		}
	}
	if err := s.databaseRepository.Delete(ctx, databaseID); err != nil {
		return err
	}
	if s.sessionService != nil {
		for _, userID := range userIDs {
			if err := s.sessionService.RevokeUser(ctx, userID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *databaseService) GetDatabasesByUserID(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	return s.databaseRepository.FindAllByUserID(ctx, userID, params)
}

func (s *databaseService) GetDatabaseByIDAndUserID(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error) {
	return s.databaseRepository.FindByIDAndUserID(ctx, databaseID, userID)
}
