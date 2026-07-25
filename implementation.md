# Demand Forecasting Implementation

## Purpose

This document describes the demand-forecasting implementation added for the Week 2 Engineer C scope. The feature estimates expected item demand from recent sales history, projects revenue and profit for a requested forecast window, optionally asks an AI service for a bounded adjustment, persists the result, and exposes the feature through `POST /api/forecast`.

The implementation deliberately uses a transparent statistical approach rather than an opaque machine-learning model:

1. Aggregate sales into daily quantities.
2. Give more recent days more influence with normalized exponential decay.
3. Multiply the expected daily quantity by the requested horizon.
4. Optionally apply an AI multiplier between `0.5` and `1.5`.
5. Calculate revenue and profit in integer cents.

## Changed Files

| File | Responsibility |
|---|---|
| `cmd/api/main.go` | Opens PostgreSQL, constructs the AI and forecasting services, and registers the forecast endpoint. |
| `go.mod`, `go.sum` | Adds the pgx PostgreSQL driver and its transitive dependencies. |
| `internal/ai/client.go` | Implements the configurable Gemini-compatible AI client. |
| `internal/demand_forecast/types.go` | Defines request, response, and transaction JSON contracts. |
| `internal/demand_forecast/handler.go` | Handles HTTP decoding, validation, error mapping, and JSON responses. |
| `internal/demand_forecast/service.go` | Implements loading history, the forecast model, AI adjustment, calculations, and persistence. |
| `internal/demand_forecast/handler_test.go` | Verifies valid and invalid HTTP requests. |
| `internal/demand_forecast/service_test.go` | Verifies forecast math, horizon handling, zero-sale days, and missing history. |
| `migrations/000002_create_forecasts.*.sql` | Creates and rolls back the `forecasts` table. |
| `migrations/000003_add_transaction_forecast_index.*.sql` | Adds and rolls back the transaction-history query index. |

## Runtime Wiring

`cmd/api/main.go` is the composition root for the API process.

1. It reads `DATABASE_URL`. If absent, it uses the local Docker-development PostgreSQL URL.
2. It opens a `pgx`-backed `database/sql` connection and calls `Ping` so the process fails early when PostgreSQL is unavailable.
3. It creates an AI client using `AI_API_KEY` and `AI_ENDPOINT` from the environment.
4. It injects the database and AI client into `demand_forecast.NewServiceWithDB`.
5. It wraps that service in `demand_forecast.NewHandler`.
6. It registers the handler as `POST /api/forecast`.

This dependency injection means the forecasting package does not open its own database connection or read environment variables directly. The AI package owns AI environment configuration, and `main` owns application construction.

## HTTP API Contract

### Endpoint

```text
POST /api/forecast
Content-Type: application/json
```

The endpoint accepts a single JSON object. It does not accept a bare array, multiple JSON values, or unknown JSON fields.

### Request body

```json
{
  "items": [
    {
      "item_id": "11111111-1111-4111-8111-111111111111",
      "item_name": "Salted Egg Coffee",
      "forecast_horizon_days": 7,
      "price_cents": 650,
      "estimated_cogs_cents": 200,
      "historical_transactions": [
        {
          "timestamp": "2026-06-05T00:00:00Z",
          "quantity": 12
        }
      ]
    }
  ]
}
```

`historical_transactions` is optional. If it is omitted or empty, the service queries the previous 30 days of that menu item's transaction history from PostgreSQL. If the request supplies it, it is used instead; this is useful for testing or for forecasting from prepared comparable-item history.

Money is expressed in **integer cents**. For example, `$6.50` is `650` and `$2.00` is `200`. This avoids binary floating-point rounding during revenue and profit calculations.

### Validation

The handler performs the following checks before any forecast work begins:

- HTTP method must be `POST`.
- The body must be one valid JSON object with known fields only.
- `items` must contain at least one request.
- `item_id` must be a non-empty UUID.
- `item_name` must be non-empty.
- `forecast_horizon_days` must be greater than zero.
- `price_cents` must be greater than zero.
- `estimated_cogs_cents` cannot be negative.
- Each supplied transaction timestamp must be RFC3339-compatible and non-zero.
- Each supplied quantity cannot be negative.

Validation failures return a JSON `400 Bad Request` response, for example:

```json
{
  "error": "items[0].price_cents must be greater than zero"
}
```

If neither supplied history nor database history contains sales, the endpoint returns `422 Unprocessable Entity` because a meaningful baseline cannot be calculated.

### Success response

```json
{
  "forecasts": [
    {
      "item_id": "11111111-1111-4111-8111-111111111111",
      "item_name": "Salted Egg Coffee",
      "baseline_units": 11.29,
      "forecasted_units": 79,
      "forecasted_revenue_cents": 51350,
      "projected_profit_cents": 35550,
      "forecast_window_days": 7,
      "model": "normalized_exponential_decay_moving_average",
      "ai_adjustment_status": "not_configured"
    }
  ]
}
```

`baseline_units` is expected **daily** demand. `forecasted_units` is expected demand over the full requested window. Revenue and profit are cents.

## Forecast Service

`internal/demand_forecast/service.go` contains the domain workflow in `ForecastMenuItems`.

### 1. Select transaction history

For each requested item, the service first uses `historical_transactions` when present. Otherwise it calls `loadDailyTransactionHistory` with a 30-day history window.

The service uses the incoming request context throughout. This allows a disconnected HTTP client to cancel DB and AI work. It also applies independent timeouts:

- database history query: five seconds;
- AI call: five seconds;
- persistence insert: three seconds.

### 2. Load a complete daily series from PostgreSQL

The database query uses PostgreSQL `generate_series` to create every calendar day in the history window. It then left-joins the item's transactions and uses `COALESCE(SUM(quantity), 0)`.

This is important. A simpler `GROUP BY sold_at` query would return only dates on which the item sold. Missing zero-sales days would artificially increase the calculated daily average. The generated series makes a day with no sales an explicit `quantity = 0` observation.

The query groups by day and orders newest to oldest. A migration adds an index on:

```sql
transactions(menu_item_id, sold_at DESC)
```

This supports filtering a menu item over a recent time range.

### 3. Normalize supplied history

`calculateNormalizedDecayAverage` also handles request-supplied history safely:

1. It truncates timestamps to UTC calendar days.
2. It combines multiple transaction rows from the same day.
3. It finds the oldest and newest observed dates.
4. It creates each day between them, including any zero-sales gaps.
5. It processes days from newest to oldest.

As a result, callers cannot alter the result merely by sending rows in a different order or by splitting one day into many rows.

### 4. Calculate normalized exponential-decay baseline

The newest day is given a weight of `1.0`. Each preceding day receives 70% of the weight of the day after it:

```text
newest day:       1.0000
one day older:    0.7000
two days older:   0.4900
three days older: 0.3430
```

For daily quantity `q_i` and its recency weight `w_i`, the baseline is:

```text
baseline daily units = sum(q_i × w_i) / sum(w_i)
```

The division by total weight makes the model normalized. Adding more historic days changes the model based on their sales, not because the raw sum of weights grows.

The model name persisted and returned by the API is:

```text
normalized_exponential_decay_moving_average
```

### 5. Apply optional AI adjustment

The statistical baseline is always the source of truth. AI is an optional adjustment layer.

The service begins with `multiplier = 1.0`. It calls AI only when a client exists and both `AI_API_KEY` and `AI_ENDPOINT` are present. The AI receives the baseline, item ID, item name, and forecast horizon.

The result is accepted only when it is valid and within the permitted range. The final calculation is:

```text
forecasted units = round(baseline daily units × horizon days × AI multiplier)
```

The response and saved assumptions expose one of these statuses:

| Status | Meaning |
|---|---|
| `not_configured` | No configured AI client was available; multiplier stayed at `1.0`. |
| `applied` | AI returned and passed validation for a multiplier. |
| `failed` | AI was configured but timed out, returned an HTTP/parsing error, or returned an invalid multiplier; multiplier fell back to `1.0`. |

This fallback keeps the forecast endpoint available when AI is unavailable, while the status prevents a fallback from being mistaken for an intentional AI recommendation.

### 6. Calculate financial projection in cents

All financial calculations use `int64` cents:

```text
forecasted revenue cents = forecasted units × price cents
projected profit cents   = forecasted revenue cents − (forecasted units × COGS cents)
```

Integer arithmetic is exact and avoids values such as `$12.999999` that can occur when `float64` is used for currency.

### 7. Persist the forecast

When a database is configured, every successful forecast is inserted into `forecasts`.

The database schema stores price, COGS, revenue, and profit in `NUMERIC` columns expressed in normal currency units. At the insert boundary, the SQL divides the integer cent values by `100` using PostgreSQL `numeric` arithmetic. The rest of the Go code therefore remains exact integer-cent arithmetic.

The `assumptions` JSONB field records:

```json
{
  "ai_multiplier": 1.0,
  "ai_adjustment_status": "not_configured",
  "baseline_unit": "daily_units",
  "history_days": 30
}
```

Persistence errors are not discarded. A failed insert returns an error rather than claiming a forecast was saved.

## Gemini-Compatible AI Client

`internal/ai/client.go` contains a small HTTP client that is compatible with a Gemini `generateContent` endpoint.

### Configuration

```text
AI_API_KEY=<Gemini API key>
AI_ENDPOINT=https://generativelanguage.googleapis.com/v1beta/models/<model>:generateContent
```

`AI_ENDPOINT` must be the full generate-content URL, not only a base domain or model name.

### Request

The client sends an HTTP `POST` with:

```text
Content-Type: application/json
x-goog-api-key: <AI_API_KEY>
```

Its body uses the Gemini content/parts form:

```json
{
  "contents": [
    {
      "parts": [
        {
          "text": "Return only a decimal multiplier between 0.5 and 1.5 ..."
        }
      ]
    }
  ]
}
```

The client expects the first generated text at:

```text
candidates[0].content.parts[0].text
```

It parses that text as a decimal multiplier. Values below `0.5`, above `1.5`, non-numeric text, missing candidates, and non-2xx responses are all rejected.

The service treats those errors as an AI adjustment failure and returns the statistical forecast with multiplier `1.0`.

## Database Migrations

### `000002_create_forecasts.up.sql`

Creates the `forecasts` table with:

- UUID primary key;
- foreign key to `menu_items`;
- selected model name;
- baseline and forecasted units;
- window length;
- price, COGS, revenue, and projected profit;
- JSONB assumptions for auditability;
- generation timestamp.

It also creates `idx_forecasts_menu_item_id` for forecast lookup by menu item.

### `000002_create_forecasts.down.sql`

Drops the `forecasts` table during rollback. Adding this file makes migration pair `000002` reversible.

### `000003_add_transaction_forecast_index.up.sql`

Creates `idx_transactions_menu_item_sold_at` on `transactions(menu_item_id, sold_at DESC)`. The forecast query uses these columns to find recent history efficiently.

### `000003_add_transaction_forecast_index.down.sql`

Drops the transaction-history index during rollback.

## Automated Tests

Run the focused tests with:

```bash
go test ./internal/demand_forecast ./internal/ai
```

`handler_test.go` verifies:

- a correctly shaped request returns `200 OK`;
- an invalid request returns `400 Bad Request` with a field-level error.

`service_test.go` verifies:

- same-day transactions are combined;
- recent observations receive greater weight;
- the normalized baseline matches the expected calculation;
- forecast horizon changes total forecasted units;
- missing history produces `ErrNoHistoricalTransactions`;
- gaps between supplied transaction dates become zero-sales days in the calculation.

The tests intentionally run without PostgreSQL or Gemini credentials. They validate deterministic forecast logic only. Database-query/persistence integration tests and mocked AI HTTP tests are recommended next additions.

## Current Limits and Future Work

- Database forecasts use the target item's own history. For a completely new menu item, the caller must supply comparable history or the API must be extended with reference-item IDs.
- An AI failure is exposed as `ai_adjustment_status: "failed"`, but the API does not yet provide detailed failure diagnostics or metrics. Add structured logs/metrics for operations.
- The AI prompt currently has limited contextual information. Seasonality, weather, social trends, and review data can be added later when those datasets are available.
- Batch requests are persisted item-by-item. A database transaction would be needed for all-or-nothing batch persistence.
- The main HTTP server has no explicit read/write/idle timeouts yet; configure an `http.Server` before production deployment.

