package ginutil_test

import (
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/ginutil"
	"github.com/gin-gonic/gin"
)

func newTestContext(query string, params gin.Params) *gin.Context {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest("GET", "/test?"+query, nil)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = request
	c.Params = params
	return c
}

func TestParseQueryInt_EmptyQueryParam_ReturnsZeroAndNilError(t *testing.T) {
	// Arrange
	ctx := newTestContext("", nil)

	// Act
	val, err := ginutil.ParseQueryInt(ctx, "page")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0 {
		t.Fatalf("expected 0 for empty query, got %d", val)
	}
}

func TestParseQueryInt_ValidIntegerQueryParam_ReturnsParsedInt(t *testing.T) {
	// Arrange
	ctx := newTestContext("page=5", nil)

	// Act
	val, err := ginutil.ParseQueryInt(ctx, "page")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 5 {
		t.Fatalf("expected 5, got %d", val)
	}
}

func TestParsePagination_ValidQueryParams_ReturnsParsedPaginationParams(t *testing.T) {
	// Arrange
	ctx := newTestContext("page=3&limit=25", nil)

	// Act
	params, err := ginutil.ParsePagination(ctx)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Page != 3 || params.Limit != 25 {
		t.Fatalf("unexpected pagination params: %+v, want page=3, limit=25", params)
	}
}

func TestParsePagination_InvalidQueryParameter_ReturnsError(t *testing.T) {
	// Arrange
	ctx := newTestContext("page=invalid", nil)

	// Act
	_, err := ginutil.ParsePagination(ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid page parameter, got nil")
	}
}

func TestParseUintParam_ValidPositiveInteger_ReturnsParsedUint(t *testing.T) {
	// Arrange
	ctx := newTestContext("", gin.Params{{Key: "id", Value: "42"}})

	// Act
	id, err := ginutil.ParseUintParam(ctx, "id")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Fatalf("got id = %d, want 42", id)
	}
}

func TestParseUintParam_ZeroOrNonNumericParam_ReturnsError(t *testing.T) {
	testCases := []string{"0", "invalid", "-5"}
	for _, val := range testCases {
		t.Run("value="+val, func(t *testing.T) {
			// Arrange
			ctx := newTestContext("", gin.Params{{Key: "id", Value: val}})

			// Act
			_, err := ginutil.ParseUintParam(ctx, "id")

			// Assert
			if err == nil {
				t.Fatalf("expected error for parameter value %q, got nil", val)
			}
		})
	}
}
