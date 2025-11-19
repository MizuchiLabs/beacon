-- name: CreateMonitor :one
INSERT INTO
  monitors (name, url, check_interval)
VALUES
  (?, ?, ?) RETURNING *;

-- name: GetMonitor :one
SELECT
  *
FROM
  monitors
WHERE
  id = ?;

-- name: GetMonitors :many
SELECT
  *
FROM
  monitors;

-- name: UpdateMonitor :one
UPDATE monitors
SET
  name = COALESCE(?, name),
  url = COALESCE(?, url),
  check_interval = COALESCE(?, check_interval)
WHERE
  id = ? RETURNING *;

-- name: DeleteMonitor :exec
DELETE FROM monitors
WHERE
  id = ?;
