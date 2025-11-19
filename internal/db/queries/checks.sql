-- name: CreateCheck :one
INSERT INTO
  checks (
    monitor_id,
    status_code,
    response_time,
    error,
    is_up
  )
VALUES
  (?, ?, ?, ?, ?) RETURNING *;

-- name: GetCheck :one
SELECT
  *
FROM
  checks
WHERE
  id = ?;

-- name: GetChecks :many
SELECT
  *
FROM
  checks
WHERE
  monitor_id = ?;

-- name: UpdateCheck :one
UPDATE checks
SET
  status_code = COALESCE(?, status_code),
  response_time = COALESCE(?, response_time),
  error = COALESCE(?, error),
  is_up = COALESCE(?, is_up)
WHERE
  id = ? RETURNING *;

-- name: DeleteCheck :exec
DELETE FROM checks
WHERE
  id = ?;
