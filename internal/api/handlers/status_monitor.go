package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// StatusMonitorResult is the per-provider status returned to the frontend.
type StatusMonitorResult struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	Status      string `json:"status"`      // operational, degraded, partial, major, maintenance, unknown
	Description string `json:"description"` // e.g. "All Systems Operational"
	StatusURL   string `json:"statusUrl"`
	Homepage    string `json:"homepage"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	Custom      bool   `json:"custom,omitempty"`
}

type builtInStatusProvider struct {
	Name     string
	Title    string
	Category string
	// StatusURL is the base URL of an Atlassian Statuspage instance.
	// The handler appends /api/v2/status.json to check status.
	StatusURL string
	Homepage  string
}

// builtInStatusProviders lists well-known services that expose an Atlassian
// Statuspage-compatible JSON API at {StatusURL}/api/v2/status.json.
var builtInStatusProviders = []builtInStatusProvider{
	// Developer Tools (all confirmed Atlassian Statuspage)
	{Name: "github", Title: "GitHub", Category: "developer-tools", StatusURL: "https://www.githubstatus.com", Homepage: "https://github.com"},
	{Name: "bitbucket", Title: "Bitbucket", Category: "developer-tools", StatusURL: "https://bitbucket.status.atlassian.com", Homepage: "https://bitbucket.org"},
	{Name: "atlassian", Title: "Atlassian (Jira/Confluence)", Category: "developer-tools", StatusURL: "https://status.atlassian.com", Homepage: "https://atlassian.com"},
	{Name: "linear", Title: "Linear", Category: "developer-tools", StatusURL: "https://linearstatus.com", Homepage: "https://linear.app"},
	{Name: "notion", Title: "Notion", Category: "developer-tools", StatusURL: "https://www.notion-status.com", Homepage: "https://notion.so"},
	{Name: "figma", Title: "Figma", Category: "developer-tools", StatusURL: "https://status.figma.com", Homepage: "https://figma.com"},
	{Name: "hashicorp", Title: "HashiCorp", Category: "developer-tools", StatusURL: "https://status.hashicorp.com", Homepage: "https://hashicorp.com"},

	// CI/CD & Hosting
	{Name: "vercel", Title: "Vercel", Category: "ci-cd", StatusURL: "https://www.vercel-status.com", Homepage: "https://vercel.com"},
	{Name: "netlify", Title: "Netlify", Category: "ci-cd", StatusURL: "https://www.netlifystatus.com", Homepage: "https://netlify.com"},
	{Name: "circleci", Title: "CircleCI", Category: "ci-cd", StatusURL: "https://status.circleci.com", Homepage: "https://circleci.com"},
	{Name: "travisci", Title: "Travis CI", Category: "ci-cd", StatusURL: "https://www.traviscistatus.com", Homepage: "https://travis-ci.com"},
	{Name: "render", Title: "Render", Category: "ci-cd", StatusURL: "https://status.render.com", Homepage: "https://render.com"},

	// CDN & Edge
	{Name: "cloudflare", Title: "Cloudflare", Category: "cdn-edge", StatusURL: "https://www.cloudflarestatus.com", Homepage: "https://cloudflare.com"},

	// Monitoring & Observability
	{Name: "datadog", Title: "Datadog", Category: "monitoring", StatusURL: "https://status.datadoghq.com", Homepage: "https://datadoghq.com"},
	{Name: "snyk", Title: "Snyk", Category: "monitoring", StatusURL: "https://status.snyk.io", Homepage: "https://snyk.io"},
	{Name: "newrelic", Title: "New Relic", Category: "monitoring", StatusURL: "https://status.newrelic.com", Homepage: "https://newrelic.com"},
	{Name: "sentry", Title: "Sentry", Category: "monitoring", StatusURL: "https://status.sentry.io", Homepage: "https://sentry.io"},
	{Name: "launchdarkly", Title: "LaunchDarkly", Category: "monitoring", StatusURL: "https://status.launchdarkly.com", Homepage: "https://launchdarkly.com"},

	// Infrastructure
	{Name: "digitalocean", Title: "DigitalOcean", Category: "infrastructure", StatusURL: "https://status.digitalocean.com", Homepage: "https://digitalocean.com"},
	{Name: "supabase", Title: "Supabase", Category: "infrastructure", StatusURL: "https://status.supabase.com", Homepage: "https://supabase.com"},
	{Name: "confluent", Title: "Confluent Cloud", Category: "infrastructure", StatusURL: "https://status.confluent.cloud", Homepage: "https://confluent.io"},

	// Communication
	{Name: "discord", Title: "Discord", Category: "communication", StatusURL: "https://discordstatus.com", Homepage: "https://discord.com"},
	{Name: "zoom", Title: "Zoom", Category: "communication", StatusURL: "https://www.zoomstatus.com", Homepage: "https://zoom.us"},
	{Name: "intercom", Title: "Intercom", Category: "communication", StatusURL: "https://www.intercomstatus.com", Homepage: "https://intercom.com"},

	// Package Registries
	{Name: "npm", Title: "npm", Category: "package-registry", StatusURL: "https://status.npmjs.org", Homepage: "https://npmjs.com"},
	{Name: "rubygems", Title: "RubyGems", Category: "package-registry", StatusURL: "https://status.rubygems.org", Homepage: "https://rubygems.org"},

	// E-Commerce & Payments
	{Name: "shopify", Title: "Shopify", Category: "payments", StatusURL: "https://www.shopifystatus.com", Homepage: "https://shopify.com"},

	// Other SaaS
	{Name: "reddit", Title: "Reddit", Category: "other", StatusURL: "https://www.redditstatus.com", Homepage: "https://reddit.com"},
	{Name: "dropbox", Title: "Dropbox", Category: "other", StatusURL: "https://status.dropbox.com", Homepage: "https://dropbox.com"},
	{Name: "twilio", Title: "Twilio", Category: "other", StatusURL: "https://status.twilio.com", Homepage: "https://twilio.com"},
}

// In-memory cache for status results.
var (
	smCacheMu      sync.RWMutex
	smCacheResults []StatusMonitorResult
	smCacheTime    time.Time
)

// GetStatusMonitorStatuses checks status for all configured providers.
func (h *Handlers) GetStatusMonitorStatuses(w http.ResponseWriter, r *http.Request) {
	p, err := h.DB.GetPlugin(r.Context(), "status-monitor")
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "status-monitor plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "status-monitor plugin is not enabled")
		return
	}

	// Return cached results if fresh (60s TTL).
	smCacheMu.RLock()
	if time.Since(smCacheTime) < 60*time.Second && len(smCacheResults) > 0 {
		results := smCacheResults
		smCacheMu.RUnlock()
		writeJSON(w, http.StatusOK, results)
		return
	}
	smCacheMu.RUnlock()

	// Build provider list from built-in + custom.
	providers := make([]StatusMonitorResult, 0, len(builtInStatusProviders))
	for _, bp := range builtInStatusProviders {
		providers = append(providers, StatusMonitorResult{
			Name:     bp.Name,
			Title:    bp.Title,
			Category: bp.Category,
			StatusURL: bp.StatusURL,
			Homepage: bp.Homepage,
			Status:   "unknown",
		})
	}

	// Append custom providers from plugin config.
	if p.Config != nil {
		if custom, ok := p.Config["customProviders"]; ok {
			if arr, ok := custom.([]any); ok {
				for _, item := range arr {
					m, ok := item.(map[string]any)
					if !ok {
						continue
					}
					name, _ := m["name"].(string)
					statusURL, _ := m["statusUrl"].(string)
					homepage, _ := m["homepageUrl"].(string)
					if name == "" || statusURL == "" {
						continue
					}
					providers = append(providers, StatusMonitorResult{
						Name:      strings.ToLower(strings.ReplaceAll(name, " ", "-")),
						Title:     name,
						Category:  "custom",
						StatusURL: statusURL,
						Homepage:  homepage,
						Status:    "unknown",
						Custom:    true,
					})
				}
			}
		}
	}

	// Check all providers concurrently.
	var wg sync.WaitGroup
	var mu sync.Mutex
	client := &http.Client{Timeout: 5 * time.Second}

	for i := range providers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result := checkStatusPageAPI(client, providers[idx].StatusURL)
			mu.Lock()
			providers[idx].Status = result.status
			providers[idx].Description = result.description
			providers[idx].UpdatedAt = result.updatedAt
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	// Cache results.
	smCacheMu.Lock()
	smCacheResults = providers
	smCacheTime = time.Now()
	smCacheMu.Unlock()

	writeJSON(w, http.StatusOK, providers)
}

// GetStatusMonitorProviders returns the list of built-in providers (no status check).
func (h *Handlers) GetStatusMonitorProviders(w http.ResponseWriter, r *http.Request) {
	type providerInfo struct {
		Name     string `json:"name"`
		Title    string `json:"title"`
		Category string `json:"category"`
		Homepage string `json:"homepage"`
	}
	out := make([]providerInfo, 0, len(builtInStatusProviders))
	for _, bp := range builtInStatusProviders {
		out = append(out, providerInfo{
			Name:     bp.Name,
			Title:    bp.Title,
			Category: bp.Category,
			Homepage: bp.Homepage,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type statusResult struct {
	status      string
	description string
	updatedAt   string
}

// checkStatusPageAPI tries the Atlassian Statuspage JSON API at baseURL/api/v2/status.json.
// Falls back to a simple HTTP reachability check if the response is not valid Statuspage JSON.
func checkStatusPageAPI(client *http.Client, baseURL string) statusResult {
	apiURL := strings.TrimRight(baseURL, "/") + "/api/v2/status.json"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return statusResult{status: "unknown", description: "Unable to reach status page"}
	}
	req.Header.Set("User-Agent", "Gantry/1.0 StatusMonitor")

	resp, err := client.Do(req)
	if err != nil {
		return statusResult{status: "unknown", description: "Unable to reach status page"}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return statusResult{status: "unknown", description: "Error reading response"}
	}

	var sp struct {
		Page struct {
			UpdatedAt string `json:"updated_at"`
		} `json:"page"`
		Status struct {
			Indicator   string `json:"indicator"`
			Description string `json:"description"`
		} `json:"status"`
	}

	if err := json.Unmarshal(body, &sp); err != nil || sp.Status.Indicator == "" {
		// Not a valid Statuspage response; fall back to HTTP status check.
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return statusResult{status: "operational", description: "Status page reachable"}
		}
		return statusResult{status: "unknown", description: "Unable to parse status"}
	}

	status := "unknown"
	switch sp.Status.Indicator {
	case "none":
		status = "operational"
	case "minor":
		status = "degraded"
	case "major":
		status = "partial"
	case "critical":
		status = "major"
	case "maintenance":
		status = "maintenance"
	}

	return statusResult{
		status:      status,
		description: sp.Status.Description,
		updatedAt:   sp.Page.UpdatedAt,
	}
}
