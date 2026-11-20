.PHONY: up down reset logs run test

BIN ?= ./rockets
DURATION ?= 60
TEST_DATABASE_URL ?= postgres://rockets:rockets@localhost:5433/rockets?sslmode=disable

up:
	docker compose up -d --build

down:
	docker compose down

reset:
	docker compose down -v

logs:
	docker compose logs -f app

run: up
	@sleep 3
	scripts/smoke.sh $(BIN) $(DURATION)

test:
	docker compose up -d postgres migrate
	@sleep 3
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test -race -count=1 ./...
