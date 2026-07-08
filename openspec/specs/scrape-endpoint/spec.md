# scrape-endpoint Specification

## Purpose

Defines the behavior of the `/metrics` multi-target scrape endpoint: request
parameters, the device login/fetch flow, failure reporting, timeout
handling, and password confidentiality.

## Requirements

### Requirement: Multi-target metrics endpoint
The exporter SHALL serve `GET /metrics` accepting query parameters `target` (device host or `host:port`) and `password` (device admin password). For each request it SHALL perform a full device round-trip — login, stats fetch — and return the resulting metrics for that target only, in Prometheus exposition format.

#### Scenario: Successful scrape
- **WHEN** a request `GET /metrics?target=192.168.0.10&password=pass` is received and the device responds to `POST /login` with a token and to `GET /api/stats` with valid JSON
- **THEN** the exporter returns HTTP 200 with metrics translated from the stats JSON, including `meshcoretel_up 1` and `meshcoretel_scrape_duration_seconds`

#### Scenario: Target with explicit port
- **WHEN** `target=192.168.0.10:8443` is provided
- **THEN** device requests are sent to `https://192.168.0.10:8443`

### Requirement: Parameter validation
The exporter SHALL return HTTP 400 with a plain-text error when the `target` or `password` parameter is missing or empty, without contacting any device.

#### Scenario: Missing target
- **WHEN** `GET /metrics?password=pass` is received
- **THEN** the exporter responds HTTP 400 and makes no outbound request

#### Scenario: Missing password
- **WHEN** `GET /metrics?target=192.168.0.10` is received
- **THEN** the exporter responds HTTP 400 and makes no outbound request

### Requirement: Device login flow
For each scrape the exporter SHALL authenticate by sending `POST https://<target>/login` with the password as the plain-text request body, read the session token from the plain-text response body, and send `GET https://<target>/api/stats` with header `X-Auth-Token: <token>`. The exporter SHALL NOT cache tokens or any other per-target state between requests.

#### Scenario: Fresh login per scrape
- **WHEN** two consecutive scrapes for the same target are received
- **THEN** each scrape performs its own `POST /login` before fetching stats

#### Scenario: Self-signed device certificate
- **WHEN** the device presents a self-signed TLS certificate
- **THEN** the exporter completes the connection (device connections skip certificate verification)

### Requirement: Failure reporting via up gauge
When the device round-trip fails — connection error, TLS failure, timeout, HTTP 401 from `/login` (wrong password), HTTP 503 from `/api/stats` (stats disabled), or malformed JSON — the exporter SHALL respond HTTP 200 containing `meshcoretel_up 0` and `meshcoretel_scrape_duration_seconds`, and SHALL NOT emit device metrics for that scrape. The failure cause SHALL be logged at warning level without including the password.

#### Scenario: Unreachable device
- **WHEN** the target does not accept connections
- **THEN** the response is HTTP 200 with `meshcoretel_up 0`

#### Scenario: Wrong password
- **WHEN** the device answers `POST /login` with HTTP 401
- **THEN** the response is HTTP 200 with `meshcoretel_up 0` and the log entry does not contain the password

#### Scenario: Stats disabled on device
- **WHEN** the device answers `GET /api/stats` with HTTP 503
- **THEN** the response is HTTP 200 with `meshcoretel_up 0`

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

### Requirement: Password confidentiality in logs
The exporter SHALL never write the `password` parameter value, full request URLs containing query strings, or the device session token to logs or error responses.

#### Scenario: Request logging
- **WHEN** any `/metrics` request is processed with log level debug
- **THEN** emitted log entries contain the target but neither the password nor the token
