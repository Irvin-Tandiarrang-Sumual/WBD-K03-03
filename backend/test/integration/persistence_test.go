//go:build integration

package integration_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func TestPersistence_GivenRegisteredTargetDatabase_WhenPasswordStored_ThenEncryptedWithAESGCMInPostgreSQL(t *testing.T) {
	// Arrange
	stack := setupTestStack(t)
	ctx := context.Background()
	plainPassword := "super-secret-target-db-password"

	targetInput := domain.Database{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "target_user",
		Password: plainPassword,
		Type:     "postgresql",
		Name:     "company_payroll",
	}

	// Act
	createdDB, err := stack.databaseRepo.Save(ctx, &targetInput)
	if err != nil {
		t.Fatalf("failed to create database via repository: %v", err)
	}

	// Assert - Direct raw SQL query to verify the column in PostgreSQL is encrypted ciphertext
	var rawRecord model.Database
	err = stack.db.Raw("SELECT id, name, password FROM databases WHERE id = ?", createdDB.ID).Scan(&rawRecord).Error
	if err != nil {
		t.Fatalf("failed to query raw database record: %v", err)
	}

	if rawRecord.Password == plainPassword {
		t.Fatalf("security violation: target database password was stored in plaintext!")
	}
	if !strings.HasPrefix(rawRecord.Password, "v1:") && len(rawRecord.Password) < 20 {
		t.Fatalf("expected encrypted ciphertext format, got %q", rawRecord.Password)
	}

	// Assert - Reading back via repository transparently decrypts using AES-GCM
	retrievedDB, err := stack.databaseRepo.FindWithCredentialByID(ctx, createdDB.ID)
	if err != nil {
		t.Fatalf("failed to retrieve database: %v", err)
	}
	if retrievedDB.Password != plainPassword {
		t.Fatalf("expected decrypted password %q, got %q", plainPassword, retrievedDB.Password)
	}
}

func TestPersistence_GivenExistingUserAndGroup_WhenUserAssignedToGroup_ThenCascadeIntegrityMaintainedOnDeletion(t *testing.T) {
	// Arrange
	stack := setupTestStack(t)
	ctx := context.Background()

	group, err := stack.groupRepo.Save(ctx, domain.Group{Name: "engineering-team"})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user, err := stack.userRepo.Save(ctx, domain.User{
		Username: "engineer_bob",
		Password: "hashed_dummy_password",
		Role:     domain.RoleUser,
	}, []uint{group.ID})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	secondGroup, err := stack.groupRepo.Save(ctx, domain.Group{Name: "devops-team"})
	if err != nil {
		t.Fatalf("failed to create second group: %v", err)
	}

	err = stack.userGroupRepo.AddUser(ctx, secondGroup.ID, user.ID)
	if err != nil {
		t.Fatalf("failed to add user to second group: %v", err)
	}

	// Act - Delete user from platform DB
	err = stack.userRepo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Assert - PostgreSQL foreign key relation in user_groups should be cleaned up
	count, err := stack.userGroupRepo.CountByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to count groups: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 user_group associations after user deletion, got %d", count)
	}
}

func TestPersistence_GivenActiveSession_WhenUserDeleted_ThenSessionCascadesInPlatformDB(t *testing.T) {
	// Arrange
	stack := setupTestStack(t)
	ctx := context.Background()

	group, err := stack.groupRepo.Save(ctx, domain.Group{Name: "session-team"})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user, err := stack.userRepo.Save(ctx, domain.User{
		Username: "alice_session_test",
		Password: "hashed_dummy_password",
		Role:     domain.RoleUser,
	}, []uint{group.ID})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	createdSession, err := stack.sessionService.CreateSession(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Act - Delete the user
	err = stack.userRepo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Assert - Session should no longer be resolvable
	_, err = stack.sessionRepo.FindByToken(ctx, createdSession.OpaqueToken)
	if err == nil {
		t.Fatalf("expected error finding session for deleted user, got nil")
	}
}

func TestPersistence_GivenDuplicateUsername_WhenInsertAttempted_ThenPostgreSQLUniqueConstraintViolated(t *testing.T) {
	// Arrange
	stack := setupTestStack(t)
	ctx := context.Background()

	group, err := stack.groupRepo.Save(ctx, domain.Group{Name: "unique-team"})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	firstUser := domain.User{
		Username: "unique_charlie",
		Password: "password_hash",
		Role:     domain.RoleUser,
	}
	_, err = stack.userRepo.Save(ctx, firstUser, []uint{group.ID})
	if err != nil {
		t.Fatalf("failed to insert initial user: %v", err)
	}

	// Act - Attempt to insert second user with identical username
	secondUser := domain.User{
		Username: "unique_charlie",
		Password: "different_password_hash",
		Role:     domain.RoleUser,
	}
	_, err = stack.userRepo.Save(ctx, secondUser, []uint{group.ID})

	// Assert - Must return ErrUsernameTaken or unique constraint error
	if err == nil {
		t.Fatalf("expected unique constraint violation error on duplicate username, got nil")
	}
	if err != domain.ErrUsernameTaken && !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Fatalf("expected ErrUsernameTaken or unique error, got %v", err)
	}
}
