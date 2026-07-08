package device

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func newMockDevice(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func targetFor(srv *httptest.Server) string {
	return strings.TrimPrefix(srv.URL, "https://")
}

func TestLoginSuccess(t *testing.T) {
	srv := newMockDevice(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/login" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "secret" {
			t.Fatalf("unexpected password body: %q", body)
		}
		_, _ = w.Write([]byte("token-123"))
	})

	c := NewClient(2 * time.Second)
	token, err := c.Login(context.Background(), targetFor(srv), "secret")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token != "token-123" {
		t.Fatalf("Login() token = %q, want token-123", token)
	}
}

func TestLoginUnauthorized(t *testing.T) {
	srv := newMockDevice(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	c := NewClient(2 * time.Second)
	_, err := c.Login(context.Background(), targetFor(srv), "wrong")
	if err != ErrUnauthorized {
		t.Fatalf("Login() error = %v, want ErrUnauthorized", err)
	}
}

func TestFetchStatsSuccess(t *testing.T) {
	statsJSON, err := os.ReadFile("../../examples/stats.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := newMockDevice(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stats" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Auth-Token") != "token-123" {
			t.Fatalf("missing/incorrect auth token header: %q", r.Header.Get("X-Auth-Token"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(statsJSON)
	})

	c := NewClient(2 * time.Second)
	body, err := c.FetchStats(context.Background(), targetFor(srv), "token-123")
	if err != nil {
		t.Fatalf("FetchStats() error = %v", err)
	}
	if len(body) == 0 {
		t.Fatal("FetchStats() returned empty body")
	}
}

func TestFetchStatsUnavailable(t *testing.T) {
	srv := newMockDevice(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	c := NewClient(2 * time.Second)
	_, err := c.FetchStats(context.Background(), targetFor(srv), "token-123")
	if err != ErrStatsUnavailable {
		t.Fatalf("FetchStats() error = %v, want ErrStatsUnavailable", err)
	}
}

func TestFetchStatsMalformedJSON(t *testing.T) {
	srv := newMockDevice(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{not valid json"))
	})

	c := NewClient(2 * time.Second)
	body, err := c.FetchStats(context.Background(), targetFor(srv), "token-123")
	if err != nil {
		t.Fatalf("FetchStats() error = %v", err)
	}
	if _, err := ParseStats(body); err == nil {
		t.Fatal("ParseStats() expected error for malformed JSON, got nil")
	}
}

func TestLoginTimeout(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	srv := newMockDevice(t, func(w http.ResponseWriter, r *http.Request) {
		<-block
	})

	c := NewClient(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Login(ctx, targetFor(srv), "secret")
	if err == nil {
		t.Fatal("Login() expected timeout error, got nil")
	}
}

func TestParseStatsGoldenFixture(t *testing.T) {
	body, err := os.ReadFile("../../examples/stats.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	stats, err := ParseStats(body)
	if err != nil {
		t.Fatalf("ParseStats() error = %v", err)
	}

	if stats.Core.BatteryMV != 4296 {
		t.Errorf("Core.BatteryMV = %d, want 4296", stats.Core.BatteryMV)
	}
	if len(stats.NeighborsDetail) != 10 {
		t.Errorf("len(NeighborsDetail) = %d, want 10", len(stats.NeighborsDetail))
	}
	if !stats.NeighborsDetail[0].IsAdvertSentinel() {
		t.Errorf("expected first neighbor advert_secs_ago to be a sentinel value")
	}
}
