package handlers

import (
	"net/http"
)

// AuthProviderInfo describes a single auth provider the login page can offer.
type AuthProviderInfo struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	IconURL  string `json:"iconUrl,omitempty"`
	LoginURL string `json:"loginUrl"`
}

// GetAuthProviders returns all enabled auth-provider plugins and their login URLs.
// This is a public endpoint used by the login page to render SSO buttons dynamically.
func (h *Handlers) GetAuthProviders(w http.ResponseWriter, r *http.Request) {
	allPlugins, err := h.DB.ListPlugins(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list plugins")
		return
	}

	providers := make([]AuthProviderInfo, 0)
	for _, p := range allPlugins {
		if p.Manifest == nil || p.Manifest.Category != "auth-provider" || !p.Enabled {
			continue
		}

		var loginURL string

		switch p.Name {
		case "github":
			ssoEnabled, _ := p.Config["ssoEnabled"].(bool)
			clientID, _ := p.Config["oauthClientId"].(string)
			if ssoEnabled && clientID != "" {
				loginURL = "/api/v1/auth/github"
			}
		case "microsoft-azure":
			ssoEnabled, _ := p.Config["ssoEnabled"].(bool)
			clientID, _ := p.Config["clientId"].(string)
			clientSecret, _ := p.Config["clientSecret"].(string)
			if ssoEnabled && clientID != "" && clientSecret != "" {
				loginURL = "/api/v1/auth/azure"
			}
		default:
			if p.Manifest.Source == "external" {
				// External auth-provider plugins get a generic OIDC login URL.
				// The plugin must have oidcIssuerUrl (or authentikUrl) and clientId configured.
				issuer, _ := p.Config["oidcIssuerUrl"].(string)
				if issuer == "" {
					issuer, _ = p.Config["authentikUrl"].(string)
				}
				clientID, _ := p.Config["clientId"].(string)
				if issuer != "" && clientID != "" {
					loginURL = "/api/v1/auth/plugin/" + p.Name
				}
			} else if p.Manifest.AuthBeginPath != "" {
				loginURL = p.Manifest.AuthBeginPath
			}
		}

		if loginURL == "" {
			continue
		}

		providers = append(providers, AuthProviderInfo{
			Name:     p.Name,
			Title:    p.Manifest.Title,
			IconURL:  p.Manifest.IconURL,
			LoginURL: loginURL,
		})
	}

	writeJSON(w, http.StatusOK, providers)
}
