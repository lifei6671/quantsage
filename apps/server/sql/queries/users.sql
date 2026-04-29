-- name: GetUserByID :one
SELECT id, username, display_name, password_hash, status, role, last_login_at, created_at, updated_at
FROM app_user
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, username, display_name, password_hash, status, role, last_login_at, created_at, updated_at
FROM app_user
WHERE username = $1;

-- name: CreateUser :one
INSERT INTO app_user (
    username,
    display_name,
    password_hash,
    status,
    role
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING id, username, display_name, password_hash, status, role, last_login_at, created_at, updated_at;

-- name: UpdateBootstrapUser :one
UPDATE app_user
SET display_name = $2,
    password_hash = $3,
    status = $4,
    role = $5,
    updated_at = now()
WHERE username = $1
RETURNING id, username, display_name, password_hash, status, role, last_login_at, created_at, updated_at;

-- name: TouchUserLastLogin :exec
UPDATE app_user
SET last_login_at = $2,
    updated_at = now()
WHERE id = $1;
