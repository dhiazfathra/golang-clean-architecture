.PHONY: infra-up infra-down build test migrate seed

infra-up:
	docker compose -f deployments/docker-compose.yaml up -d

infra-down:
	docker compose -f deployments/docker-compose.yaml down

build:
	go build ./...

test:
	go test ./...

migrate:
	@for f in migrations/*.up.sql; do \
		echo "Applying $$f"; \
		psql "$$DATABASE_URL" -f "$$f"; \
	done

seed:
	go run cmd/server/main.go --seed-only
