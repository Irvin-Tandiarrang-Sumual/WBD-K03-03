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
	userGroupsRepository     ports.UserGroupRepository
	sessionService           ports.SessionService
	databasePoolInvalidator  ports.DatabasePoolInvalidator
}

var _ ports.DatabaseService = (*databaseService)(nil)

func NewDatabaseService(
	databaseRepository ports.DatabaseRepository,
	databaseConnectionTester ports.DatabaseConnectionTester,
	userGroupsRepository ports.UserGroupRepository,
	sessionService ports.SessionService,
	databasePoolInvalidator ports.DatabasePoolInvalidator,
) ports.DatabaseService {
	return &databaseService{
		databaseRepository:       databaseRepository,
		databaseConnectionTester: databaseConnectionTester,
		userGroupsRepository:     userGroupsRepository,
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

	databaseType := strings.ToLower(strings.TrimSpace(databaseInput.Type))
	if databaseType == "" {
		databaseType = domain.DatabaseTypePostgreSQL
	}

	database := domain.Database{
		Port:     databaseInput.Port,
		Host:     strings.TrimSpace(databaseInput.Host),
		User:     strings.TrimSpace(databaseInput.User),
		Password: databaseInput.Password,
		Type:     databaseType,
		Name:     strings.TrimSpace(databaseInput.Name),
	}

	if err := database.Validate(); err != nil {
		return nil, err
	}

	s.enrichMissingCredential(ctx, &database)

	if s.databaseConnectionTester != nil {
		if err := s.databaseConnectionTester.Test(ctx, database); err != nil {
			return nil, err
		}
	}

	created, err := s.databaseRepository.Save(ctx, &database)
	if err != nil {
		return nil, err
	}

	if created != nil {
		created.Password = ""
	}

	return created, nil
}

func (s *databaseService) DeleteDatabase(ctx context.Context, databaseID uint) error {
	if _, err := s.databaseRepository.FindById(ctx, databaseID); err != nil {
		return err
	}

	if err := s.ensureDatabaseNotInUse(ctx, databaseID); err != nil {
		return err
	}

	var userIDs []uint
	if s.userGroupsRepository != nil {
		var err error
		userIDs, err = s.userGroupsRepository.FindAllUserIDsByDatabaseID(ctx, databaseID)
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

	if s.sessionService != nil && len(userIDs) > 0 {
		return s.sessionService.RevokeUsers(ctx, userIDs)
	}

	return nil
}

func (s *databaseService) GetDatabasesByUserID(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	databases, total, err := s.databaseRepository.FindAllByUserID(ctx, userID, params)
	if err != nil {
		return nil, 0, err
	}

	for _, db := range databases {
		s.populateAccessType(ctx, userID, db)
	}

	return databases, total, nil
}

func (s *databaseService) GetDatabaseByIDAndUserID(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error) {
	db, err := s.databaseRepository.FindByIDAndUserID(ctx, databaseID, userID)
	if err != nil {
		return nil, err
	}

	s.populateAccessType(ctx, userID, db)

	return db, nil
}

func (s *databaseService) ensureDatabaseNotInUse(ctx context.Context, databaseID uint) error {
	if s.userGroupsRepository == nil {
		return nil
	}
	leavesEmptyGroup, err := s.userGroupsRepository.HasGroupWithOnlyThisDatabase(ctx, databaseID)
	if err != nil {
		return err
	}
	if leavesEmptyGroup {
		return domain.ErrDatabaseInUse
	}
	return nil
}

func (s *databaseService) enrichMissingCredential(ctx context.Context, database *domain.Database) {
	if database.Password != "" || s.databaseRepository == nil {
		return
	}
	pwd, err := s.databaseRepository.FindCredentialByHostAndPort(ctx, database.Host, database.Port, database.User)
	if err == nil && pwd != "" {
		database.Password = pwd
	}
}

func (s *databaseService) populateAccessType(ctx context.Context, userID uint, db *domain.Database) {
	if db == nil || s.userGroupsRepository == nil {
		return
	}
	accessTypes, err := s.userGroupsRepository.FindAllAccessTypes(ctx, userID, db.ID)
	if err != nil {
		return
	}
	for _, access := range accessTypes {
		if access == domain.AccessViewEdit {
			db.AccessType = domain.AccessViewEdit
			return
		}
		if access == domain.AccessViewOnly {
			db.AccessType = domain.AccessViewOnly
		}
	}
}
