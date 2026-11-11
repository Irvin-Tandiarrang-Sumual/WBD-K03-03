# Backend Developer Guidance (WBD-K03-03)

This is technical guidelines for backend development. It covers architectural patterns, directory structure, testing standards, Swagger/OpenAPI documentation workflow, and naming conventions.

---

## 1. Core Architecture

The backend follows the **Hexagonal Architecture (Ports and Adapters)** pattern to separate business logic from external frameworks:

### Core Concepts Explained Simply
- **`domain` (`internal/core/domain`):** Contains the core business entities (e.g., `User`, `Group`, `Database`), business rules, and domain errors. It has zero external dependencies on frameworks or database libraries.
- **`ports` (`internal/core/ports`):** Defines the interfaces (contracts) for services and repositories. It specifies what capabilities the core needs (e.g., `UserRepository`) and what capabilities use cases expose (e.g., `UserService`), completely decoupling business logic from implementation details.
- **`service` (`internal/core/service`):** Implements the application use cases and business workflows. Services orchestrate domain entities and call repositories strictly through the interfaces declared in `ports`.
- **`adapters` (`internal/adapters`):** Implements external infrastructure and technical details. This includes the HTTP layer (Gin handlers, middleware, routes), database storage (GORM PostgreSQL repositories and models), engine drivers (PostgreSQL & MySQL target clients), and encryption ciphers (AES-GCM).

---

## 2. API Documentation (Swagger / OpenAPI)

### Accessing Swagger UI
When the backend service is running, the interactive documentation is available at:
```text
http://localhost:8080/swagger/index.html
```

### Documenting New Endpoints with Swaggo Annotations
Each HTTP handler must declare declarative Swaggo comments. Example standard format:
```go
// AddUser adds a user to a group.
//
//	@Summary		Add user to group
//	@Description	Adds an existing user to an existing group. Requires admin privileges.
//	@Tags			Admin - Groups
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id		path	int	true	"Group ID"
//	@Param			user_id	path	int	true	"User ID"
//	@Success		204		"User added to group"
//	@Failure		400		"Invalid input"
//	@Failure		401		"Unauthorized"
//	@Failure		403		"Forbidden (not admin)"
//	@Failure		404		"Group or user not found"
//	@Failure		409		"Membership already exists"
//	@Failure		500		"Internal server error"
//	@Router			/admin/groups/{id}/users/{user_id} [post]
func (h *UserGroupHandler) AddUser(c *gin.Context) { ... }
```

### Regenerating Swagger Documentation
When annotations or DTO structures are modified, regenerate the OpenAPI files:
```bash
make swagger
# or directly inside the backend directory:
cd backend && swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

---

## 3. Testing Strategy & Standards

### A. Unit Testing

Unit tests verify pure business logic in total isolation without accessing external databases or Docker.

- **File Location:** Placed alongside the code being tested with a `_test.go` suffix (e.g., `service_test.go`, `csrf_test.go`).
- **Principles:** Dependencies must be mocked via interfaces defined in `ports`.
- **Mandatory Structure: AAA (Arrange, Act, Assert) Pattern:**
  Every unit test function must follow the three-phase AAA structure clearly labeled with comments:
  1. `Arrange`: Prepare input data, instantiate mock objects, declare method call expectations, and initialize context.
  2. `Act`: Execute the specific function or service method under test.
  3. `Assert`: Validate return values, inspect errors (`errors.Is`), and verify that all mock expectations were satisfied.
- **Function Naming Convention:**
  ```text
  Test<MethodName>_<Scenario/Condition>_<ExpectedResult>
  ```
  Examples:
  - `TestCreateUser_ValidInput_ReturnsCreatedUser`
  - `TestDeleteGroup_HasOrphanedUsers_ReturnsErrUserMustHaveGroup`
  - `TestAuthenticate_MissingCookie_ReturnsUnauthorized`

- **Execution Commands:**
  ```bash
  make test-unit
  # or with code coverage analysis:
  make test-unit-cover
  ```

---

### B. Integration Testing
#### 1. Mandatory Header
Every file located in `test/integration/` **must** include the following build tag on line 1:
```go
//go:build integration
```
This ensures integration tests are never executed unintentionally during normal unit test runs (`go test ./...`).

#### 2. Type
The integration test suite uses Incremental Bottom-Up

#### 3. Naming Convention
All integration test functions must follow this convention:
```text
Test<Component>_Given_<Condition>_When_<Action>_Then_<ExpectedResult>
```
Examples:
- `TestPersistence_GivenRegisteredTargetDatabase_WhenPasswordStored_ThenEncryptedWithAESGCMInPostgreSQL`
- `TestPostgresTarget_GivenUserWithReadOnlyAccess_WhenExecuteSelectQuery_ThenReturnRowsSuccessfully`
- `TestMySQLTarget_GivenAppliedMigration_WhenRollbackTriggered_ThenCompensatingSQLDownRestoresPreviousSchema`
- `TestAuthJourney_GivenValidCredentials_WhenLogin_ThenSetOpaqueTokenCookieAndCreateSessionInPlatformDB`
- `TestAdminRBACJourney_GivenActiveUserSession_WhenAdminRevokesGroupAccess_ThenUserSessionInvalidatedAndAccessDenied`

#### 4. Execution Commands:
```bash
make test-integration
# or directly from backend directory:
cd backend && go test -tags=integration -v ./test/integration/... -count=1
```

---

## 4. Source Code Naming Conventions
- **Files:** Always use `snake_case.go` (e.g., `user_handler.go`, `admin_routes.go`, `user_group_service.go`). Test files end with `_test.go`.
- **Domain Entities & Types:** Use `PascalCase` nouns (e.g., `User`, `Group`, `DatabaseAccess`, `Migration`).
- **Domain Errors:** Always prefix with `Err` in `PascalCase` describing the business rule violation (e.g., `ErrUserMustHaveGroup`, `ErrInvalidCredentials`, `ErrUsernameTaken`).
- **Interfaces (`ports`):** Use `PascalCase` without an `I` prefix (e.g., `UserRepository`, `UserService`, `DatabaseExecutorRepository`).
- **Concrete Implementations:** Use unexported `camelCase` structs (e.g., `userService`, `userRepository`), instantiated through exported constructors returning the interface:
  ```go
  func NewUserService(userRepo ports.UserRepository, groupRepo ports.GroupRepository) ports.UserService
  ```
- **HTTP Handlers & DTOs:**
  - Handlers are named `<Entity>Handler` (e.g., `UserHandler`, `GroupHandler`).
  - Request DTOs are named `<Action><Entity>Request` (e.g., `CreateUserRequest`, `LoginRequest`).
  - Response DTOs are named `<Entity>Response` (e.g., `UserResponse`, `GroupResponse`).
- **Constants & Enums:** Use `PascalCase` grouped by concept with a common prefix (e.g., `RoleAdmin`, `RoleUser`, `AccessViewOnly`, `AccessViewEdit`).
- **Repository Methods (`ports.*Repository` & storage implementations):**
  Follow standard data-access semantics prefixed with clear intent:
  - Queries / Retrieval: Use `Find...` prefix (e.g., `FindByID(ctx, id)`, `FindAll(ctx, params)`, `FindByToken(ctx, token)`, `FindWithCredentialByID(ctx, id)`).
  - Persistence / Insert: Use `Save(ctx, entity)` (or `Save(ctx, entity, associations)`).
  - Updates: Use `Update(ctx, id, ...)` or `UpdateRole(ctx, id, role)`.
  - Deletions: Use `Delete(ctx, id)`.
  - Counting / Aggregations: Use `CountBy...` prefix (e.g., `CountByRole(ctx, role)`, `CountByUserID(ctx, userID)`).
  - Relational operations: Use descriptive action verbs (e.g., `AddUser(ctx, groupID, userID)`, `RemoveUser(ctx, groupID, userID)`).
- **Service & Port Methods (`ports.*Service` & business logic implementations):**
  Represent business operations and use-case workflows:
  - Standard resource workflows: `Create<Entity>(ctx, input)`, `Get<Entity>ByID(ctx, id)`, `GetAll<Entities>(ctx, params)`, `Update<Entity>(ctx, id, input)`, `Delete<Entity>(ctx, id)`.
  - Specialized business actions: Use domain action verbs (e.g., `Login(ctx, username, password)`, `Logout(ctx, token)`, `Resolve(ctx, token)`, `RevokeUser(ctx, userID)`, `Authorize(ctx, userID, dbID, accessType)`, `RollbackMigrations(ctx, input)`, `CompareMigrations(ctx, dbID, v1, v2)`).
- **HTTP Handler Methods (`internal/adapters/http/handlers`):**
  All handler methods take a single parameter `(c *gin.Context)` and reflect the HTTP action and target resource:
  - List collections (GET): `GetAllUsers(c *gin.Context)`, `GetAllGroups(c *gin.Context)`, `GetAllDatabases(c *gin.Context)`.
  - Get single resource (GET): `GetUserByID(c *gin.Context)`, `GetGroupByID(c *gin.Context)`, `GetDatabaseByID(c *gin.Context)`, `GetDatabasesByUserID(c *gin.Context)`.
  - Create resource (POST): `CreateUser(c *gin.Context)`, `CreateGroup(c *gin.Context)`, `CreateDatabase(c *gin.Context)`, `CreateMigration(c *gin.Context)`.
  - Modify resource (PATCH/PUT): `UpdateUser(c *gin.Context)`, `UpdateGroup(c *gin.Context)`, `UpdateUserRole(c *gin.Context)`.
  - Delete resource (DELETE): `DeleteUser(c *gin.Context)`, `DeleteGroup(c *gin.Context)`, `DeleteDatabase(c *gin.Context)`.
  - Specific action handlers: `AddUser(c *gin.Context)`, `RemoveUser(c *gin.Context)`, `Login(c *gin.Context)`, `Logout(c *gin.Context)`, `ExecuteDatabaseQuery(c *gin.Context)`, `ExplainDatabaseQuery(c *gin.Context)`, `RollbackMigrations(c *gin.Context)`.
- **Database Engine Driver Methods (`ports.DatabaseExecutorRepository`):**
  Define execution capabilities across client engines:
  - `ExecuteQuery(ctx, database, query)`
  - `ExecuteCommand(ctx, database, command)`
  - `Explain(ctx, database, query)`
  - `ExplainAnalyze(ctx, database, query)`
  - `GetDatabaseMetadata(ctx, database)`