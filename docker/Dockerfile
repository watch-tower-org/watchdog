# Stage 1: Build the web dashboard
FROM node:22-alpine AS web-builder

WORKDIR /app/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Stage 2: Build the Go backend (embeds the web dist)
FROM golang:1.26.1-alpine AS builder

ARG VERSION=dev

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first (cached independently). The backend module replaces
# the SDK with a local path (../sdk/go), so the SDK's go.mod must be present
# before the module graph is resolved.
COPY backend/go.mod backend/go.sum ./backend/
COPY sdk/go/go.mod ./sdk/go/
RUN cd backend && go mod download

# Copy all source (SDK included so the replace directive resolves)
COPY backend/ ./backend/
COPY sdk/ ./sdk/

# Embed the built frontend
COPY --from=web-builder /app/web/dist ./backend/web/dist/

RUN cd backend && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/watch-tower-org/watchtower/backend/internal/version.Version=$(VERSION)" -tags netgo -o /app/main .

# Stage 3: Runtime
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/main /main

EXPOSE 8080

CMD ["/main"]
