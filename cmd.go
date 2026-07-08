package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/krom/vbart-meshcoretel-exporter/internal/exporter"
)

// version, commit, and date are injected at build time via -ldflags, e.g.:
//
//	go build -ldflags "-X main.version=v1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func run() {
	var (
		listenAddress  = flag.String("web.listen-address", ":9642", "Address on which to expose the web interface and metrics.")
		scrapeTimeout  = flag.Duration("scrape.timeout", 10*time.Second, "Maximum time allowed for a single device scrape (login + stats fetch).")
		logLevel       = flag.String("log.level", "info", "Minimum log level to emit: debug, info, warn, or error.")
		logFormat      = flag.String("log.format", "text", "Log output format: text or json.")
		showVersion    = flag.Bool("version", false, "Print version information and exit.")
		collectVersion = flag.Bool("collect.version", false, "Fetch and expose meshcoretel_build_info via an extra device round-trip per scrape. Off by default: see README before enabling.")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("vbart-meshcoretel-exporter %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	resolvedCollectVersion := resolveCollectVersion(*collectVersion, flagWasSet(flag.CommandLine, "collect.version"), os.LookupEnv)

	logger, err := newLogger(*logLevel, *logFormat)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vbart-meshcoretel-exporter:", err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	cfg := exporter.Config{
		ListenAddress:  *listenAddress,
		ScrapeTimeout:  *scrapeTimeout,
		Version:        version,
		Logger:         logger,
		CollectVersion: resolvedCollectVersion,
	}

	if err := exporter.Run(cfg); err != nil {
		logger.Error("exporter exited with error", "error", err.Error())
		os.Exit(1)
	}
}

// collectVersionEnvVar is checked when --collect.version was not explicitly
// passed on the command line.
const collectVersionEnvVar = "VBART_COLLECT_VERSION"

// flagWasSet reports whether name was explicitly passed on the command
// line, as opposed to left at its default value.
func flagWasSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

// resolveCollectVersion applies the --collect.version / VBART_COLLECT_VERSION
// precedence rule: the flag wins whenever it was explicitly set; otherwise
// the environment variable is consulted, falling back to the flag's default
// (false) if the environment variable is unset or unparseable.
func resolveCollectVersion(flagValue, flagExplicit bool, lookupEnv func(string) (string, bool)) bool {
	if flagExplicit {
		return flagValue
	}
	if raw, ok := lookupEnv(collectVersionEnvVar); ok {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			return parsed
		}
	}
	return flagValue
}

func newLogger(level, format string) (*slog.Logger, error) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log.level %q: must be debug, info, warn, or error", level)
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		return nil, fmt.Errorf("invalid log.format %q: must be text or json", format)
	}

	return slog.New(handler), nil
}
