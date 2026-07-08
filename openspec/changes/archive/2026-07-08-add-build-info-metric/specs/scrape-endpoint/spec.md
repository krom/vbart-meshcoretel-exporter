## MODIFIED Requirements

### Requirement: Scrape timeout budget
The exporter SHALL enforce a total per-scrape timeout covering login plus stats fetch — and, when build-info collection is enabled, the additional `ver` command fetch — defaulting to 10 seconds and configurable via a CLI flag. When the request carries an `X-Prometheus-Scrape-Timeout-Seconds` header smaller than the configured timeout, the smaller value SHALL be used (minus a small offset).

#### Scenario: Device hangs
- **WHEN** the device accepts the connection but never responds
- **THEN** the scrape completes within the timeout budget returning `meshcoretel_up 0`

#### Scenario: Prometheus header shortens timeout
- **WHEN** a request carries `X-Prometheus-Scrape-Timeout-Seconds: 5` and the configured timeout is 10s
- **THEN** the effective device timeout is derived from 5 seconds

#### Scenario: Version fetch shares the same budget
- **WHEN** build-info collection is enabled and login plus stats consume most of the configured timeout
- **THEN** the `ver` command is only attempted within the remaining budget and is abandoned without affecting `meshcoretel_up` if the budget runs out
