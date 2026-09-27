CREATE TABLE monitors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  url TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT 'http', -- http, tcp, ssl, dns, ping, push
  group_name TEXT NOT NULL DEFAULT '',
  check_interval INTEGER NOT NULL DEFAULT 60, -- in seconds
  retries INTEGER NOT NULL DEFAULT 1, -- extra attempts before a check counts as down
  degraded_threshold INTEGER NOT NULL DEFAULT 500, -- ms, slower up checks count as degraded
  expected_status INTEGER NOT NULL DEFAULT 0, -- http: required status, 0 accepts any 2xx or 3xx
  keyword TEXT NOT NULL DEFAULT '', -- http: text the response body must contain
  ignore_cert_expiry BOOLEAN NOT NULL DEFAULT 0, -- keep cert expiry from degrading status or sending warnings
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE checks (
  monitor_id INTEGER NOT NULL REFERENCES monitors (id) ON DELETE CASCADE,
  status_code INTEGER NOT NULL,
  response_time INTEGER NOT NULL, -- in ms
  days_remaining INTEGER, -- ssl: days until certificate expiry, null otherwise
  error TEXT,
  is_up BOOLEAN NOT NULL,
  checked_at INTEGER NOT NULL DEFAULT (unixepoch()), -- unix seconds
  PRIMARY KEY (monitor_id, checked_at)
);

-- Hourly totals of the checks table. Raw checks are cleaned up early, these
-- stay around much longer and back every window of a day or more.
CREATE TABLE check_rollups (
  monitor_id INTEGER NOT NULL REFERENCES monitors (id) ON DELETE CASCADE,
  hour INTEGER NOT NULL, -- unix seconds at the start of the hour
  total INTEGER NOT NULL,
  up INTEGER NOT NULL, -- up and faster than the degraded threshold
  degraded INTEGER NOT NULL, -- up but slower than the degraded threshold
  down INTEGER NOT NULL,
  sum_ms INTEGER NOT NULL, -- response time sum of up and degraded checks
  PRIMARY KEY (monitor_id, hour)
) WITHOUT ROWID;

-- Browser notification subscriptions
CREATE TABLE push_subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  monitor_id INTEGER NOT NULL REFERENCES monitors (id) ON DELETE CASCADE,
  endpoint TEXT NOT NULL,
  p256dh_key TEXT NOT NULL, -- encryption key
  auth_key TEXT NOT NULL, -- authentication secret
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

-- VAPID keys for push notifications (singleton)
CREATE TABLE vapid_keys (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  public_key TEXT NOT NULL,
  private_key TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX idx_checks_checked_at ON checks (checked_at);

CREATE INDEX idx_checks_monitor_time_up ON checks (monitor_id, checked_at, is_up, response_time);

CREATE INDEX idx_check_rollups_hour ON check_rollups (hour);

CREATE UNIQUE INDEX idx_push_sub_endpoint ON push_subscriptions (endpoint, monitor_id);

CREATE INDEX idx_push_sub_monitor ON push_subscriptions (monitor_id);

-- Every stored check is folded into its hourly rollup in the same statement,
-- so the rollups can never drift from the raw checks.
CREATE TRIGGER checks_rollup AFTER INSERT ON checks
BEGIN
  INSERT INTO
    check_rollups (monitor_id, hour, total, up, degraded, down, sum_ms)
  SELECT
    NEW.monitor_id,
    NEW.checked_at - NEW.checked_at % 3600,
    1,
    NEW.is_up AND NEW.response_time <= m.degraded_threshold,
    NEW.is_up AND NEW.response_time > m.degraded_threshold,
    NOT NEW.is_up,
    IIF(NEW.is_up, NEW.response_time, 0)
  FROM
    monitors m
  WHERE
    m.id = NEW.monitor_id
  ON CONFLICT (monitor_id, hour) DO UPDATE
  SET
    total = total + 1,
    up = up + excluded.up,
    degraded = degraded + excluded.degraded,
    down = down + excluded.down,
    sum_ms = sum_ms + excluded.sum_ms;
END;
