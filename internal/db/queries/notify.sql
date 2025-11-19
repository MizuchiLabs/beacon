-- name: CreatePushSubscription :exec
INSERT INTO
  push_subscriptions (monitor_id, endpoint, p256dh_key, auth_key)
VALUES
  (?, ?, ?, ?);

-- name: DeletePushSubscription :exec
DELETE FROM push_subscriptions
WHERE
  endpoint = ?
  AND monitor_id = ?;

-- name: GetPushSubscriptionsByMonitor :many
SELECT
  id,
  monitor_id,
  endpoint,
  p256dh_key,
  auth_key,
  created_at
FROM
  push_subscriptions
WHERE
  monitor_id = ?;

-- name: DeletePushSubscriptionByEndpoint :exec
DELETE FROM push_subscriptions
WHERE
  endpoint = ?;

-- name: GetVAPIDKeys :one
SELECT
  id,
  public_key,
  private_key,
  created_at
FROM
  vapid_keys
WHERE
  id = 1;

-- name: CreateVAPIDKeys :exec
INSERT INTO
  vapid_keys (id, public_key, private_key)
VALUES
  (1, ?, ?);

-- name: VAPIDKeysExist :one
SELECT
  COUNT(*) as count
FROM
  vapid_keys
WHERE
  id = 1;
