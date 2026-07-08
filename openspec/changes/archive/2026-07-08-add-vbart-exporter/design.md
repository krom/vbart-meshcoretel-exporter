## Context

MeshCoreTel firmware devices expose an HTTPS API with a self-signed certificate:

- `POST /login` — admin password as plain-text request body → session token as plain-text response body (`401` on wrong password).
- `GET /api/stats` — requires `X-Auth-Token: <token>` header → hierarchical stats JSON (`503` if stats disabled). A reference response is committed at `examples/stats.json`.
- Firmware guidance: poll at most once per minute; avoid parallel requests to the same device.

This is a greenfield repository (only `examples/` and `openspec/` exist). The reference architecture is `prometheus/snmp_exporter`: one exporter instance serves many devices; Prometheus passes the target per scrape and rewrites `instance` via `relabel_configs`.

## Goals / Non-Goals

**Goals:**

- Stateless multi-target exporter: everything a scrape needs comes in the request; no config files, storage, or caches.
- Faithful, idiomatic Prometheus translation of the stats JSON (base units, `_total` counters, 0/1 gauges, `_info` metrics, labeled per-neighbor series).
- Production-ready packaging: multi-stage Docker image, compose demo, CI, releases, GPL-3.0 license, docs, Grafana dashboard.

**Non-Goals:**

- No support for `/api/command`, trend series (`/api/stats?series=...`), or device configuration.
- No result caching or rate limiting toward devices (Prometheus `scrape_interval` is the rate limiter; docs recommend ≥60s).
- No authentication on the exporter's own HTTP server (deploy-time concern; documented).
- No historical event export (the `events` array is unbounded, timestamp-relative data unsuited to scrape-based metrics; only an event-count-style signal if trivially available).

## Decisions

### 1. Multi-target pattern, snmp_exporter style

`GET /metrics?target=<host>&password=<pass>` performs login + stats fetch per scrape and renders metrics for that device only. Exporter self-metrics (Go runtime, scrape counters) live on a separate path so they don't pollute device series. Alternative — one exporter per device with env-var config — rejected: contradicts the stateless/no-config requirement and scales poorly.

**Endpoints:** `/` landing page, `/metrics` device scrape, `/-/metrics` self-metrics, `/-/healthy` liveness. Default listen `:9642` (not in the Prometheus port allocation registry as of writing; flag-overridable).

### 2. Per-scrape registry with prometheus/client_golang

Each `/metrics` request builds a fresh `prometheus.Registry`, populated by a collector that performs the device round-trip, then rendered with `promhttp.HandlerFor`. This gives correct exposition-format output (incl. content negotiation) for free and guarantees no state leaks between scrapes. Alternative — hand-writing the text format — rejected: reinvents escaping/negotiation; client_golang is the ecosystem standard.

### 3. Device client: fresh login every scrape, InsecureSkipVerify

The device presents a self-signed cert, so the device-facing `http.Client` uses `TLS InsecureSkipVerify: true` (documented; devices are on trusted LANs). Tokens are not cached — statelessness beats saving one request, and token lifetime semantics are undocumented. One combined timeout budget per scrape (default 10s, flag-overridable) covers login + stats; the handler honors Prometheus's `X-Prometheus-Scrape-Timeout-Seconds` header when present and smaller.

Validated against real hardware: the firmware's embedded TLS stack (ESP32/mbedTLS) negotiated only `TLS_RSA_WITH_AES_128_GCM_SHA256` over TLS 1.2 — a non-ECDHE, non-forward-secret cipher suite that Go's `crypto/tls` does not offer by default. Without an explicit `CipherSuites` list the handshake fails (`tls: handshake failure`) before certificate verification is even reached. The device `tls.Config` therefore also pins `MaxVersion: tls.VersionTLS12` and an explicit `CipherSuites` list covering both ECDHE and legacy RSA-kex AES-GCM/CBC suites, to tolerate this and similar embedded TLS stacks.

### 4. Error semantics: `up` gauge, not HTTP errors

Unreachable device, TLS failure, timeout, or `503` (stats disabled) → HTTP 200 with `meshcoretel_up 0` plus `meshcoretel_scrape_duration_seconds`, so Prometheus records the outage as data. Exceptions that do return HTTP errors: missing/invalid `target` or `password` parameters → `400`; wrong password (`401` from device) → also `up 0` with an error-type label rather than surfacing 401, keeping alerting uniform. Rationale: matches snmp_exporter/blackbox_exporter conventions.

### 5. JSON flattening via typed structs, not generic reflection

The stats schema is known and versioned with the firmware, so decode into Go structs and map explicitly to metric descriptors. Explicit mapping lets us assign correct types/units per field (e.g. `uptime_secs` → `meshcoretel_core_uptime_seconds` counter; `battery_mv` → `meshcoretel_core_battery_volts` gauge ×0.001). A generic walker was considered and rejected: it can't distinguish counters from gauges, can't convert units, and produces unstable names when firmware adds fields. Unknown JSON fields are ignored (forward compatibility).

**Naming rules** (detailed in the `metrics-translation` spec): prefix `meshcoretel_`, subsystem from the JSON section, base units (seconds/bytes/volts/celsius/ratio), `_total` for monotonic counters, booleans as 0/1 gauges, string enums as either `_info` labels or per-state gauges, `neighbors_detail[]` → per-neighbor metrics labeled by `neighbor` (short id).

### 6. Stack and structure

Go (latest stable), stdlib `net/http` + `log/slog`; only external runtime dep is `prometheus/client_golang` (+ its transitive deps). Layout: `main.go` thin entrypoint, `internal/exporter` (HTTP handlers, collector), `internal/device` (API client), `internal/metrics` (flattening/mapping). Version/commit injected via `-ldflags`. Flags follow Prometheus conventions (`--web.listen-address`, `--scrape.timeout`, `--log.level`).

### 7. Packaging and CI

- Dockerfile: `golang:<ver>` build stage (CGO_ENABLED=0, tests run at build) → `gcr.io/distroless/static` (or `scratch` + CA certs) final; runs as non-root.
- `docker-compose.yml`: exporter alone; `examples/` adds Prometheus + Grafana with provisioned datasource and the shipped dashboard for a turnkey demo.
- GitHub Actions: `ci.yml` (vet, golangci-lint, `go test -race`, build) on push/PR; `docker.yml` publishes multi-arch (amd64/arm64) images to `ghcr.io` on tags and `main`; `release.yml` runs GoReleaser on tags for binary artifacts.
- License GPL-3.0-only. Compatibility verified: Apache-2.0 (client_golang) and BSD-3-Clause (Go stdlib/x libs) are one-way compatible into GPLv3 works.

### 8. Password in query string

Kept per requirements and snmp_exporter-parity (`auth` module analogue is out of scope for a no-config exporter). Mitigations documented: exporter should not log full URLs (log target only), deploy on a trusted network next to Prometheus, Prometheus config carries the password via `params` and should be permission-restricted. A future `password` via HTTP header or env fallback is noted as an open extension.

## Risks / Trade-offs

- [Firmware JSON schema drift] → typed decoding ignores unknown fields; flattener unit tests pin `examples/stats.json`; docs state supported firmware API version.
- [Password visible in query string/logs] → never log query strings; documentation section on threat model; recommend network isolation.
- [InsecureSkipVerify weakens transport security] → acceptable for LAN devices with self-signed certs; documented; certificate pinning listed as possible future flag.
- [Concurrent Prometheus scrapes of the same target violate firmware "no parallel requests" guidance] → docs mandate a single scrape job per device with `scrape_interval >= 60s`; exporter itself stays stateless (no per-target locking) to honor the no-state requirement.
- [`advert_secs_ago` contains sentinel values near 2^32 (never advertised)] → flattener maps sentinels (> 2^31) to metric omission for that sample; noted in metric reference.
- [Counter vs gauge misclassification (e.g. `errors`, `recv`)] → treat device-lifetime totals as counters with `_total`; they reset on reboot, which Prometheus handles via `rate()`/`resets()`.

## Migration Plan

Greenfield — no migration. Rollout: publish image to ghcr.io, users add one scrape job. Rollback: stop the container; no state to clean up.

## Open Questions

- Exact default port if 9642 conflicts once the exporter is registered in the Prometheus exporter port list (cosmetic; flag exists).
- Whether `history`/`archive` sections warrant metrics beyond a few capacity gauges (initial release: minimal set; extend on demand).
