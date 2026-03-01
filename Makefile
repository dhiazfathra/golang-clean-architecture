.PHONY: infra-up infra-down build test cover migrate seed generate lint run vet install-hooks sql-validate

SERVICE_NAME ?= golang-clean-arch
ENV          ?= development

infra-up:
	docker compose -f deployments/docker-compose.yaml up -d

infra-down:
	docker compose -f deployments/docker-compose.yaml down

build:
	go build ./...

test:
	go test ./...

cover:
	go test -coverprofile=cover.out ./...
	go tool cover -func=cover.out
	@echo "Coverage: $$(go tool cover -func=cover.out | grep total | awk '{print $$3}' | tr -d '%')%"

cover-html:
	go test ./...  -coverpkg=./... -coverprofile ./cover.out && go tool cover -html ./cover.out

migrate:
	@echo "Applying migrations..."
	@for f in $$(ls migrations/*.up.sql | sort); do \
		echo "  $$f"; \
		psql "$(DATABASE_URL)" -f "$$f" 2>/dev/null || true; \
	done

seed:
	SEED_ONLY=true go run cmd/server/main.go

generate:
	go run cmd/generate/main.go -module=$(module) -fields=$(fields)
# Usage: make generate module=product fields="name:string,price:float64,sku:string,active:bool"

lint:
	golangci-lint run ./...

run:
	go run cmd/server/main.go

vet:
	go vet ./...

install-hooks:
	@bash scripts/install-hooks.sh

sql-validate:
	@bash -c 'for f in migrations/*.up.sql; do \
		base=$$(basename "$$f"); \
		if ! echo "$$base" | grep -qE "^[0-9]{14}_[a-z_]+\.up\.sql$$"; then \
			echo "Invalid: $$base"; exit 1; \
		fi; \
	done && echo "All migration filenames valid"'
