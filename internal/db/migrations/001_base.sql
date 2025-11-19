-- +goose Up
-- monitors table stores sites being monitored
CREATE TABLE monitors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  url TEXT NOT NULL UNIQUE,
  check_interval INTEGER NOT NULL DEFAULT 60, -- in seconds
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- checks table stores individual check results
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

-- incidents table tracks downtime periods
CREATE TABLE incidents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  monitor_id INTEGER NOT NULL,
  reason TEXT,
  resolved_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (monitor_id) REFERENCES monitors (id) ON DELETE CASCADE
);

-- indexes for common queries
CREATE INDEX idx_checks_monitor_id ON checks (monitor_id, checked_at DESC);

CREATE INDEX idx_checks_checked_at ON checks (checked_at);

CREATE INDEX idx_incidents_monitor_id ON incidents (monitor_id);

CREATE INDEX idx_incidents_active ON incidents (monitor_id, resolved_at)
WHERE
  resolved_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS monitors;

DROP TABLE IF EXISTS checks;

DROP TABLE IF EXISTS incidents;
