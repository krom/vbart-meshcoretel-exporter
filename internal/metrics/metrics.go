// Package metrics translates MeshCoreTel device stats JSON into Prometheus metrics.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/krom/vbart-meshcoretel-exporter/internal/device"
)

const namespace = "meshcoretel"

// Collect builds a slice of ready-to-register Prometheus metrics from a
// decoded Stats document. Each call produces independent metric objects so
// callers can register them into a fresh, per-scrape registry.
func Collect(stats *device.Stats) []prometheus.Metric {
	var m []prometheus.Metric

	m = append(m, collectCore(stats.Core)...)
	m = append(m, collectRadio(stats.Radio)...)
	m = append(m, collectPackets(stats.Packets)...)
	m = append(m, collectMemory(stats.Memory)...)
	m = append(m, collectWifi(stats.Wifi)...)
	m = append(m, collectServices(stats.Services)...)
	m = append(m, collectSensors(stats.Sensors)...)
	m = append(m, collectHistory(stats.History)...)
	m = append(m, collectNeighbors(stats.NeighborsDetail)...)

	return m
}

func gauge(name, help string, value float64, labels prometheus.Labels) prometheus.Metric {
	return mustConst(name, help, prometheus.GaugeValue, value, labels)
}

func counter(name, help string, value float64, labels prometheus.Labels) prometheus.Metric {
	return mustConst(name, help, prometheus.CounterValue, value, labels)
}

func boolGauge(name, help string, value bool) prometheus.Metric {
	v := 0.0
	if value {
		v = 1.0
	}
	return gauge(name, help, v, nil)
}

func mustConst(name, help string, valueType prometheus.ValueType, value float64, labels prometheus.Labels) prometheus.Metric {
	labelNames := make([]string, 0, len(labels))
	labelValues := make([]string, 0, len(labels))
	for k, v := range labels {
		labelNames = append(labelNames, k)
		labelValues = append(labelValues, v)
	}
	desc := prometheus.NewDesc(name, help, labelNames, nil)
	metric, err := prometheus.NewConstMetric(desc, valueType, value, labelValues...)
	if err != nil {
		// Only fails on malformed descriptors, which is a programming error
		// caught immediately by unit tests; panicking here surfaces it loudly.
		panic(err)
	}
	return metric
}

func fq(section, field string) string {
	return namespace + "_" + section + "_" + field
}

func collectCore(c device.CoreStats) []prometheus.Metric {
	return []prometheus.Metric{
		gauge(fq("core", "battery_volts"), "Battery voltage in volts.", float64(c.BatteryMV)/1000, nil),
		gauge(fq("core", "battery_percent"), "Battery charge percentage as reported by the device (-1 if unknown).", float64(c.BatteryPct), nil),
		gauge(fq("core", "battery_display_percent"), "Battery charge percentage shown on the device UI.", float64(c.BatteryDisplayPct), nil),
		gauge(fq("core", "battery_min_volts"), "Configured minimum battery voltage in volts.", float64(c.BatteryMinMV)/1000, nil),
		gauge(fq("core", "battery_max_volts"), "Configured maximum battery voltage in volts.", float64(c.BatteryMaxMV)/1000, nil),
		counter(fq("core", "uptime_seconds_total"), "Device uptime in seconds since last boot.", float64(c.UptimeSecs), nil),
		gauge(fq("core", "cpu0_utilization_ratio"), "Core 0 CPU utilization as a ratio between 0 and 1.", c.Core0Util/100, nil),
		counter(fq("core", "errors_total"), "Total internal error count since last boot.", float64(c.Errors), nil),
		gauge(fq("core", "queue_length"), "Current internal work queue length.", float64(c.QueueLen), nil),
		boolGauge(fq("core", "external_power"), "Whether external power is connected (1) or not (0).", c.ExternalPower),
		boolGauge(fq("core", "charging"), "Whether the battery is currently charging (1) or not (0).", c.Charging),
		boolGauge(fq("core", "vbus"), "Whether USB VBUS power is present (1) or not (0).", c.VBus),
	}
}

func collectRadio(r device.RadioStats) []prometheus.Metric {
	return []prometheus.Metric{
		gauge(fq("radio", "noise_floor_dbm"), "LoRa radio noise floor in dBm.", float64(r.NoiseFloor), nil),
		gauge(fq("radio", "last_rssi_dbm"), "RSSI of the last received packet in dBm.", r.LastRSSI, nil),
		gauge(fq("radio", "last_snr_db"), "SNR of the last received packet in dB.", r.LastSNR, nil),
		counter(fq("radio", "tx_air_seconds_total"), "Total transmit air-time in seconds since last boot.", float64(r.TXAirSecs), nil),
		counter(fq("radio", "rx_air_seconds_total"), "Total receive air-time in seconds since last boot.", float64(r.RXAirSecs), nil),
	}
}

func collectPackets(p device.PacketsStats) []prometheus.Metric {
	return []prometheus.Metric{
		counter(fq("packets", "recv_total"), "Total packets received since last boot.", float64(p.Recv), nil),
		counter(fq("packets", "sent_total"), "Total packets sent since last boot.", float64(p.Sent), nil),
		counter(fq("packets", "flood_tx_total"), "Total flood-routed packets transmitted since last boot.", float64(p.FloodTX), nil),
		counter(fq("packets", "direct_tx_total"), "Total direct-routed packets transmitted since last boot.", float64(p.DirectTX), nil),
		counter(fq("packets", "flood_rx_total"), "Total flood-routed packets received since last boot.", float64(p.FloodRX), nil),
		counter(fq("packets", "direct_rx_total"), "Total direct-routed packets received since last boot.", float64(p.DirectRX), nil),
		counter(fq("packets", "recv_errors_total"), "Total packet receive errors since last boot.", float64(p.RecvErrors), nil),
		counter(fq("packets", "direct_dups_total"), "Total duplicate direct-routed packets since last boot.", float64(p.DirectDups), nil),
		counter(fq("packets", "flood_dups_total"), "Total duplicate flood-routed packets since last boot.", float64(p.FloodDups), nil),
		gauge(fq("packets", "neighbors"), "Current number of known mesh neighbors.", float64(p.Neighbors), nil),
	}
}

func collectMemory(mem device.MemoryStats) []prometheus.Metric {
	return []prometheus.Metric{
		gauge(fq("memory", "heap_free_bytes"), "Free heap memory in bytes.", float64(mem.HeapFree), nil),
		gauge(fq("memory", "heap_min_bytes"), "Minimum observed free heap memory in bytes.", float64(mem.HeapMin), nil),
		gauge(fq("memory", "heap_max_bytes"), "Maximum observed free heap memory in bytes.", float64(mem.HeapMax), nil),
		gauge(fq("memory", "psram_free_bytes"), "Free PSRAM in bytes.", float64(mem.PSRAMFree), nil),
		gauge(fq("memory", "psram_min_bytes"), "Minimum observed free PSRAM in bytes.", float64(mem.PSRAMMin), nil),
		gauge(fq("memory", "psram_max_bytes"), "Maximum observed free PSRAM in bytes.", float64(mem.PSRAMMax), nil),
	}
}

func collectWifi(w device.WifiStats) []prometheus.Metric {
	info := gauge(fq("wifi", "info"), "Wi-Fi station identity information; value is always 1.", 1, prometheus.Labels{
		"ssid":      w.SSID,
		"ip":        w.IP,
		"status":    w.Status,
		"state":     w.State,
		"signal":    w.Signal,
		"powersave": w.Powersave,
	})
	return []prometheus.Metric{
		info,
		boolGauge(fq("wifi", "connected"), "Whether the device is connected to Wi-Fi (1) or not (0).", w.Connected),
		gauge(fq("wifi", "rssi_dbm"), "Wi-Fi RSSI in dBm.", float64(w.RSSI), nil),
		gauge(fq("wifi", "quality_percent"), "Wi-Fi link quality percentage.", float64(w.Quality), nil),
		gauge(fq("wifi", "status_code"), "Raw Wi-Fi status code reported by the device.", float64(w.Code), nil),
	}
}

func collectServices(s device.ServicesStats) []prometheus.Metric {
	info := gauge(fq("services", "info"), "Service state identity information; value is always 1.", 1, prometheus.Labels{
		"mqtt_state": s.MQTTState,
		"web_auth":   s.WebAuth,
	})
	return []prometheus.Metric{
		info,
		boolGauge(fq("services", "mqtt_connected"), "Whether the MQTT client is connected (1) or not (0).", s.MQTTConnected),
		boolGauge(fq("services", "web_enabled"), "Whether the web panel is enabled (1) or not (0).", s.WebEnabled),
		boolGauge(fq("services", "web_panel_up"), "Whether the web panel is up (1) or not (0).", s.WebPanelUp),
		boolGauge(fq("services", "archive_available"), "Whether the archive backend is available (1) or not (0).", s.ArchiveAvailable),
	}
}

func collectSensors(s device.SensorsStats) []prometheus.Metric {
	return []prometheus.Metric{
		boolGauge(fq("sensors", "gps_enabled"), "Whether the GPS sensor is enabled (1) or not (0).", s.GPSEnabled),
		boolGauge(fq("sensors", "gps_fix"), "Whether the GPS sensor has a fix (1) or not (0).", s.GPSFix),
		gauge(fq("sensors", "supply_voltage_volts"), "Measured supply voltage in volts.", s.SupplyVoltageV, nil),
		gauge(fq("sensors", "mcu_temp_celsius"), "MCU temperature in degrees Celsius.", s.MCUTempC, nil),
	}
}

func collectHistory(h device.HistoryStats) []prometheus.Metric {
	return []prometheus.Metric{
		boolGauge(fq("history", "active"), "Whether the in-memory sample history is active (1) or not (0).", h.Active),
		gauge(fq("history", "samples"), "Number of samples currently held in history.", float64(h.Samples), nil),
		gauge(fq("history", "sample_capacity"), "Maximum number of samples the history buffer can hold.", float64(h.SampleCapacity), nil),
		gauge(fq("history", "events"), "Number of events currently held in history.", float64(h.Events), nil),
		gauge(fq("history", "event_capacity"), "Maximum number of events the history buffer can hold.", float64(h.EventCapacity), nil),
	}
}

func collectNeighbors(neighbors []device.NeighborDetail) []prometheus.Metric {
	m := make([]prometheus.Metric, 0, len(neighbors)*3)
	for _, n := range neighbors {
		labels := prometheus.Labels{"neighbor": n.ID}
		m = append(m,
			gauge(namespace+"_neighbor_snr_db", "Signal-to-noise ratio last heard from this neighbor, in dB.", n.SNRDb, labels),
			gauge(namespace+"_neighbor_heard_seconds", "Seconds since this neighbor was last heard.", float64(n.HeardSecsAgo), labels),
		)
		if !n.IsAdvertSentinel() {
			m = append(m, gauge(namespace+"_neighbor_advert_seconds", "Seconds since this neighbor last sent an advertisement.", float64(n.AdvertSecsAgo), labels))
		}
	}
	return m
}

// Up returns the meshcoretel_up meta metric.
func Up(up bool) prometheus.Metric {
	v := 0.0
	if up {
		v = 1.0
	}
	return gauge(namespace+"_up", "Whether the last scrape of the device succeeded (1) or not (0).", v, nil)
}

// ScrapeDuration returns the meshcoretel_scrape_duration_seconds meta metric.
func ScrapeDuration(seconds float64) prometheus.Metric {
	return gauge(namespace+"_scrape_duration_seconds", "Duration of the device scrape (login + stats fetch) in seconds.", seconds, nil)
}

// BuildInfo returns the meshcoretel_build_info info-style metric, carrying
// the device's firmware build/version string as a label (value is always
// 1), following the node_exporter build_info convention.
func BuildInfo(version string) prometheus.Metric {
	return gauge(namespace+"_build_info", "Firmware build/version string reported by the device; value is always 1.", 1, prometheus.Labels{"version": version})
}
