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
  monitor_id = ?
ORDER BY
  checked_at DESC
LIMIT
  ?;

-- name: CleanupChecks :exec
DELETE FROM checks
WHERE
  checked_at < ?;
