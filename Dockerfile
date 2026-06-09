# Stage 1: build React frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
# Output goes directly into internal/web/dist (Vite configured for this)
COPY internal/web/ ../internal/web/
RUN npm run build

# Stage 2: build Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /apt-repository ./cmd/server

# Stage 3: minimal runtime image
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /apt-repository .
VOLUME ["/data"]
ENV DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/app/apt-repository"]
