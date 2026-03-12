COMPOSE ?= docker compose

.PHONY: up down reset migrate-up migrate-down logs test fmt vet

up:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

reset:
	$(COMPOSE) down -v --remove-orphans

migrate-up:
	$(COMPOSE) run --rm migrate

migrate-down:
	$(COMPOSE) run --rm migrate -path /migrations -database "postgres://$${DB_USER}:$${DB_PASSWORD}@postgres:5432/$${DB_NAME}?sslmode=$${DB_SSLMODE}" down 1

logs:
	$(COMPOSE) logs -f

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...

test:
	go test ./...
