package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// newPluginReverseProxy creates a reverse proxy that forwards requests to the
// plugin's embedded HTTP server at upstream. The full request path is preserved
// so the plugin can route against its own declared paths.
func newPluginReverseProxy(upstream, pathPrefix string) http.Handler {
	target, _ := url.Parse(upstream)
	proxy := httputil.NewSingleHostReverseProxy(target)

	orig := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalHost := req.Host
		orig(req)

		if originalHost != "" {
			req.Header.Set("X-Forwarded-Host", originalHost)
		}

		proto := "http"
		if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
			proto = "https"
		}
		req.Header.Set("X-Forwarded-Proto", proto)
	}

	return proxy
}
