CREATE INDEX IF NOT EXISTS idx_transactions_menu_item_sold_at ON transactions(menu_item_id, sold_at DESC);
