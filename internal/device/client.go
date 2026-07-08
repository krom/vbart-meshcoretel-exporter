// Package device implements the HTTPS client for MeshCoreTel firmware devices.
package device

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultPort is used when a target does not specify a port.
const DefaultPort = 443

// ErrUnauthorized is returned when the device rejects the supplied password.
var ErrUnauthorized = errors.New("device: invalid password")

// ErrStatsUnavailable is returned when the device reports stats collection
// is disabled (HTTP 503 from /api/stats).
var ErrStatsUnavailable = errors.New("device: stats unavailable")

// Client talks to a single MeshCoreTel device over HTTPS. Devices present a
// self-signed certificate, so certificate verification is intentionally
// disabled for device connections.
type Client struct {
	httpClient *http.Client
}

// NewClient returns a Client using the given per-request timeout as the
// default for its underlying transport. Callers should still pass a
// context with a deadline to Login/FetchStats to bound the whole scrape.
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec // device uses a self-signed cert by design
					// Devices run an embedded TLS stack (ESP32/mbedTLS) that only
					// speaks TLS 1.2 and, on many firmware builds, only offers
					// non-ECDHE RSA cipher suites. Go's default cipher suite list
					// excludes those (no forward secrecy), so they must be listed
					// explicitly or the handshake fails outright.
					MaxVersion: tls.VersionTLS12,
					CipherSuites: []uint16{
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_RSA_WITH_AES_128_CBC_SHA,
						tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					},
				},
			},
		},
	}
}

// baseURL normalizes a target (host or host:port) into a base HTTPS URL.
func baseURL(target string) (string, error) {
	if target == "" {
		return "", errors.New("device: empty target")
	}
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		// No port present; use the target as-is with the default port.
		host = target
		port = strconv.Itoa(DefaultPort)
	}
	if host == "" {
		return "", errors.New("device: empty target host")
	}
	if port == strconv.Itoa(DefaultPort) {
		return fmt.Sprintf("https://%s", host), nil
	}
	return fmt.Sprintf("https://%s", net.JoinHostPort(host, port)), nil
}

// Login authenticates against the device's /login endpoint, returning the
// session token to use for subsequent requests.
func (c *Client) Login(ctx context.Context, target, password string) (string, error) {
	base, err := baseURL(target)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/login", strings.NewReader(password))
	if err != nil {
		return "", fmt.Errorf("device: build login request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("device: login request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("device: read login response: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return "", ErrUnauthorized
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("device: login returned unexpected status %d", resp.StatusCode)
	}

	token := strings.TrimSpace(string(body))
	if token == "" {
		return "", errors.New("device: empty session token")
	}
	return token, nil
}

// FetchStats retrieves the raw stats JSON payload from the device using the
// given session token.
func (c *Client) FetchStats(ctx context.Context, target, token string) ([]byte, error) {
	base, err := baseURL(target)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/stats", nil)
	if err != nil {
		return nil, fmt.Errorf("device: build stats request: %w", err)
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device: stats request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("device: read stats response: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusServiceUnavailable:
		return nil, ErrStatsUnavailable
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("device: stats returned unexpected status %d", resp.StatusCode)
	}

	return body, nil
}

// FetchVersion runs the "ver" CLI command through the device's command
// channel, using the given session token. The response is returned as-is
// for the caller to validate with ParseVersion.
func (c *Client) FetchVersion(ctx context.Context, target, token string) ([]byte, error) {
	base, err := baseURL(target)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/command", strings.NewReader("ver"))
	if err != nil {
		return nil, fmt.Errorf("device: build command request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Auth-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device: command request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("device: read command response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device: command returned unexpected status %d", resp.StatusCode)
	}

	return body, nil
}

// FetchStatsDecoded logs in and fetches stats, decoding the JSON response
// into a Stats struct in one call.
func (c *Client) FetchStatsDecoded(ctx context.Context, target, password string) (*Stats, error) {
	token, err := c.Login(ctx, target, password)
	if err != nil {
		return nil, err
	}

	body, err := c.FetchStats(ctx, target, token)
	if err != nil {
		return nil, err
	}

	stats, err := ParseStats(body)
	if err != nil {
		return nil, fmt.Errorf("device: parse stats: %w", err)
	}
	return stats, nil
}
