-- name: InsertCheck :exec
INSERT INTO
  checks (monitor_id, status_code, response_time, days_remaining, error, is_up, checked_at)
VALUES
  (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (monitor_id, checked_at) DO NOTHING;

-- name: CleanupChecks :exec
DELETE FROM checks
WHERE
  checked_at < sqlc.arg (cutoff);

-- name: CleanupRollups :exec
DELETE FROM check_rollups
WHERE
  hour < sqlc.arg (cutoff);

-- name: BackfillRollups :exec
-- Rolls up raw checks older than the first rollup. Only does work on the
-- first start after rollups were introduced, the trigger keeps up after that.
INSERT INTO
  check_rollups (monitor_id, hour, total, up, degraded, down, sum_ms)
SELECT
  c.monitor_id,
  c.checked_at - c.checked_at % 3600 AS hour,
  COUNT(*),
  SUM(c.is_up AND c.response_time <= m.degraded_threshold),
  SUM(c.is_up AND c.response_time > m.degraded_threshold),
  SUM(NOT c.is_up),
  SUM(IIF(c.is_up, c.response_time, 0))
FROM
  checks c
  JOIN monitors m ON m.id = c.monitor_id
WHERE
  c.checked_at < COALESCE(
    (
      SELECT
        MIN(hour)
      FROM
        check_rollups
    ),
    9223372036854775807
  )
GROUP BY
  c.monitor_id,
  hour
ON CONFLICT (monitor_id, hour) DO NOTHING;

-- name: GetLatestChecks :many
SELECT
  c.monitor_id,
  c.status_code,
  c.response_time,
  c.days_remaining,
  c.is_up,
  c.checked_at
FROM
  monitors m
  JOIN checks c ON c.monitor_id = m.id
WHERE
  c.checked_at = (
    SELECT
      MAX(c2.checked_at)
    FROM
      checks c2
    WHERE
      c2.monitor_id = m.id
  );

-- name: GetCheckWindow :many
SELECT
  c.monitor_id,
  c.checked_at - (c.checked_at % sqlc.arg (step)) AS ts,
  COUNT(*) AS total,
  CAST(SUM(c.is_up AND c.response_time <= m.degraded_threshold) AS INTEGER) AS up,
  CAST(SUM(c.is_up AND c.response_time > m.degraded_threshold) AS INTEGER) AS degraded,
  CAST(SUM(NOT c.is_up) AS INTEGER) AS down,
  CAST(SUM(IIF(c.is_up, c.response_time, 0)) AS INTEGER) AS sum_ms
FROM
  checks c
  JOIN monitors m ON m.id = c.monitor_id
WHERE
  c.checked_at >= sqlc.arg (from_ts)
GROUP BY
  c.monitor_id,
  ts
ORDER BY
  c.monitor_id,
  ts;

-- name: GetRollupWindow :many
SELECT
  monitor_id,
  hour - (hour % sqlc.arg (step)) AS ts,
  CAST(SUM(total) AS INTEGER) AS total,
  CAST(SUM(up) AS INTEGER) AS up,
  CAST(SUM(degraded) AS INTEGER) AS degraded,
  CAST(SUM(down) AS INTEGER) AS down,
  CAST(SUM(sum_ms) AS INTEGER) AS sum_ms
FROM
  check_rollups
WHERE
  hour >= sqlc.arg (from_ts)
GROUP BY
  monitor_id,
  ts
ORDER BY
  monitor_id,
  ts;

-- name: GetUptime :one
SELECT
  CAST(COALESCE(SUM(total), 0) AS INTEGER) AS total,
  CAST(COALESCE(SUM(up + degraded), 0) AS INTEGER) AS alive
FROM
  check_rollups
WHERE
  monitor_id = sqlc.arg (monitor_id)
  AND hour >= sqlc.arg (from_ts);

-- name: GetResponseTimes :many
SELECT
  response_time
FROM
  checks
WHERE
  monitor_id = sqlc.arg (monitor_id)
  AND checked_at >= sqlc.arg (from_ts)
  AND is_up
ORDER BY
  response_time;
