-- name: UpsertMonitor :one
INSERT INTO
  monitors (
    name,
    url,
    type,
    group_name,
    check_interval,
    retries,
    degraded_threshold,
    expected_status,
    keyword,
    ignore_cert_expiry
  )
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (name) DO UPDATE
SET
  url = excluded.url,
  type = excluded.type,
  group_name = excluded.group_name,
  check_interval = excluded.check_interval,
  retries = excluded.retries,
  degraded_threshold = excluded.degraded_threshold,
  expected_status = excluded.expected_status,
  keyword = excluded.keyword,
  ignore_cert_expiry = excluded.ignore_cert_expiry RETURNING *;

-- name: GetMonitors :many
SELECT
  *
FROM
  monitors
ORDER BY
  id;

-- name: DeleteMonitor :exec
DELETE FROM monitors
WHERE
  id = ?;
