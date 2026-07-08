## Why

Devices running MeshCoreTel firmware (LoRa mesh repeaters) expose rich runtime statistics via an authenticated HTTPS JSON API, but there is no way to monitor them with Prometheus/Grafana today. This change creates `vbart-exporter` from scratch: a stateless Go exporter that translates the device's `/api/stats` JSON into flat Prometheus metrics on demand, following the proven `snmp_exporter` multi-target pattern.

## What Changes

- New Go application `vbart-exporter`: an HTTP server that, per scrape request (`GET /metrics?target=<ip>&password=<pass>`), logs into the target device (`POST /login`), fetches `GET /api/stats` with the returned `X-Auth-Token`, flattens the hierarchical JSON into Prometheus metrics, and returns them in the scrape response.
- Stateless by design: no configuration files, no persistent storage, no token caching between scrapes. All per-scrape parameters arrive via query string.
- Landing page on `/`, exporter self-metrics on `/-/metrics` (separate from proxied device metrics), health endpoint on `/-/healthy`.
- Multi-stage Dockerfile (Go builder → minimal static final image) and a `docker-compose.yml` demo stack (exporter, optionally Prometheus + Grafana).
- Documentation: README with usage, security notes on password-in-URL, Prometheus scrape config with `relabel_configs` (snmp_exporter style), and a metric reference table.
- Grafana dashboard JSON covering battery, radio, packet counters, memory, Wi-Fi, and neighbor metrics.
- GitHub Actions CI: test + lint (golangci-lint), multi-arch Docker image publish to ghcr.io, tagged releases (GoReleaser).
- License: GPL-3.0-only (compatible with dependencies: Go stdlib is BSD-3-Clause, prometheus/client_golang is Apache-2.0; both may be combined into a GPLv3 work).
- Unit tests: JSON flattener tested against `examples/stats.json`, HTTP handlers tested with `httptest` mock device.

## Capabilities

### New Capabilities

- `scrape-endpoint`: The `/metrics` multi-target HTTP endpoint — parameter handling, device login flow, error semantics (`up` gauge instead of HTTP errors), timeouts, and concurrency behavior.
- `metrics-translation`: Rules for flattening the device stats JSON into Prometheus metrics — naming, units, metric types, boolean/enum/string handling, per-neighbor labeled metrics.
- `exporter-service`: The exporter process itself — CLI flags, listen address, landing page, self-metrics, health endpoint, logging, graceful shutdown, version info.
- `packaging-and-deployment`: Docker image, docker-compose stack, GitHub Actions CI/CD, release artifacts, licensing.
- `observability-content`: Shipped documentation and Grafana dashboard requirements.

### Modified Capabilities

_None — greenfield project, no existing specs._

## Impact

- New Go module `github.com/<owner>/vbart-exporter`; new source tree, tests, Dockerfile, compose file, CI workflows, dashboard JSON, README, LICENSE.
- New runtime dependency: `github.com/prometheus/client_golang` (Apache-2.0).
- Network: exporter makes outbound HTTPS requests to devices with self-signed certificates (TLS verification disabled for device connections); listens on `:9642` by default.
- Security consideration: device admin password transits as a query parameter from Prometheus to the exporter; documented with mitigation guidance (run exporter co-located/behind TLS, restrict network access).
