.PHONY: db-up db-down dev-web dev-api test test-web test-api build migrate docker-up docker-down docker-logs

db-up:
	docker compose -f docker-compose.dev.yml up -d mysql

db-down:
	docker compose -f docker-compose.dev.yml down

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f --tail=100

dev-web:
	cd web && npm run dev

dev-api:
	cd server && set -a && . ./.env && set +a && go run ./cmd/api

test: test-api test-web

test-api:
	cd server && go test ./...

test-web:
	cd web && npm test -- --run

build:
	cd server && go build ./cmd/api
	cd web && npm run build

migrate:
	cd server && set -a && . ./.env && set +a && go run ./cmd/api migrate
