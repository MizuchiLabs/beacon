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

-- name: CleanupChecks :exec
DELETE FROM checks
WHERE
  checked_at < datetime('now', '-' || sqlc.arg (days) || ' days');

-- name: GetMonitorStats :many
SELECT
  m.id,
  m.name,
  m.url,
  m.check_interval,
  COUNT(c.id) AS total_checks,
  CAST(
    ROUND(
      COALESCE(
        SUM(
          CASE
            WHEN c.is_up THEN 1
            ELSE 0
          END
        ) * 100.0 / COUNT(c.id),
        100.0
      ),
      2
    ) AS REAL
  ) AS uptime_pct,
  CAST(COALESCE(AVG(c.response_time), 0.0) AS INTEGER) AS avg_response_time
FROM
  monitors m
  LEFT JOIN checks c ON c.monitor_id = m.id
  AND c.checked_at >= datetime('now', '-' || sqlc.arg (seconds) || ' seconds')
GROUP BY
  m.id
ORDER BY
  m.id;

-- name: GetStatusDataPoints :many
SELECT
  monitor_id,
  CAST(
    CAST(strftime('%s', checked_at) AS INTEGER) / CAST(sqlc.arg (bucket_size) AS INTEGER) * CAST(sqlc.arg (bucket_size) AS INTEGER) AS INTEGER
  ) AS bucket_ts,
  COUNT(*) AS total_count,
  CAST(
    SUM(
      CASE
        WHEN NOT is_up THEN 1
        ELSE 0
      END
    ) AS REAL
  ) AS down_count,
  CAST(
    SUM(
      CASE
        WHEN is_up
        AND response_time > sqlc.arg (degraded_threshold) THEN 1
        ELSE 0
      END
    ) AS REAL
  ) AS degraded_count,
  CAST(
    SUM(
      CASE
        WHEN is_up
        AND (
          response_time IS NULL
          OR response_time <= sqlc.arg (degraded_threshold)
        ) THEN 1
        ELSE 0
      END
    ) AS REAL
  ) AS up_count
FROM
  checks
WHERE
  checked_at >= sqlc.arg (since)
  AND checked_at IS NOT NULL
GROUP BY
  monitor_id,
  bucket_ts
HAVING
  bucket_ts IS NOT NULL
ORDER BY
  monitor_id,
  bucket_ts;

-- name: GetTimeSeriesDataPoints :many
SELECT
  monitor_id,
  CAST(
    CAST(strftime('%s', checked_at) AS INTEGER) / CAST(sqlc.arg (bucket_size) AS INTEGER) * CAST(sqlc.arg (bucket_size) AS INTEGER) AS INTEGER
  ) AS bucket_ts,
  CAST(COALESCE(AVG(response_time), 0.0) AS INTEGER) AS avg_response_time,
  CAST(
    SUM(
      CASE
        WHEN is_up THEN 1
        ELSE 0
      END
    ) AS REAL
  ) AS up_count,
  COUNT(*) AS total_count
FROM
  checks
WHERE
  checked_at >= sqlc.arg (since)
  AND checked_at IS NOT NULL
GROUP BY
  monitor_id,
  bucket_ts
HAVING
  bucket_ts IS NOT NULL
ORDER BY
  monitor_id,
  bucket_ts;

-- name: GetPercentiles :many
WITH
  ordered AS (
    SELECT
      monitor_id,
      response_time,
      PERCENT_RANK() OVER (
        PARTITION BY
          monitor_id
        ORDER BY
          response_time
      ) AS pct
    FROM
      checks
    WHERE
      checked_at >= sqlc.arg (since)
      AND is_up = 1
      AND response_time IS NOT NULL
  )
SELECT
  monitor_id,
  CAST(
    MAX(
      CASE
        WHEN pct <= 0.50 THEN response_time
      END
    ) AS INTEGER
  ) AS p50,
  CAST(
    MAX(
      CASE
        WHEN pct <= 0.75 THEN response_time
      END
    ) AS INTEGER
  ) AS p75,
  CAST(
    MAX(
      CASE
        WHEN pct <= 0.90 THEN response_time
      END
    ) AS INTEGER
  ) AS p90,
  CAST(
    MAX(
      CASE
        WHEN pct <= 0.95 THEN response_time
      END
    ) AS INTEGER
  ) AS p95,
  CAST(
    MAX(
      CASE
        WHEN pct <= 0.99 THEN response_time
      END
    ) AS INTEGER
  ) AS p99
FROM
  ordered
GROUP BY
  monitor_id;
