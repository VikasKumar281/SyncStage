# Convenience targets. Nothing here is required — every command it wraps can be
# typed directly — but `make dev` is the fastest path from clone to running.

.PHONY: help install dev backend frontend build test verify docker clean

help:
	@echo "make install   Install frontend dependencies"
	@echo "make backend   Run the Go API on :8080"
	@echo "make frontend  Run the React dev server on :5173"
	@echo "make dev       Run both (backend in the background)"
	@echo "make test      Run the Go test suite"
	@echo "make verify    Check the Go and JS timelines agree (needs a running backend)"
	@echo "make build     Build the frontend bundle and the Go binary"
	@echo "make docker    Build and start the single-container deployment"
	@echo "make clean     Remove build output and the local state file"

install:
	cd frontend && npm install

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

dev:
	@echo "Starting backend on :8080 and frontend on :5173 — Ctrl-C stops both"
	@cd backend && go run ./cmd/server & \
	cd frontend && npm run dev; \
	kill %1 2>/dev/null || true

test:
	cd backend && go vet ./... && go test ./... -count=1

verify:
	cd frontend && node scripts/verify-parity.mjs http://localhost:8080

build:
	cd frontend && npm run build
	cd backend && go build -trimpath -o bin/server ./cmd/server

docker:
	docker compose up --build

clean:
	rm -rf frontend/dist backend/bin backend/data/state.json
