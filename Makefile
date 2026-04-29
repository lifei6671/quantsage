.PHONY: build test race lint fmt tidy migrate-up migrate-down

GO_CACHE := /tmp/quantsage-go-build

build:
	cd apps/server && GOCACHE=$(GO_CACHE) go build ./...

test:
	cd apps/server && GOCACHE=$(GO_CACHE) go test -timeout 120s ./...

race:
	cd apps/server && GOCACHE=$(GO_CACHE) go test -race -timeout 120s ./...

lint:
	cd apps/server && GOCACHE=$(GO_CACHE) GOLANGCI_LINT_CACHE=/tmp/golangci-lint golangci-lint run ./...

fmt:
	gofmt -w apps/server

tidy:
	cd apps/server && GOCACHE=$(GO_CACHE) go mod tidy

migrate-up:
	cd apps/server && goose -dir ../../migrations/postgres postgres "$$QUANTSAGE_DATABASE_DSN" up

migrate-down:
	cd apps/server && goose -dir ../../migrations/postgres postgres "$$QUANTSAGE_DATABASE_DSN" down
