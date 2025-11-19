-- name: CreateIncident :one
INSERT INTO
  incidents (monitor_id, reason)
VALUES
  (?, ?) RETURNING *;

-- name: GetIncident :one
SELECT
  *
FROM
  incidents
WHERE
  id = ?;

-- name: GetIncidents :many
SELECT
  *
FROM
  incidents
WHERE
  monitor_id = ?;

-- name: GetMonitorIncidents :many
SELECT
  *
FROM
  incidents
WHERE
  monitor_id = ?
ORDER BY
  created_at DESC
LIMIT
  ?;

-- name: UpdateIncident :one
UPDATE incidents
SET
  reason = COALESCE(?, reason)
WHERE
  id = ? RETURNING *;

-- name: ResolveIncident :one
UPDATE incidents
SET
  resolved_at = CURRENT_TIMESTAMP
WHERE
  id = ? RETURNING *;

-- name: DeleteIncident :exec
DELETE FROM incidents
WHERE
  id = ?;
