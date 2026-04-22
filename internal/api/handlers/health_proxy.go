package handlers

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"time"
)

var healthCheckBlockedPrefixes = []netip.Prefix{
	mustParsePrefix("0.0.0.0/8"),
	mustParsePrefix("10.0.0.0/8"),
	mustParsePrefix("100.64.0.0/10"),
	mustParsePrefix("127.0.0.0/8"),
	mustParsePrefix("169.254.0.0/16"),
	mustParsePrefix("172.16.0.0/12"),
	mustParsePrefix("192.0.0.0/24"),
	mustParsePrefix("192.168.0.0/16"),
	mustParsePrefix("198.18.0.0/15"),
	mustParsePrefix("224.0.0.0/4"),
	mustParsePrefix("240.0.0.0/4"),
	mustParsePrefix("::/128"),
	mustParsePrefix("::1/128"),
	mustParsePrefix("fe80::/10"),
	mustParsePrefix("fc00::/7"),
	mustParsePrefix("ff00::/8"),
}

func mustParsePrefix(value string) netip.Prefix {
	prefix, err := netip.ParsePrefix(value)
	if err != nil {
		panic(err)
	}
	return prefix
}

func isBlockedHealthCheckIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	addr = addr.Unmap()
	if !addr.IsValid() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || !addr.IsGlobalUnicast() {
		return true
	}
	for _, prefix := range healthCheckBlockedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func stableLookupIPAddrs(ctx context.Context, host string) ([]net.IPAddr, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return net.DefaultResolver.LookupIPAddr(lookupCtx, host)
}

func normalizeResolvedIPs(addrs []net.IPAddr) []string {
	set := make(map[string]struct{}, len(addrs))
	for _, addr := range addrs {
		ip := addr.IP
		if ip == nil {
			continue
		}
		if v4 := ip.To4(); v4 != nil {
			ip = v4
		}
		set[ip.String()] = struct{}{}
	}
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func sameResolvedIPs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func resolveHealthCheckDialTarget(r *http.Request, parsed *url.URL) (string, string, error) {
	host := parsed.Hostname()
	if host == "" {
		return "", "", fmt.Errorf("url must include a hostname")
	}
	port := parsed.Port()
	if port == "" {
		switch parsed.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			return "", "", fmt.Errorf("url must be an absolute http or https URL")
		}
	}

	if literalIP := net.ParseIP(host); literalIP != nil {
		if isBlockedHealthCheckIP(literalIP) {
			return "", "", fmt.Errorf("health check target resolves to a blocked IP range")
		}
		return net.JoinHostPort(literalIP.String(), port), host, nil
	}

	first, err := stableLookupIPAddrs(r.Context(), host)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve health check target: %w", err)
	}
	second, err := stableLookupIPAddrs(r.Context(), host)
	if err != nil {
		return "", "", fmt.Errorf("failed to verify health check target: %w", err)
	}
	firstIPs := normalizeResolvedIPs(first)
	secondIPs := normalizeResolvedIPs(second)
	if len(firstIPs) == 0 {
		return "", "", fmt.Errorf("health check target did not resolve to any IP addresses")
	}
	if !sameResolvedIPs(firstIPs, secondIPs) {
		return "", "", fmt.Errorf("health check target resolution is unstable")
	}

	hasBlocked := false
	hasAllowed := false
	for _, ipText := range firstIPs {
		ip := net.ParseIP(ipText)
		if ip == nil || isBlockedHealthCheckIP(ip) {
			hasBlocked = true
			continue
		}
		hasAllowed = true
	}
	if hasBlocked && hasAllowed {
		return "", "", fmt.Errorf("health check target resolves to mixed public and private IP addresses")
	}
	if hasBlocked {
		return "", "", fmt.Errorf("health check target resolves to a blocked IP range")
	}

	return net.JoinHostPort(firstIPs[0], port), host, nil
}

// HealthCheckProxy fetches an external health-check URL on behalf of the
// frontend (avoiding CORS issues) and returns the upstream status, latency,
// and body.
func (h *Handlers) HealthCheckProxy(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")
	if target == "" {
		writeError(w, http.StatusBadRequest, "url query parameter is required")
		return
	}

	parsed, err := url.Parse(target)
	if err != nil || !parsed.IsAbs() || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		writeError(w, http.StatusBadRequest, "url must be an absolute http or https URL")
		return
	}
	if parsed.User != nil {
		writeError(w, http.StatusBadRequest, "url must not contain userinfo")
		return
	}

	dialTarget, serverName, err := resolveHealthCheckDialTarget(r, parsed)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, dialTarget)
	}
	if parsed.Scheme == "https" {
		transport.TLSClientConfig = &tls.Config{ServerName: serverName}
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}
	req.Header.Set("User-Agent", "gantry-health-check/1.0")
	resp, err := client.Do(req)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"reachable": false,
			"error":     err.Error(),
			"latencyMs": latencyMs,
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	writeJSON(w, http.StatusOK, map[string]any{
		"reachable":  resp.StatusCode >= 200 && resp.StatusCode < 400,
		"statusCode": resp.StatusCode,
		"latencyMs":  latencyMs,
		"body":       string(body),
	})
}
