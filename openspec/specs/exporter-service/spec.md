# exporter-service Specification

## Purpose

Defines the exporter as a stateless, single-binary Go service: its HTTP
surface, CLI flags, logging behavior, and graceful shutdown.

## Requirements

### Requirement: Stateless single-binary service
The exporter SHALL be a single statically linked Go binary configured exclusively via CLI flags. It SHALL NOT read configuration files, write to disk, or keep per-target state in memory between requests.

#### Scenario: Run with no arguments
- **WHEN** the binary is started with no flags
- **THEN** it listens on the default address `:9642` and serves requests using built-in defaults

### Requirement: HTTP surface
The exporter SHALL expose: `/` — an HTML landing page linking to the endpoints and showing the version; `/metrics` — the multi-target device scrape endpoint; `/-/metrics` — exporter self-metrics (Go runtime and process metrics plus exporter scrape counters); `/-/healthy` — liveness endpoint returning HTTP 200.

#### Scenario: Landing page
- **WHEN** `GET /` is requested
- **THEN** an HTML page with links to `/metrics`, `/-/metrics`, and `/-/healthy` is returned

#### Scenario: Health check
- **WHEN** `GET /-/healthy` is requested
- **THEN** the exporter responds HTTP 200 regardless of device availability

#### Scenario: Self-metrics separation
- **WHEN** `GET /-/metrics` is requested
- **THEN** the response contains Go runtime metrics and no `meshcoretel_*` device metrics

### Requirement: CLI flags
The exporter SHALL support at minimum: `--web.listen-address` (default `:9642`), `--scrape.timeout` (default `10s`), `--log.level` (`debug|info|warn|error`, default `info`), `--log.format` (`text|json`), and `--version` printing version, commit, and build date injected at build time via ldflags.

#### Scenario: Version output
- **WHEN** the binary is invoked with `--version`
- **THEN** it prints the semantic version, git commit, and build date, then exits 0

### Requirement: Structured logging
The exporter SHALL log via `log/slog` with the configured level and format: one startup line (version, listen address), one warning per failed device scrape (target and error class), and no logging of passwords, tokens, or full query strings.

#### Scenario: Startup log
- **WHEN** the exporter starts
- **THEN** it logs an info entry containing the version and listen address

### Requirement: Graceful shutdown
On SIGINT or SIGTERM the exporter SHALL stop accepting new connections, allow in-flight scrapes up to a bounded drain period, then exit 0.

#### Scenario: SIGTERM during scrape
- **WHEN** SIGTERM arrives while a `/metrics` request is in flight
- **THEN** the in-flight response completes (within the drain period) and the process exits 0
