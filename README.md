# Restos

A small site for the restaurants we've eaten at: rate them, write a review, add photos of the dishes, and keep them in our own order of preference. Anyone can browse; an admin panel is used to enter the data.

- **Backend:** Go (stdlib `net/http`), SQLite (pure Go, no CGO), photos on local disk
- **Frontend:** Nuxt 4 + Nuxt UI, built as a static SPA and embedded in the Go binary
- **Ratings:** food, service, ambiance, value (1–5 in half steps); the overall score is their average
- **Ranking:** manual drag-and-drop order, independent of the scores
- **Visits:** a restaurant has one rating/review and many visits; each visit has dishes and dish photos

## Run it (Docker)

```sh
docker compose up --build -d
docker compose exec restos /restos user create --email you@example.com --name "You"   # prompts for a password
```

Open http://localhost:8080 and sign in at `/admin/login`. If you serve it over HTTPS, set `RESTOS_COOKIE_SECURE: "true"` in `docker-compose.yml`.

The distroless image has no TTY for password prompts in some setups; use `-e RESTOS_PASSWORD=...` with `docker compose exec` if the prompt isn't available.

Data lives in the `restos-data` volume (`/data`: `restos.db` and `media/`). Back it up with e.g.
`docker run --rm -v restos_restos-data:/data -v "$PWD":/out alpine tar czf /out/restos-backup.tgz -C /data .`

## Develop

Requires Go 1.26+ and Node 24+.

```sh
cd backend && go run ./cmd/restos user create --email you@example.com --name "You"
make dev            # API on :8080, Nuxt on :3000 (proxies /api and /media); open http://localhost:3000
make test           # go tests + frontend typecheck
make build          # single binary with the frontend embedded: bin/restos
```

Configuration (environment variables): `RESTOS_ADDR` (default `:8080`), `RESTOS_DATA_DIR` (default `./data`), `RESTOS_COOKIE_SECURE` (`true` behind HTTPS).

CLI: `restos serve | migrate | user create|list|delete`.

## Layout

```
backend/    Go API: cmd/restos, internal/{db,auth,media,httpapi,web}
frontend/   Nuxt app: app/{pages,components,layouts,composables}, shared/types.ts
```

Notes: photos are re-encoded to JPEG (max 2000 px, plus a 480 px thumbnail), which also strips EXIF/GPS data. Login attempts are rate-limited per IP, and admin requests need a session cookie plus an `X-Requested-With` header. Behind a reverse proxy, the rate limiter sees the proxy's address.
