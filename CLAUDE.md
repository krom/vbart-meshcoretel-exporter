# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go build ./...                 # build everything
go vet ./...                   # vet
golangci-lint run ./...        # lint (config: .golangci.yml)
go test -race ./...            # full test suite
go test -race ./internal/metrics/...        # single package
go test -race -run TestScrapeHandlerSuccess ./internal/exporter/...  # single test

go run . --version             # print version/commit/date
go run . --web.listen-address=:9642 --log.level=debug   # run locally

docker build -t vbart-meshcoretel-exporter .
docker compose up -d           # exporter alone (root docker-compose.yml)
cd examples/compose && docker compose up -d   # full demo: exporter + Prometheus + Grafana
```

There is no configuration file for the exporter itself — everything is a
CLI flag (see `cmd.go`) or a per-request query parameter. Don't add one.

## Architecture

`vbart-meshcoretel-exporter` is a stateless, multi-target Prometheus exporter for
devices running MeshCoreTel firmware (LoRa mesh repeaters), modeled on
`prometheus/snmp_exporter`'s scrape pattern. One running instance serves
any number of devices; Prometheus supplies the target device and its admin
password per scrape via query parameters, never via exporter config.

Request flow for `GET /metrics?target=<host>&password=<pw>`
(`internal/exporter/handler.go`, `ScrapeHandler`):

1. Validate `target`/`password` are present (400 if not).
2. `device.Client.Login` → `POST /login` with the password as plain body,
   returns a session token (device uses a self-signed cert, so the device
   HTTP client sets `InsecureSkipVerify`).
3. `device.Client.FetchStats` → `GET /api/stats` with `X-Auth-Token`,
   returns the raw JSON, decoded via `device.ParseStats` into `device.Stats`.
4. `internal/metrics.Collect` flattens `device.Stats` into a slice of
   `prometheus.Metric`, registered into a **fresh `prometheus.Registry`
   created per request** (no shared/global registry — this is what keeps
   the exporter stateless across scrapes) and rendered via
   `promhttp.HandlerFor`.
5. Any failure at steps 2-4 (unreachable device, wrong password, 503
   stats-disabled, timeout, bad JSON) is *not* surfaced as an HTTP error —
   the handler still returns 200 with `meshcoretel_up 0` and
   `meshcoretel_scrape_duration_seconds`, so Prometheus records the outage
   as data rather than a failed scrape. Only missing/empty `target`/
   `password` params return HTTP 400.
6. `ScrapeHandler.effectiveTimeout` honors Prometheus's
   `X-Prometheus-Scrape-Timeout-Seconds` header when it's smaller than the
   configured `--scrape.timeout`, minus a small safety margin.

Package layout:

- `internal/device` — HTTP client for the physical device (`login`,
  `stats`) plus the `Stats` struct tree mirroring `/api/stats`'s JSON.
  Unknown JSON fields are ignored by design (forward-compat with firmware
  updates); adding a new metric means adding a field here first.
- `internal/metrics` — pure translation from `device.Stats` to
  `[]prometheus.Metric`. All naming/unit-conversion rules live in
  `metrics.go` (`fq()` builds `meshcoretel_<section>_<field>` names). No
  device I/O in this package — it's tested purely against
  `examples/stats.json` as a golden fixture.
- `internal/exporter` — HTTP surface: `ScrapeHandler` (the `/metrics`
  proxy logic) and `server.go` (routing for `/`, `/-/metrics`,
  `/-/healthy`, plus `Run()`'s graceful-shutdown loop). `DeviceClient` is
  an interface here so handler tests inject a mock instead of hitting a
  real device.
- `cmd.go` / `main.go` — flag parsing, `log/slog` setup, wiring into
  `exporter.Run`. `version`/`commit`/`date` are package vars injected via
  `-ldflags` at build time (see `Dockerfile` and `.goreleaser.yaml`).

**Adding a new metric**: add the field to the relevant struct in
`internal/device/stats.go`, add a `gauge()`/`counter()` call in the
matching `collect*` function in `internal/metrics/metrics.go`, then update
the metric reference table in `README.md` and, if relevant, add a panel to
`dashboards/vbart-meshcoretel-exporter.json`. `events[]` is intentionally *not*
exported (unbounded, relative-timestamp data doesn't fit a scrape model).

**Security-relevant invariants** (see README's Security Notes section):
never log the password, session token, or a full query string
(`ScrapeHandler` logs `target` only); device connections always skip TLS
verification (self-signed device certs), exporter-to-Prometheus has no
such exception.

## OpenSpec

This repo uses [OpenSpec](https://github.com/Fission-AI/openspec) for
spec-driven change proposals under `openspec/`. The original build of this
exporter is fully specified in
`openspec/changes/add-vbart-exporter/` (proposal, design, per-capability
specs, tasks) — read `design.md` there for the rationale behind decisions
like per-scrape registries, fresh logins with no token caching, and the
`up`-gauge-instead-of-HTTP-error failure model before changing that
behavior.
