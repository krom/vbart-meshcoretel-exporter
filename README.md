# vbart-meshcoretel-exporter

[![CI](https://github.com/krom/vbart-meshcoretel-exporter/actions/workflows/ci.yml/badge.svg)](https://github.com/krom/vbart-meshcoretel-exporter/actions/workflows/ci.yml)
[![Docker](https://github.com/krom/vbart-meshcoretel-exporter/actions/workflows/docker.yml/badge.svg)](https://github.com/krom/vbart-meshcoretel-exporter/actions/workflows/docker.yml)
[![GHCR](https://img.shields.io/badge/GHCR-ghcr.io%2Fkrom%2Fvbart--meshcoretel--exporter-blue?logo=docker)](https://github.com/krom/vbart-meshcoretel-exporter/pkgs/container/vbart-meshcoretel-exporter)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

A stateless [Prometheus](https://prometheus.io/) exporter for devices running
[MeshCoreTel firmware](https://github.com/VBart/MeshCoreTel-firmware) (LoRa
mesh repeaters). It follows the same multi-target pattern as
[`snmp_exporter`](https://github.com/prometheus/snmp_exporter): one exporter
instance serves many devices, with the target and credentials supplied per
scrape by Prometheus.

## How it works

For every `GET /metrics?target=<device>&password=<password>` request, the
exporter:

1. Logs into the device (`POST /login`, password as the plain-text body).
2. Fetches `GET /api/stats` using the returned session token
   (`X-Auth-Token` header).
3. Flattens the hierarchical stats JSON into Prometheus metrics.
4. Returns the metrics for **that device only**, in the scrape response.

There is no configuration file, no database, and no cache: every scrape is
an independent round-trip to the device. See the firmware's
[API documentation](https://vbart.github.io/MeshCoreTel-firmware/api/) for
the underlying device endpoints.

## Quick start

### Binary

Prebuilt `linux`/`darwin`/`windows` (`amd64`/`arm64`) binaries are attached
to every [GitHub release](https://github.com/krom/vbart-meshcoretel-exporter/releases).
Or build from source:

```sh
go build -o vbart-meshcoretel-exporter .
./vbart-meshcoretel-exporter --web.listen-address=:9642
curl 'http://localhost:9642/metrics?target=192.168.0.10&password=admin-password'
```

### Docker (GHCR image)

Prebuilt multi-arch (`amd64`/`arm64`) images are published to
[`ghcr.io/krom/vbart-meshcoretel-exporter`](https://github.com/krom/vbart-meshcoretel-exporter/pkgs/container/vbart-meshcoretel-exporter)
on every version tag, tagged `latest`, `<major>`, `<major>.<minor>`, and
`<major>.<minor>.<patch>`:

```sh
docker run -p 9642:9642 ghcr.io/krom/vbart-meshcoretel-exporter:latest
```

### Docker (build locally)

```sh
docker build -t vbart-meshcoretel-exporter .
docker run -p 9642:9642 vbart-meshcoretel-exporter
```

### Docker Compose

Using the published GHCR image:

```yaml
services:
  vbart-meshcoretel-exporter:
    image: ghcr.io/krom/vbart-meshcoretel-exporter:latest
    container_name: vbart-meshcoretel-exporter
    restart: unless-stopped
    ports:
      - "9642:9642"
```

Or build locally with the repo's own [`docker-compose.yml`](docker-compose.yml):

```sh
docker compose up -d
```

A full demo stack (exporter + Prometheus + Grafana, with the dashboard
below pre-provisioned) lives in [`examples/compose/`](examples/compose/):

```sh
cd examples/compose
docker compose up -d
```

Edit [`examples/compose/prometheus/prometheus.yml`](examples/compose/prometheus/prometheus.yml)
to point at your real device before starting.

## CLI flags

| Flag | Default | Description |
|---|---|---|
| `--web.listen-address` | `:9642` | Address to expose the web interface and metrics on. |
| `--scrape.timeout` | `10s` | Maximum time allowed for a single device scrape (login + stats fetch). |
| `--log.level` | `info` | `debug`, `info`, `warn`, or `error`. |
| `--log.format` | `text` | `text` or `json`. |
| `--version` | — | Print version information and exit. |
| `--collect.version` | `false` | Expose `meshcoretel_build_info` by running the device's `ver` CLI command. Can also be set via the `VBART_COLLECT_VERSION` environment variable; the flag wins if both are given. **The maintainer strongly recommends leaving this off in normal operation** — it adds a third HTTP round-trip to every scrape against a resource-constrained, battery-powered LoRa device. Enable it only temporarily, e.g. for fleet inventory/auditing. |

## HTTP endpoints

| Path | Purpose |
|---|---|
| `/` | Landing page. |
| `/metrics?target=<host>&password=<password>` | Scrape a MeshCoreTel device. `target` may include a port (`host:port`); default port is 443 (HTTPS). |
| `/-/metrics` | Exporter's own process/runtime metrics (Go collector). |
| `/-/healthy` | Liveness check; always returns HTTP 200. |

## Prometheus configuration

Pass the device address and password via scrape `params`, and relabel
`instance` to the device address — the same pattern used by
`snmp_exporter` and `blackbox_exporter`. Internal `__param_*` labels never
reach the stored time series, so the password is not persisted in
Prometheus's TSDB.

```yaml
scrape_configs:
  - job_name: meshcoretel
    metrics_path: /metrics
    scrape_interval: 60s   # firmware recommends polling no more than once/minute
    static_configs:
      - targets:
          - 192.168.0.10
        labels:
          __param_password: "the-device-admin-password"
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: vbart-meshcoretel-exporter:9642   # the exporter's own host:port
```

Add one `static_configs` entry (with its own `__param_password` label) per
device. Avoid scraping the same device concurrently from multiple jobs —
the firmware serves one request at a time.

## Security notes

- The device admin password travels as a query parameter from Prometheus to
  the exporter. Run the exporter on a trusted network next to Prometheus,
  and restrict access to Prometheus's own configuration (it already holds
  the password in the scrape config above).
- The exporter never logs the password, the device session token, or full
  request query strings.
- MeshCoreTel devices present a self-signed TLS certificate; the exporter
  intentionally disables certificate verification for device connections
  only (`InsecureSkipVerify`). This is safe on a trusted LAN but means the
  exporter does not authenticate the device's identity.
- Device connections are additionally pinned to TLS 1.2 with an explicit
  cipher suite list (including non-ECDHE RSA suites). This firmware's
  embedded TLS stack has been observed to only offer
  `TLS_RSA_WITH_AES_128_GCM_SHA256`, which Go's default client configuration
  does not send in its `ClientHello` (no forward secrecy); omitting this
  causes a `tls: handshake failure` before any certificate is even
  exchanged.
- The exporter's own HTTP server has no authentication. Do not expose it
  directly to untrusted networks.

## Metric reference

All metrics use the `meshcoretel_` prefix. Values shown for "Example" are
taken from [`examples/stats.json`](examples/stats.json).

### Meta metrics

| Metric | Type | Description |
|---|---|---|
| `meshcoretel_up` | gauge | `1` if the last scrape of the device succeeded, `0` otherwise. |
| `meshcoretel_scrape_duration_seconds` | gauge | Duration of the device scrape (login + stats fetch). |
| `meshcoretel_build_info{version}` | gauge, always 1 | Firmware build/version string, from the device's `ver` command. Only present when `--collect.version`/`VBART_COLLECT_VERSION` is enabled **and** the device returned a usable response; see [CLI flags](#cli-flags) for why this is opt-in. |

### `core`

| Metric | Type | Unit | Source field |
|---|---|---|---|
| `meshcoretel_core_battery_volts` | gauge | volts | `battery_mv` |
| `meshcoretel_core_battery_percent` | gauge | percent | `battery_pct` |
| `meshcoretel_core_battery_display_percent` | gauge | percent | `battery_display_pct` |
| `meshcoretel_core_battery_min_volts` | gauge | volts | `battery_min_mv` |
| `meshcoretel_core_battery_max_volts` | gauge | volts | `battery_max_mv` |
| `meshcoretel_core_uptime_seconds_total` | counter | seconds | `uptime_secs` |
| `meshcoretel_core_cpu0_utilization_ratio` | gauge | ratio (0-1) | `core0_util` |
| `meshcoretel_core_errors_total` | counter | count | `errors` |
| `meshcoretel_core_queue_length` | gauge | count | `queue_len` |
| `meshcoretel_core_external_power` | gauge (bool) | 0/1 | `external_power` |
| `meshcoretel_core_charging` | gauge (bool) | 0/1 | `charging` |
| `meshcoretel_core_vbus` | gauge (bool) | 0/1 | `vbus` |

### `radio`

| Metric | Type | Unit | Source field |
|---|---|---|---|
| `meshcoretel_radio_noise_floor_dbm` | gauge | dBm | `noise_floor` |
| `meshcoretel_radio_last_rssi_dbm` | gauge | dBm | `last_rssi` |
| `meshcoretel_radio_last_snr_db` | gauge | dB | `last_snr` |
| `meshcoretel_radio_tx_air_seconds_total` | counter | seconds | `tx_air_secs` |
| `meshcoretel_radio_rx_air_seconds_total` | counter | seconds | `rx_air_secs` |

### `packets`

| Metric | Type | Source field |
|---|---|---|
| `meshcoretel_packets_recv_total` | counter | `recv` |
| `meshcoretel_packets_sent_total` | counter | `sent` |
| `meshcoretel_packets_flood_tx_total` | counter | `flood_tx` |
| `meshcoretel_packets_direct_tx_total` | counter | `direct_tx` |
| `meshcoretel_packets_flood_rx_total` | counter | `flood_rx` |
| `meshcoretel_packets_direct_rx_total` | counter | `direct_rx` |
| `meshcoretel_packets_recv_errors_total` | counter | `recv_errors` |
| `meshcoretel_packets_direct_dups_total` | counter | `direct_dups` |
| `meshcoretel_packets_flood_dups_total` | counter | `flood_dups` |
| `meshcoretel_packets_neighbors` | gauge | `neighbors` |

### `memory`

| Metric | Type | Unit | Source field |
|---|---|---|---|
| `meshcoretel_memory_heap_free_bytes` | gauge | bytes | `heap_free` |
| `meshcoretel_memory_heap_min_bytes` | gauge | bytes | `heap_min` |
| `meshcoretel_memory_heap_max_bytes` | gauge | bytes | `heap_max` |
| `meshcoretel_memory_psram_free_bytes` | gauge | bytes | `psram_free` |
| `meshcoretel_memory_psram_min_bytes` | gauge | bytes | `psram_min` |
| `meshcoretel_memory_psram_max_bytes` | gauge | bytes | `psram_max` |

### `wifi`

| Metric | Type | Unit | Source field(s) |
|---|---|---|---|
| `meshcoretel_wifi_info{ssid,ip,status,state,signal,powersave}` | gauge, always 1 | — | `ssid`, `ip`, `status`, `state`, `signal`, `powersave` |
| `meshcoretel_wifi_connected` | gauge (bool) | 0/1 | `connected` |
| `meshcoretel_wifi_rssi_dbm` | gauge | dBm | `rssi` |
| `meshcoretel_wifi_quality_percent` | gauge | percent | `quality` |
| `meshcoretel_wifi_status_code` | gauge | raw code | `code` |

### `services`

| Metric | Type | Unit | Source field(s) |
|---|---|---|---|
| `meshcoretel_services_info{mqtt_state,web_auth}` | gauge, always 1 | — | `mqtt_state`, `web_auth` |
| `meshcoretel_services_mqtt_connected` | gauge (bool) | 0/1 | `mqtt_connected` |
| `meshcoretel_services_web_enabled` | gauge (bool) | 0/1 | `web_enabled` |
| `meshcoretel_services_web_panel_up` | gauge (bool) | 0/1 | `web_panel_up` |
| `meshcoretel_services_archive_available` | gauge (bool) | 0/1 | `archive_available` |

### `sensors`

| Metric | Type | Unit | Source field |
|---|---|---|---|
| `meshcoretel_sensors_gps_enabled` | gauge (bool) | 0/1 | `gps_enabled` |
| `meshcoretel_sensors_gps_fix` | gauge (bool) | 0/1 | `gps_fix` |
| `meshcoretel_sensors_supply_voltage_volts` | gauge | volts | `supply_voltage_v` |
| `meshcoretel_sensors_mcu_temp_celsius` | gauge | °C | `mcu_temp_c` |

### `history`

| Metric | Type | Source field |
|---|---|---|
| `meshcoretel_history_active` | gauge (bool) | `active` |
| `meshcoretel_history_samples` | gauge | `samples` |
| `meshcoretel_history_sample_capacity` | gauge | `sample_capacity` |
| `meshcoretel_history_events` | gauge | `events` |
| `meshcoretel_history_event_capacity` | gauge | `event_capacity` |

The `events` array itself (individual timestamped event log entries) is
**not** exported — it is unbounded, relative-timestamp data unsuited to a
scrape-based model.

### `neighbors_detail` (per mesh neighbor)

Each entry is labeled `neighbor="<short id>"`.

| Metric | Type | Unit | Source field |
|---|---|---|---|
| `meshcoretel_neighbor_snr_db{neighbor}` | gauge | dB | `snr_db` |
| `meshcoretel_neighbor_heard_seconds{neighbor}` | gauge | seconds | `heard_secs_ago` |
| `meshcoretel_neighbor_advert_seconds{neighbor}` | gauge | seconds | `advert_secs_ago` |

`meshcoretel_neighbor_advert_seconds` is **omitted** for neighbors that have
never sent an advertisement — the firmware represents this with a sentinel
value near the `uint32` maximum (≥ 2³¹ in this exporter's check) rather than
a real elapsed time.

## Grafana dashboard

A ready-to-import dashboard is at
[`dashboards/vbart-meshcoretel-exporter.json`](dashboards/vbart-meshcoretel-exporter.json), covering
battery, uptime, MCU temperature, radio RSSI/SNR/noise floor, air-time,
packet rates/errors/duplicates, heap/PSRAM, Wi-Fi, neighbor count, and a
per-neighbor SNR/last-heard table. It is auto-provisioned in the
[demo stack](examples/compose/).

## Development

```sh
go build ./...
go vet ./...
golangci-lint run ./...
go test -race ./...
```

Unit tests decode [`examples/stats.json`](examples/stats.json) as a golden
fixture for the metrics flattener, and exercise the HTTP handler against a
mock device client covering success, missing parameters, wrong password,
unreachable target, disabled stats (503), and timeout scenarios.

## Firmware compatibility

Targets the `/login` and `/api/stats` endpoints documented at
<https://vbart.github.io/MeshCoreTel-firmware/api/>. Unknown JSON fields
are ignored, so newer firmware versions that add fields should continue to
work; new sections require an exporter update to be exposed as metrics.

## License

GPL-3.0-only — see [LICENSE](LICENSE). Runtime dependencies
(`prometheus/client_golang` and its transitive dependencies) are licensed
Apache-2.0, BSD-3-Clause, or MIT, all of which are compatible with
distribution under GPLv3.
