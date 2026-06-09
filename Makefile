.PHONY: build ui dev clean

ui:
	cd web && npm run build

build: ui
	go build -o apt-repository ./cmd/server

dev-backend:
	go run ./cmd/server

dev-ui:
	cd web && npm run dev

clean:
	rm -rf apt-repository data/ internal/web/dist/ web/dist/
