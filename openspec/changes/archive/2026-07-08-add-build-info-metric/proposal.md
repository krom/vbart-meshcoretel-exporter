## Why

There is no way to know which firmware build a device is running from the
data the exporter already collects — `/api/stats` carries no version field,
and the upstream API docs confirm no dedicated version endpoint exists. The
device does expose its build string via the CLI command channel
(`POST /api/command` with body `ver`, returning e.g.
`v1.16.0-vbart-meshcoretel-v1.2.0-1817248 (Build: 07-Jun-2026)`), which is
useful for fleet inventory/auditing when diagnosing behavior differences
across firmware versions. It provides no compatibility guarantee (a version
label doesn't prevent metric drift from field renames or semantic changes),
so it should be opt-in rather than part of the default scrape path.

## What Changes

- Add an opt-in `meshcoretel_build_info{version="..."} 1` info-style gauge
  (node_exporter `build_info` convention), populated by running the `ver`
  CLI command through the device's existing command channel.
- Add a `--collect.version` CLI flag and a `VBART_COLLECT_VERSION`
  environment variable to control it; flag takes precedence when both are
  set. Default is **off**.
- When enabled, `ScrapeHandler` performs a third device round-trip per
  scrape (after login, after stats): send `ver`, parse the response text.
  Parsing tolerates the documented format; if the response doesn't parse,
  the metric is omitted and the scrape still succeeds — this never
  downgrades `meshcoretel_up`.
- Document the flag/env var in README, with an explicit maintainer
  recommendation **against** leaving it enabled permanently: it adds a
  third HTTP round-trip to every Prometheus scrape against a
  resource-constrained, battery-powered LoRa device. Recommended use is
  temporary, for fleet inventory/auditing, not as a steady-state setting.
- Add a version panel to `dashboards/vbart-meshcoretel-exporter.json`, guarded so it
  renders "no data" gracefully rather than erroring when the metric is
  absent (collection disabled) — no separate dashboard variant needed.

## Capabilities

### New Capabilities
- `build-info-collection`: opt-in collection and exposition of the
  firmware build/version string as an info-style metric, gated by a
  flag/env var, additive to (not replacing) the existing stats scrape.

### Modified Capabilities
- `scrape-endpoint`: `ScrapeHandler` gains a conditional third device
  round-trip (the `ver` command) when version collection is enabled, with
  its own failure isolation (a parse/command failure must not affect
  `meshcoretel_up` or the rest of the scrape).

## Impact

- `cmd.go`: new flag `--collect.version` + env var `VBART_COLLECT_VERSION`
  binding (flag wins if both set).
- `internal/device`: new client method to send the `ver` command via
  `/api/command` and a parser for its response format.
- `internal/exporter/handler.go`: `ScrapeHandler` conditionally invokes the
  new device method and feeds the result into metrics collection; failures
  here are isolated from the rest of the scrape outcome.
- `internal/metrics`: new `meshcoretel_build_info` gauge construction from
  the parsed version string.
- `README.md`: document the flag/env var and the maintainer's
  don't-leave-this-on-by-default guidance.
- `dashboards/vbart-meshcoretel-exporter.json`: add a version info panel.
