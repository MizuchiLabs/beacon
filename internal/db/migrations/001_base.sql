-- +goose Up
-- Monitors 
CREATE TABLE monitors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  url TEXT NOT NULL UNIQUE,
  check_interval INTEGER NOT NULL DEFAULT 60, -- in seconds
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Checks are individual check results
CREATE TABLE checks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  monitor_id INTEGER NOT NULL,
  status_code INTEGER,
  response_time INTEGER, -- in ms
  error TEXT,
  is_up BOOLEAN NOT NULL,
  checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (monitor_id) REFERENCES monitors (id) ON DELETE CASCADE
);

-- Incidents track downtime periods
CREATE TABLE incidents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  monitor_id INTEGER NOT NULL,
  reason TEXT,
  resolved_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (monitor_id) REFERENCES monitors (id) ON DELETE CASCADE
);

-- Browser notification subscriptions
CREATE TABLE push_subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  monitor_id INTEGER NOT NULL,
  endpoint TEXT NOT NULL,
  p256dh_key TEXT NOT NULL, -- encryption key
  auth_key TEXT NOT NULL, -- authentication secret
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (monitor_id) REFERENCES monitors (id) ON DELETE CASCADE
);

-- VAPID keys for push notifications (single row table)
CREATE TABLE vapid_keys (
  id INTEGER PRIMARY KEY CHECK (id = 1), -- ensure only one row
  public_key TEXT NOT NULL,
  private_key TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- indexes for common queries
CREATE INDEX idx_checks_monitor_id ON checks (monitor_id, checked_at DESC);

CREATE INDEX idx_checks_checked_at ON checks (checked_at);

CREATE INDEX idx_incidents_monitor_id ON incidents (monitor_id);

CREATE INDEX idx_incidents_active ON incidents (monitor_id, resolved_at)
WHERE
  resolved_at IS NULL;

CREATE UNIQUE INDEX idx_push_sub_endpoint ON push_subscriptions (endpoint, monitor_id);

CREATE INDEX idx_push_sub_monitor ON push_subscriptions (monitor_id);

-- +goose Down
DROP TABLE IF EXISTS monitors;

DROP TABLE IF EXISTS checks;

DROP TABLE IF EXISTS incidents;

DROP TABLE IF EXISTS push_subscriptions;

DROP TABLE IF EXISTS vapid_keys;
