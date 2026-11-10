package response_test

import (
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
)

func TestSuccess_PayloadProvided_ReturnsSuccessResponse(t *testing.T) {
	// Arrange
	message := "data retrieved successfully"
	data := []string{"item1", "item2"}

	// Act
	res := response.Success(message, data)

	// Assert
	if !res.Success {
		t.Fatal("expected Success to be true")
	}
	if res.Message != message {
		t.Fatalf("expected message %q, got %q", message, res.Message)
	}
	if len(res.Data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Data))
	}
}

func TestError_MessageProvided_ReturnsFailureResponse(t *testing.T) {
	// Arrange
	errorMessage := "invalid resource identifier"

	// Act
	res := response.Error(errorMessage)

	// Assert
	if res.Success {
		t.Fatal("expected Success to be false")
	}
	if res.Message != errorMessage {
		t.Fatalf("expected message %q, got %q", errorMessage, res.Message)
	}
	if res.Data != nil {
		t.Fatalf("expected nil data, got %+v", res.Data)
	}
}

func TestNewPagedResponse_DatasetAndPaginationParams_CalculatesPagingMetadata(t *testing.T) {
	// Arrange
	code := 200
	status := "success"
	data := []string{"row1", "row2"}
	params := pagination.NewPaginationParams(2, 2)
	totalData := int64(5) // 5 items with limit 2 -> 3 pages

	// Act
	res := response.NewPagedResponse(code, status, data, params, totalData)

	// Assert
	if res.Code != 200 || res.Status != "success" {
		t.Fatalf("unexpected code/status: code=%d status=%s", res.Code, res.Status)
	}
	if res.Paging.CurrentPage != 2 {
		t.Fatalf("expected CurrentPage=2, got %d", res.Paging.CurrentPage)
	}
	if res.Paging.Limit != 2 {
		t.Fatalf("expected Limit=2, got %d", res.Paging.Limit)
	}
	if res.Paging.TotalPage != 3 {
		t.Fatalf("expected TotalPage=3, got %d", res.Paging.TotalPage)
	}
	if res.Paging.TotalData != 5 {
		t.Fatalf("expected TotalData=5, got %d", res.Paging.TotalData)
	}
}
