//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/security"
	databaserepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/database"
	grouprepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/group"
	sessionrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/session"
	userrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	models "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
)

func TestIntegrationEffectiveAccessUsesMaximumAcrossGroups(t *testing.T) {
	db := openTestDB(t)

	seedTestDB(t, db)

	ctx := context.Background()

	var user models.User

	if err := db.
		Where(
			"username = ?",
			"user",
		).
		First(&user).Error; err != nil {
		t.Fatal(err)
	}

	var database models.Database

	if err := db.
		Where(
			"name = ?",
			"db1_pg",
		).
		First(&database).Error; err != nil {
		t.Fatal(err)
	}

	editGroup := models.Group{
		Name: "editors",
	}

	if err := db.Create(
		&editGroup,
	).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(
		&models.GroupDatabase{
			GroupID:    editGroup.ID,
			DatabaseID: database.ID,
			AccessType: string(
				domain.AccessViewEdit,
			),
		},
	).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(
		&models.UserGroup{
			UserID:  user.ID,
			GroupID: editGroup.ID,
		},
	).Error; err != nil {
		t.Fatal(err)
	}

	cipher, err := security.NewAESGCMCipher(
		[]byte(
			"12345678901234567890123456789012",
		),
	)

	if err != nil {
		t.Fatal(err)
	}

	groupRepo := 	grouprepository.NewGroupRepository(db)

	userRepo := 	userrepository.NewUserRepository(db)

	databaseRepo := 	databaserepository.NewDatabaseRepository(
		db,
		cipher,
	)

	sessionService := 	sessionservice.NewSessionService(
		sessionrepository.NewSessionRepository(db),
	)

	service := 	groupservice.NewGroupService(
		groupRepo,
		userRepo,
		databaseRepo,
		sessionService,
	)

	access, err := service.EffectiveDatabaseAccess(
		ctx,
		user.ID,
		database.ID,
	)

	if err != nil {
		t.Fatalf(
			"EffectiveDatabaseAccess() error = %v",
			err,
		)
	}

	if access != domain.AccessViewEdit {
		t.Fatalf(
			"access = %q, want %q",
			access,
			domain.AccessViewEdit,
		)
	}
}

func TestIntegrationGroupFindByIDLoadsAllDatabaseAccesses(t *testing.T) {
	db := openTestDB(t)

	seedTestDB(t, db)

	var adminGroup models.Group

	if err := db.
		Where(
			"name = ?",
			"admin",
		).
		First(&adminGroup).Error; err != nil {
		t.Fatal(err)
	}

	groupRepo := 	grouprepository.NewGroupRepository(db)

	group, err := groupRepo.FindByID(
		context.Background(),
		adminGroup.ID,
	)

	if err != nil {
		t.Fatalf(
			"FindByID() error = %v",
			err,
		)
	}

	if len(group.DatabaseAccesses) != 2 {
		t.Fatalf(
			"database access count = %d, want 2: %+v",
			len(group.DatabaseAccesses),
			group.DatabaseAccesses,
		)
	}
}

func TestIntegrationDatabaseCredentialIsEncryptedAtRest(t *testing.T) {
	db := openTestDB(t)

	ctx := context.Background()

	cipher, err := security.NewAESGCMCipher(
		[]byte(
			"12345678901234567890123456789012",
		),
	)

	if err != nil {
		t.Fatal(err)
	}

	repo := 	databaserepository.NewDatabaseRepository(
		db,
		cipher,
	)

	created, err := repo.CreateDatabase(
		ctx,
		&domain.Database{
			Port:     15432,
			Host:     "localhost",
			User:     "postgres",
			Password: "very-secret",
			Type:     "postgresql",
			Name:     "encrypted_db",
		},
	)

	if err != nil {
		t.Fatalf(
			"Create() error = %v",
			err,
		)
	}

	var stored models.Database

	if err := db.First(
		&stored,
		created.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Password == "very-secret" {
		t.Fatal(
			"database credential was stored as plaintext",
		)
	}

	if stored.Password == "" {
		t.Fatal(
			"encrypted credential is empty",
		)
	}

	withCredential, err :=
		repo.FindWithCredentialByID(
			ctx,
			created.ID,
		)

	if err != nil {
		t.Fatalf(
			"FindWithCredentialByID() error = %v",
			err,
		)
	}

	if withCredential.Password !=
		"very-secret" {
		t.Fatalf(
			"decrypted password = %q",
			withCredential.Password,
		)
	}
}