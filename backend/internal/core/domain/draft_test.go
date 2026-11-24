package domain

import (
	"errors"
	"testing"
)

func TestValidate_ValidCreateDraftInput_ReturnsNil(t *testing.T) {
	// Arrange
	input := CreateDraftInput{
		DatabaseID: 1,
		UserID:     2,
		SqlUp:      "ALTER TABLE users ADD COLUMN age INT;",
	}

	// Act
	err := input.Validate()

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidate_EmptyCreateDraftInputFields_ReturnsErrInvalidDraftInput(t *testing.T) {
	testCases := []struct {
		name  string
		input CreateDraftInput
	}{
		{
			name:  "missing database id",
			input: CreateDraftInput{DatabaseID: 0, UserID: 1, SqlUp: "CREATE TABLE t (id INT);"},
		},
		{
			name:  "missing user id",
			input: CreateDraftInput{DatabaseID: 1, UserID: 0, SqlUp: "CREATE TABLE t (id INT);"},
		},
		{
			name:  "empty sql up",
			input: CreateDraftInput{DatabaseID: 1, UserID: 1, SqlUp: "   "},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange & Act
			err := tc.input.Validate()

			// Assert
			if !errors.Is(err, ErrInvalidDraftInput) {
				t.Fatalf("expected ErrInvalidDraftInput, got %v", err)
			}
		})
	}
}

func TestValidate_ValidSubmitApprovalInput_ReturnsNil(t *testing.T) {
	testCases := []struct {
		name   string
		status string
	}{
		{name: "approved status", status: ApprovalStatusApproved},
		{name: "rejected status", status: ApprovalStatusRejected},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			input := SubmitApprovalInput{
				DraftID: 1,
				UserID:  2,
				Status:  tc.status,
			}

			// Act
			err := input.Validate()

			// Assert
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidate_InvalidSubmitApprovalInput_ReturnsErrInvalidDraftInput(t *testing.T) {
	testCases := []struct {
		name  string
		input SubmitApprovalInput
	}{
		{
			name:  "missing draft id",
			input: SubmitApprovalInput{DraftID: 0, UserID: 1, Status: ApprovalStatusApproved},
		},
		{
			name:  "missing user id",
			input: SubmitApprovalInput{DraftID: 1, UserID: 0, Status: ApprovalStatusApproved},
		},
		{
			name:  "invalid status",
			input: SubmitApprovalInput{DraftID: 1, UserID: 1, Status: "invalid_status"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange & Act
			err := tc.input.Validate()

			// Assert
			if !errors.Is(err, ErrInvalidDraftInput) {
				t.Fatalf("expected ErrInvalidDraftInput, got %v", err)
			}
		})
	}
}

func TestIsTerminal_TerminalStatus_ReturnsTrue(t *testing.T) {
	terminalStatuses := []string{DraftStatusRejected, DraftStatusInvalidated}

	for _, status := range terminalStatuses {
		t.Run(status, func(t *testing.T) {
			// Arrange
			d := Draft{Status: status}

			// Act
			res := d.IsTerminal()

			// Assert
			if !res {
				t.Fatalf("expected IsTerminal for %s to be true, got false", status)
			}
		})
	}
}

func TestIsTerminal_NonTerminalStatus_ReturnsFalse(t *testing.T) {
	nonTerminalStatuses := []string{
		DraftStatusPendingApproval,
		DraftStatusApproved,
		DraftStatusExecuting,
		DraftStatusApplied,
		DraftStatusFailed,
		DraftStatusPartiallyReverted,
	}

	for _, status := range nonTerminalStatuses {
		t.Run(status, func(t *testing.T) {
			// Arrange
			d := Draft{Status: status}

			// Act
			res := d.IsTerminal()

			// Assert
			if res {
				t.Fatalf("expected IsTerminal for %s to be false, got true", status)
			}
		})
	}
}
