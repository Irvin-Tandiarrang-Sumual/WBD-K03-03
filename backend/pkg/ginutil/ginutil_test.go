package ginutil

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestContext(query string, params gin.Params) *gin.Context {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest("GET", "/test?"+query, nil)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request
	context.Params = params
	return context
}

func TestParsePagination(t *testing.T) {
	context := newTestContext("page=3&limit=25", nil)

	params, err := ParsePagination(context)
	if err != nil {
		t.Fatalf("ParsePagination returned error: %v", err)
	}
	if params.Page != 3 || params.Limit != 25 {
		t.Fatalf("unexpected pagination params: %+v", params)
	}
}

func TestParsePaginationRejectsInvalidQuery(t *testing.T) {
	context := newTestContext("page=invalid", nil)

	if _, err := ParsePagination(context); err == nil {
		t.Fatal("expected invalid page to return an error")
	}
}

func TestParseUintParam(t *testing.T) {
	context := newTestContext("", gin.Params{{Key: "id", Value: "42"}})

	id, err := ParseUintParam(context, "id")
	if err != nil || id != 42 {
		t.Fatalf("got id=%d err=%v, want id=42", id, err)
	}
}

func TestParseUintParamRejectsZeroAndInvalidValues(t *testing.T) {
	for _, value := range []string{"0", "invalid"} {
		context := newTestContext("", gin.Params{{Key: "id", Value: value}})
		if _, err := ParseUintParam(context, "id"); err == nil {
			t.Fatalf("expected value %q to be rejected", value)
		}
	}
}
