# Baratie

Self-hosted kitchen companion: recipes, pantry, and weekly meal planning. Same
stack and conventions as the sibling project `../finance-buddy`, trimmed to a
clean scaffold.

**Tech Stack:** SvelteKit 2 + Svelte 5 (runes) + TypeScript, Go 1.26
(chi + pgx), PostgreSQL 18, Tailwind 4 + Skeleton (sahara theme).

> Version numbers are indicative. `package.json` and `backend-go/go.mod` are
> the single source of truth — check them when exact versions matter.

---

## Project Structure

```
/backend-go/         # Go backend (chi router + pgx)
  /cmd/api/          # main — server wiring, schema bootstrap, healthcheck cmd
  /internal/
    /<domain>/       # one package per endpoint group: store.go (pgx),
                     # handler.go (chi), validation as needed
    /db/             # pool wiring + schema.sql (baseline, applied on first start)
    /auth/           # users table, bcrypt, JWT session tokens, chi middleware
    /httputil/       # JSON + error response helpers
    /dbutil/         # pgx scan/collect helpers
    /wire/           # JSON wire types (IsoDate, IsoNaive)
    /metrics/        # Prometheus collectors + /metrics handler
    /server/         # route registration + middleware stack
/src/                # SvelteKit frontend
  /routes/           # file-based routing (+page.svelte, +page.ts)
  /lib/
    /components/     # UI components (Modal, Toast, Confirm, BottomSheet, ...)
    /stores/         # rune-based stores (toast, confirm, crudForm, navPrefs)
    /utils/          # formatters
    /nav/            # nav route table
/e2e/                # Playwright suite (auth.setup + specs)
```

### Domains

- **recipes** (`/api/recipes`) — name, description, instructions, `ingredients`
  + `tags` as Postgres `text[]`, servings, prep/cook minutes. Full CRUD.
- **pantry** (`/api/pantry`) — name, quantity, unit, category, optional expiry.
  Full CRUD.
- **mealplan** (`/api/meal-plan`) — dated entries (breakfast/lunch/dinner/snack)
  optionally linked to a recipe; supports `date_from`/`date_to` filtering.
- **nutrition** (`/api/nutrition/summary`) — read-only macro aggregation from the
  meal plan over `date_from`/`date_to`: per-day + period totals and average, with
  optional `target_*` query params yielding per-day deltas. No own table.
- **auth** (`/api/auth/*`, `/api/users`) — login/logout/me, admin-only user CRUD.

---

## Conventions

### Backend (Go)

- One package per endpoint group: `store.go` (pgx queries), `handler.go` (chi
  handlers + wire types), validation inline or in `validation.go`.
- **Sentinel errors** (`ErrNotFound`, ...) so handlers map to HTTP status
  without sniffing pg text; `dbutil.MapErr` does the pgx.ErrNoRows → sentinel
  mapping.
- **Validation returns only `*httputil.ValidationError`** and normalizes the
  request in place; a separate `toX()` builder always returns a non-nil entity.
  This keeps `nilaway` happy (no `(entity, err)` pairs the analyzer can't prove
  mutually exclusive).
- **PUT is a full replace**, DELETE is a hard delete.
- Linters: `golangci-lint` (`.golangci.yml`) + `nilaway`. **Never** add
  suppression directives — fix the root cause. `funlen` caps handlers at
  100 lines / 70 statements.

### Frontend (Svelte 5 runes — no legacy syntax)

- Props via `$props()`, state via `$state`, computed via `$derived`/`$derived.by`,
  effects via `$effect`. Events as attributes (`onclick={fn}`).
- Data fetching in `load()` (`+page.ts`) via `$lib/apiClient` (`api.get/post/...`).
- Browser API calls route through the SvelteKit `/api` proxy so the JWT never
  reaches client JS. SSR calls hit the backend directly (see `$lib/api`).
- No `any`. Formatter prettier, linter oxlint. Tabs.
- UI labels are Polish; no i18n.

### Auth

- Backend: `BRT_JWT_SECRET`, `BRT_ADMIN_USERNAME`, `BRT_ADMIN_PASSWORD` (admin
  re-seeded on every startup), `BRT_COOKIE_SECURE=true` over HTTPS. Backend
  hard-fails at startup without `BRT_JWT_SECRET` / `BRT_ADMIN_PASSWORD`.
- Session cookie `brt_token`; `hooks.server.ts` gates page navigation and the
  `/api` proxy.

---

## Common Commands

```bash
mise run dev        # docker-compose.dev.yml (postgres + backend + frontend)
mise run backend    # go run ./cmd/api (needs DATABASE_URL)
mise run frontend   # vite dev (:5173)

npm run check                       # svelte-check
npm run lint                        # oxlint + prettier --check
npm run test:coverage               # vitest (coverage gated on utils + stores)
cd backend-go && go test ./...      # Go unit tests
golangci-lint run ./...             # Go lint
./scripts/run-nilaway.sh backend-go github.com/Automaat/baratie/backend-go
```

---

## Deployment

Runs on a self-hosted NAS (Ansible repo `~/sideprojects/home-nas`), behind
Traefik with TLS, same topology as finance-buddy: Postgres 18 + `backend-go` +
`frontend`. Images: `ghcr.io/automaat/baratie-{backend-go,frontend}`.

Release: run `release.yml` (empty input auto-bumps minor) → tags `vX.Y.Z`, cuts
a GitHub release, pushes `:latest` + `:vX.Y.Z` to ghcr.io. Then bump the tags in
the home-nas stack file and deploy.

---

## Deliberate Simplifications vs finance-buddy

This is a fresh scaffold, so a few finance-buddy mechanisms were intentionally
left out (add them back if the need arises):

- **No OpenAPI codegen.** Frontend types are hand-written per domain in the
  `+page.ts` files rather than generated from a Go-emitted spec.
- **No Python black-box test suite.** Backend correctness is covered by Go unit
  tests + the Playwright e2e flow against a real backend.
- **No schedulers / external data fetchers.** The backend is request/response
  only.

---

## Git & CI

- Conventional commits, `type(scope): description`. Sign with `-s -S`.
- CI gates: frontend (lint + check + coverage + build), Go (vet + test + lint +
  nilaway), Playwright e2e, then docker-publish to ghcr.io on `main`.
- Branch naming: `feat/*`, `fix/*`, `chore/*`.
