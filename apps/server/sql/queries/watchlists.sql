-- name: ListWatchlistGroups :many
SELECT id, user_id, name, sort_order, created_at, updated_at
FROM watchlist_group
WHERE user_id = $1
ORDER BY sort_order ASC, created_at ASC, id ASC;

-- name: GetWatchlistGroup :one
SELECT id, user_id, name, sort_order, created_at, updated_at
FROM watchlist_group
WHERE id = $1 AND user_id = $2;

-- name: CreateWatchlistGroup :one
INSERT INTO watchlist_group (
    user_id,
    name,
    sort_order
) VALUES (
    $1,
    $2,
    $3
)
RETURNING id, user_id, name, sort_order, created_at, updated_at;

-- name: UpdateWatchlistGroup :one
UPDATE watchlist_group
SET name = $3,
    sort_order = $4,
    updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, name, sort_order, created_at, updated_at;

-- name: DeleteWatchlistGroup :execrows
DELETE FROM watchlist_group
WHERE id = $1 AND user_id = $2;

-- name: ListWatchlistItems :many
SELECT i.id, i.group_id, i.ts_code, i.note, i.created_at
FROM watchlist_item AS i
INNER JOIN watchlist_group AS g ON g.id = i.group_id
WHERE i.group_id = $1 AND g.user_id = $2
ORDER BY i.created_at ASC, i.id ASC;

-- name: CreateWatchlistItem :one
INSERT INTO watchlist_item (
    group_id,
    ts_code,
    note
)
SELECT g.id, $2, $3
FROM watchlist_group AS g
WHERE g.id = $1 AND g.user_id = $4
RETURNING id, group_id, ts_code, note, created_at;

-- name: DeleteWatchlistItem :execrows
DELETE FROM watchlist_item AS i
USING watchlist_group AS g
WHERE i.id = $1
  AND i.group_id = $2
  AND g.id = i.group_id
  AND g.user_id = $3;
