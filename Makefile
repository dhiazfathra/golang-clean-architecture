.PHONY: infra-up infra-down build test migrate seed generate

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

generate:
	go run cmd/generate/main.go -module=$(module) -fields=$(fields)
# Usage: make generate module=product fields="name:string,price:float64,sku:string,active:bool"
