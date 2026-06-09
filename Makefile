.PHONY: build ui dev clean

ui:
	cd web && npm run build

build: ui
	go build -o aptify ./cmd/server

dev-backend:
	go run ./cmd/server

dev-ui:
	cd web && npm run dev

clean:
	rm -rf aptify data/ internal/web/dist/ web/dist/
