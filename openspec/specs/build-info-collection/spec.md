# build-info-collection Specification

## Purpose

Defines the opt-in collection of firmware build-version information from
the device and its exposition as the `meshcoretel_build_info` metric.

## Requirements

### Requirement: Opt-in build-info collection toggle
The exporter SHALL support enabling firmware build-info collection via a
CLI flag `--collect.version` and an environment variable
`VBART_COLLECT_VERSION`. Collection SHALL default to disabled. When both
the flag and the environment variable are set, the flag's value SHALL take
precedence.

#### Scenario: Default disabled
- **WHEN** the exporter starts with neither `--collect.version` nor
  `VBART_COLLECT_VERSION` set
- **THEN** scrapes perform only login and stats round-trips, and
  `meshcoretel_build_info` is never emitted

#### Scenario: Enabled via flag
- **WHEN** the exporter starts with `--collect.version=true`
- **THEN** scrapes additionally query the device's build version and may
  emit `meshcoretel_build_info`

#### Scenario: Enabled via environment variable
- **WHEN** the exporter starts with `VBART_COLLECT_VERSION=true` and no
  `--collect.version` flag is passed
- **THEN** scrapes additionally query the device's build version and may
  emit `meshcoretel_build_info`

#### Scenario: Flag overrides environment variable
- **WHEN** the exporter starts with `VBART_COLLECT_VERSION=true` and
  `--collect.version=false` explicitly passed
- **THEN** scrapes do not query the device's build version

### Requirement: Build version retrieval via device command channel
When build-info collection is enabled, the exporter SHALL retrieve the
firmware build string for a scrape's target by sending `POST /api/command`
with body `ver` to the device, authenticated with the same session token
obtained from that scrape's login, after the stats fetch succeeds.

#### Scenario: Version fetched after successful stats
- **WHEN** collection is enabled and `GET /api/stats` succeeds
- **THEN** the exporter sends `POST /api/command` with body `ver` using
  the existing session token, without performing a second login

#### Scenario: Version not fetched when stats fails
- **WHEN** collection is enabled and the device round-trip fails before
  stats succeeds (login failure, unreachable device, stats error)
- **THEN** the exporter does not send the `ver` command for that scrape

### Requirement: Build-info metric exposition
When a usable build version string is obtained, the exporter SHALL expose
it as `meshcoretel_build_info{version="<value>"} 1`, following the
info-metric convention (constant value of 1, data carried in the label).

#### Scenario: Successful version parse
- **WHEN** the device responds to the `ver` command with
  `v1.16.0-vbart-meshcoretel-v1.2.0-1817248 (Build: 07-Jun-2026)`
- **THEN** the exporter emits
  `meshcoretel_build_info{version="v1.16.0-vbart-meshcoretel-v1.2.0-1817248 (Build: 07-Jun-2026)"} 1`

### Requirement: Graceful degradation on unusable version response
The exporter SHALL omit `meshcoretel_build_info` for a scrape, without
altering `meshcoretel_up` or failing the scrape, if the `ver` command
fails, times out, or returns an empty, whitespace-only, non-printable, or
oversized (over 256 bytes) response. The failure SHALL be logged at
warning level without including the password or session token.

#### Scenario: Command times out
- **WHEN** collection is enabled and the `ver` command does not respond
  within the scrape's timeout budget
- **THEN** the scrape still returns HTTP 200 with `meshcoretel_up 1` and
  stats metrics, but no `meshcoretel_build_info`

#### Scenario: Unparseable response
- **WHEN** the device returns an empty body or a body exceeding 256 bytes
  for the `ver` command
- **THEN** `meshcoretel_build_info` is omitted and the rest of the scrape
  is unaffected

### Requirement: Combined timeout budget includes version fetch
The exporter SHALL cover the `ver` command within its existing per-scrape
timeout budget (login + stats, honoring
`X-Prometheus-Scrape-Timeout-Seconds`) when build-info collection is
enabled, rather than introducing a second independent timeout.

#### Scenario: Version fetch respects remaining budget
- **WHEN** login and stats together consume most of the configured
  timeout
- **THEN** the `ver` command is attempted only within the remaining budget
  and is abandoned (metric omitted) if the budget is exhausted
