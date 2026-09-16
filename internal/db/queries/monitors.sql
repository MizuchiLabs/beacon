-- name: CreateMonitor :one
INSERT INTO
  monitors (name, url, type, ignore_cert_expiry, check_interval)
VALUES
  (?, ?, ?, ?, ?) RETURNING *;

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
  type = COALESCE(?, type),
  ignore_cert_expiry = COALESCE(?, ignore_cert_expiry),
  check_interval = COALESCE(?, check_interval)
WHERE
  id = ? RETURNING *;

-- name: DeleteMonitor :exec
DELETE FROM monitors
WHERE
  id = ?;
