## 1. Project scaffolding

- [x] 1.1 Initialize git repository and Go module (`go.mod`, latest stable Go); add `.gitignore`, `LICENSE` (GPL-3.0-only text), and repo skeleton (`main.go`, `internal/device`, `internal/metrics`, `internal/exporter`)
- [x] 1.2 Add `prometheus/client_golang` dependency; verify all module licenses are GPLv3-compatible (`go-licenses` or manual review)
- [x] 1.3 Add `.golangci.yml` with a sensible linter set and make sure `golangci-lint run` passes on the skeleton

## 2. Device client (`internal/device`)

- [x] 2.1 Implement the HTTPS client with `InsecureSkipVerify` for device connections, context-based timeout, and target normalization (host vs host:port)
- [x] 2.2 Implement `Login(ctx, target, password)` — `POST /login`, plain-text body/response, typed error for 401
- [x] 2.3 Implement `FetchStats(ctx, target, token)` — `GET /api/stats` with `X-Auth-Token`, typed error for 503, JSON decode into stats structs
- [x] 2.4 Define stats structs covering all sections of `examples/stats.json` (unknown fields ignored)
- [x] 2.5 Unit tests with `httptest.NewTLSServer` mock device: success, 401, 503, timeout, malformed JSON

## 3. Metrics translation (`internal/metrics`)

- [x] 3.1 Define the metric mapping table (name, type, unit conversion, HELP) for core, radio, packets, memory, wifi, services, sensors, history sections per the metrics-translation spec
- [x] 3.2 Implement the flattener producing client_golang collectors: unit conversions (mv→volts, secs→seconds), counters with `_total`, booleans as 0/1, `_info` metrics for string fields
- [x] 3.3 Implement per-neighbor metrics from `neighbors_detail[]` with `neighbor` label and sentinel (≥2^31) omission
- [x] 3.4 Implement meta metrics `meshcoretel_up` and `meshcoretel_scrape_duration_seconds`
- [x] 3.5 Golden-file test: flatten `examples/stats.json`, compare against checked-in expected exposition output; add focused tests for conversions and sentinels

## 4. HTTP server (`internal/exporter`, `main.go`)

- [x] 4.1 Implement `/metrics` handler: param validation (400), per-scrape registry, device round-trip, `up 0` on failures, `X-Prometheus-Scrape-Timeout-Seconds` handling
- [x] 4.2 Implement `/` landing page, `/-/metrics` self-metrics (Go/process collectors + exporter counters), `/-/healthy`
- [x] 4.3 Implement CLI flags (`--web.listen-address`, `--scrape.timeout`, `--log.level`, `--log.format`, `--version`), slog setup, version via ldflags
- [x] 4.4 Implement graceful shutdown on SIGINT/SIGTERM with drain period
- [x] 4.5 Handler tests via `httptest`: successful scrape end-to-end against mock device, missing params, wrong password, unreachable target, 503, timeout; verify no password/token in logs

## 5. Packaging

- [x] 5.1 Write multi-stage `Dockerfile` (builder runs `go test`, final distroless/static non-root, EXPOSE 9642); verify image builds and `/-/healthy` responds
- [x] 5.2 Write root `docker-compose.yml` for the exporter alone
- [x] 5.3 Write demo stack under `examples/compose/`: exporter + Prometheus (scrape config with params/relabel_configs) + Grafana (provisioned datasource and dashboard)

## 6. CI/CD

- [x] 6.1 `ci.yml`: go vet, golangci-lint, `go test -race`, build on push/PR
- [x] 6.2 `docker.yml`: buildx multi-arch (amd64/arm64) publish to ghcr.io on main and tags
- [x] 6.3 `.goreleaser.yaml` + `release.yml`: binaries and release notes on `v*` tags

## 7. Documentation and dashboard

- [x] 7.1 Write README: overview, quick start (binary/Docker/compose), flags, Prometheus scrape_config example with relabeling, security section (password-in-URL, InsecureSkipVerify, network isolation), firmware compatibility note
- [x] 7.2 Generate the metric reference table in the README covering everything emitted from `examples/stats.json`
- [x] 7.3 Build the Grafana dashboard JSON (datasource + instance variables; panels: battery, uptime, MCU temp, RSSI/SNR/noise floor, air-time, packet rates/errors/dups, heap/PSRAM, Wi-Fi, neighbor count, per-neighbor table); validate by import against the demo stack
- [x] 7.4 Add repository meta: CONTRIBUTING/badges as appropriate, verify license notices

## 8. Verification

- [x] 8.1 Run full local check: `go vet`, `golangci-lint run`, `go test -race ./...`, `docker build`, compose demo smoke test
- [x] 8.2 If a real device is available, scrape it end-to-end and reconcile output with the metric reference table
