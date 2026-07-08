package metrics

import (
	"os"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/krom/vbart-meshcoretel-exporter/internal/device"
)

func loadFixture(t *testing.T) *device.Stats {
	t.Helper()
	body, err := os.ReadFile("../../examples/stats.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	stats, err := device.ParseStats(body)
	if err != nil {
		t.Fatalf("ParseStats() error = %v", err)
	}
	return stats
}

func TestCollectGoldenFixture(t *testing.T) {
	stats := loadFixture(t)
	ms := Collect(stats)

	if len(ms) == 0 {
		t.Fatal("Collect() returned no metrics")
	}

	names := map[string]bool{}
	for _, m := range ms {
		names[m.Desc().String()] = true
	}

	// Spot-check presence of at least one metric per required section.
	requiredSubstrings := []string{
		"meshcoretel_core_",
		"meshcoretel_radio_",
		"meshcoretel_packets_",
		"meshcoretel_memory_",
		"meshcoretel_wifi_",
		"meshcoretel_services_",
		"meshcoretel_sensors_",
		"meshcoretel_neighbor_",
	}
	for _, sub := range requiredSubstrings {
		found := false
		for n := range names {
			if strings.Contains(n, sub) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no metric found containing %q", sub)
		}
	}
}

func TestUnitConversions(t *testing.T) {
	stats := loadFixture(t)
	ms := Collect(stats)

	want := map[string]float64{
		"meshcoretel_core_battery_volts":        4.296,
		"meshcoretel_core_uptime_seconds_total": 894275,
		"meshcoretel_memory_heap_free_bytes":    150628,
		"meshcoretel_sensors_mcu_temp_celsius":  54.0,
		"meshcoretel_packets_recv_total":        64875,
	}

	got := map[string]float64{}
	for _, m := range ms {
		var pb dto.Metric
		if err := m.Write(&pb); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		name := metricBaseName(m)
		if len(pb.Label) > 0 {
			continue // skip labeled metrics for this exact-value check
		}
		if pb.Gauge != nil {
			got[name] = pb.Gauge.GetValue()
		}
		if pb.Counter != nil {
			got[name] = pb.Counter.GetValue()
		}
	}

	for name, wantVal := range want {
		gotVal, ok := got[name]
		if !ok {
			t.Errorf("metric %s not found", name)
			continue
		}
		if gotVal != wantVal {
			t.Errorf("metric %s = %v, want %v", name, gotVal, wantVal)
		}
	}
}

func TestNeighborSentinelOmitted(t *testing.T) {
	stats := loadFixture(t)
	ms := Collect(stats)

	firstNeighborID := stats.NeighborsDetail[0].ID
	if !stats.NeighborsDetail[0].IsAdvertSentinel() {
		t.Fatalf("fixture assumption changed: first neighbor is no longer a sentinel")
	}

	for _, m := range ms {
		if metricBaseName(m) != "meshcoretel_neighbor_advert_seconds" {
			continue
		}
		var pb dto.Metric
		if err := m.Write(&pb); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		for _, l := range pb.Label {
			if l.GetName() == "neighbor" && l.GetValue() == firstNeighborID {
				t.Errorf("expected no advert_seconds metric for sentinel neighbor %s", firstNeighborID)
			}
		}
	}
}

func TestBooleanTranslation(t *testing.T) {
	stats := loadFixture(t)
	ms := Collect(stats)

	for _, m := range ms {
		if metricBaseName(m) != "meshcoretel_core_charging" {
			continue
		}
		var pb dto.Metric
		if err := m.Write(&pb); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if pb.Gauge.GetValue() != 0 {
			t.Errorf("meshcoretel_core_charging = %v, want 0 (fixture has charging=false)", pb.Gauge.GetValue())
		}
		return
	}
	t.Fatal("meshcoretel_core_charging metric not found")
}

func TestUpAndScrapeDuration(t *testing.T) {
	up := Up(true)
	var pb dto.Metric
	if err := up.Write(&pb); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if pb.Gauge.GetValue() != 1 {
		t.Errorf("Up(true) = %v, want 1", pb.Gauge.GetValue())
	}

	down := Up(false)
	var pbDown dto.Metric
	if err := down.Write(&pbDown); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if pbDown.Gauge.GetValue() != 0 {
		t.Errorf("Up(false) = %v, want 0", pbDown.Gauge.GetValue())
	}

	dur := ScrapeDuration(0.25)
	var pbDur dto.Metric
	if err := dur.Write(&pbDur); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if pbDur.Gauge.GetValue() != 0.25 {
		t.Errorf("ScrapeDuration(0.25) = %v, want 0.25", pbDur.Gauge.GetValue())
	}
}

func TestBuildInfo(t *testing.T) {
	const version = "v1.16.0-vbart-meshcoretel-v1.2.0-1817248 (Build: 07-Jun-2026)"
	m := BuildInfo(version)

	var pb dto.Metric
	if err := m.Write(&pb); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if pb.Gauge.GetValue() != 1 {
		t.Errorf("BuildInfo() value = %v, want 1", pb.Gauge.GetValue())
	}

	found := false
	for _, l := range pb.Label {
		if l.GetName() == "version" && l.GetValue() == version {
			found = true
		}
	}
	if !found {
		t.Errorf("BuildInfo() missing version label %q, got %v", version, pb.Label)
	}
}

func metricBaseName(m prometheus.Metric) string {
	// Desc().String() looks like: Desc{fqName: "name", help: "...", ...}
	s := m.Desc().String()
	const marker = `fqName: "`
	i := strings.Index(s, marker)
	if i < 0 {
		return ""
	}
	rest := s[i+len(marker):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return ""
	}
	return rest[:j]
}
