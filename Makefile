.PHONY: dev dev-backend dev-frontend build test docker clean

# Run API on :8080 and Nuxt on :3000 (Nuxt proxies /api and /media to the API).
dev:
	$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	cd backend && go run ./cmd/restos serve

dev-frontend:
	cd frontend && npm run dev

# Production binary with the frontend embedded: bin/restos
build:
	cd frontend && npm ci && npx nuxt generate
	find backend/internal/web/dist -mindepth 1 ! -name .gitkeep -delete
	cp -r frontend/.output/public/. backend/internal/web/dist/
	mkdir -p bin && cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../bin/restos ./cmd/restos

test:
	cd backend && go test ./...
	cd frontend && npx nuxt typecheck

docker:
	docker compose up --build

clean:
	rm -rf bin frontend/.output frontend/.nuxt
