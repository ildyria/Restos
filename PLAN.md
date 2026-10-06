# Restos: a personal restaurant ranking site

## Context
You want a website that records the restaurants you've had lunch or dinner at. For each one you rate it, write a review and add photos of the dishes you ordered. You also keep your own ranked order of the restaurants. The public site is read-only, and an admin panel is used to enter the data. No metrics for now. The stack is a **Go backend**, a **Vue frontend (Nuxt + Nuxt UI)** and a modern look that doesn't feel AI-generated. The project directory is empty, so everything below is new.

**Decisions taken**
- Ranking is a manual drag-and-drop order set by you. The public list can also be sorted by rating, by most recent visit or by name.
- One shared rating per restaurant, scored on 4 criteria (food, service, ambiance, value), each 1–5 in half steps. The overall score is the average of the criteria that are filled in.
- A restaurant has **many visits but a single rating and review**. Each visit records a date, lunch or dinner, short notes, and the dishes ordered with their photos.
- The public site is read-only. The admin panel needs a login.
- Multiple admin users. The first one is created with a CLI command, and further admins can be added from an admin page.
- UI in English.
- Deployment: one Docker container running a single Go binary that serves both the API and the embedded Nuxt SPA. Data is stored in SQLite, and photos go on local disk under `/data`.

## Architecture

```
Restos/
  backend/                     Go 1.24 module "restos"
    cmd/restos/main.go         subcommands: serve | migrate | user create|list|delete
    internal/config/           env config (RESTOS_ADDR, RESTOS_DATA_DIR, RESTOS_COOKIE_SECURE)
    internal/db/               open SQLite (WAL, foreign_keys=on), goose migrations (embedded)
      migrations/0001_init.sql
      store.go restaurants.go visits.go dishes.go photos.go users.go sessions.go
    internal/auth/             bcrypt, session tokens, RequireUser middleware, login rate limit
    internal/media/            decode → auto-orient → resize (2000px + 480px thumb) → JPEG, strips EXIF
    internal/httpapi/          stdlib net/http ServeMux (Go 1.22 method patterns), JSON helpers, handlers
    internal/web/              //go:embed dist → SPA handler with index.html fallback
  frontend/                    Nuxt 4 + @nuxt/ui v4, ssr:false, `nuxi generate`
    app/app.config.ts          theme colors
    app/assets/css/main.css    @import "tailwindcss"; @import "@nuxt/ui"; font + token overrides
    app/pages/                 public + admin pages
    app/layouts/               default (public), admin (dashboard)
    app/components/            RestaurantCard, ScoreBreakdown, RatingInput, PhotoGallery, VisitEditor…
    app/composables/useApi.ts  $fetch wrapper (credentials, X-Requested-With, error toasts)
    app/middleware/admin.ts    redirects to /admin/login if /api/auth/me returns 401
    shared/types.ts            API DTO types
  Dockerfile                   node build → copy dist into backend/internal/web/dist → go build → distroless
  docker-compose.yml           one service, volume ./data:/data
  Makefile                     dev, build, test, docker
```

**Backend libraries** (kept to a minimum): `modernc.org/sqlite` (pure Go, no CGO), `github.com/pressly/goose/v3`, `golang.org/x/crypto/bcrypt`, `github.com/disintegration/imaging`, `golang.org/x/image/webp`. Routing uses the standard library only, with no framework.

**Dev workflow:** `make dev` runs `go run ./cmd/restos serve` on :8080 and `nuxt dev` on :3000. Nuxt `nitro.devProxy` forwards `/api` and `/media` to :8080, so the browser only ever sees one origin and there is no CORS to configure.

## Data model (`0001_init.sql`)
- `users(id, email UNIQUE, name, password_hash, created_at)`
- `sessions(token_hash PK, user_id FK→users ON DELETE CASCADE, expires_at)`
- `restaurants(id, slug UNIQUE, name, cuisine, address, city, website, price_level 1–4, rating_food, rating_service, rating_ambiance, rating_value REAL NULL, review TEXT, rank_position INT NULL, cover_photo_id NULL, created_at, updated_at)`. A `rank_position` of NULL means the restaurant is unranked.
- `visits(id, restaurant_id FK CASCADE, visited_on DATE, meal CHECK IN ('lunch','dinner'), notes, created_at)`
- `dishes(id, visit_id FK CASCADE, name, course NULL ('starter','main','dessert','drink','other'), notes, position)`
- `photos(id, dish_id FK CASCADE, file_key, width, height, position, created_at)`. Files are stored as `/data/media/{full,thumb}/{file_key}.jpg`. Deleting a row also deletes its files, and cascades are handled by collecting the file keys before the delete.

## API
Public (no auth):
- `GET /api/restaurants?sort=rank|rating|recent|name&q=&cuisine=` returns summaries: scores, overall, cover thumb, last visit date and visit count. Ranked restaurants come first, then unranked ones by rating.
- `GET /api/restaurants/{slug}` returns the detail view: visits sorted newest first, each with its dishes and photos.
- `GET /api/cuisines` returns the distinct values, used for the filter.
- `GET /media/{full|thumb}/{key}.jpg` is a static file server with long-lived cache headers.

Auth:
- `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/me`
- The session cookie is a random 32-byte token, stored hashed with SHA-256. It is HttpOnly, SameSite=Lax, Secure by config, and lasts 30 days.
- CSRF protection: every mutating `/api/admin/*` and auth request must carry `X-Requested-With: restos`, and the `Origin` header is checked when present.
- Login attempts are rate-limited per IP in memory.

Admin (requires a session):
- `POST/PUT/DELETE /api/admin/restaurants[/{id}]`. The slug is generated from the name and kept unique.
- `PUT /api/admin/ranking {ids:[...]}` rewrites `rank_position` in one transaction. Ids that are not in the list become unranked.
- `POST /api/admin/restaurants/{id}/visits`, `PUT/DELETE /api/admin/visits/{id}`
- `POST /api/admin/visits/{id}/dishes`, `PUT/DELETE /api/admin/dishes/{id}`
- `POST /api/admin/dishes/{id}/photos` takes a multipart upload of one or more files, up to 20 MB each. Accepted formats are JPEG, PNG and WebP. `PUT /api/admin/photos/order` and `DELETE /api/admin/photos/{id}` handle reordering and deletion.
- `PUT /api/admin/restaurants/{id}/cover {photo_id}`
- `GET/POST/DELETE /api/admin/users`. You cannot delete yourself or the last remaining user.

## Frontend

**Look ("modern, not AI"):** an editorial food-magazine style.
- Warm neutrals: Nuxt UI `neutral: 'stone'`, with one muted accent (a terracotta or olive custom color in `main.css`). No gradients, glassmorphism, sparkle icons or emoji.
- Fonts via `@nuxt/fonts`: a serif display face (Fraunces or Instrument Serif) for headings and rank numerals, and Inter for body text.
- Photo-first cards with generous whitespace and thin borders instead of heavy shadows.
- Light and dark modes using Nuxt UI's color mode.

**Public pages**
- `/` shows "Our list". Ranked restaurants appear as large rows: a big serif rank numeral, the cover photo, then name, cuisine, city, price, overall score and a compact four-criteria breakdown. The page has a sort toggle (Our ranking / Top rated / Recent / A–Z), search and a cuisine filter. Unranked restaurants are shown in an "Unranked yet" section below.
- `/r/[slug]` has a hero cover photo, name and meta info, a `ScoreBreakdown` (four bars plus the overall score), the review, and a visit timeline. Each visit has a date and a lunch or dinner badge, with its dishes and photo grid. Clicking a photo opens a `UModal` + `UCarousel` lightbox.

**Admin pages** (`layouts/admin.vue` built on `UDashboardGroup` / `UDashboardSidebar` / `UDashboardPanel`)
- `/admin/login`
- `/admin` is a restaurants `UTable` with search, a "New restaurant" button and links to edit each one.
- `/admin/restaurants/new` and `/admin/restaurants/[id]` use a `UForm` with a zod schema for the info fields, four `RatingInput`s (half-star, built as a custom component) and a review textarea. Below the form is a visits list. Adding or editing a visit opens a `USlideover` with the date, meal and notes, plus inline dish rows, each with a `UFileUpload` for its photos. The slideover also lets you pick the cover photo.
- `/admin/ranking` has two columns, "Ranked" and "Unranked". Drag and drop between and within them uses `useSortable` from `@vueuse/integrations` (SortableJS). Saving calls `PUT /api/admin/ranking`.
- `/admin/users` lists the admins and has an add form (email, name, password) and a delete action.
- Feedback uses toasts (`useToast`), and deletes ask for confirmation in a modal.

## Implementation order
1. Backend scaffold: config, DB open, migrations, users and sessions store, `user create` CLI, auth endpoints and middleware.
2. CRUD for restaurants, visits and dishes, plus the ranking endpoint.
3. Media pipeline (upload, orient, resize, thumb), photo endpoints and the `/media` static handler.
4. Nuxt scaffold: Nuxt UI, fonts, theme tokens, `useApi`, the dev proxy and shared types.
5. Public pages: list and detail.
6. Admin: layout, login, restaurant table and form, visit and dish editor with uploads, ranking page, users page.
7. SPA embedding and fallback in Go, the Dockerfile, docker-compose, Makefile and a README with setup steps.

## Notes / risks
- **HEIC from iPhones:** iOS Safari converts HEIC to JPEG on upload when the file input uses `accept="image/*"`. The backend rejects anything else with a clear message.
- **SPA vs SSR:** a static SPA keeps deployment to one binary. The trade-off is weak SEO and link previews, which seems fine for a personal site.
- **Go embed:** `internal/web/dist` gets a placeholder `index.html` committed so that `go build` works before the frontend has been built.

## Verification
- `go test ./...` runs the store tests against a temp SQLite file: CRUD, cascade deletes that remove photo files, a ranking rewrite, and session expiry. It also runs `httptest` handler tests for the auth flow, CSRF header rejection, the 401 on admin routes without a session, and a multipart upload that produces both the full and thumb files.
- `cd frontend && npx nuxi typecheck`
- Manual end-to-end with `make dev`:
  1. Run `go run ./cmd/restos user create --email … --name …`.
  2. Log in at `/admin/login`.
  3. Create 3 restaurants with ratings.
  4. Add a visit with 2 dishes and photos, including a portrait phone photo, to confirm the orientation is correct.
  5. Set a cover photo.
  6. Drag to reorder on `/admin/ranking`.
  7. Check `/` in each sort mode, and check the detail page with the lightbox, in both light and dark mode and at mobile width.
- Run `docker compose up --build`, then confirm the same flow on :8080 and that data survives a container restart (the `/data` volume).
