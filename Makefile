WEB_DIR   := web
BACKEND   := backend
EMBED_DIR := $(BACKEND)/web/dist
BIN       := bin/watchtower

.PHONY: all dev dev-web build docker clean

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

# Build a production single binary with the frontend embedded.
build:
	npm --prefix $(WEB_DIR) run build
	rm -rf $(EMBED_DIR) && cp -R $(WEB_DIR)/dist $(EMBED_DIR)
	mkdir -p $(BACKEND)/$(dir $(BIN))
	cd $(BACKEND) && go build -ldflags="-s -w" -o $(BIN) .

# Build the production Docker image (multi-stage: web -> go -> scratch).
docker:
	docker build -f docker/Dockerfile -t watchtower .

clean:
	rm -rf $(WEB_DIR)/dist $(EMBED_DIR)
	rm -f $(BACKEND)/$(BIN)
