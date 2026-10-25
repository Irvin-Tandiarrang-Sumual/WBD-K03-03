.PHONY: test test-unit test-unit-cover test-integration test-all

test: test-unit

test-unit:
	cd backend && go test ./...

test-unit-cover:
	cd backend && go test ./internal/adapters/driver/postgresql ./internal/adapters/http/handlers/auth ./internal/adapters/http/handlers/database_executor ./internal/adapters/http/middleware ./internal/adapters/security ./internal/core/service/... ./pkg/... -coverprofile=coverage.out
	cd backend && go tool cover -func=coverage.out

test-integration:
	cd backend && go test -tags=integration ./integration ./internal/adapters/driver/postgresql

test-all: test-unit test-integration
