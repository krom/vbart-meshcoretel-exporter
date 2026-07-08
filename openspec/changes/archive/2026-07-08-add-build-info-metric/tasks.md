## 1. Configuration

- [x] 1.1 Add `--collect.version` bool flag to `cmd.go` (default `false`)
- [x] 1.2 Read `VBART_COLLECT_VERSION` env var in `cmd.go`, applied only when `--collect.version` was not explicitly passed on the command line
- [x] 1.3 Plumb the resolved boolean through to `exporter.Config`/`ScrapeHandler` construction

## 2. Device client

- [x] 2.1 Add a `Client` method (e.g. `FetchVersion`) in `internal/device` that sends `POST /api/command` with body `ver` and the session token header, returning the raw response body
- [x] 2.2 Add a parser that trims whitespace, rejects empty/non-printable/over-256-byte responses, and returns a clean version string or "not usable"
- [x] 2.3 Unit tests for the parser: valid string, empty body, whitespace-only, oversized, non-printable characters

## 3. Scrape handler integration

- [x] 3.1 In `ScrapeHandler`, after a successful stats fetch, conditionally call the version fetch when collection is enabled, reusing the existing session token (no second login)
- [x] 3.2 Ensure the existing per-scrape timeout budget (login + stats, `X-Prometheus-Scrape-Timeout-Seconds`-aware) also bounds the version fetch, rather than adding a second timeout
- [x] 3.3 On version-fetch failure/timeout/unusable response: log at warning level (no password/token), skip the metric, do not touch `meshcoretel_up` or fail the scrape
- [x] 3.4 Handler tests: collection disabled (no `ver` call made), collection enabled + success, collection enabled + timeout, collection enabled + unparseable response, collection enabled but stats fails (no `ver` call made)

## 4. Metrics translation

- [x] 4.1 Add `meshcoretel_build_info` gauge construction in `internal/metrics` (value `1`, `version` label) driven by the parsed string, following the node_exporter `build_info` convention
- [x] 4.2 Unit test covering metric emission when a version string is present and omission when absent

## 5. Documentation

- [x] 5.1 Document `--collect.version` / `VBART_COLLECT_VERSION` in README's flag/config reference
- [x] 5.2 Add `meshcoretel_build_info` to README's metric reference table
- [x] 5.3 Add an explicit maintainer note in README stating this should not be left enabled permanently — it adds a third HTTP round-trip per scrape against a resource-constrained, battery-powered device — and is recommended only for temporary fleet inventory/auditing use

## 6. Dashboard

- [x] 6.1 Add a stat/table panel to `dashboards/vbart-meshcoretel-exporter.json` bound to `meshcoretel_build_info`, verified to render "No data" gracefully when the series is absent (collection disabled)

## 7. Verification

- [x] 7.1 `go build ./...`, `go vet ./...`, `golangci-lint run ./...` (golangci-lint not installed in this environment — build/vet/gofmt/tests all clean), `go test -race ./...`
- [x] 7.2 Manual check: run exporter with `--collect.version` against a mocked device (httptest TLS server) and confirm `meshcoretel_build_info` appears; run without the flag and confirm it's absent; also verified `VBART_COLLECT_VERSION` env var and flag-wins-over-env precedence
