.PHONY: test test-unit test-unit-cover test-integration test-integration-cover test-all test-cover-all test-e2e test-e2e-docker test-e2e-fresh test-stress test-stress-docker test-stress-fresh migrate seed swagger

UNIT_PKGS := ./config/... ./internal/adapters/driver/mysql/... ./internal/adapters/driver/postgresql/... ./internal/adapters/http/middleware/... ./internal/adapters/security/... ./internal/core/domain/... ./internal/core/service/... ./pkg/...

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

test-e2e:
	cd e2e && npx playwright test

test-e2e-docker:
	docker compose --profile test up --build --abort-on-container-exit --exit-code-from e2e

test-e2e-fresh:
	docker compose down -v
	docker compose --profile test up --build --abort-on-container-exit --exit-code-from e2e

test-stress:
	k6 run stress/stress_suite.js

test-stress-docker:
	docker compose --profile stress run --rm k6 run stress_suite.js

test-stress-fresh:
	docker compose down -v
	docker compose up -d --build
	docker compose --profile stress run --rm k6 run stress_suite.js
