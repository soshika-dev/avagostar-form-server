.PHONY: run migrate-up migrate-down test

run:
	go run ./cmd/server

migrate-up:
	docker compose --profile tools run --rm migrate

migrate-down:
	docker run --rm -v $(PWD)/internal/db/migrations:/migrations migrate/migrate \
		-path /migrations -database "$${DATABASE_URL}" down 1

test:
	go test ./...
