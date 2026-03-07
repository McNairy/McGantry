package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go2engle/gantry/internal/plugins"
	argocd "github.com/go2engle/gantry/internal/plugins/argocd"
	ghplugin "github.com/go2engle/gantry/internal/plugins/github"
	k8s "github.com/go2engle/gantry/internal/plugins/kubernetes"
)

// ListPlugins returns all plugins: registry entries merged with installed state.
func (h *Handlers) ListPlugins(w http.ResponseWriter, r *http.Request) {
	// Load bundled registry.
	registry, err := plugins.BundledRegistry()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load plugin registry: "+err.Error())
		return
	}

	// Load installed plugins from DB.
	installed, err := h.DB.ListPlugins(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build a map of installed plugins by name for quick lookup.
	installedMap := make(map[string]*plugins.Plugin, len(installed))
	for i := range installed {
		installedMap[installed[i].Name] = &installed[i]
	}

	type pluginListItem struct {
		plugins.RegistryEntry
		Installed   bool   `json:"installed"`
		Enabled     bool   `json:"enabled"`
		InstalledAt string `json:"installedAt,omitempty"`
	}

	items := make([]pluginListItem, 0, len(registry))
	for _, entry := range registry {
		item := pluginListItem{RegistryEntry: entry}
		if p, ok := installedMap[entry.Name]; ok {
			item.Installed = true
			item.Enabled = p.Enabled
			item.InstalledAt = p.InstalledAt
		}
		items = append(items, item)
	}

	// Also append any installed plugins not in the bundled registry (e.g. community).
	for _, p := range installed {
		found := false
		for _, entry := range registry {
			if entry.Name == p.Name {
				found = true
				break
			}
		}
		if !found && p.Manifest != nil {
			items = append(items, pluginListItem{
				RegistryEntry: plugins.RegistryEntry{
					Name:        p.Manifest.Name,
					Title:       p.Manifest.Title,
					Description: p.Manifest.Description,
					Version:     p.Manifest.Version,
					Author:      p.Manifest.Author,
					Category:    p.Manifest.Category,
				},
				Installed:   true,
				Enabled:     p.Enabled,
				InstalledAt: p.InstalledAt,
			})
		}
	}

	writeJSON(w, http.StatusOK, items)
}

// GetPlugin returns full detail for a single installed plugin.
func (h *Handlers) GetPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	p, err := h.DB.GetPlugin(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "plugin not installed")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// InstallPlugin "installs" a plugin from the bundled registry into the DB.
func (h *Handlers) InstallPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Look up in bundled registry.
	entry, err := plugins.FindInRegistry(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entry == nil {
		writeError(w, http.StatusNotFound, "plugin not found in registry")
		return
	}

	// Build a manifest from the registry entry.
	manifest := &plugins.Manifest{
		Name:         entry.Name,
		Title:        entry.Title,
		Description:  entry.Description,
		Version:      entry.Version,
		Author:       entry.Author,
		Category:     entry.Category,
		Homepage:     entry.Homepage,
		ConfigSchema: entry.ConfigSchema,
		EntityPanels: entry.EntityPanels,
		ActionTypes:  entry.ActionTypes,
	}

	p := &plugins.Plugin{
		ID:       newShortID(),
		Name:     entry.Name,
		Version:  entry.Version,
		Enabled:  false,
		Manifest: manifest,
	}

	if err := h.DB.UpsertPlugin(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// UninstallPlugin removes an installed plugin.
func (h *Handlers) UninstallPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.DB.DeletePlugin(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// EnablePlugin enables or disables a plugin.
func (h *Handlers) EnablePlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.DB.UpdatePluginEnabled(r.Context(), name, body.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetPluginConfig returns the config schema and current values for a plugin.
func (h *Handlers) GetPluginConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	p, err := h.DB.GetPlugin(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "plugin not installed")
		return
	}

	var schema map[string]any
	// Always prefer the registry schema so config forms stay up-to-date
	// even if the plugin was installed with an older manifest.
	if entry, _ := plugins.FindInRegistry(name); entry != nil {
		schema = entry.ConfigSchema
	} else if p.Manifest != nil {
		schema = p.Manifest.ConfigSchema
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"schema": schema,
		"values": p.Config,
	})
}

// UpdatePluginConfig saves plugin configuration.
func (h *Handlers) UpdatePluginConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var config map[string]any
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.DB.UpdatePluginConfig(r.Context(), name, config); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SyncPlugin triggers a plugin-specific sync operation (e.g. Kubernetes discovery).
// Currently only the "kubernetes" plugin is supported.
func (h *Handlers) SyncPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	p, err := h.DB.GetPlugin(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "plugin is not enabled")
		return
	}

	switch name {
	case "kubernetes":
		clusters := allClusterConfigs(p.Config)
		if len(clusters) == 0 {
			writeError(w, http.StatusBadRequest, "no clusters configured")
			return
		}
		combined := &k8s.SyncResult{}
		for _, cfg := range clusters {
			result, err := k8s.Sync(r.Context(), cfg, h.DB)
			if err != nil {
				combined.Errors = append(combined.Errors, err.Error())
				continue
			}
			combined.Namespaces += result.Namespaces
			combined.Deployments += result.Deployments
			combined.Services += result.Services
			combined.Created += result.Created
			combined.Updated += result.Updated
			combined.Errors = append(combined.Errors, result.Errors...)
		}
		for _, e := range combined.Errors {
			log.Printf("[kubernetes-sync] error: %s", e)
		}
		writeJSON(w, http.StatusOK, combined)
	case "github":
		result, err := ghplugin.Sync(r.Context(), p.Config, h.DB)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "github sync failed: "+err.Error())
			return
		}
		for _, e := range result.Errors {
			log.Printf("[github-sync] error: %s", e)
		}
		writeJSON(w, http.StatusOK, result)
	case "argocd":
		result, err := argocd.Sync(r.Context(), p.Config, h.DB)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "argocd sync failed: "+err.Error())
			return
		}
		for _, e := range result.Errors {
			log.Printf("[argocd-sync] error: %s", e)
		}
		writeJSON(w, http.StatusOK, result)
	default:
		writeError(w, http.StatusNotImplemented, "sync not supported for plugin: "+name)
	}
}

// argoCDClientForInstance looks up the ArgoCD plugin, validates it is enabled,
// and returns a client for the instance named by the ?instance= query param
// (or the first/only instance if omitted).
func (h *Handlers) argoCDClientForInstance(r *http.Request) (*argocd.Client, error) {
	p, err := h.DB.GetPlugin(r.Context(), "argocd")
	if err != nil || p == nil {
		return nil, fmt.Errorf("argocd plugin not installed")
	}
	if !p.Enabled {
		return nil, fmt.Errorf("argocd plugin is not enabled")
	}

	instanceName := r.URL.Query().Get("instance")
	instances := argocd.AllInstanceConfigs(p.Config)
	cfg := instances[0] // default: first instance
	for _, inst := range instances {
		if n, _ := inst["instanceName"].(string); n == instanceName || instanceName == "" {
			cfg = inst
			break
		}
	}
	return argocd.NewClient(cfg)
}

// GetArgoCDEntityApps returns live status for all ArgoCD apps associated with
// an entity. The appNames query param is a comma-separated list of
// "instanceName:appName" pairs stored in the argocd.io/appNames annotation.
func (h *Handlers) GetArgoCDEntityApps(w http.ResponseWriter, r *http.Request) {
	appNamesParam := r.URL.Query().Get("appNames")
	if appNamesParam == "" {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	p, err := h.DB.GetPlugin(r.Context(), "argocd")
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "argocd plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "argocd plugin is not enabled")
		return
	}

	// Build instance name → config map for quick lookup.
	instanceMap := make(map[string]map[string]any)
	for _, inst := range argocd.AllInstanceConfigs(p.Config) {
		if name, _ := inst["instanceName"].(string); name != "" {
			instanceMap[name] = inst
		}
	}

	var results []*argocd.AppWithInstance
	for _, pair := range strings.Split(appNamesParam, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		instanceName, appName := parts[0], parts[1]
		cfg, ok := instanceMap[instanceName]
		if !ok {
			continue
		}
		client, err := argocd.NewClient(cfg)
		if err != nil {
			continue
		}
		app, err := client.GetApplication(appName)
		if err != nil {
			continue
		}
		results = append(results, &argocd.AppWithInstance{
			Instance:          instanceName,
			AppStatusResponse: argocd.AppToStatusResponse(app),
		})
	}
	if results == nil {
		results = []*argocd.AppWithInstance{}
	}
	writeJSON(w, http.StatusOK, results)
}

// GetArgoCDApp returns live status for a single ArgoCD Application by name.
// An optional ?instance= query param selects the ArgoCD instance; defaults to the first.
func (h *Handlers) GetArgoCDApp(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")

	client, err := h.argoCDClientForInstance(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	app, err := client.GetApplication(appName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "argocd get app: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, argocd.AppToStatusResponse(app))
}

// SyncArgoCDApp triggers a sync (and optional hard sync) for an ArgoCD Application.
// An optional ?instance= query param selects the ArgoCD instance.
func (h *Handlers) SyncArgoCDApp(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")

	var body struct {
		Hard bool `json:"hard"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	client, err := h.argoCDClientForInstance(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := client.SyncApplication(appName, body.Hard); err != nil {
		writeError(w, http.StatusInternalServerError, "argocd sync: "+err.Error())
		return
	}

	app, err := client.GetApplication(appName)
	if err != nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, http.StatusOK, argocd.AppToStatusResponse(app))
}

// RefreshArgoCDApp triggers a git refresh (no sync) for an ArgoCD Application.
// An optional ?instance= query param selects the ArgoCD instance.
func (h *Handlers) RefreshArgoCDApp(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")

	client, err := h.argoCDClientForInstance(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	app, err := client.RefreshApplication(appName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "argocd refresh: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, argocd.AppToStatusResponse(app))
}

// allClusterConfigs extracts every cluster from the plugin config and returns
// each as a flat map with the keys expected by k8s.NewClient:
// clusterUrl, token, caData (optional), namespace (optional).
// It handles both the clusters-array format and the legacy flat format.
func allClusterConfigs(raw map[string]any) []map[string]any {
	if raw == nil {
		return nil
	}
	if clusters, ok := raw["clusters"]; ok {
		if arr, ok := clusters.([]any); ok && len(arr) > 0 {
			var out []map[string]any
			for _, item := range arr {
				c, ok := item.(map[string]any)
				if !ok {
					continue
				}
				cfg := map[string]any{}
				if u, _ := c["url"].(string); u != "" {
					cfg["clusterUrl"] = u
				}
				if t, _ := c["token"].(string); t != "" {
					cfg["token"] = t
				}
				if ca, _ := c["caData"].(string); ca != "" {
					cfg["caData"] = ca
				}
				if ns, _ := c["namespaces"].(string); ns != "" {
					cfg["namespace"] = ns
				}
				if name, _ := c["name"].(string); name != "" {
					cfg["clusterName"] = name
				}
				if cfg["clusterUrl"] != nil && cfg["token"] != nil {
					out = append(out, cfg)
				}
			}
			return out
		}
	}
	// Flat format (legacy): treat the whole config as a single cluster.
	cfg := map[string]any{}
	if u, _ := raw["clusterUrl"].(string); u != "" {
		cfg["clusterUrl"] = u
	}
	if t, _ := raw["token"].(string); t != "" {
		cfg["token"] = t
	}
	if ca, _ := raw["caData"].(string); ca != "" {
		cfg["caData"] = ca
	}
	if ns, _ := raw["namespace"].(string); ns != "" {
		cfg["namespace"] = ns
	}
	if cfg["clusterUrl"] != nil && cfg["token"] != nil {
		return []map[string]any{cfg}
	}
	return nil
}

// normalizeK8sConfig is kept for callers that still need a single-cluster flat config.
// It returns the first cluster config or the flat config as-is.
func normalizeK8sConfig(raw map[string]any) map[string]any {
	cfgs := allClusterConfigs(raw)
	if len(cfgs) > 0 {
		return cfgs[0]
	}
	return raw
}

// GetKubernetesWorkload returns deployment and pod info for an app across its namespaces.
func (h *Handlers) GetKubernetesWorkload(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")

	var namespaces []string
	if ns := r.URL.Query().Get("namespaces"); ns != "" {
		for _, n := range strings.Split(ns, ",") {
			if n = strings.TrimSpace(n); n != "" {
				namespaces = append(namespaces, n)
			}
		}
	}

	p, err := h.DB.GetPlugin(r.Context(), "kubernetes")
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "kubernetes plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "kubernetes plugin is not enabled")
		return
	}

	clusters := allClusterConfigs(p.Config)
	if len(clusters) == 0 {
		writeError(w, http.StatusBadRequest, "no clusters configured")
		return
	}

	combined := &k8s.WorkloadInfo{AppName: appName, Deployments: []k8s.DeploymentInfo{}, Pods: []k8s.PodInfo{}}
	for _, cfg := range clusters {
		clusterNS := namespaces
		if len(clusterNS) == 0 {
			if ns, _ := cfg["namespace"].(string); ns != "" {
				for _, n := range strings.Split(ns, ",") {
					if n = strings.TrimSpace(n); n != "" {
						clusterNS = append(clusterNS, n)
					}
				}
			}
		}
		client, err := k8s.NewClient(cfg)
		if err != nil {
			continue
		}
		info, err := client.GetWorkload(appName, clusterNS)
		if err != nil {
			continue
		}
		// Tag each pod with its source cluster so the frontend can route log requests.
		if cname, _ := cfg["clusterName"].(string); cname != "" {
			for i := range info.Pods {
				info.Pods[i].ClusterName = cname
			}
		}
		combined.Deployments = append(combined.Deployments, info.Deployments...)
		combined.Pods = append(combined.Pods, info.Pods...)
	}
	writeJSON(w, http.StatusOK, combined)
}

// StreamKubernetesPodLogs streams logs from a pod container as plain text.
// An optional ?cluster= query param selects the target cluster by name; defaults to the first cluster.
func (h *Handlers) StreamKubernetesPodLogs(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	pod := chi.URLParam(r, "pod")
	container := chi.URLParam(r, "container")
	clusterParam := r.URL.Query().Get("cluster")

	p, err := h.DB.GetPlugin(r.Context(), "kubernetes")
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "kubernetes plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "kubernetes plugin is not enabled")
		return
	}

	// Find the cluster config matching the requested cluster name.
	// Falls back to the first cluster if no name is provided or no match found.
	var config map[string]any
	for _, cfg := range allClusterConfigs(p.Config) {
		name, _ := cfg["clusterName"].(string)
		if clusterParam == "" || name == clusterParam {
			config = cfg
			break
		}
	}
	if config == nil {
		writeError(w, http.StatusBadRequest, "cluster not found: "+clusterParam)
		return
	}

	client, err := k8s.NewClient(config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	client.StreamLogs(w, namespace, pod, container, 200)
}

// newShortID generates a short random ID for plugin records.
func newShortID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
