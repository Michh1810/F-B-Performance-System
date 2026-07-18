CREATE TABLE forecasts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    menu_item_id UUID NOT NULL REFERENCES menu_items(id),
    model TEXT NOT NULL,
    baseline NUMERIC,
    forecasted_units INTEGER,
    forecast_window_days INTEGER,
    price NUMERIC,
    estimated_cogs NUMERIC,
    forecasted_revenue NUMERIC,
    projected_profit NUMERIC,
    assumptions JSONB DEFAULT '{}',
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_forecasts_menu_item_id ON forecasts(menu_item_id);
