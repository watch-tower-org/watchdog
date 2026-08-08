WEB_DIR   := web
BACKEND   := backend
EMBED_DIR := $(BACKEND)/web/dist
BIN       := bin/watchtower

.PHONY: all dev dev-all dev-web build docker docker-compose-up docker-compose-down docker-compose-logs clean

all: build

# Build the frontend and embed it into the backend, then run the backend
# (production-style single binary serving the dashboard on :8080).
dev:
	npm --prefix $(WEB_DIR) run build
	rm -rf $(EMBED_DIR) && cp -R $(WEB_DIR)/dist $(EMBED_DIR)
	cd $(BACKEND) && go run .

# Frontend dev server with hot reload (proxies /api to localhost:8080).
# Run `make dev` in a second terminal to serve the API.
dev-web:
	npm --prefix $(WEB_DIR) run dev

# Run the backend API (:8080) and the Vite dev server (:5173) together.
# Stop both at once with Ctrl+C.
dev-all:
	@set -m; \
	trap 'kill -TERM -$$backend_pid -$$web_pid 2>/dev/null' INT TERM EXIT; \
	(cd $(BACKEND) && go run .) & backend_pid=$$!; \
	npm --prefix $(WEB_DIR) run dev & web_pid=$$!; \
	wait

# Build a production single binary with the frontend embedded.
build:
	npm --prefix $(WEB_DIR) run build
	rm -rf $(EMBED_DIR) && cp -R $(WEB_DIR)/dist $(EMBED_DIR)
	mkdir -p $(BACKEND)/$(dir $(BIN))
	cd $(BACKEND) && go build -ldflags="-s -w" -o $(BIN) .

# Build the production Docker image (multi-stage: web -> go -> scratch).
docker:
	docker build -f docker/Dockerfile -t watchtower .

# Bring up the full stack (Postgres + WatchTower) via docker compose.
docker-compose-up:
	docker compose up -d --build

docker-compose-down:
	docker compose down

docker-compose-logs:
	docker compose logs -f watchtower

clean:
	rm -rf $(WEB_DIR)/dist $(EMBED_DIR)
	rm -f $(BACKEND)/$(BIN)
