package response

import (
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

func TestSuccessAndErrorResponses(t *testing.T) {
	success := Success("loaded", []string{"one"})
	if !success.Success || success.Message != "loaded" || len(success.Data) != 1 {
		t.Fatalf("unexpected success response: %+v", success)
	}

	failure := Error("invalid input")
	if failure.Success || failure.Message != "invalid input" || failure.Data != nil {
		t.Fatalf("unexpected error response: %+v", failure)
	}
}

func TestNewPagedResponseCalculatesMetadata(t *testing.T) {
	result := NewPagedResponse(
		200,
		"success",
		[]string{"a", "b"},
		pagination.NewPaginationParams(2, 2),
		5,
	)

	if result.Code != 200 || result.Status != "success" {
		t.Fatalf("unexpected response status: %+v", result)
	}
	if result.Paging.CurrentPage != 2 ||
		result.Paging.Limit != 2 ||
		result.Paging.TotalPage != 3 ||
		result.Paging.TotalData != 5 {
		t.Fatalf("unexpected paging metadata: %+v", result.Paging)
	}
}
