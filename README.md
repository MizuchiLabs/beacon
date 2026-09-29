<p align="center">
<img src="./.github/logo.svg" width="80">
<br><br>
<img alt="GitHub Tag" src="https://img.shields.io/github/v/tag/MizuchiLabs/beacon?label=Version">
<img alt="GitHub License" src="https://img.shields.io/github/license/MizuchiLabs/beacon">
<img alt="GitHub Issues or Pull Requests" src="https://img.shields.io/github/issues/MizuchiLabs/beacon">
</p>

# Beacon

A lightweight, self-hosted uptime monitoring solution that keeps track of your websites and services.

## Features

- HTTP/HTTPS, TCP, TLS certificate, DNS and ping checks
- Heartbeat monitors for cron jobs and backups
- Retries, expected status codes and keyword checks
- Alerts through browser push and generic webhooks (Discord, ntfy, Slack, ...)
- Uptime history up to a year, response times and percentiles
- Incidents from YAML files or a git repository
- Status and uptime badges for your READMEs
- Config reloads on save, no restart needed
- One binary with SQLite, Docker-ready

## Quick Start

### Using Docker

```bash
docker run -d \
   -p 3000:3000 \
   -v $(pwd)/data:/data \
   -e BEACON_DATA_DIR=/data \
   -e BEACON_CONFIG=/data/config.yaml \
   ghcr.io/mizuchilabs/beacon:latest
```

### Using Binary

```bash
beacon --config config.yaml
```

## Configuration

Create a `config.yaml`. Only `name` and `url` are required:

```yaml
monitors:
  - name: "My Website"
    url: "https://example.com"
  - name: "API"
    url: "https://api.example.com/health"
    group: "Backend"
    check_interval: 30
    expected_status: 200
    keyword: "ok"
  - name: "Postgres"
    url: "tcp://db.example.com:5432"
    group: "Backend"
    retries: 2
  - name: "Pi-hole"
    url: "dns://example.com?server=192.168.1.2"
  - name: "Router"
    url: "ping://192.168.1.1"
  - name: "Nightly backup"
    url: "push://some-long-random-token"
    check_interval: 86400
```

Beacon watches the file and applies changes a few seconds after you save it.
Monitors are matched by name, so changing a url keeps the history. Renaming
a monitor starts a fresh one.

| Field                | Default     | Description                                                     |
| -------------------- | ----------- | --------------------------------------------------------------- |
| `name`               | -           | Shown on the dashboard, must be unique                          |
| `url`                | -           | What to check, the scheme picks the check type                  |
| `group`              | -           | Monitors with the same group are shown together                 |
| `check_interval`     | `60`        | Seconds between checks, 10 to 86400                             |
| `retries`            | `1`         | Extra attempts, 5s apart, before a check counts as down         |
| `degraded_threshold` | `500`       | Response time in ms above which an up check is degraded         |
| `expected_status`    | any 2xx/3xx | http: the exact status code that counts as up                   |
| `keyword`            | -           | http: text the response body must contain                       |
| `ignore_cert_expiry` | `false`     | Keep certificate expiry from degrading status or sending alerts |

| Scheme         | Check                                                                  |
| -------------- | ---------------------------------------------------------------------- |
| `http`/`https` | HTTP GET, up on any 2xx or 3xx status                                  |
| `tcp`          | Plain TCP connect to `host:port`, port is required                     |
| `ssl`          | TLS handshake and certificate validity, port defaults to 443           |
| `dns`          | Resolves the host, `?server=` sends the query to a specific DNS server |
| `ping`         | One ICMP echo, see the note below                                      |
| `push`         | Waits for heartbeats instead of checking anything, see below           |

`https` monitors track certificate expiry automatically, no second monitor
needed. Any monitor with certificate data turns degraded 30 days before expiry
and subscribers get a daily reminder until it is renewed.

Ping uses unprivileged ICMP sockets, so no root is needed. Docker allows them
by default. On a bare Linux host the user running Beacon must be inside
`net.ipv4.ping_group_range`.

For containers you can also pass the whole config as a YAML string in
`BEACON_MONITORS`. It is not reloaded on change.

### Heartbeats

A `push://<token>` monitor goes down when it hears nothing for one and a half
intervals. Ping it from the end of your job:

```bash
./backup.sh && curl -fsS https://status.example.com/api/push/<token>
# Or report a failure with a reason
curl -fsS "https://status.example.com/api/push/<token>?status=down&msg=disk%20full"
```

The token works like a password. It is never shown on the dashboard.

### Webhooks

Every alert is POSTed to each webhook. Without a `body` the event is sent as
JSON:

```json
{
  "event": "down",
  "monitor": "API",
  "url": "https://api.example.com",
  "title": "🔴 API is down",
  "message": "API is down: HTTP 502",
  "time": "..."
}
```

`event` is `down`, `up`, `cert_expiry` or `test`. A `body` is a Go template
over those fields, `{{json .Message}}` quotes a value for JSON:

```yaml
webhooks:
  - url: "https://discord.com/api/webhooks/..."
    body: '{"content": {{json .Message}}}'
  - url: "https://hooks.slack.com/services/..."
    body: '{"text": {{json .Message}}}'
  - url: "https://ntfy.sh/my-beacon-alerts"
    headers:
      Title: "Beacon"
    body: "{{.Message}}"
```

Run `beacon test-notify` to send a test event to all of them.

### Badges

```markdown
![API](https://status.example.com/api/badge/API)
![API uptime](https://status.example.com/api/badge/API?kind=uptime)
```

The uptime badge covers the last 30 days, change it with `&seconds=`.

## Environment Variables

| Variable                 | Default             | Description                                         |
| ------------------------ | ------------------- | --------------------------------------------------- |
| `BEACON_PORT`            | `3000`              | Server port                                         |
| `BEACON_CONFIG`          | `config.yaml`       | Path to the config file                             |
| `BEACON_MONITORS`        | -                   | Config as a YAML string, instead of the file        |
| `BEACON_DATA_DIR`        | `data`              | Directory for `beacon.db` and local incident files  |
| `BEACON_TIMEOUT`         | `30s`               | Check timeout for every check type                  |
| `BEACON_INSECURE`        | `false`             | Skip TLS certificate verification                   |
| `BEACON_RETENTION_DAYS`  | `30`                | Days to keep raw checks, percentiles use these      |
| `BEACON_HISTORY_DAYS`    | `365`               | Days to keep the hourly history behind long windows |
| `BEACON_TITLE`           | `Beacon`            | Dashboard title                                     |
| `BEACON_LOGO_URL`        | -                   | Custom logo shown in the header                     |
| `BEACON_PUSH_SUBSCRIBER` | `mailto:beacon@...` | VAPID contact sent with push notifications          |
| `BEACON_DEBUG`           | `false`             | Enable debug logging                                |
| `BEACON_TRUSTED_PROXIES` | `direct`            | Client IP source for rate limiting, see below       |

Beacon backs up `beacon.db` to `beacon.db.backup` before it changes the schema
on an upgrade.

`BEACON_TRUSTED_PROXIES` controls how client IPs are resolved for the API rate
limit. With the default `direct`, the TCP peer address is used, which cannot be
spoofed. Behind a reverse proxy, set it so each real client gets its own rate
limit bucket instead of sharing the proxy's:

- `cloudflare`: read `CF-Connecting-IP`
- `nginx` or `traefik`: read `X-Real-IP`
- CIDR list, e.g. `10.0.0.0/8,172.16.0.0/12`: resolve via `X-Forwarded-For`
  from those trusted hops

Only set this when the proxy actually sets the matching header, otherwise
clients can spoof their IP and dodge the rate limit. Header modes also assume
the port is not reachable without going through the proxy.

## Docker Compose Example

```yaml
services:
  beacon:
    image: ghcr.io/mizuchilabs/beacon:latest
    container_name: beacon
    ports:
      - "3000:3000"
    volumes:
      - ./data:/data
      - ./config.yaml:/config.yaml:ro
    environment:
      - BEACON_DATA_DIR=/data
      - BEACON_CONFIG=/config.yaml
      - BEACON_TITLE=My Status Page
      - TZ=America/New_York
    restart: unless-stopped
```

## Incidents

Drop incident files into `data/incidents` (inside `BEACON_DATA_DIR`) and they
show up within a few minutes:

```yaml
title: Database Connection Issues
description: Users experiencing intermittent connection errors
severity: major # minor, major, critical, maintenance
status: resolved # investigating, identified, monitoring, resolved
affected_monitors:
  - My Website
  - API
started_at: 2025-01-15T14:30:00Z
resolved_at: 2025-01-15T16:45:00Z
updates:
  - message: Investigating connection timeouts
    status: investigating
    created_at: 2025-01-15T14:30:00Z
  - message: All systems operational
    status: resolved
    created_at: 2025-01-15T16:45:00Z
```

The file name is the incident id unless the file sets `id`. Files that don't
parse are skipped with a warning in the log.

To keep incidents in git instead, point Beacon at the repository. It is cloned
into the incident directory and synced every few minutes:

| Variable               | Default                | Description                                                                        |
| ---------------------- | ---------------------- | ---------------------------------------------------------------------------------- |
| `BEACON_INCIDENT_REPO` | -                      | Git repository URL for incidents, private repos use `https://user:token@host/repo` |
| `BEACON_INCIDENT_PATH` | `<data dir>/incidents` | Local directory for incident files                                                 |
| `BEACON_INCIDENT_SYNC` | `5m`                   | How often to sync and reload                                                       |

## Keyboard shortcuts

`1` to `6` switch the time range, `/` filters the monitors.

## Screenshots

![Dashboard](./.github/screenshots/dashboard.png)
![Incidents](./.github/screenshots/incidents.png)

## License

Apache License 2.0 - See [LICENSE](LICENSE)
