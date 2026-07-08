## Context

`internal/exporter/handler.go`'s `ScrapeHandler` currently performs exactly
two device round-trips per scrape: `POST /login`, then `GET /api/stats`.
`openspec/changes/add-vbart-exporter/design.md` establishes that this is
deliberate — no token caching, no result caching, no extra per-scrape
device traffic — because devices are resource-constrained, battery-powered
LoRa repeaters and Prometheus's `scrape_interval` (recommended ≥60s) is the
only rate limiter the exporter relies on.

The device exposes its firmware build string only through the CLI command
channel: `POST /api/command` with body `ver`, authenticated the same way
as `/api/stats` (`X-Auth-Token` from the same login), returning plain text
like `v1.16.0-vbart-meshcoretel-v1.2.0-1817248 (Build: 07-Jun-2026)`. There
is no lighter-weight way to get this data — no field in `/api/stats`, no
dedicated version endpoint (confirmed against the upstream API docs).

This proposal is a deliberate, narrow, opt-in exception to the "no extra
round-trips" principle, not a reversal of it: the extra request only
happens when an operator explicitly turns it on.

## Goals / Non-Goals

**Goals:**
- Let operators optionally see which firmware build a device is running,
  as a standard Prometheus info-metric (`meshcoretel_build_info`).
- Keep it fully opt-in and off by default, with both a flag and an env var
  for operators who set config via container env rather than flags.
- Isolate failures: an unparseable or failed `ver` command must never
  affect `meshcoretel_up` or the rest of the scrape's metrics.
- Make the operational cost (a third round-trip per scrape) explicit and
  hard to miss in the README.

**Non-Goals:**
- Using the version to detect or guard against firmware/API drift. As
  established in prior discussion, a version label has no protective
  value against renamed/re-typed/re-scaled fields — this is a
  purely-informational, fleet-inventory feature.
- Caching the version across scrapes to reduce the round-trip cost. This
  would reintroduce per-target state, which conflicts with the exporter's
  stateless design and its no-caching principle; if the cost is
  unacceptable for continuous scraping, the answer is "leave it off," not
  "cache it."
- Parsing/exposing structured sub-fields (base MeshCore version vs.
  MeshCoreTel fork version vs. build hash) as separate labels/metrics.
  Ship the whole string as one label; revisit only if a concrete need for
  structured fields shows up.

## Decisions

**1. Flag + env var, flag wins.** `--collect.version` (bool flag, default
`false`) and `VBART_COLLECT_VERSION` (env var, any of the standard
truthy strings). Precedence: if the flag is explicitly set on the command
line, it wins over the env var; otherwise the env var is read at startup.
This matches common CLI/env precedence conventions and lets container
deployments (env-var-first) and direct CLI use (flag-first) both work
naturally. Alternative considered: env var only — rejected, this project's
`cmd.go` already exposes all config as flags and a flag-only escape hatch
for quick manual testing (`go run . --collect.version`) is worth keeping.

**2. Third round-trip is synchronous and gated at the top of the scrape,
after stats succeeds.** `ScrapeHandler` runs login → stats → (if enabled)
`ver`, reusing the same session token from login (no second login). If
`ver` fails or times out, the handler logs at warning level and proceeds
without the build-info metric — it does not set `meshcoretel_up 0`, since
the stats round-trip (the thing `up` represents) already succeeded.
Alternative considered: run `ver` in parallel with stats — rejected,
adds concurrency complexity for a low-frequency, purely-informational
metric, and the design doc for this project already prefers one combined
sequential timeout budget over concurrent device calls.

**3. Timeout budget includes the `ver` call when enabled.** The existing
per-scrape timeout (login + stats, default 10s, `X-Prometheus-Scrape-Timeout-Seconds`-aware)
is extended to cover the `ver` call too when collection is enabled, rather
than adding a second independent timeout. Keeps the single-deadline model
from the existing design instead of introducing a second budget to reason
about.

**4. Version string is stored verbatim as the `version` label; parsing is
best-effort trim/validate, not structural decomposition.** The device
response `v1.16.0-vbart-meshcoretel-v1.2.0-1817248 (Build: 07-Jun-2026)` is
trimmed of whitespace/newlines and used as-is if non-empty and within a
sane length bound (e.g. ≤256 bytes, to guard against a misbehaving device
flooding a label value). If the response is empty, whitespace-only, or
exceeds the bound, the metric is omitted for that scrape — this is a
graceful degrade, not a scrape failure.

**5. Dashboard: one panel, not a second dashboard.** Add a single stat/table
panel bound to `meshcoretel_build_info` that shows "No data" when the
series is absent (collection disabled or scrape didn't produce it), rather
than maintaining a parallel "with version" dashboard. Alternative
considered: a second dashboard variant — rejected as needless duplication
to maintain; Grafana panels already handle absent series gracefully, and
operators who never enable `--collect.version` simply see an empty panel
instead of a broken one.

## Risks / Trade-offs

- [Extra round-trip adds latency/load even for operators who only need it
  occasionally] → Off by default; README explicitly recommends enabling
  only temporarily for inventory/auditing, not as steady state.
- [Device CLI command responses are undocumented/unversioned themselves —
  the `ver` output format could change] → Best-effort parsing with a
  length/emptiness guard; unparseable output degrades to "metric omitted,"
  never a scrape failure.
- [A misbehaving or compromised device could return an oversized or
  control-character-laden string as a Prometheus label value] → Enforce
  a max length and strip/reject non-printable characters before using it
  as a label value, since Prometheus label values ARE arbitrary UTF-8 but
  a malformed/huge label is still worth guarding against defensively.
- [Operators might miss the "don't leave this on" guidance and run it
  continuously against many devices] → Called out explicitly in README
  next to the flag/env var documentation, not just in a general notes
  section.

## Open Questions

None — scope, gating, and failure isolation are all settled above.
