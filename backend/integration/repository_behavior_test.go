//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

func TestIntegrationRepositoriesPersistAndReadRelationships(t *testing.T) {
	stack := newIntegrationStack(t, domain.AccessViewEdit)
	ctx := context.Background()

	listedDatabases, totalDatabases, err := stack.databaseRepo.FindAll(ctx, pagination.NewPaginationParams(1, 10))
	if err != nil || totalDatabases != 1 || len(listedDatabases) != 1 {
		t.Fatalf("FindAll databases=%v total=%d err=%v", listedDatabases, totalDatabases, err)
	}
	if listedDatabases[0].Password != "" {
		t.Fatal("normal database reads must not expose password")
	}
	withCredential, err := stack.databaseRepo.FindWithCredentialByID(ctx, stack.target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if withCredential.Password != stack.target.Password {
		t.Fatalf("decrypted password=%q want=%q", withCredential.Password, stack.target.Password)
	}
	if exists, err := stack.databaseRepo.ExistsAll(ctx, []uint{stack.target.ID, stack.target.ID}); err != nil || !exists {
		t.Fatalf("ExistsAll duplicate IDs exists=%v err=%v", exists, err)
	}
	if exists, err := stack.databaseRepo.ExistsAll(ctx, []uint{stack.target.ID, stack.target.ID + 9999}); err != nil || exists {
		t.Fatalf("ExistsAll missing ID exists=%v err=%v", exists, err)
	}

	groups, totalGroups, err := stack.groupRepo.FindAll(ctx, pagination.NewPaginationParams(1, 10))
	if err != nil || totalGroups != 1 || len(groups) != 1 {
		t.Fatalf("groups=%v total=%d err=%v", groups, totalGroups, err)
	}
	members, err := stack.userGroupRepo.FindAllUserIDsByGroupID(ctx, stack.group.ID)
	if err != nil || len(members) != 1 || members[0] != stack.user.ID {
		t.Fatalf("members=%v err=%v", members, err)
	}
	accesses, err := stack.userGroupRepo.FindAllAccessTypes(ctx, stack.user.ID, stack.target.ID)
	if err != nil || len(accesses) != 1 || accesses[0] != domain.AccessViewEdit {
		t.Fatalf("accesses=%v err=%v", accesses, err)
	}
	count, err := stack.userGroupRepo.CountByUserID(ctx, stack.user.ID)
	if err != nil || count != 1 {
		t.Fatalf("group count=%d err=%v", count, err)
	}
	userIDs, err := stack.userGroupRepo.FindAllUserIDsByDatabaseID(ctx, stack.target.ID)
	if err != nil || len(userIDs) != 1 || userIDs[0] != stack.user.ID {
		t.Fatalf("database users=%v err=%v", userIDs, err)
	}
	leavesEmpty, err := stack.userGroupRepo.HasGroupWithOnlyThisDatabase(ctx, stack.target.ID)
	if err != nil || !leavesEmpty {
		t.Fatalf("HasGroupWithOnlyDatabase=%v err=%v", leavesEmpty, err)
	}

	users, totalUsers, err := stack.userRepo.FindAll(ctx, pagination.NewPaginationParams(1, 10))
	if err != nil || totalUsers != 1 || len(users) != 1 || users[0].Password != "" {
		t.Fatalf("users=%v total=%d err=%v", users, totalUsers, err)
	}
	loginUser, err := stack.userRepo.FindByUsernameWithPassword(ctx, stack.user.Username)
	if err != nil || loginUser.Password == "" {
		t.Fatalf("login user=%+v err=%v", loginUser, err)
	}
}

func TestIntegrationSessionRepositoryLifecycle(t *testing.T) {
	stack := newIntegrationStack(t, domain.AccessViewOnly)
	ctx := context.Background()

	session, err := stack.sessionService.CreateSession(ctx, stack.user.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := stack.sessionRepo.FindByToken(ctx, session.OpaqueToken)
	if err != nil || stored.UserID != stack.user.ID {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
	newLastActive := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	if err := stack.sessionRepo.Touch(ctx, session.ID, newLastActive); err != nil {
		t.Fatal(err)
	}
	stored, err = stack.sessionRepo.FindByToken(ctx, session.OpaqueToken)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LastActive.Before(newLastActive.Add(-time.Millisecond)) || stored.LastActive.After(newLastActive.Add(time.Millisecond)) {
		t.Fatalf("last active=%v want around %v", stored.LastActive, newLastActive)
	}
	if err := stack.sessionRepo.DeleteByToken(ctx, session.OpaqueToken); err != nil {
		t.Fatal(err)
	}
	if _, err := stack.sessionRepo.FindByToken(ctx, session.OpaqueToken); !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("error after delete=%v", err)
	}
	if err := stack.sessionRepo.Touch(ctx, session.ID, time.Now()); !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("touch deleted session error=%v", err)
	}
}

func TestIntegrationMigrationRepositoryLifecycle(t *testing.T) {
	stack := newIntegrationStack(t, domain.AccessViewEdit)
	ctx := context.Background()

	created, err := stack.migrationRepo.Save(ctx, &domain.Migration{
		DatabaseID: stack.target.ID,
		Version:    "1",
		Status:     "success",
		SqlUp:      "CREATE TABLE repository_only(id int)",
		SqlDown:    "DROP TABLE repository_only",
		UserID:     stack.user.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatal("saved migration did not receive ID")
	}
	byID, err := stack.migrationRepo.FindByIDAndDatabaseID(ctx, created.ID, stack.target.ID)
	if err != nil || byID.Version != "1" {
		t.Fatalf("byID=%+v err=%v", byID, err)
	}
	byVersion, err := stack.migrationRepo.FindByVersionAndDatabaseID(ctx, "1", stack.target.ID)
	if err != nil || byVersion.ID != created.ID {
		t.Fatalf("byVersion=%+v err=%v", byVersion, err)
	}
	latest, err := stack.migrationRepo.FindLatestByDatabaseID(ctx, stack.target.ID)
	if err != nil || latest.ID != created.ID {
		t.Fatalf("latest=%+v err=%v", latest, err)
	}
	list, total, err := stack.migrationRepo.FindAllByDatabaseID(ctx, stack.target.ID, pagination.NewPaginationParams(1, 10))
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("list=%v total=%d err=%v", list, total, err)
	}
	if _, err := stack.migrationRepo.Save(ctx, nil); !errors.Is(err, domain.ErrInvalidMigrationInput) {
		t.Fatalf("nil migration error=%v", err)
	}
}
