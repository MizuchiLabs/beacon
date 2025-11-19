-- name: GetCheckStats :many
SELECT
  hour_timestamp,
  total_checks,
  successful_checks,
  avg_response_time
FROM
  check_stats
WHERE
  monitor_id = ?
  AND hour_timestamp >= datetime ('now', ?)
ORDER BY
  hour_timestamp ASC;

-- name: UpsertCheckStats :exec
INSERT INTO
  check_stats (
    monitor_id,
    hour_timestamp,
    total_checks,
    successful_checks,
    avg_response_time
  )
VALUES
  (?, ?, ?, ?, ?) ON CONFLICT (monitor_id, hour_timestamp) DO
UPDATE
SET
  total_checks = total_checks + excluded.total_checks,
  successful_checks = successful_checks + excluded.successful_checks,
  avg_response_time = (avg_response_time + excluded.avg_response_time) / 2;
