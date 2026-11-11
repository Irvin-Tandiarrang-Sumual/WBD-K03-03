.PHONY: test test-unit test-unit-cover test-integration test-integration-cover test-all test-cover-all migrate seed swagger

UNIT_PKGS := ./config/... ./internal/adapters/driver/... ./internal/adapters/http/... ./internal/adapters/security/... ./internal/core/... ./pkg/...

migrate:
	cd backend && go run ./cmd/migrate

seed:
	cd backend && go run ./cmd/seed

swagger:
	cd backend && swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

test: test-unit

test-unit:
	cd backend && go test $(UNIT_PKGS) -count=1

test-unit-cover:
	cd backend && go test $(UNIT_PKGS) -coverprofile=coverage.unit.out -count=1
	cd backend && go tool cover -func coverage.unit.out

test-integration:
	cd backend && go test -tags=integration -v ./test/integration/... -count=1

test-integration-cover:
	cd backend && go test -tags=integration -coverpkg=./... -coverprofile=coverage.integration.out -v ./test/integration/... -count=1
	cd backend && go tool cover -func coverage.integration.out

test-all: test-unit test-integration

test-cover-all: test-unit-cover test-integration-cover
