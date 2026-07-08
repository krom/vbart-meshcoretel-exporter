# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-07-08

First release of `vbart-meshcoretel-exporter`.

### Added

- Stateless multi-target Prometheus exporter for devices running
  MeshCoreTel firmware, modeled on `snmp_exporter`'s scrape pattern:
  `GET /metrics?target=<host>&password=<pw>` logs into the device, fetches
  `/api/stats`, and translates the response into `meshcoretel_*` Prometheus
  metrics on a fresh registry per request.
- `meshcoretel_up` and `meshcoretel_scrape_duration_seconds` gauges so
  device/scrape failures surface as data rather than failed scrapes.
- `/-/metrics`, `/-/healthy` exporter self-observability endpoints.
- CLI flags for listen address, scrape timeout, and log level (see
  `cmd.go`); no configuration file.
- Multi-stage `Dockerfile` and root `docker-compose.yml`, plus a full demo
  stack under `examples/compose/` (exporter + Prometheus + Grafana with a
  pre-provisioned dashboard).
- GitHub Actions CI (`go vet`, `golangci-lint`, `go test -race`) on every
  push and pull request.
- GitHub Actions release automation: on `v*.*.*` tags, multi-arch
  (`linux/amd64`, `linux/arm64`) images are published to
  `ghcr.io/krom/vbart-meshcoretel-exporter` tagged with semantic versioning
  (`X.Y.Z`, `X.Y`, `X`, `latest`, no `v` prefix), and GoReleaser builds and
  attaches `linux`/`darwin`/`windows` (`amd64`/`arm64`) binaries to the
  GitHub release.

[Unreleased]: https://github.com/krom/vbart-meshcoretel-exporter/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/krom/vbart-meshcoretel-exporter/releases/tag/v1.0.0
