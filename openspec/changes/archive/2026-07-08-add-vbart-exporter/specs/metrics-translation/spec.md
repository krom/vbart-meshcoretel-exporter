## ADDED Requirements

### Requirement: Flat Prometheus naming from hierarchical JSON
The exporter SHALL translate the device stats JSON into flat metric names of the form `meshcoretel_<section>_<field>[_<unit>|_total]`, where `<section>` is the top-level JSON object name (`core`, `radio`, `packets`, `memory`, `wifi`, `services`, `sensors`, `history`). Names SHALL follow Prometheus conventions: lowercase snake_case, base units, no unit prefixes in values.

#### Scenario: Nested numeric field
- **WHEN** the stats JSON contains `"radio": {"noise_floor": -88}`
- **THEN** the output contains `meshcoretel_radio_noise_floor_dbm -88`

#### Scenario: Unknown fields ignored
- **WHEN** the stats JSON contains a field not known to the exporter
- **THEN** translation succeeds and the unknown field produces no metric and no error

### Requirement: Unit normalization
Values SHALL be converted to Prometheus base units: `*_secs` fields to `_seconds`, `*_mv` to `_volts` (×0.001), byte quantities to `_bytes`, temperatures to `_celsius`, percentages to `_ratio` (0–1) or kept as `_percent` only where the source is already a display percentage, dBm/dB values suffixed `_dbm`/`_db`.

#### Scenario: Millivolts to volts
- **WHEN** the stats JSON contains `"battery_mv": 4296`
- **THEN** the output contains `meshcoretel_core_battery_volts 4.296`

#### Scenario: Uptime seconds
- **WHEN** the stats JSON contains `"uptime_secs": 894275`
- **THEN** the output contains `meshcoretel_core_uptime_seconds_total 894275` as a counter

### Requirement: Metric type assignment
Monotonically increasing device-lifetime totals (packet counters, air-time seconds, error counts) SHALL be exposed as counters with the `_total` suffix. Point-in-time values (battery, memory, RSSI/SNR, utilization, queue length, neighbor count) SHALL be gauges. Every metric SHALL carry a HELP string.

#### Scenario: Packet counter
- **WHEN** the stats JSON contains `"packets": {"recv": 64875}`
- **THEN** the output contains counter `meshcoretel_packets_recv_total 64875`

#### Scenario: Memory gauge
- **WHEN** the stats JSON contains `"memory": {"heap_free": 150628}`
- **THEN** the output contains gauge `meshcoretel_memory_heap_free_bytes 150628`

### Requirement: Boolean and enum translation
Boolean JSON fields SHALL become 0/1 gauges (e.g. `meshcoretel_core_external_power`, `meshcoretel_wifi_connected`, `meshcoretel_services_mqtt_connected`). String enumerations and identity strings (Wi-Fi `ssid`, `ip`, `status`, `signal`, services `mqtt_state`, `web_auth`) SHALL be exposed as labels on `_info`-style gauges with value 1 (e.g. `meshcoretel_wifi_info{ssid="...",ip="...",status="..."} 1`), never as metric values.

#### Scenario: Boolean field
- **WHEN** the stats JSON contains `"charging": false`
- **THEN** the output contains `meshcoretel_core_charging 0`

#### Scenario: String fields as info labels
- **WHEN** the stats JSON contains `"wifi": {"ssid": "KRIOT01", "ip": "10.110.6.39", "connected": true}`
- **THEN** the output contains `meshcoretel_wifi_info{ssid="KRIOT01",ip="10.110.6.39"} 1` and `meshcoretel_wifi_connected 1`

### Requirement: Per-neighbor metrics
Each entry of `neighbors_detail[]` SHALL produce gauges labeled with the neighbor's short id: `meshcoretel_neighbor_snr_db{neighbor="<id>"}`, `meshcoretel_neighbor_heard_seconds{neighbor="<id>"}`, and `meshcoretel_neighbor_advert_seconds{neighbor="<id>"}`. Sentinel values (fields ≥ 2^31, meaning "never") SHALL cause that sample to be omitted rather than emitted.

#### Scenario: Neighbor gauges
- **WHEN** `neighbors_detail` contains `{"id": "10103E", "heard_secs_ago": 874, "snr_db": -10.75}`
- **THEN** the output contains `meshcoretel_neighbor_heard_seconds{neighbor="10103E"} 874` and `meshcoretel_neighbor_snr_db{neighbor="10103E"} -10.75`

#### Scenario: Sentinel advert age
- **WHEN** a neighbor entry has `"advert_secs_ago": 4294967270`
- **THEN** no `meshcoretel_neighbor_advert_seconds` sample is emitted for that neighbor

### Requirement: Meta metrics
Every scrape response SHALL include `meshcoretel_up` (1 on success, 0 on failure) and `meshcoretel_scrape_duration_seconds` (wall time of the device round-trip).

#### Scenario: Meta metrics present on success
- **WHEN** a scrape succeeds
- **THEN** the output contains `meshcoretel_up 1` and a positive `meshcoretel_scrape_duration_seconds`

### Requirement: Reference sample coverage
The translation SHALL be verified against `examples/stats.json`: decoding that document SHALL produce metrics covering at minimum the `core`, `radio`, `packets`, `memory`, `wifi`, `services`, `sensors`, and `neighbors_detail` sections, and the repository documentation SHALL contain a metric reference table generated from the same mapping.

#### Scenario: Golden file translation
- **WHEN** `examples/stats.json` is passed through the flattener in tests
- **THEN** the emitted metric set matches the checked-in expected output
