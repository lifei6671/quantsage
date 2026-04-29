-- name: ListUserPositions :many
SELECT id, user_id, ts_code, position_date, quantity, cost_price, note, created_at, updated_at
FROM user_position
WHERE user_id = $1
ORDER BY position_date DESC, id DESC;

-- name: CreateUserPosition :one
INSERT INTO user_position (
    user_id,
    ts_code,
    position_date,
    quantity,
    cost_price,
    note
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING id, user_id, ts_code, position_date, quantity, cost_price, note, created_at, updated_at;

-- name: UpdateUserPosition :one
UPDATE user_position
SET ts_code = $3,
    position_date = $4,
    quantity = $5,
    cost_price = $6,
    note = $7,
    updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, ts_code, position_date, quantity, cost_price, note, created_at, updated_at;

-- name: DeleteUserPosition :execrows
DELETE FROM user_position
WHERE id = $1 AND user_id = $2;
