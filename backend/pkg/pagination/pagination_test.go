package pagination_test

import (
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

func TestNewPaginationParams_ZeroOrNegativeInputs_DefaultsToPageOneAndLimitTen(t *testing.T) {
	// Arrange
	zeroPage, zeroLimit := 0, 0
	negPage, negLimit := -5, -10

	// Act
	paramsZero := pagination.NewPaginationParams(zeroPage, zeroLimit)
	paramsNeg := pagination.NewPaginationParams(negPage, negLimit)

	// Assert
	if paramsZero.Page != 1 || paramsZero.Limit != 10 || paramsZero.Offset() != 0 {
		t.Fatalf("unexpected defaults for zero values: %+v, offset=%d", paramsZero, paramsZero.Offset())
	}
	if paramsNeg.Page != 1 || paramsNeg.Limit != 10 || paramsNeg.Offset() != 0 {
		t.Fatalf("unexpected defaults for negative values: %+v, offset=%d", paramsNeg, paramsNeg.Offset())
	}
}

func TestNewPaginationParams_ValidInputs_PreservesValuesAndCalculatesOffset(t *testing.T) {
	// Arrange
	page := 3
	limit := 25
	expectedOffset := 50 // (3 - 1) * 25

	// Act
	params := pagination.NewPaginationParams(page, limit)

	// Assert
	if params.Page != 3 || params.Limit != 25 {
		t.Fatalf("expected page=3, limit=25, got page=%d, limit=%d", params.Page, params.Limit)
	}
	if params.Offset() != expectedOffset {
		t.Fatalf("expected offset=%d, got %d", expectedOffset, params.Offset())
	}
}

func TestNewPaginationParams_LimitExceedsMaximum_CapsLimitToOneHundred(t *testing.T) {
	// Arrange
	page := 2
	exorbitantLimit := 500
	expectedLimit := 100
	expectedOffset := 100 // (2 - 1) * 100

	// Act
	params := pagination.NewPaginationParams(page, exorbitantLimit)

	// Assert
	if params.Limit != expectedLimit {
		t.Fatalf("expected limit capped at %d, got %d", expectedLimit, params.Limit)
	}
	if params.Offset() != expectedOffset {
		t.Fatalf("expected offset=%d, got %d", expectedOffset, params.Offset())
	}
}
