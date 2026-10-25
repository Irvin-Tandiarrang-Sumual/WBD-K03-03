.PHONY: test test-unit test-unit-cover test-integration test-integration-cover test-all test-cover-all

test: test-unit

test-unit:
	cd backend && go test ./... -count=1

test-unit-cover:
	cd backend && go test ./... -coverprofile=coverage.unit.out -count=1
	cd backend && go tool cover -func=coverage.unit.out

test-integration:
	cd backend && go test -tags=integration ./integration ./internal/adapters/driver/postgresql ./internal/adapters/storage/postgresql -count=1

test-integration-cover:
	cd backend && go test -tags=integration -coverpkg=./... -coverprofile=coverage.integration.out ./integration ./internal/adapters/driver/postgresql ./internal/adapters/storage/postgresql -count=1
	cd backend && go tool cover -func=coverage.integration.out

test-all: test-unit test-integration

test-cover-all: test-unit-cover test-integration-cover
