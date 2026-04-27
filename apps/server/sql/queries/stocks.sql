-- name: ListStocks :many
SELECT ts_code, symbol, name, area, industry, market, exchange, list_date, delist_date, is_active, source, updated_at
FROM stock_basic
WHERE ($1::text = '' OR ts_code ILIKE '%' || $1 || '%' OR symbol ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%')
ORDER BY ts_code
LIMIT $2 OFFSET $3;

-- name: GetStock :one
SELECT ts_code, symbol, name, area, industry, market, exchange, list_date, delist_date, is_active, source, updated_at
FROM stock_basic
WHERE ts_code = $1;

-- name: ListStockDaily :many
SELECT ts_code, trade_date, open, high, low, close, pre_close, change, pct_chg, vol, amount, source, data_status
FROM stock_daily
WHERE ts_code = $1 AND trade_date BETWEEN $2 AND $3
ORDER BY trade_date;
