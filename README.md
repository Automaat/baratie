# 🍳 Baratie

Self-hosted kitchen companion — manage **recipes**, track your **pantry**, and
plan **meals** for the week. Runs on a home NAS, same stack and look as
[finance-buddy](../finance-buddy).

## Tech Stack

- **Frontend:** SvelteKit 2 + Svelte 5 (runes) + TypeScript
- **Backend:** Go (chi + pgx) — `backend-go/`
- **Database:** PostgreSQL 18
- **UI:** Tailwind CSS 4 + Skeleton (sahara theme) + lucide icons
- **Charts:** Apache ECharts 6
- **Auth:** JWT session cookie, admin seeded on startup
- **Deployment:** Docker Compose, images on ghcr.io

> Exact versions live in `package.json` and `backend-go/go.mod` — those
> manifests are the single source of truth.

## Features

- **Przepisy (Recipes):** full CRUD with ingredients, instructions, tags,
  servings and prep/cook times.
- **Spiżarnia (Pantry):** stock items with quantity, unit, category and
  expiry dates; the dashboard flags items expiring within 7 days.
- **Plan posiłków (Meal plan):** dated entries (breakfast/lunch/dinner/snack)
  optionally linked to a recipe.
- **Pulpit (Dashboard):** counts, upcoming meals and a pantry-by-category chart.
- **Users:** admin-managed accounts behind a JWT session cookie.

## Development

### Prerequisites

- [mise](https://mise.jdx.dev/) — manages tool versions (node 24, go 1.26) and
  task runners.

### Setup

```bash
mise install          # node, go, golangci-lint
npm install           # frontend deps
cp .env.example .env  # set POSTGRES_PASSWORD, BRT_JWT_SECRET, BRT_ADMIN_PASSWORD
mise run dev          # postgres + backend-go + frontend via docker-compose.dev.yml
```

Or run pieces individually:

```bash
mise run backend      # go run ./cmd/api  (needs DATABASE_URL)
mise run frontend     # vite dev server on :5173
```

### Common commands

```bash
# Frontend
npm run dev | build | preview
npm run check          # svelte-check (types)
npm run lint           # oxlint + prettier --check
npm run test:coverage  # vitest + coverage
npm run test:e2e       # playwright

# Backend
cd backend-go
go build ./... && go test ./... && gofmt -w .
golangci-lint run ./...
```

## Deployment

`docker-compose.yml` runs the frontend, backend-go and PostgreSQL together from
the published `ghcr.io/automaat/baratie-*` images. backend-go applies the
database schema on first start and seeds the admin user from `BRT_ADMIN_*`.

```bash
export POSTGRES_PASSWORD="a-strong-password"
export BRT_JWT_SECRET="a-long-random-secret"
export BRT_ADMIN_PASSWORD="a-strong-admin-password"
docker-compose up -d
```

### Release → deploy flow

1. Run the `release.yml` workflow (`gh workflow run release.yml`). An empty
   input auto-bumps the minor version — it tags `vX.Y.Z`, cuts a GitHub
   release, and pushes `:latest` + `:vX.Y.Z` images to ghcr.io.
2. Bump both image tags in the home-nas stack file to the new `vX.Y.Z`.
3. Deploy via the home-nas Ansible playbook.

`ORIGIN`, `CORS_ORIGINS`, and `PUBLIC_API_URL_BROWSER` can be overridden for
deployments behind a custom domain. Set `BRT_COOKIE_SECURE=true` when served
over HTTPS.

## License

Private project
