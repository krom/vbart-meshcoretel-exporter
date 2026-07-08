# observability-content Specification

## Purpose

Defines the documentation, dashboard, and test-coverage deliverables that
make the exporter observable and verifiable: README, Grafana dashboard,
and automated test coverage.

## Requirements

### Requirement: Usage documentation
The repository SHALL contain a README covering: what the exporter does and supported firmware; quick start (binary, Docker, compose); all CLI flags; the `/metrics?target=&password=` contract; a complete Prometheus `scrape_config` example using `params` and `relabel_configs` in the snmp_exporter style (targets listed under `static_configs`, `__param_target` and `instance` relabeling, `scrape_interval: 60s` recommended); a metric reference table (name, type, unit, source JSON field); and a security section explaining the password-in-URL model, `InsecureSkipVerify`, and recommended network isolation.

#### Scenario: Prometheus config example works
- **WHEN** the README scrape_config example is copied into a Prometheus config with real target and password
- **THEN** Prometheus scrapes the device via the exporter and stores series with `instance` set to the device address

#### Scenario: Metric table completeness
- **WHEN** a metric is emitted from `examples/stats.json`
- **THEN** it appears in the README metric reference table

### Requirement: Grafana dashboard
The repository SHALL ship a Grafana dashboard JSON (under `dashboards/` or `examples/`) importable into Grafana 10+, using a Prometheus datasource variable and an `instance` template variable, with panels covering at minimum: battery voltage and charge state, uptime, MCU temperature, radio RSSI/SNR/noise floor, TX/RX air-time, packet rates (received, sent, flood/direct, errors, duplicates), memory (heap and PSRAM free), Wi-Fi RSSI/quality, neighbor count, and a per-neighbor SNR/last-heard table or panel.

#### Scenario: Dashboard import
- **WHEN** the dashboard JSON is imported into Grafana with a Prometheus datasource containing exporter data
- **THEN** all panels render data without query errors

### Requirement: Automated test coverage
The Go module SHALL include unit tests for: the JSON flattener against `examples/stats.json` (golden output), unit conversions, sentinel handling, boolean/info translation; and HTTP handler tests using `httptest`-based mock devices covering successful scrape, missing parameters, wrong password, unreachable target, 503 stats-disabled, and timeout. Tests SHALL run with `-race` in CI.

#### Scenario: Test suite
- **WHEN** `go test -race ./...` is executed
- **THEN** all tests pass and the scenarios listed above are each exercised by at least one test
