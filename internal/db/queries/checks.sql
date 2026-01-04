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
  (?, ?, ?, ?, ?)
RETURNING
  *;

-- name: GetChecks :many
SELECT
  *
FROM
  checks
WHERE
  checked_at >= datetime('now', '-' || sqlc.arg (seconds) || ' seconds')
ORDER BY
  checked_at DESC;

-- name: CleanupChecks :exec
DELETE FROM checks
WHERE
  checked_at < datetime('now', '-' || sqlc.arg (days) || ' days');
