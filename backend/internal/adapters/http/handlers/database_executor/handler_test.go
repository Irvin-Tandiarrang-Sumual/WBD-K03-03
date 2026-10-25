package database_executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type fakeExecutorService struct {
	ports.DatabaseExecutorService
	queryResult *ports.DatabaseQueryResult
	queryErr    error
	explainErr  error
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
	return f.queryResult, f.queryErr
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
	return f.queryResult, f.explainErr
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
	var responseBody struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, exists := responseBody.Data["rows"]; !exists {
		t.Fatalf("response data does not contain lowercase rows: %s", recorder.Body.String())
	}
	if _, exists := responseBody.Data["Rows"]; exists {
		t.Fatalf("response data contains legacy uppercase Rows: %s", recorder.Body.String())
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

func TestExecuteDatabaseQueryRejectsInvalidJSON(t *testing.T) {
	service := &fakeExecutorService{}
	handler := NewDatabaseExecutorHandler(service)
	userID := uint(7)
	context, recorder := executorContext("/databases/3/query", &userID)
	context.Params = gin.Params{{Key: "database_id", Value: "3"}}
	context.Request = httptest.NewRequest(http.MethodPost, "/databases/3/query", bytes.NewBufferString(`{"query":`))
	context.Request.Header.Set("Content-Type", "application/json")

	handler.ExecuteDatabaseQuery(context)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestExplainDatabaseQueryRejectsInvalidAnalyzer(t *testing.T) {
	service := &fakeExecutorService{}
	handler := NewDatabaseExecutorHandler(service)
	userID := uint(7)
	context, recorder := executorContext("/databases/3/explain?analyzer=maybe", &userID)
	context.Params = gin.Params{{Key: "database_id", Value: "3"}}
	context.Request = httptest.NewRequest(http.MethodPost, "/databases/3/explain?analyzer=maybe", bytes.NewBufferString(`{"query":"SELECT 1"}`))
	context.Request.Header.Set("Content-Type", "application/json")

	handler.ExplainDatabaseQuery(context)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestDatabaseExecutorHandlerMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid query", err: domain.ErrInvalidDatabaseQuery, want: http.StatusBadRequest},
		{name: "database missing", err: domain.ErrDatabaseNotFound, want: http.StatusNotFound},
		{name: "access denied", err: domain.ErrDatabaseAccessDenied, want: http.StatusForbidden},
		{name: "internal", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeExecutorService{queryErr: tt.err}
			handler := NewDatabaseExecutorHandler(service)
			userID := uint(7)
			context, recorder := executorContext("/databases/3/query", &userID)
			context.Params = gin.Params{{Key: "database_id", Value: "3"}}
			context.Request = httptest.NewRequest(http.MethodPost, "/databases/3/query", bytes.NewBufferString(`{"query":"SELECT 1"}`))
			context.Request.Header.Set("Content-Type", "application/json")

			handler.ExecuteDatabaseQuery(context)
			if recorder.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, tt.want, recorder.Body.String())
			}
		})
	}
}

func TestExplainDatabaseQueryMapsServiceError(t *testing.T) {
	service := &fakeExecutorService{explainErr: domain.ErrDatabaseAccessDenied}
	handler := NewDatabaseExecutorHandler(service)
	userID := uint(7)
	context, recorder := executorContext("/databases/3/explain", &userID)
	context.Params = gin.Params{{Key: "database_id", Value: "3"}}
	context.Request = httptest.NewRequest(http.MethodPost, "/databases/3/explain", bytes.NewBufferString(`{"query":"SELECT 1"}`))
	context.Request.Header.Set("Content-Type", "application/json")

	handler.ExplainDatabaseQuery(context)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestExplainDatabaseQueryRejectsInvalidInputs(t *testing.T) {
	handler := NewDatabaseExecutorHandler(&fakeExecutorService{})
	userID := uint(7)

	// Invalid database ID
	ctx1, rec1 := executorContext("/databases/bad/explain", &userID)
	ctx1.Params = gin.Params{{Key: "database_id", Value: "bad"}}
	ctx1.Request = httptest.NewRequest(http.MethodPost, "/databases/bad/explain", bytes.NewBufferString(`{"query":"SELECT 1"}`))
	ctx1.Request.Header.Set("Content-Type", "application/json")
	handler.ExplainDatabaseQuery(ctx1)
	if rec1.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=400", rec1.Code)
	}

	// Missing user context
	ctx2, rec2 := executorContext("/databases/1/explain", nil)
	ctx2.Params = gin.Params{{Key: "database_id", Value: "1"}}
	ctx2.Request = httptest.NewRequest(http.MethodPost, "/databases/1/explain", bytes.NewBufferString(`{"query":"SELECT 1"}`))
	ctx2.Request.Header.Set("Content-Type", "application/json")
	handler.ExplainDatabaseQuery(ctx2)
	if rec2.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d want=501", rec2.Code)
	}

	// Invalid JSON body
	ctx3, rec3 := executorContext("/databases/1/explain", &userID)
	ctx3.Params = gin.Params{{Key: "database_id", Value: "1"}}
	ctx3.Request = httptest.NewRequest(http.MethodPost, "/databases/1/explain", bytes.NewBufferString(`{"query":`))
	ctx3.Request.Header.Set("Content-Type", "application/json")
	handler.ExplainDatabaseQuery(ctx3)
	if rec3.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=400", rec3.Code)
	}
}
