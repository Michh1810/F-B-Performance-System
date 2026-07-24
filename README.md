# FBPerformance Backend

This is the backend service for FBPerformance, built with Golang, PostgreSQL, and Redis. 

The entire development environment is Dockerized. You do **not** need to install Go, PostgreSQL, or Redis on your local machine to work on this project!

## Prerequisites

1. Install [Docker](https://docs.docker.com/get-docker/) (Docker Desktop for Mac/Windows).
2. (Optional but recommended) Install `golang-migrate` to run database migrations locally.
   - macOS: `brew install golang-migrate`

## Getting Started

### 1. Start the Environment
To spin up the Go backend, PostgreSQL database, and Redis cache, run:

```bash
docker-compose up -d --build
```
*Note: The Go server uses `air` for hot-reloading. Any changes you make to `.go` files will instantly restart the server in the container!*

### 2. Verify the Server
Open your browser and navigate to:
[http://localhost:8080](http://localhost:8080)

### 3. Run Database Migrations
Before the application can work properly, you need to set up the database tables.

Run the following command from the root directory to apply the latest database schema:
```bash
migrate -path migrations -database "postgres://postgres:devpassword@localhost:5440/fbperformance?sslmode=disable" up
```

## Useful Commands

- **Stop the environment:** `docker-compose down`
- **View backend logs:** `docker logs fbperformance-backend -f`
- **Access PostgreSQL CLI:** `docker exec -it fbperformance-postgres psql -U postgres -d fbperformance`

## Database Connection
If you want to use a GUI like TablePlus or DBeaver to view the database:
- **Host:** `localhost`
- **Port:** `5440`
- **User:** `postgres`
- **Password:** `devpassword`
- **Database:** `fbperformance`

## TikTok Trend Retrieval

The Trend Agent (`/api/ai/recommendation`) evaluates a menu item — existing or hypothetical — by embedding it as a semantic query and searching a TikTok trend-signal corpus (`trend_signals`, a `pgvector`-backed table) for the closest matches. This happens **live, per request**: it embeds the query, searches, aggregates, classifies sentiment, computes growth against the item's last evaluation, and writes a fresh `trend_snapshots` row — all inside the `/recommendation` call. There is no separate batch job that precomputes snapshots per item.

The corpus itself (`trend_signals`) is populated separately by a one-shot binary, `cmd/trend-ingest`, which sweeps a curated, **menu-item-agnostic** list of broad food-trend hashtags via Apify's `clockworks/tiktok-scraper` actor, embeds each video's caption (Gemini's `gemini-embedding-001`), and upserts it into the corpus:

```bash
go run ./cmd/trend-ingest
```

Requires `DATABASE_URL`, `GEMINI_API_KEY`, and `APIFY_API_TOKEN` set (see `.env.example`). The hashtag list defaults to a small built-in set; override with `TREND_INGEST_HASHTAGS` (comma-separated).

**Trend coverage is bounded by that hashtag list.** This is semantic matching over a curated corpus, not open-ended discovery across all of TikTok — a real trend that's never swept under one of the curated hashtags won't be in `trend_signals` at all, no matter how well it would semantically match a query.

This binary is **not self-scheduling** — run it periodically (e.g. daily) via an external scheduler (cron, a Kubernetes `CronJob`, etc.). The scheduler is responsible for preventing overlapping runs (e.g. `flock` around a cron entry, or `concurrencyPolicy: Forbid` on a `CronJob`) — the binary itself has no lock against concurrent invocations. Overlapping runs don't corrupt data, they just double Apify/Gemini spend.

**Evaluating a hypothetical (not-yet-sold) item**: `/recommendation` requires a real `menu_items` row to exist (its `id` is the request's `menu_item_id`, a foreign key on `trend_snapshots`). There's currently no API endpoint to create one — insert a "candidate" row manually:
```sql
INSERT INTO menu_items (name, category, current_price, cogs, is_active)
VALUES ('Salted Egg Coffee', 'Beverage', 0, 0, false)
RETURNING id;
```
A proper creation endpoint is a separate follow-up, not yet built.
