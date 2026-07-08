package exporter

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/krom/vbart-meshcoretel-exporter/internal/device"
	"github.com/krom/vbart-meshcoretel-exporter/internal/metrics"
)

// DeviceClient is the subset of device.Client used by the scrape handler,
// allowing tests to substitute a mock device.
type DeviceClient interface {
	Login(ctx context.Context, target, password string) (string, error)
	FetchStats(ctx context.Context, target, token string) ([]byte, error)
	FetchVersion(ctx context.Context, target, token string) ([]byte, error)
}

// ScrapeHandler serves GET /metrics?target=&password= by performing a
// fresh device login and stats fetch on every request.
type ScrapeHandler struct {
	Client         DeviceClient
	DefaultTimeout time.Duration
	Logger         *slog.Logger

	// CollectVersion enables an additional device round-trip per scrape
	// (the "ver" command) to expose meshcoretel_build_info. Off by
	// default: see README for why this should not be left on permanently.
	CollectVersion bool
}

func (h *ScrapeHandler) logger() *slog.Logger {
	if h.Logger != nil {
		return h.Logger
	}
	return slog.Default()
}

// ServeHTTP implements http.Handler.
func (h *ScrapeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	password := r.URL.Query().Get("password")

	if target == "" || password == "" {
		http.Error(w, "target and password query parameters are required", http.StatusBadRequest)
		return
	}

	timeout := h.effectiveTimeout(r)
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	reg := prometheus.NewRegistry()
	start := time.Now()

	stats, token, err := h.fetchDeviceStats(ctx, target, password)
	duration := time.Since(start).Seconds()

	if err != nil {
		h.logger().Warn("scrape failed", "target", target, "error", err.Error())
		reg.MustRegister(constCollector{metrics.Up(false), metrics.ScrapeDuration(duration)})
	} else {
		ms := metrics.Collect(stats)
		ms = append(ms, metrics.Up(true), metrics.ScrapeDuration(duration))
		if h.CollectVersion {
			if bi, ok := h.fetchBuildInfo(ctx, target, token); ok {
				ms = append(ms, bi)
			}
		}
		reg.MustRegister(constCollector(ms))
	}

	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

func (h *ScrapeHandler) fetchDeviceStats(ctx context.Context, target, password string) (*device.Stats, string, error) {
	token, err := h.Client.Login(ctx, target, password)
	if err != nil {
		return nil, "", err
	}

	body, err := h.Client.FetchStats(ctx, target, token)
	if err != nil {
		return nil, "", err
	}

	stats, err := device.ParseStats(body)
	if err != nil {
		return nil, "", errors.New("parse stats: " + err.Error())
	}
	return stats, token, nil
}

// fetchBuildInfo runs the "ver" command using the scrape's existing session
// token (no second login) and returns a meshcoretel_build_info metric. Any
// failure or unusable response is logged and reported as ok=false, so the
// caller can skip the metric without affecting the rest of the scrape.
func (h *ScrapeHandler) fetchBuildInfo(ctx context.Context, target, token string) (prometheus.Metric, bool) {
	raw, err := h.Client.FetchVersion(ctx, target, token)
	if err != nil {
		h.logger().Warn("build-info fetch failed", "target", target, "error", err.Error())
		return nil, false
	}

	version, ok := device.ParseVersion(raw)
	if !ok {
		h.logger().Warn("build-info response unusable", "target", target)
		return nil, false
	}

	return metrics.BuildInfo(version), true
}

// effectiveTimeout honors Prometheus's scrape timeout header when it is
// smaller than the configured default, leaving a small safety margin.
func (h *ScrapeHandler) effectiveTimeout(r *http.Request) time.Duration {
	def := h.DefaultTimeout
	if def <= 0 {
		def = 10 * time.Second
	}

	raw := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds")
	if raw == "" {
		return def
	}

	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil || seconds <= 0 {
		return def
	}

	headerTimeout := time.Duration(seconds*float64(time.Second)) - 500*time.Millisecond
	if headerTimeout <= 0 {
		return def
	}
	if headerTimeout < def {
		return headerTimeout
	}
	return def
}

// constCollector adapts a fixed slice of prometheus.Metric to the
// prometheus.Collector interface for one-shot, per-scrape registries.
type constCollector []prometheus.Metric

func (c constCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range c {
		ch <- m.Desc()
	}
}

func (c constCollector) Collect(ch chan<- prometheus.Metric) {
	for _, m := range c {
		ch <- m
	}
}
