package exporter

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/krom/vbart-meshcoretel-exporter/internal/device"
)

type mockClient struct {
	loginToken string
	loginErr   error
	statsBody  []byte
	statsErr   error

	loginDelay time.Duration

	versionBody        []byte
	versionErr         error
	versionDelay       time.Duration
	fetchVersionCalled bool
}

func (m *mockClient) Login(ctx context.Context, target, password string) (string, error) {
	if m.loginDelay > 0 {
		select {
		case <-time.After(m.loginDelay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	if m.loginErr != nil {
		return "", m.loginErr
	}
	return m.loginToken, nil
}

func (m *mockClient) FetchStats(ctx context.Context, target, token string) ([]byte, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.statsBody, nil
}

func (m *mockClient) FetchVersion(ctx context.Context, target, token string) ([]byte, error) {
	m.fetchVersionCalled = true
	if m.versionDelay > 0 {
		select {
		case <-time.After(m.versionDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.versionErr != nil {
		return nil, m.versionErr
	}
	return m.versionBody, nil
}

func testLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestScrapeHandlerSuccess(t *testing.T) {
	statsJSON, err := os.ReadFile("../../examples/stats.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	h := &ScrapeHandler{
		Client:         &mockClient{loginToken: "tok", statsBody: statsJSON},
		DefaultTimeout: 2 * time.Second,
		Logger:         testLogger(&bytes.Buffer{}),
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics?target=192.168.0.10&password=secret", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "meshcoretel_up 1") {
		t.Errorf("body missing meshcoretel_up 1:\n%s", body)
	}
	if !strings.Contains(body, "meshcoretel_core_battery_volts") {
		t.Errorf("body missing device metrics:\n%s", body)
	}
	if !strings.Contains(body, "meshcoretel_scrape_duration_seconds") {
		t.Errorf("body missing scrape duration metric:\n%s", body)
	}
}

func TestScrapeHandlerMissingParams(t *testing.T) {
	h := &ScrapeHandler{Client: &mockClient{}, DefaultTimeout: time.Second}

	cases := []string{
		"/metrics",
		"/metrics?target=192.168.0.10",
		"/metrics?password=secret",
	}
	for _, url := range cases {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("url %q: status = %d, want 400", url, rec.Code)
		}
	}
}

func TestScrapeHandlerWrongPassword(t *testing.T) {
	var logBuf bytes.Buffer
	h := &ScrapeHandler{
		Client:         &mockClient{loginErr: device.ErrUnauthorized},
		DefaultTimeout: time.Second,
		Logger:         testLogger(&logBuf),
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics?target=192.168.0.10&password=wrong", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "meshcoretel_up 0") {
		t.Errorf("body missing meshcoretel_up 0:\n%s", rec.Body.String())
	}
	if strings.Contains(logBuf.String(), "wrong") {
		t.Errorf("log output leaked password: %s", logBuf.String())
	}
}

func TestScrapeHandlerUnreachable(t *testing.T) {
	h := &ScrapeHandler{
		Client:         &mockClient{loginErr: errors.New("dial tcp: connection refused")},
		DefaultTimeout: time.Second,
		Logger:         testLogger(&bytes.Buffer{}),
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics?target=10.0.0.99&password=secret", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "meshcoretel_up 0") {
		t.Errorf("body missing meshcoretel_up 0:\n%s", rec.Body.String())
	}
}

func TestScrapeHandlerStatsUnavailable(t *testing.T) {
	h := &ScrapeHandler{
		Client:         &mockClient{loginToken: "tok", statsErr: device.ErrStatsUnavailable},
		DefaultTimeout: time.Second,
		Logger:         testLogger(&bytes.Buffer{}),
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics?target=192.168.0.10&password=secret", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "meshcoretel_up 0") {
		t.Errorf("body missing meshcoretel_up 0:\n%s", rec.Body.String())
	}
}

func TestScrapeHandlerTimeout(t *testing.T) {
	h := &ScrapeHandler{
		Client:         &mockClient{loginToken: "tok", loginDelay: 200 * time.Millisecond},
		DefaultTimeout: 20 * time.Millisecond,
		Logger:         testLogger(&bytes.Buffer{}),
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics?target=192.168.0.10&password=secret", nil)
	rec := httptest.NewRecorder()

	start := time.Now()
	h.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if elapsed > 150*time.Millisecond {
		t.Fatalf("handler took %v, expected it to time out near 20ms", elapsed)
	}
	if !strings.Contains(rec.Body.String(), "meshcoretel_up 0") {
		t.Errorf("body missing meshcoretel_up 0:\n%s", rec.Body.String())
	}
}

func TestScrapeHandlerVersionCollectionDisabled(t *testing.T) {
	statsJSON, err := os.ReadFile("../../examples/stats.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	client := &mockClient{loginToken: "tok", statsBody: statsJSON, versionBody: []byte("v1.16.0")}
	h := &ScrapeHandler{
		Client:         client,
		DefaultTimeout: 2 * time.Second,
		Logger:         testLogger(&bytes.Buffer{}),
		CollectVersion: false,
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics?target=192.168.0.10&password=secret", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if client.fetchVersionCalled {
		t.Error("FetchVersion called even though collection is disabled")
	}
	if strings.Contains(rec.Body.String(), "meshcoretel_build_info") {
		t.Errorf("body contains meshcoretel_build_info with collection disabled:\n%s", rec.Body.String())
	}
}

func TestScrapeHandlerVersionCollectionSuccess(t *testing.T) {
	statsJSON, err := os.ReadFile("../../examples/stats.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	const version = "v1.16.0-vbart-meshcoretel-v1.2.0-1817248 (Build: 07-Jun-2026)"

	h := &ScrapeHandler{
		Client:         &mockClient{loginToken: "tok", statsBody: statsJSON, versionBody: []byte(version)},
		DefaultTimeout: 2 * time.Second,
		Logger:         testLogger(&bytes.Buffer{}),
		CollectVersion: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics?target=192.168.0.10&password=secret", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "meshcoretel_build_info") || !strings.Contains(body, version) {
		t.Errorf("body missing meshcoretel_build_info with version %q:\n%s", version, body)
	}
}

func TestScrapeHandlerVersionCollectionTimeout(t *testing.T) {
	statsJSON, err := os.ReadFile("../../examples/stats.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	h := &ScrapeHandler{
		Client:         &mockClient{loginToken: "tok", statsBody: statsJSON, versionDelay: 200 * time.Millisecond},
		DefaultTimeout: 50 * time.Millisecond,
		Logger:         testLogger(&bytes.Buffer{}),
		CollectVersion: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics?target=192.168.0.10&password=secret", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "meshcoretel_up 1") {
		t.Errorf("body missing meshcoretel_up 1:\n%s", body)
	}
	if strings.Contains(body, "meshcoretel_build_info") {
		t.Errorf("body contains meshcoretel_build_info despite version fetch timing out:\n%s", body)
	}
}

func TestScrapeHandlerVersionCollectionUnparseable(t *testing.T) {
	statsJSON, err := os.ReadFile("../../examples/stats.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	h := &ScrapeHandler{
		Client:         &mockClient{loginToken: "tok", statsBody: statsJSON, versionBody: []byte("   ")},
		DefaultTimeout: 2 * time.Second,
		Logger:         testLogger(&bytes.Buffer{}),
		CollectVersion: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics?target=192.168.0.10&password=secret", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "meshcoretel_up 1") {
		t.Errorf("body missing meshcoretel_up 1:\n%s", body)
	}
	if strings.Contains(body, "meshcoretel_build_info") {
		t.Errorf("body contains meshcoretel_build_info despite unparseable version response:\n%s", body)
	}
}

func TestScrapeHandlerVersionNotFetchedWhenStatsFails(t *testing.T) {
	client := &mockClient{loginErr: device.ErrUnauthorized, versionBody: []byte("v1.16.0")}
	h := &ScrapeHandler{
		Client:         client,
		DefaultTimeout: time.Second,
		Logger:         testLogger(&bytes.Buffer{}),
		CollectVersion: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics?target=192.168.0.10&password=wrong", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if client.fetchVersionCalled {
		t.Error("FetchVersion called even though the stats round-trip failed")
	}
	if !strings.Contains(rec.Body.String(), "meshcoretel_up 0") {
		t.Errorf("body missing meshcoretel_up 0:\n%s", rec.Body.String())
	}
}

func TestEffectiveTimeoutHonorsPrometheusHeader(t *testing.T) {
	h := &ScrapeHandler{DefaultTimeout: 10 * time.Second}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("X-Prometheus-Scrape-Timeout-Seconds", "2")

	got := h.effectiveTimeout(req)
	want := 1500 * time.Millisecond
	if got != want {
		t.Errorf("effectiveTimeout() = %v, want %v", got, want)
	}
}

func TestEffectiveTimeoutIgnoresLargerHeader(t *testing.T) {
	h := &ScrapeHandler{DefaultTimeout: 5 * time.Second}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("X-Prometheus-Scrape-Timeout-Seconds", "30")

	got := h.effectiveTimeout(req)
	if got != 5*time.Second {
		t.Errorf("effectiveTimeout() = %v, want 5s", got)
	}
}
