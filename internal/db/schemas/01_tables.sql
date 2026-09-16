CREATE TABLE monitors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  url TEXT NOT NULL UNIQUE,
  type TEXT NOT NULL DEFAULT 'http', -- http, tcp, ssl
  check_interval INTEGER NOT NULL DEFAULT 60, -- in seconds
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

CREATE UNIQUE INDEX idx_push_sub_endpoint ON push_subscriptions (endpoint, monitor_id);

CREATE INDEX idx_push_sub_monitor ON push_subscriptions (monitor_id);
