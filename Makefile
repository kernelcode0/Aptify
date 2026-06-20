.PHONY: build build-cli ui dev clean

ui:
	cd web && npm run build

build: ui
	go build -o bin/aptify ./cmd/server

build-cli:
	go build -o bin/aptify-cli ./cmd/aptify-cli

dev-backend:
	go run ./cmd/server

dev-ui:
	cd web && npm run dev

clean:
	rm -rf bin/ data/ internal/web/dist/ web/dist/
