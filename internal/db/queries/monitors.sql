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

-- name: GetMonitorStatus :one
SELECT
  m.*,
  c.is_up,
  c.checked_at,
  c.response_time
FROM
  monitors m
  LEFT JOIN checks c ON c.id = (
    SELECT
      id
    FROM
      checks
    WHERE
      monitor_id = m.id
    ORDER BY
      checked_at DESC
    LIMIT
      1
  )
WHERE
  m.id = ?;

-- name: GetUptimeStats :one
SELECT
  COUNT(*) as total_checks,
  SUM(
    CASE
      WHEN is_up THEN 1
      ELSE 0
    END
  ) as successful_checks,
  AVG(response_time) as avg_response_time
FROM
  checks
WHERE
  monitor_id = ?
  AND checked_at > datetime ('now', '-24 hours');
