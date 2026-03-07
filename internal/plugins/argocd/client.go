// Package argocd implements ArgoCD application discovery and sync triggering
// for the Gantry plugin system. It uses the ArgoCD REST API directly via
// net/http so there is no dependency on the ArgoCD Go client.
package argocd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a minimal ArgoCD REST API client.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a client from plugin config values.
//
//	config["argocdUrl"]        — required, e.g. "https://argocd.example.com"
//	config["token"]            — API token. Either this or username+password is required.
//	config["username"]         — ArgoCD username (alternative to token)
//	config["password"]         — ArgoCD password (alternative to token)
//	config["insecureSkipTLS"]  — optional bool, skip TLS verification (default false)
//
// When username and password are provided instead of a token, NewClient calls
// the ArgoCD session API to exchange them for a JWT which is used for all
// subsequent requests.
func NewClient(config map[string]any) (*Client, error) {
	argoURL, _ := config["argocdUrl"].(string)
	if argoURL == "" {
		return nil, fmt.Errorf("argocd plugin: argocdUrl is required")
	}
	argoURL = strings.TrimRight(argoURL, "/")

	insecure, _ := config["insecureSkipTLS"].(bool)
	tlsCfg := &tls.Config{InsecureSkipVerify: insecure} //nolint:gosec

	httpClient := &http.Client{
		Timeout:   20 * time.Second,
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
	}

	authMode, _ := config["authMode"].(string)
	token, _ := config["token"].(string)

	// Use credentials auth when authMode is "credentials" OR when no token is set.
	if authMode == "credentials" || (token == "" && authMode != "token") {
		username, _ := config["username"].(string)
		password, _ := config["password"].(string)
		if username == "" || password == "" {
			return nil, fmt.Errorf("argocd plugin: username and password are required when using credentials auth")
		}
		var err error
		token, err = createSession(httpClient, argoURL, username, password)
		if err != nil {
			return nil, fmt.Errorf("argocd plugin: authentication failed: %w", err)
		}
	} else if token == "" {
		return nil, fmt.Errorf("argocd plugin: token is required when using token auth")
	}

	return &Client{
		baseURL:    argoURL,
		token:      token,
		httpClient: httpClient,
	}, nil
}

// createSession authenticates with ArgoCD using username/password and returns
// the JWT token from the session API (POST /api/v1/session).
func createSession(httpClient *http.Client, baseURL, username, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req, err := http.NewRequest("POST", baseURL+"/api/v1/session", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("session request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse session response: %w", err)
	}
	if result.Token == "" {
		return "", fmt.Errorf("empty token in session response")
	}
	return result.Token, nil
}

// ListApplications fetches all ArgoCD Applications the token has access to.
// An optional project filter can be passed to restrict results.
func (c *Client) ListApplications(project string) ([]Application, error) {
	path := "/api/v1/applications"
	if project != "" {
		path += "?projects=" + project
	}
	var list ApplicationList
	if err := c.get(path, &list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetApplication fetches a single ArgoCD Application by name.
func (c *Client) GetApplication(name string) (*Application, error) {
	var app Application
	if err := c.get("/api/v1/applications/"+name, &app); err != nil {
		return nil, err
	}
	return &app, nil
}

// SyncApplication triggers an ArgoCD sync for the named application.
// If hard is true, a hard refresh (cache-busting) is performed first.
func (c *Client) SyncApplication(name string, hard bool) error {
	body := map[string]any{}
	if hard {
		body["prune"] = true
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/applications/"+name+"/sync", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("argocd sync %s: %w", name, err)
	}
	defer resp.Body.Close()
	body2, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("argocd sync %s: HTTP %d: %s", name, resp.StatusCode, string(body2))
	}
	return nil
}

// RefreshApplication triggers an ArgoCD refresh (re-fetch from git without sync).
func (c *Client) RefreshApplication(name string) (*Application, error) {
	// ArgoCD refresh: GET /api/v1/applications/{name}?refresh=normal|hard
	var app Application
	if err := c.get("/api/v1/applications/"+name+"?refresh=normal", &app); err != nil {
		return nil, err
	}
	return &app, nil
}

// AppToStatusResponse converts an Application to the lightweight status struct.
func AppToStatusResponse(app *Application) *AppStatusResponse {
	resp := &AppStatusResponse{
		AppName:       app.Metadata.Name,
		SyncStatus:    app.Status.Sync.Status,
		HealthStatus:  app.Status.Health.Status,
		HealthMessage: app.Status.Health.Message,
		SyncRevision:  app.Status.Sync.Revision,
		RepoURL:       app.Spec.Source.RepoURL,
		TargetRevision: app.Spec.Source.TargetRevision,
		Path:          app.Spec.Source.Path,
		Chart:         app.Spec.Source.Chart,
		Project:       app.Spec.Project,
		DestServer:    app.Spec.Destination.Server,
		DestNamespace: app.Spec.Destination.Namespace,
		Images:        app.Status.Summary.Images,
		Resources:     app.Status.Resources,
	}
	if app.Status.OperationState != nil {
		resp.OperationPhase = app.Status.OperationState.Phase
		resp.OperationMsg = app.Status.OperationState.Message
	}
	return resp
}

// AllInstanceConfigs extracts every ArgoCD instance from the plugin config and
// returns each as a flat map compatible with NewClient, plus an "instanceName" key.
// It handles both the instances-array format and the legacy flat format.
func AllInstanceConfigs(raw map[string]any) []map[string]any {
	if raw == nil {
		return nil
	}
	if instances, ok := raw["instances"]; ok {
		if arr, ok := instances.([]any); ok && len(arr) > 0 {
			var out []map[string]any
			for _, item := range arr {
				c, ok := item.(map[string]any)
				if !ok {
					continue
				}
				cfg := map[string]any{}
				for _, key := range []string{"argocdUrl", "token", "username", "password", "authMode", "insecureSkipTLS", "project"} {
					if v, ok := c[key]; ok {
						cfg[key] = v
					}
				}
				if name, _ := c["name"].(string); name != "" {
					cfg["instanceName"] = name
				}
				if _, ok := cfg["argocdUrl"]; ok {
					out = append(out, cfg)
				}
			}
			return out
		}
	}
	// Flat / legacy format: treat the whole config as a single instance.
	if url, _ := raw["argocdUrl"].(string); url != "" {
		cfg := map[string]any{}
		for _, key := range []string{"argocdUrl", "token", "username", "password", "authMode", "insecureSkipTLS", "project"} {
			if v, ok := raw[key]; ok {
				cfg[key] = v
			}
		}
		cfg["instanceName"] = "default"
		return []map[string]any{cfg}
	}
	return nil
}

// get performs a GET request to the ArgoCD API and decodes the JSON response.
func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("argocd request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("argocd %s: HTTP %d: %s", path, resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, out)
}
