package exporter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLandingPage(t *testing.T) {
	mux := NewServeMux(Config{ListenAddress: ":9642", ScrapeTimeout: time.Second, Version: "test"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "vbart-meshcoretel-exporter") {
		t.Errorf("landing page missing title: %s", rec.Body.String())
	}
}

func TestHealthy(t *testing.T) {
	mux := NewServeMux(Config{ListenAddress: ":9642", ScrapeTimeout: time.Second, Version: "test"})

	req := httptest.NewRequest(http.MethodGet, "/-/healthy", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestSelfMetrics(t *testing.T) {
	mux := NewServeMux(Config{ListenAddress: ":9642", ScrapeTimeout: time.Second, Version: "test"})

	req := httptest.NewRequest(http.MethodGet, "/-/metrics", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "go_goroutines") {
		t.Errorf("self metrics missing go runtime metrics: %s", body)
	}
	if strings.Contains(body, "meshcoretel_") {
		t.Errorf("self metrics should not contain device metrics: %s", body)
	}
}
