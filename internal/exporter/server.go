package exporter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/krom/vbart-meshcoretel-exporter/internal/device"
)

// Config holds the exporter's runtime configuration, populated from CLI
// flags. There is no configuration file.
type Config struct {
	ListenAddress string
	ScrapeTimeout time.Duration
	Version       string
	Logger        *slog.Logger
	DrainTimeout  time.Duration

	// CollectVersion enables the opt-in meshcoretel_build_info metric; see
	// ScrapeHandler.CollectVersion.
	CollectVersion bool
}

const landingPageHTML = `<!DOCTYPE html>
<html lang="en">
<head><title>vbart-meshcoretel-exporter</title></head>
<body>
<h1>vbart-meshcoretel-exporter</h1>
<p>Prometheus exporter for MeshCoreTel firmware devices (version %s).</p>
<ul>
<li><a href="/metrics?target=192.168.0.10&password=CHANGEME">/metrics?target=&lt;device&gt;&amp;password=&lt;password&gt;</a> &mdash; scrape a device</li>
<li><a href="/-/metrics">/-/metrics</a> &mdash; exporter self-metrics</li>
<li><a href="/-/healthy">/-/healthy</a> &mdash; liveness check</li>
</ul>
</body>
</html>
`

// NewServeMux builds the exporter's HTTP routes.
func NewServeMux(cfg Config) *http.ServeMux {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()

	scrapeHandler := &ScrapeHandler{
		Client:         device.NewClient(cfg.ScrapeTimeout),
		DefaultTimeout: cfg.ScrapeTimeout,
		Logger:         logger,
		CollectVersion: cfg.CollectVersion,
	}
	mux.Handle("/metrics", scrapeHandler)

	selfRegistry := prometheus.NewRegistry()
	selfRegistry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	mux.Handle("/-/metrics", promhttp.HandlerFor(selfRegistry, promhttp.HandlerOpts{}))

	mux.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, landingPageHTML, cfg.Version)
	})

	return mux
}

// Run starts the HTTP server and blocks until it receives SIGINT/SIGTERM,
// then drains in-flight requests before returning.
func Run(cfg Config) error {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	drain := cfg.DrainTimeout
	if drain <= 0 {
		drain = 10 * time.Second
	}

	srv := &http.Server{
		Addr:         cfg.ListenAddress,
		Handler:      NewServeMux(cfg),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: cfg.ScrapeTimeout + 5*time.Second,
	}

	logger.Info("starting vbart-meshcoretel-exporter", "version", cfg.Version, "listen_address", cfg.ListenAddress)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		logger.Info("shutting down", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), drain)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return nil
	}
}
