// Package security provides security utilities for the application
package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Private/Reserved IP ranges that should be blocked to prevent SSRF
var privateIPBlocks []*net.IPNet

func init() {
	// Initialize private IP blocks
	// These ranges should not be accessible via user-controlled URLs
	privateRanges := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918 private
		"172.16.0.0/12",  // RFC1918 private
		"192.168.0.0/16", // RFC1918 private
		"169.254.0.0/16", // Link-local / Cloud metadata
		"0.0.0.0/8",      // Current network
		"224.0.0.0/4",    // Multicast
		"240.0.0.0/4",    // Reserved
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local
	}

	for _, cidr := range privateRanges {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			privateIPBlocks = append(privateIPBlocks, block)
		}
	}
}

// SSRFError represents a Server-Side Request forgery attempt
type SSRFError struct {
	URL    string
	Reason string
}

func (e *SSRFError) Error() string {
	return fmt.Sprintf("SSRF blocked: %s - %s", e.URL, e.Reason)
}

// isPrivateIP checks if an IP address is in a private/reserved range
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	if ip.IsLoopback() {
		return true
	}

	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	if ip.IsPrivate() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

// Return an error if the URL is potentially dangerous
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return &SSRFError{URL: rawURL, Reason: "empty URL"}
	}

	parseURL, err := url.Parse(rawURL)
	if err != nil {
		return &SSRFError{URL: rawURL, Reason: "invalid URL format"}
	}

	schema := strings.ToLower(parseURL.Scheme)
	if schema != "http" && schema != "https" {
		return &SSRFError{URL: rawURL, Reason: fmt.Sprintf("unsupported schema: %s", schema)}
	}

	host := parseURL.Hostname()
	if host == "" {
		return &SSRFError{URL: rawURL, Reason: "empty hostname"}
	}

	lowerHost := strings.ToLower(host)
	blockedHosts := []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"0.0.0.0",
		"metadata.google.internal",
		"metadata.google",
		"instance-data",
	}
	for _, blocked := range blockedHosts {
		if lowerHost == blocked {
			return &SSRFError{URL: rawURL, Reason: fmt.Sprintf("blocked hostname: %s", host)}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolver := net.Resolver{}
	ips, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		ip := net.ParseIP(host)
		if ip != nil {
			if isPrivateIP(ip) {
				return &SSRFError{URL: rawURL, Reason: "resolves to private IP address"}
			}
			return nil
		}
		return nil
	}

	// check all resolved IPs
	for _, ipAddr := range ips {
		if isPrivateIP(ipAddr.IP) {
			return &SSRFError{URL: rawURL, Reason: fmt.Sprintf("resolves to private IP: %s", &ipAddr.IP)}
		}
	}

	return nil
}

// validatingTransport validates the request URL (SSRF) then delegates to the
// base transport. Using DefaultTransport underneath keeps HTTPS_PROXY /
// HTTP_PROXY / ALL_PROXY working — the previous DialContext SSRF check dialed
// directly and blocked private proxy hosts (e.g. 127.0.0.1:7890).
type validatingTransport struct {
	base http.RoundTripper
}

func (t *validatingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("SSRF protection: empty request URL")
	}
	if err := ValidateURL(req.URL.String()); err != nil {
		return nil, err
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// SafeHTTPClient returns an HTTP client with SSRF protection on the request
// URL while still honoring proxy environment variables.
func SafeHTTPClient(timeout time.Duration) *http.Client {
	var base http.RoundTripper = http.DefaultTransport
	if dt, ok := http.DefaultTransport.(*http.Transport); ok {
		cloned := dt.Clone()
		// Explicit — DefaultTransport already does this, keep it obvious.
		cloned.Proxy = http.ProxyFromEnvironment
		base = cloned
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: &validatingTransport{base: base},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if err := ValidateURL(req.URL.String()); err != nil {
				return fmt.Errorf("SSRF protection: redirect blocked - %w", err)
			}
			return nil
		},
	}
}

func SafeGet(rawUrl string, timeout time.Duration) (*http.Response, error) {
	if err := ValidateURL(rawUrl); err != nil {
		return nil, err
	}

	client := SafeHTTPClient(timeout)
	return client.Get(rawUrl)
}
