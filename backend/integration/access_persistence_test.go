//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/security"
	models "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	databaserepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/database"
	grouprepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/group"
	sessionrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/session"
	userrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
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

	groupRepo := grouprepository.NewGroupRepository(db)

	userRepo := userrepository.NewUserRepository(db)

	databaseRepo := databaserepository.NewDatabaseRepository(
		db,
		cipher,
	)

	sessionService := sessionservice.NewSessionService(
		sessionrepository.NewSessionRepository(db),
	)

	service := groupservice.NewGroupService(
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

	groupRepo := grouprepository.NewGroupRepository(db)

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

	repo := databaserepository.NewDatabaseRepository(
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

func TestIntegrationMultiGroupPerDatabaseAccessIsolation(t *testing.T) {
	db := openTestDB(t)
	seedTestDB(t, db)
	ctx := context.Background()

	cipher, err := security.NewAESGCMCipher([]byte(integrationCredentialKey))
	if err != nil {
		t.Fatal(err)
	}
	databaseRepo := databaserepository.NewDatabaseRepository(db, cipher)
	groupRepo := grouprepository.NewGroupRepository(db)
	userRepo := userrepository.NewUserRepository(db)
	sessionRepo := sessionrepository.NewSessionRepository(db)
	sessionService := sessionservice.NewSessionService(sessionRepo)
	groupService := groupservice.NewGroupService(groupRepo, userRepo, databaseRepo, sessionService)

	// Create 2 separate target databases in PostgreSQL
	db1, err := databaseRepo.CreateDatabase(ctx, &domain.Database{
		Host: "localhost", Port: 5432, User: "postgres", Name: "app_db1", Type: "postgresql",
	})
	if err != nil {
		t.Fatal(err)
	}
	db2, err := databaseRepo.CreateDatabase(ctx, &domain.Database{
		Host: "localhost", Port: 5432, User: "postgres", Name: "app_db2", Type: "postgresql",
	})
	if err != nil {
		t.Fatal(err)
	}
	db3, err := databaseRepo.CreateDatabase(ctx, &domain.Database{
		Host: "localhost", Port: 5432, User: "postgres", Name: "app_db3", Type: "postgresql",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create group 1 with View Only access to DB 1
	g1, err := groupService.Create(ctx, domain.CreateGroupInput{
		Name: "viewers_group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: db1.ID, AccessType: domain.AccessViewOnly},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create group 2 with View + Edit access to DB 2
	g2, err := groupService.Create(ctx, domain.CreateGroupInput{
		Name: "editors_group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: db2.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Fetch test user and add to both groups
	var user models.User
	if err := db.Where("username = ?", "user").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := groupService.AddUser(ctx, g1.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	if err := groupService.AddUser(ctx, g2.ID, user.ID); err != nil {
		t.Fatal(err)
	}

	// 1. Effective access for DB 1 must be View Only
	acc1, err := groupService.EffectiveDatabaseAccess(ctx, user.ID, db1.ID)
	if err != nil || acc1 != domain.AccessViewOnly {
		t.Fatalf("db1 effective access = %q err = %v, want view_only", acc1, err)
	}

	// 2. Effective access for DB 2 must be View + Edit
	acc2, err := groupService.EffectiveDatabaseAccess(ctx, user.ID, db2.ID)
	if err != nil || acc2 != domain.AccessViewEdit {
		t.Fatalf("db2 effective access = %q err = %v, want view_edit", acc2, err)
	}

	// 3. Effective access for DB 3 must be denied (user in no group that grants DB 3)
	_, err = groupService.EffectiveDatabaseAccess(ctx, user.ID, db3.ID)
	if err != domain.ErrDatabaseAccessDenied {
		t.Fatalf("db3 access err = %v, want ErrDatabaseAccessDenied", err)
	}
}
