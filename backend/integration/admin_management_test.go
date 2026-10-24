package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/security"
	databaserepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/database"
	grouprepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/group"
	sessionrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/session"
	userrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"
	models "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
)

func TestIntegrationUserAndGroupManagementPersists(t *testing.T) {
	db := openTestDB(t)

	seedTestDB(t, db)

	ctx := context.Background()

	cipher, err := security.NewAESGCMCipher(
		[]byte(
			"12345678901234567890123456789012",
		),
	)

	if err != nil {
		t.Fatal(err)
	}

	userRepo := 	userrepository.NewUserRepository(db)

	groupRepo := 	grouprepository.NewGroupRepository(db)

	databaseRepo := 	databaserepository.NewDatabaseRepository(
		db,
		cipher,
	)

	sessionService := 	sessionservice.NewSessionService(
		sessionrepository.NewSessionRepository(db),
	)

	userService := 	userservice.NewUserService(
		userRepo,
		groupRepo,
		sessionService,
	)

	groupService := 	groupservice.NewGroupService(
		groupRepo,
		userRepo,
		databaseRepo,
		sessionService,
	)

	var targetDB models.Database

	if err := db.
		Where(
			"name = ?",
			"db1_pg",
		).
		First(&targetDB).Error; err != nil {
		t.Fatal(err)
	}

	group, err := groupService.Create(
		ctx,
		domain.CreateGroupInput{
			Name: "qa-team",
			DatabaseAccesses: []domain.DatabaseAccess{
				{
					DatabaseID: targetDB.ID,
					AccessType: domain.AccessViewOnly,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf(
			"Create group: %v",
			err,
		)
	}

	user, err := userService.Create(
		ctx,
		domain.CreateUserInput{
			Username: "integration-user",
			Password: "secret123",
			GroupIDs: []uint{
				group.ID,
			},
		},
	)

	if err != nil {
		t.Fatalf(
			"Create user: %v",
			err,
		)
	}

	if user.Role != domain.RoleUser {
		t.Fatalf(
			"new user role = %q, want user",
			user.Role,
		)
	}

	newName := "integration-user-renamed"

	updated, err := userService.Update(
		ctx,
		user.ID,
		domain.UpdateUserInput{
			Username: &newName,
		},
	)

	if err != nil {
		t.Fatalf(
			"Update user: %v",
			err,
		)
	}

	if updated.Username != newName {
		t.Fatalf(
			"username = %q, want %q",
			updated.Username,
			newName,
		)
	}

	updated, err = userService.SetRole(
		ctx,
		user.ID,
		domain.RoleAdmin,
	)

	if err != nil {
		t.Fatalf(
			"SetRole: %v",
			err,
		)
	}

	if updated.Role != domain.RoleAdmin {
		t.Fatalf(
			"role = %q, want admin",
			updated.Role,
		)
	}

	var adminGroup models.Group

	if err := db.
		Where(
			"name = ?",
			"admin",
		).
		First(&adminGroup).Error; err != nil {
		t.Fatal(err)
	}

	if err := groupService.AddUser(
		ctx,
		adminGroup.ID,
		user.ID,
	); err != nil {
		t.Fatalf(
			"AddUser: %v",
			err,
		)
	}

	count, err := groupRepo.CountUserGroups(
		ctx,
		user.ID,
	)

	if err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Fatalf(
			"group count after AddUser = %d, want 2",
			count,
		)
	}

	if err := groupService.RemoveUser(
		ctx,
		adminGroup.ID,
		user.ID,
	); err != nil {
		t.Fatalf(
			"RemoveUser: %v",
			err,
		)
	}

	count, err = groupRepo.CountUserGroups(
		ctx,
		user.ID,
	)

	if err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf(
			"group count after RemoveUser = %d, want 1",
			count,
		)
	}

	if err := userService.Delete(
		ctx,
		user.ID,
	); err != nil {
		t.Fatalf(
			"Delete user: %v",
			err,
		)
	}

	_, err = userRepo.FindByID(
		ctx,
		user.ID,
	)

	if !errors.Is(
		err,
		domain.ErrUserNotFound,
	) {
		t.Fatalf(
			"FindByID after delete error = %v, want ErrUserNotFound",
			err,
		)
	}
}