VERSION := $(shell tr -d '[:space:]' < VERSION)
VERSION_LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build build-cli ui dev clean

ui:
	cd web && npm run build

build: ui
	go build -ldflags="$(VERSION_LDFLAGS)" -o bin/aptify ./cmd/server

build-cli:
	go build -ldflags="$(VERSION_LDFLAGS)" -o bin/aptify-cli ./cmd/aptify-cli

dev-backend:
	go run ./cmd/server

dev-ui:
	cd web && npm run dev

clean:
	rm -rf bin/ data/ internal/web/dist/ web/dist/
