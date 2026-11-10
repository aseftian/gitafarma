.PHONY: help dev-backend dev-web build test lint bench-storage vapid-keys docker-up docker-down

help:
	@echo "Targets:"
	@echo "  dev-backend    Run Go backend API"
	@echo "  dev-web        Run Vite React dev server"
	@echo "  build          Build backend binary and web static bundle"
	@echo "  test           Run backend and web tests"
	@echo "  lint           Run backend vet and web lint"
	@echo "  bench-storage  Run SQLite vs BBolt storage benchmark spike"
	@echo "  vapid-keys     Generate Web Push VAPID public/private keys"
	@echo "  docker-up      Start services with Docker Compose"
	@echo "  docker-down    Stop Docker Compose services"

dev-backend:
	cd backend && go run ./cmd/api

dev-web:
	cd web && npm run dev

build:
	cd backend && go build -o bin/api ./cmd/api
	cd web && npm run build

test:
	cd backend && go test ./... -race
	cd web && npm test

lint:
	cd backend && go vet ./...
	cd web && npm run lint

bench-storage:
	cd backend && go test -v -bench=. -benchmem ./internal/adapter/storage/spike

vapid-keys:
	cd backend && go run ./cmd/vapidgen

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
