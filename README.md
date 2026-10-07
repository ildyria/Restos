# Restos

A small site for the restaurants we've eaten at: rate them, write a review, add photos of the dishes, and keep them in our own order of preference. Anyone can browse; an admin panel is used to enter the data.

- **Backend:** Go (stdlib `net/http`), SQLite (pure Go, no CGO), photos on local disk
- **Frontend:** Nuxt 4 + Nuxt UI, built as a static SPA and embedded in the Go binary
- **Ratings:** each person (admin user) scores food, service, ambiance, value and quietness (1–5 in half steps) and says whether they'd go back; the public scores are the averages across people, and the overall score is the average of the five
- **Photos:** uploaded to a visit, then captioned with the dish they show (typing a dish name links or creates it); each restaurant can also have one photo of the place itself, which becomes its cover; a restaurant's website picture (`og:image`) is fetched as soon as its address is entered and can be chosen as the cover
- **Map:** restaurants can be placed on a map (OpenStreetMap tiles; the admin "Find address" button geocodes through OpenStreetMap Nominatim from the browser)
- **Ranking:** manual drag-and-drop order, independent of the scores
- **Visits:** a restaurant has one rating/review and many visits; each visit has dishes and dish photos
- **About page:** Markdown edited in the admin panel (with an optional photo), rendered by the server without raw HTML; `{{names}}` and `{{restaurants}}` are filled in, and a generic default text applies until it is edited

## Run it (Docker)

```sh
docker compose up --build -d
docker compose exec restos /restos user create --email you@example.com --name "You"   # prompts for a password
```

A prebuilt image (linux/amd64) is published to GitHub Container Registry by the `Docker image` workflow on every push to `master` (`:latest`) and on `v*` tags (`:1.2.3`): use `image: ghcr.io/ildyria/restos:latest` in `docker-compose.yml` instead of `build: .` to skip building.

Open http://localhost:8080 and sign in at `/admin/login`. If you serve it over HTTPS, set `RESTOS_COOKIE_SECURE: "true"` in `docker-compose.yml`.

The distroless image has no TTY for password prompts in some setups; use `-e RESTOS_PASSWORD=...` with `docker compose exec` if the prompt isn't available.

Data lives in the `restos-data` volume (`/data`: `restos.db` and `media/`). Back it up with e.g.
`docker run --rm -v restos_restos-data:/data -v "$PWD":/out alpine tar czf /out/restos-backup.tgz -C /data .`

## Develop

Requires Go 1.26+ and Node 24+.

```sh
cd backend && go run ./cmd/restos user create --email you@example.com --name "You"
make dev            # API on :8080, Nuxt on :3000 (proxies /api and /media); open http://localhost:3000
make test           # go tests + frontend typecheck and unit tests (vitest)
make build          # single binary with the frontend embedded: bin/restos
```

Configuration (environment variables): `RESTOS_ADDR` (default `:8080`), `RESTOS_DATA_DIR` (default `./data`), `RESTOS_COOKIE_SECURE` (`true` behind HTTPS), `RESTOS_REGISTRATION` (`true` opens sign-up at `/admin/register`; off by default, since every account is a full admin: turn it on while the people who should have an account register, then off again).

CLI: `restos serve | migrate | user create|list|delete`.

## Layout

```
backend/    Go API: cmd/restos, internal/{db,auth,media,httpapi,web}
frontend/   Nuxt app: app/{pages,components,layouts,composables}, shared/types.ts
```

Notes: entering a website makes the server fetch that page and its preview picture (public addresses only, 15 s / 2 MB page / 20 MB image limits); fetched pictures not saved with a restaurant are deleted after a day. The names of admins who have scored a restaurant are public (masthead, About page, per-person scores); emails never are. Photos are re-encoded to JPEG (max 2000 px, plus a 480 px thumbnail), which also strips EXIF/GPS data. Login attempts are rate-limited per IP, and admin requests need a session cookie plus an `X-Requested-With` header. Behind a reverse proxy, the rate limiter sees the proxy's address.
