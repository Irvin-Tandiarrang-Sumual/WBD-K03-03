//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	authhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/auth"
	databasehandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database"
	databaseexecutorhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database_executor"
	grouphandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/group"
	healthhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/health"
	migrationhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/migration"
	userhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/router"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	authservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/auth"
	"github.com/gin-gonic/gin"
)

func fullIntegrationRouter(t *testing.T, stack *integrationStack) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	authService := authservice.NewAuthService(stack.userRepo, stack.sessionService)
	engine := gin.New()
	router.SetupRouter(
		engine,
		authhandler.NewAuthHandler(authService, false),
		userhandler.NewUserHandler(stack.userService),
		grouphandler.NewGroupHandler(stack.groupService),
		databasehandler.NewDatabaseHandler(stack.databaseService),
		databaseexecutorhandler.NewDatabaseExecutorHandler(stack.executorService),
		migrationhandler.NewMigrationHandler(stack.migrationService),
		healthhandler.NewHealthHandler(nil),
		stack.sessionService,
		stack.userService,
		stack.groupService,
	)
	return engine
}

func requestJSON(t *testing.T, handler http.Handler, method string, path string, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var requestBody *bytes.Reader
	if body == "" {
		requestBody = bytes.NewReader(nil)
	} else {
		requestBody = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, requestBody)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func loginFullRouter(t *testing.T, handler http.Handler, username string, password string) *http.Cookie {
	t.Helper()
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		t.Fatal(err)
	}
	response := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/login", string(body), nil)
	if response.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", response.Code, response.Body.String())
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == middleware.SessionCookieName {
			return cookie
		}
	}
	t.Fatal("login did not return session cookie")
	return nil
}

func responseID(t *testing.T, response *httptest.ResponseRecorder) uint {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response %q: %v", response.Body.String(), err)
	}
	value, ok := body["id"].(float64)
	if !ok || value <= 0 {
		t.Fatalf("response has no valid id: %v", body)
	}
	return uint(value)
}

func TestIntegrationAdminUserAndGroupHTTPFlow(t *testing.T) {
	stack := newIntegrationStack(t, domain.AccessViewEdit)

	admin, err := stack.userService.SetRole(t.Context(), stack.user.ID, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("promote integration user: %v", err)
	}

	engine := fullIntegrationRouter(t, stack)
	cookie := loginFullRouter(t, engine, admin.Username, "secret123")

	groupBody := fmt.Sprintf(`{"name":"http-team","databaseAccesses":[{"databaseId":%d,"accessType":"view_only"}]}`, stack.target.ID)
	response := requestJSON(t, engine, http.MethodPost, "/api/v1/admin/groups", groupBody, cookie)
	if response.Code != http.StatusCreated {
		t.Fatalf("create group status=%d body=%s", response.Code, response.Body.String())
	}
	groupID := responseID(t, response)

	userBody := fmt.Sprintf(`{"username":"http-user","password":"secret123","groupIds":[%d]}`, groupID)
	response = requestJSON(t, engine, http.MethodPost, "/api/v1/admin/users", userBody, cookie)
	if response.Code != http.StatusCreated {
		t.Fatalf("create user status=%d body=%s", response.Code, response.Body.String())
	}
	userID := responseID(t, response)

	response = requestJSON(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/admin/users/%d", userID), "", cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("get user status=%d body=%s", response.Code, response.Body.String())
	}

	response = requestJSON(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d", userID), `{"username":"http-user-renamed"}`, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("update user status=%d body=%s", response.Code, response.Body.String())
	}

	response = requestJSON(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/role", userID), `{"role":"admin"}`, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("set role status=%d body=%s", response.Code, response.Body.String())
	}

	response = requestJSON(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/admin/groups/%d/users/%d", stack.group.ID, userID), "", cookie)
	if response.Code != http.StatusNoContent {
		t.Fatalf("add membership status=%d body=%s", response.Code, response.Body.String())
	}

	response = requestJSON(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/admin/groups/%d/users/%d", stack.group.ID, userID), "", cookie)
	if response.Code != http.StatusNoContent {
		t.Fatalf("remove membership status=%d body=%s", response.Code, response.Body.String())
	}

	response = requestJSON(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/admin/groups/%d", groupID), `{"name":"http-team-renamed"}`, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("update group status=%d body=%s", response.Code, response.Body.String())
	}

	response = requestJSON(t, engine, http.MethodGet, "/api/v1/admin/users?page=1&limit=20", "", cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("list users status=%d body=%s", response.Code, response.Body.String())
	}
	response = requestJSON(t, engine, http.MethodGet, "/api/v1/admin/groups?page=1&limit=20", "", cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("list groups status=%d body=%s", response.Code, response.Body.String())
	}

	response = requestJSON(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/admin/users/%d", userID), "", cookie)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete user status=%d body=%s", response.Code, response.Body.String())
	}
	response = requestJSON(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/admin/users/%d", userID), "", cookie)
	if response.Code != http.StatusNotFound {
		t.Fatalf("deleted user lookup status=%d body=%s", response.Code, response.Body.String())
	}

	response = requestJSON(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/admin/groups/%d", groupID), "", cookie)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete group status=%d body=%s", response.Code, response.Body.String())
	}
	response = requestJSON(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/admin/groups/%d", groupID), "", cookie)
	if response.Code != http.StatusNotFound {
		t.Fatalf("deleted group lookup status=%d body=%s", response.Code, response.Body.String())
	}
}
