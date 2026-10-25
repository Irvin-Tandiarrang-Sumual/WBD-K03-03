package database_executor

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type fakeExecutorService struct {
	ports.DatabaseExecutorService
	queryResult *ports.DatabaseQueryResult
	queryID     uint
	queryUserID uint
	query       string
	analyze     bool
}

func (f *fakeExecutorService) ExecuteDatabaseQuery(
	_ context.Context,
	databaseID uint,
	userID uint,
	query string,
) (*ports.DatabaseQueryResult, error) {
	f.queryID = databaseID
	f.queryUserID = userID
	f.query = query
	return f.queryResult, nil
}

func (f *fakeExecutorService) ExplainDatabaseQuery(
	_ context.Context,
	databaseID uint,
	userID uint,
	query string,
	analyze bool,
) (*ports.DatabaseQueryResult, error) {
	f.queryID = databaseID
	f.queryUserID = userID
	f.query = query
	f.analyze = analyze
	return f.queryResult, nil
}

func executorContext(path string, userID *uint) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, path, nil)
	if userID != nil {
		middleware.SetAuthenticatedUserID(context, *userID)
	}
	return context, recorder
}

func TestExecuteDatabaseQueryReturnsResult(t *testing.T) {
	service := &fakeExecutorService{
		queryResult: &ports.DatabaseQueryResult{
			Columns: []string{"id"},
			Rows:    []map[string]any{{"id": 1}},
		},
	}
	handler := NewDatabaseExecutorHandler(service)
	userID := uint(7)
	context, recorder := executorContext("/databases/3/query", &userID)
	context.Params = gin.Params{{Key: "database_id", Value: "3"}}
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/databases/3/query",
		bytes.NewBufferString(`{"query":"SELECT * FROM users"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	handler.ExecuteDatabaseQuery(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if service.queryID != 3 || service.queryUserID != 7 ||
		service.query != "SELECT * FROM users" {
		t.Fatalf("unexpected service call: %+v", service)
	}
}

func TestExecuteDatabaseQueryRejectsMissingUser(t *testing.T) {
	handler := NewDatabaseExecutorHandler(&fakeExecutorService{})
	context, recorder := executorContext("/databases/3/query", nil)
	context.Params = gin.Params{{Key: "database_id", Value: "3"}}
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/databases/3/query",
		bytes.NewBufferString(`{"query":"SELECT 1"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	handler.ExecuteDatabaseQuery(context)

	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotImplemented)
	}
}

func TestExplainDatabaseQueryParsesAnalyzer(t *testing.T) {
	service := &fakeExecutorService{
		queryResult: &ports.DatabaseQueryResult{},
	}
	handler := NewDatabaseExecutorHandler(service)
	userID := uint(2)
	context, recorder := executorContext(
		"/databases/4/explain?analyzer=true",
		&userID,
	)
	context.Params = gin.Params{{Key: "database_id", Value: "4"}}
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/databases/4/explain?analyzer=true",
		bytes.NewBufferString(`{"query":"SELECT 1"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	handler.ExplainDatabaseQuery(context)

	if recorder.Code != http.StatusOK || !service.analyze {
		t.Fatalf("status=%d analyze=%v", recorder.Code, service.analyze)
	}
}

func TestExecutorErrorResponseIsJSONEnvelope(t *testing.T) {
	handler := NewDatabaseExecutorHandler(&fakeExecutorService{})
	context, recorder := executorContext("/databases/invalid/query", nil)
	context.Params = gin.Params{{Key: "database_id", Value: "invalid"}}

	handler.ExecuteDatabaseQuery(context)

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if recorder.Code != http.StatusBadRequest || body["success"] != false {
		t.Fatalf("unexpected response: status=%d body=%v", recorder.Code, body)
	}
}
