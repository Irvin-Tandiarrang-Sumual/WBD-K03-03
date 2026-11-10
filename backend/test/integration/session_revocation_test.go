//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func TestIntegrationAccessChangeRevokesExistingSession(t *testing.T) {
	stack := newIntegrationStack(t, domain.AccessViewOnly)
	ctx := context.Background()

	session, err := stack.sessionService.CreateSession(ctx, stack.user.ID)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}
	if _, err := stack.sessionService.Resolve(ctx, session.OpaqueToken); err != nil {
		t.Fatalf("session should initially resolve: %v", err)
	}

	newAccesses := []domain.DatabaseAccess{{
		DatabaseID: stack.target.ID,
		AccessType: domain.AccessViewEdit,
	}}
	updated, err := stack.groupService.UpdateGroup(ctx, stack.group.ID, domain.UpdateGroupInput{DatabaseAccesses: &newAccesses})
	if err != nil {
		t.Fatalf("Update group access: %v", err)
	}
	if len(updated.DatabaseAccesses) != 1 || updated.DatabaseAccesses[0].AccessType != domain.AccessViewEdit {
		t.Fatalf("unexpected updated group=%+v", updated)
	}

	_, err = stack.sessionService.Resolve(ctx, session.OpaqueToken)
	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("old session error=%v, want ErrInvalidSession", err)
	}
}

func TestIntegrationRoleChangeRevokesExistingSession(t *testing.T) {
	stack := newIntegrationStack(t, domain.AccessViewOnly)
	ctx := context.Background()

	session, err := stack.sessionService.CreateSession(ctx, stack.user.ID)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := stack.userService.UpdateUserRole(ctx, stack.user.ID, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("SetRole() error=%v", err)
	}
	if updated.Role != domain.RoleAdmin {
		t.Fatalf("role=%q", updated.Role)
	}
	_, err = stack.sessionService.Resolve(ctx, session.OpaqueToken)
	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("old session error=%v, want ErrInvalidSession", err)
	}
}
