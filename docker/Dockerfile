# Stage 1: Build
FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod first (cached independently)
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

# Copy all source
COPY backend/ ./backend/

RUN cd backend && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -tags netgo -o /app/main .

# Stage 2: Runtime
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/main /main

EXPOSE 8080

CMD ["/main"]
