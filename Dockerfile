# Stage 1: build React frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
# Output goes directly into internal/web/dist (Vite configured for this)
COPY internal/web/ ../internal/web/
RUN npm run build

# Stage 2: build Go binaries
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/internal/web/dist ./internal/web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o /aptify ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /aptify-cli ./cmd/aptify-cli

# Stage 3: minimal runtime image (server only)
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata wget
WORKDIR /app
COPY --from=builder /aptify .
# CLI binary is available in the builder stage and can be extracted:
#   docker create --name tmp <image> && docker cp tmp:/aptify-cli . && docker rm tmp
COPY --from=builder /aptify-cli /usr/local/bin/aptify-cli
VOLUME ["/data"]
ENV DATA_DIR=/data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["/app/aptify"]
