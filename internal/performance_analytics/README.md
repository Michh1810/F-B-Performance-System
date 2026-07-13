# Foretaste Dashboard API Contract (v2)

All endpoints accept a shared date range filter as query params:
`?from=YYYY-MM-DD&to=YYYY-MM-DD`

If omitted, the backend defaults to the last 30 days.

---

## 1. GET /api/v1/dashboard/summary (IN-CHARGE: NGU TRUONG)

Top-of-page KPI strip. Aggregate numbers only, no per-item detail.

**Response**
```json
{
  "dateRange": { "from": "2026-06-01", "to": "2026-06-30" },
  "totalRevenue": 48230.50,
  "averageRating": 4.3,
  "averageProfitMargin": 34.2,
  "totalReviews": 128
}
```

| Field | Type | Notes |
|---|---|---|
| `totalRevenue` | number | Sum of all transactions in range |
| `averageRating` | number | Blended across Yelp + Google |
| `averageProfitMargin` | number | Menu-wide average, used as the profitability threshold line for classifying items below |
| `totalReviews` | number | Count in range, for context next to the rating |

---

## 2. GET /api/v1/dashboard/menu-items

The core menu engineering table. This is what changed the most from v1.

**Additional query params:** `sortBy=revenue|unitsSold|profitMargin`, `performanceCategory=star|plowhorse|puzzle|dog`

**Response**
```json
{
  "dateRange": { "from": "2026-06-01", "to": "2026-06-30" },
  "items": [
    {
      "id": "8f14e45f-ceea-4d7a-b5c1-example",
      "name": "Salted Egg Coffee",
      "menuCategory": "Beverage",
      "unitsSold": 412,
      "popularityIndex": 8.7,
      "revenue": 2060.00,
      "foodCostPercent": 28.5,
      "contributionMargin": 1472.90,
      "performanceCategory": "star",
      "trendPercent": 18.5
    }
  ]
}
```

| Field | Type | Notes |
|---|---|---|
| `menuCategory` | string | The dish's own category (Beverage, Entree, etc). Renamed from the old `category` to avoid clashing with `performanceCategory` below. |
| `popularityIndex` | number | This item's units sold as a percent of total units sold across the whole menu in range |
| `foodCostPercent` | number | Ingredient cost divided by menu price |
| `contributionMargin` | number | Dollar profit for this item in range, `(price - cogs) * unitsSold` |
| `performanceCategory` | enum | `"star" \| "plowhorse" \| "puzzle" \| "dog"`, computed by comparing this item's popularity and margin against the menu-wide averages from the summary endpoint |
| `trendPercent` | number | Change vs the prior equivalent period |

---

## 3. GET /api/v1/dashboard/sales-trend

Powers the line/bar chart showing revenue over time.

**Additional query param:** `granularity=day|week|month`

**Response**
```json
{
  "granularity": "day",
  "points": [
    { "date": "2026-06-01", "revenue": 1580.00, "unitsSold": 132 },
    { "date": "2026-06-02", "revenue": 1690.25, "unitsSold": 140 }
  ]
}
```

---

## 4. GET /api/v1/dashboard/review-summary

Separated from `summary` because review data is heavier (text, sentiment) and not every dashboard view needs it loaded.

**Response**
```json
{
  "dateRange": { "from": "2026-06-01", "to": "2026-06-30" },
  "totalReviews": 128,
  "averageRating": 4.3,
  "sentimentBreakdown": { "positive": 78, "neutral": 32, "negative": 18 },
  "sentimentTrend": [
    { "date": "2026-06-01", "averageRating": 4.1, "reviewCount": 5 }
  ],
  "topKeywords": ["fusion", "spicy", "slow service"]
}
```

| Field | Type | Notes |
|---|---|---|
| `sentimentBreakdown` | object | Simple positive/neutral/negative counts, feeds the Trend Agent later |
| `sentimentTrend` | array | Same shape as sales-trend, so the frontend can reuse one chart component for both |
| `topKeywords` | array | Optional for v1, nice signal once you're doing any text analysis on reviews |

---

## Design notes

- **Four endpoints, not one.** Each dashboard widget fetches independently, matching the ETL plan's "pre-calculate then serve from Redis" pattern. If `menu-items` gets slow to compute, it doesn't block `summary` from rendering.
- **`performanceCategory` is derived, not stored.** Compute it in `service.go` at request time (or during the nightly aggregation job) by comparing each item's `popularityIndex` and `contributionMargin` against the menu-wide averages in `summary`. Don't hardcode it in Postgres, the averages shift every time new sales data lands.
- **Naming discipline going forward:** any field two different concepts might both want to call `category`, `status`, or `score` deserves a specific name up front. Cheap to fix now, expensive once the frontend is wired to it.