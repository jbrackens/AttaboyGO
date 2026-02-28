.PHONY: build run test vet lint migrate-up migrate-down migrate-version generate docker-up docker-down clean web-dev web-build

# ── Build ──────────────────────────────────────────────
build:
	go build -o bin/api ./cmd/api
	go build -o bin/wallet-server ./cmd/wallet-server
	go build -o bin/outbox-consumer ./cmd/outbox-consumer

run: build
	./bin/api

# ── Test ───────────────────────────────────────────────
test:
	go test ./... -count=1

test-v:
	go test ./... -v -count=1

test-integration:
	go test ./... -tags=integration -count=1

# ── Code Quality ──────────────────────────────────────
vet:
	go vet ./...

lint: vet
	@echo "go vet passed"

# ── Database Migrations ───────────────────────────────
migrate-up:
	go run ./cmd/migrate -cmd=up

migrate-down:
	go run ./cmd/migrate -cmd=down

migrate-version:
	go run ./cmd/migrate -cmd=version

migrate-step:
	go run ./cmd/migrate -cmd=step -steps=$(STEPS)

# ── Code Generation ──────────────────────────────────
generate:
	sqlc generate

# ── Docker ────────────────────────────────────────────
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-reset:
	docker compose down -v
	docker compose up -d

# ── Web Frontend ─────────────────────────────────────
web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

# ── Cleanup ───────────────────────────────────────────
clean:
	rm -rf bin/
