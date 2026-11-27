# Unit Testing Guide

Unit tests in Wedebe validate business logic, SQL query parsers, hazard classification rules, DDL reverse generators, cryptographic hashing, and permission checks in complete isolation without external database dependencies.

---

## Backend Unit Tests (Go)

### Test Scope and Packages
Backend unit tests cover 22 critical packages:
- **Core Domain & Entities**: `internal/core/domain/...` (validation rules, state machines, consensus quorum logic).
- **Domain Services**: `internal/core/service/...` (authentication, draft proposal lifecycles, user group RBAC, migration coordinator).
- **Adapters & Utilities**:
  - `internal/adapters/security/...` (session token generation and CSRF protection).
  - `internal/adapters/queue/channel/...` (in-memory job channel logic).
  - `pkg/sqlutil/...` (SQL statement tokenization, parameter binding, reverse DDL generation).
  - `pkg/hasher/...` (Argon2 / bcrypt password hashing).
  - `pkg/ginutil/...` & `pkg/response/...` (HTTP response serialization and error handling).

### How to Run

Run unit tests from the repository root:
```bash
# Run all unit tests via Makefile
make test
# or:
make test-unit

# Run unit tests with code coverage summary
make test-unit-cover
```

Or run directly from the `backend/` directory:
```bash
cd backend
go test -v ./... -count=1
```

To view HTML coverage report:
```bash
cd backend
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---