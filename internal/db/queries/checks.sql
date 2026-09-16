-- name: UpsertCheck :exec
INSERT INTO
  checks (monitor_id, status_code, response_time, days_remaining, error, is_up, checked_at)
VALUES
  (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (monitor_id, checked_at) DO UPDATE
SET
  status_code = excluded.status_code,
  response_time = excluded.response_time,
  days_remaining = excluded.days_remaining,
  error = excluded.error,
  is_up = excluded.is_up;

-- name: CleanupChecks :exec
DELETE FROM checks
WHERE
  checked_at < sqlc.arg (cutoff);

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
  monitor_id,
  checked_at - (checked_at % sqlc.arg (step)) AS ts,
  COUNT(*) AS total,
  CAST(
    SUM(
      CASE
        WHEN is_up
        AND response_time <= sqlc.arg (degraded_threshold) THEN 1
        ELSE 0
      END
    ) AS INTEGER
  ) AS up,
  CAST(
    SUM(
      CASE
        WHEN is_up
        AND response_time > sqlc.arg (degraded_threshold) THEN 1
        ELSE 0
      END
    ) AS INTEGER
  ) AS degraded,
  CAST(
    SUM(
      CASE
        WHEN NOT is_up THEN 1
        ELSE 0
      END
    ) AS INTEGER
  ) AS down,
  CAST(SUM(response_time) AS INTEGER) AS sum_ms
FROM
  checks
WHERE
  checked_at >= sqlc.arg (from_ts)
GROUP BY
  monitor_id,
  ts
ORDER BY
  monitor_id,
  ts;

-- name: GetCheckResponseTimes :many
SELECT
  monitor_id,
  response_time
FROM
  checks
WHERE
  checked_at >= sqlc.arg (from_ts)
  AND is_up
ORDER BY
  monitor_id,
  response_time;
