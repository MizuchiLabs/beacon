-- name: UpsertCheck :exec
INSERT INTO
  checks (monitor_id, status_code, response_time, error, is_up, checked_at)
VALUES
  (?, ?, ?, ?, ?, ?)
ON CONFLICT (monitor_id, checked_at) DO UPDATE
SET
  status_code = excluded.status_code,
  response_time = excluded.response_time,
  error = excluded.error,
  is_up = excluded.is_up;

-- name: CleanupChecks :exec
DELETE FROM checks
WHERE
  checked_at < sqlc.arg (cutoff);

-- name: GetMonitorStats :many
SELECT
  m.id,
  m.name,
  m.url,
  m.check_interval,
  CAST(
    ROUND(
      COALESCE(
        SUM(
          CASE
            WHEN c.is_up THEN 1
            ELSE 0
          END
        ) * 100.0 / COUNT(c.monitor_id),
        100.0
      ),
      2
    ) AS REAL
  ) AS uptime_pct,
  CAST(COALESCE(AVG(c.response_time), 0.0) AS INTEGER) AS avg_response_time
FROM
  monitors m
  LEFT JOIN checks c ON c.monitor_id = m.id
  AND c.checked_at >= sqlc.arg (since)
GROUP BY
  m.id
ORDER BY
  m.id;

-- name: GetDataPoints :many
SELECT
  monitor_id,
  checked_at - (checked_at % sqlc.arg (bucket_size)) AS bucket_ts,
  COUNT(*) AS total_count,
  CAST(COALESCE(AVG(response_time), 0.0) AS INTEGER) AS avg_response_time,
  CAST(
    SUM(
      CASE
        WHEN is_up
        AND response_time <= sqlc.arg (degraded_threshold) THEN 1
        ELSE 0
      END
    ) AS INTEGER
  ) AS up_count,
  CAST(
    SUM(
      CASE
        WHEN is_up
        AND response_time > sqlc.arg (degraded_threshold) THEN 1
        ELSE 0
      END
    ) AS INTEGER
  ) AS degraded_count,
  CAST(
    SUM(
      CASE
        WHEN NOT is_up THEN 1
        ELSE 0
      END
    ) AS INTEGER
  ) AS down_count
FROM
  checks
WHERE
  checked_at >= sqlc.arg (since)
GROUP BY
  monitor_id,
  bucket_ts
ORDER BY
  monitor_id,
  bucket_ts;

-- name: GetResponseTimes :many
SELECT
  monitor_id,
  response_time
FROM
  checks
WHERE
  checked_at >= sqlc.arg (since)
  AND is_up = 1;
