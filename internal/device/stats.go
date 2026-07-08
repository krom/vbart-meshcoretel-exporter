package device

import "encoding/json"

// SentinelThreshold is the smallest value treated as "never happened" for
// unsigned duration-since fields such as NeighborDetail.AdvertSecsAgo.
// The firmware represents "never" using values near the uint32 max.
const SentinelThreshold = 1 << 31

// Stats mirrors the JSON document returned by GET /api/stats. Unknown
// fields are ignored by encoding/json and require no special handling.
type Stats struct {
	Enabled  bool          `json:"enabled"`
	History  HistoryStats  `json:"history"`
	Archive  ArchiveStats  `json:"archive"`
	Core     CoreStats     `json:"core"`
	Radio    RadioStats    `json:"radio"`
	Packets  PacketsStats  `json:"packets"`
	Memory   MemoryStats   `json:"memory"`
	Wifi     WifiStats     `json:"wifi"`
	Services ServicesStats `json:"services"`
	Sensors  SensorsStats  `json:"sensors"`

	NeighborsDetail []NeighborDetail `json:"neighbors_detail"`
}

// HistoryStats reflects the device's in-memory sample history buffer.
type HistoryStats struct {
	Active                     bool `json:"active"`
	PSRAM                      bool `json:"psram"`
	Degraded                   bool `json:"degraded"`
	LiveOnly                   bool `json:"live_only"`
	Samples                    int  `json:"samples"`
	SampleCapacity             int  `json:"sample_capacity"`
	SampleIntervalSecs         int  `json:"sample_interval_secs"`
	ArchiveRestored            bool `json:"archive_restored"`
	ArchiveRestoredSamples     int  `json:"archive_restored_samples"`
	ArchiveSummaryIntervalSecs int  `json:"archive_summary_interval_secs"`
	Events                     int  `json:"events"`
	EventCapacity              int  `json:"event_capacity"`
}

// ArchiveStats describes the optional persistent archive backend.
type ArchiveStats struct {
	Logical    string `json:"logical"`
	Available  bool   `json:"available"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	TotalBytes int64  `json:"total_bytes"`
	UsedBytes  int64  `json:"used_bytes"`
}

// CoreStats covers battery, power, and CPU health.
type CoreStats struct {
	BatteryMV         int     `json:"battery_mv"`
	BatteryPct        int     `json:"battery_pct"`
	BatteryDisplayPct int     `json:"battery_display_pct"`
	BatteryMinMV      int     `json:"battery_min_mv"`
	BatteryMaxMV      int     `json:"battery_max_mv"`
	UptimeSecs        int64   `json:"uptime_secs"`
	Core0Util         float64 `json:"core0_util"`
	Errors            int64   `json:"errors"`
	QueueLen          int     `json:"queue_len"`
	ExternalPower     bool    `json:"external_power"`
	Charging          bool    `json:"charging"`
	VBus              bool    `json:"vbus"`
}

// RadioStats covers LoRa radio signal quality and air-time.
type RadioStats struct {
	NoiseFloor int     `json:"noise_floor"`
	LastRSSI   float64 `json:"last_rssi"`
	LastSNR    float64 `json:"last_snr"`
	TXAirSecs  int64   `json:"tx_air_secs"`
	RXAirSecs  int64   `json:"rx_air_secs"`
}

// PacketsStats covers mesh packet counters.
type PacketsStats struct {
	Recv       int64 `json:"recv"`
	Sent       int64 `json:"sent"`
	FloodTX    int64 `json:"flood_tx"`
	DirectTX   int64 `json:"direct_tx"`
	FloodRX    int64 `json:"flood_rx"`
	DirectRX   int64 `json:"direct_rx"`
	RecvErrors int64 `json:"recv_errors"`
	DirectDups int64 `json:"direct_dups"`
	FloodDups  int64 `json:"flood_dups"`
	Neighbors  int   `json:"neighbors"`
}

// MemoryStats covers heap and PSRAM utilization.
type MemoryStats struct {
	HeapFree  int64 `json:"heap_free"`
	HeapMin   int64 `json:"heap_min"`
	HeapMax   int64 `json:"heap_max"`
	PSRAMFree int64 `json:"psram_free"`
	PSRAMMin  int64 `json:"psram_min"`
	PSRAMMax  int64 `json:"psram_max"`
}

// WifiStats covers the device's Wi-Fi station connection.
type WifiStats struct {
	SSID      string `json:"ssid"`
	Status    string `json:"status"`
	Connected bool   `json:"connected"`
	State     string `json:"state"`
	Code      int    `json:"code"`
	IP        string `json:"ip"`
	RSSI      int    `json:"rssi"`
	Quality   int    `json:"quality"`
	Signal    string `json:"signal"`
	Powersave string `json:"powersave"`
}

// ServicesStats covers auxiliary service health flags.
type ServicesStats struct {
	MQTTConnected    bool   `json:"mqtt_connected"`
	MQTTState        string `json:"mqtt_state"`
	WebEnabled       bool   `json:"web_enabled"`
	WebPanelUp       bool   `json:"web_panel_up"`
	WebAuth          string `json:"web_auth"`
	ArchiveAvailable bool   `json:"archive_available"`
}

// SensorsStats covers onboard/attached sensor readings.
type SensorsStats struct {
	GPSEnabled     bool    `json:"gps_enabled"`
	GPSFix         bool    `json:"gps_fix"`
	SupplyVoltageV float64 `json:"supply_voltage_v"`
	MCUTempC       float64 `json:"mcu_temp_c"`
}

// NeighborDetail describes one entry in the device's neighbor table.
type NeighborDetail struct {
	ID            string  `json:"id"`
	FullID        string  `json:"full_id"`
	HeardSecsAgo  int64   `json:"heard_secs_ago"`
	AdvertSecsAgo int64   `json:"advert_secs_ago"`
	SNRDb         float64 `json:"snr_db"`
}

// IsAdvertSentinel reports whether AdvertSecsAgo represents "never
// advertised" rather than a real elapsed-time value.
func (n NeighborDetail) IsAdvertSentinel() bool {
	return n.AdvertSecsAgo >= SentinelThreshold
}

// ParseStats decodes a raw /api/stats JSON payload. Unknown fields are
// ignored so that future firmware additions do not break parsing.
func ParseStats(body []byte) (*Stats, error) {
	var s Stats
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
