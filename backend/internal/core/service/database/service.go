package database

import (
	"context"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type Service struct {
	databases ports.DatabaseRepository
	tester    ports.DatabaseConnectionTester
	groups    ports.GroupRepository
	sessions  ports.SessionService
	pools     ports.DatabasePoolInvalidator
}

var _ ports.DatabaseService = (*Service)(nil)

func NewService(databases ports.DatabaseRepository, tester ports.DatabaseConnectionTester, groups ports.GroupRepository, sessions ports.SessionService, pools ports.DatabasePoolInvalidator) *Service {
	return &Service{
		databases: databases,
		tester:    tester,
		groups:    groups,
		sessions:  sessions,
		pools: pools,
	}
}

func (s *Service) List(ctx context.Context) ([]domain.Database, error) {
	return s.databases.List(ctx)
}

func (s *Service) Get(ctx context.Context, id uint) (domain.Database, error) {
	return s.databases.FindByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, input domain.CreateDatabaseInput) (domain.Database, error) {
	database := domain.Database{
		Port:     input.Port,
		Host:     strings.TrimSpace(input.Host),
		User:     strings.TrimSpace(input.User),
		Password: input.Password,
		Type:     strings.ToLower(strings.TrimSpace(input.Type)),
		Name:     strings.TrimSpace(input.Name),
	}

	if database.Type == "" {
		database.Type = "postgresql"
	}

	if err := validateDatabase(database); err != nil {
		return domain.Database{}, err
	}

	if err := s.tester.Test(ctx, database); err != nil {
		return domain.Database{}, err
	}

	return s.databases.Create(ctx, database)
}

func validateDatabase(database domain.Database) error {
	if database.Port == 0 || database.Port > 65535 || database.Host == "" || database.User == "" || database.Name == "" {
		return domain.ErrInvalidInput
	}

	if database.Type != "postgresql" {
		return domain.ErrInvalidInput
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id uint) error {
	if _, err := s.databases.FindByID(ctx, id); err != nil {
		return err
	}

	leavesEmptyGroup, err := s.groups.HasGroupWithOnlyDatabase(ctx, id)
	if err != nil {
		return err
	}

	if leavesEmptyGroup {
		return domain.ErrDatabaseInUse
	}

	userIDs, err := s.groups.ListUserIDsByDatabase(ctx, id)
	if err != nil {
		return err
	}

	if err := s.pools.Invalidate(id); err != nil {
		return err
	}

	if err := s.databases.Delete(ctx, id); err != nil {
		return err
	}

	for _, userID := range userIDs {
		if err := s.sessions.RevokeUser(ctx, userID); err != nil {
			return err
		}
	}

	return nil
}

