#!/usr/bin/env python3
"""
implement-external-plugins.py

Applies the Gantry external plugin system to a fresh Gantry codebase.
Run from the root of the Gantry repository:

    python3 implement-external-plugins.py

The script is idempotent: it checks for existing markers before applying
each change and skips sections that are already present.
"""

import os
import sys
import textwrap

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def check(path):
    if not os.path.exists(path):
        print(f"  ERROR: {path} not found — are you running from the Gantry repo root?")
        sys.exit(1)

def read(path):
    with open(path, "r") as f:
        return f.read()

def write(path, content):
    dir_name = os.path.dirname(path)
    if dir_name:  # Only create directories if a directory path actually exists
        os.makedirs(dir_name, exist_ok=True)
    with open(path, "w") as f:
        f.write(content)


def patch(path, old, new, description=""):
    content = read(path)
    if new.strip() in content:
        label = description or old[:60].strip()
        print(f"  SKIP (already applied): {label}")
        return
    if old not in content:
        print(f"  ERROR: anchor not found in {path}:")
        print(f"    >>> {repr(old[:80])}")
        sys.exit(1)
    write(path, content.replace(old, new, 1))
    label = description or old[:60].strip()
    print(f"  OK: {label}")

def create(path, content, description=""):
    if os.path.exists(path):
        print(f"  SKIP (already exists): {path}")
        return
    write(path, content)
    print(f"  OK: created {path}")

def section(title):
    print(f"\n{'='*60}")
    print(f"  {title}")
    print(f"{'='*60}")

# ---------------------------------------------------------------------------
# Preflight
# ---------------------------------------------------------------------------

section("Preflight checks")
for required in [
    "go.mod", "cmd/gantry/serve.go", "internal/config/config.go",
    "internal/db/queries.go", "internal/plugins/manifest.go",
    "internal/api/handlers/handlers.go", "internal/api/handlers/plugins.go",
    "internal/api/server.go",
]:
    check(required)
    print(f"  OK: {required} found")

mod = read("go.mod")
if "github.com/go2engle/gantry" not in mod:
    print("  ERROR: go.mod does not look like a Gantry repository")
    sys.exit(1)
print("  OK: module path confirmed")

# ---------------------------------------------------------------------------
# Part 1 — New package: internal/plugins/external/
# ---------------------------------------------------------------------------

section("Part 1 — Creating internal/plugins/external/")

create("internal/plugins/external/handshake.go", '''\
package external

import "github.com/hashicorp/go-plugin"

// Handshake is the shared configuration used by both Gantry (client) and
// external plugin binaries (server). A mismatching magic cookie causes
// go-plugin to return an error immediately so accidentally-invoked binaries
// fail fast rather than hanging.
var Handshake = plugin.HandshakeConfig{
\tProtocolVersion:  1,
\tMagicCookieKey:   "GANTRY_PLUGIN",
\tMagicCookieValue: "gantry-plugin-v1",
}
''')

create("internal/plugins/external/plugin.go", '''\
// Package external implements the Gantry external plugin system.
// External plugins are separate binaries discovered at runtime from a configured
// directory. They communicate with Gantry via net/rpc using hashicorp/go-plugin
// for subprocess lifecycle management.
package external

import (
\t"encoding/gob"
\t"encoding/json"
\t"net/rpc"

\t"github.com/hashicorp/go-plugin"
)

func init() {
\t// map[string]any is not registered by default; without this, ConfigSchema
\t// values are silently dropped during gob encoding over net/rpc.
\tgob.Register(map[string]any{})
}

// Manifest is a plugin\'s self-description, returned by GetManifest.
type Manifest struct {
\tName            string        `json:"name"`
\tTitle           string        `json:"title"`
\tDescription     string        `json:"description"`
\tVersion         string        `json:"version"`
\tAuthor          string        `json:"author"`
\tCategory        string        `json:"category"`
\tIconURL         string        `json:"iconUrl,omitempty"`
\tHomepage        string        `json:"homepage,omitempty"`
\tSupportsSync    bool          `json:"supportsSync"`
\tSupportsPanels  bool          `json:"supportsPanels"`
\tSupportsActions bool          `json:"supportsActions"`
\tEntityPanels    []string      `json:"entityPanels,omitempty"`
\tActionTypes     []string      `json:"actionTypes,omitempty"`
\tRequirements    []Requirement `json:"requirements,omitempty"`
\t// ConfigSchemaJSON carries the JSON-encoded configSchema over gob.
\t// map[string]any cannot cross the gob boundary safely; a JSON string can.
\tConfigSchemaJSON string `json:"configSchemaJson,omitempty"`
\tSupportsHTTP     bool   `json:"supportsHttp,omitempty"`
\t// HTTPRoutesJSON carries the JSON-encoded []Route over gob.
\tHTTPRoutesJSON string `json:"httpRoutesJson,omitempty"`
}

// Requirement describes an infrastructure or configuration prerequisite.
type Requirement struct {
\tName        string `json:"name"`
\tDescription string `json:"description"`
\tOptional    bool   `json:"optional,omitempty"`
}

// SyncResult is returned by Sync.
type SyncResult struct {
\tCreated int      `json:"created"`
\tUpdated int      `json:"updated"`
\tErrors  []string `json:"errors,omitempty"`
}

// PanelArgs are the arguments passed to GetPanelData.
type PanelArgs struct {
\tKind      string
\tNamespace string
\tName      string
}

// ActionArgs are the arguments passed to ExecuteAction.
type ActionArgs struct {
\tActionName string
\tInputs     map[string]string
}

// ActionResult is returned by ExecuteAction.
type ActionResult struct {
\tOutputs map[string]string
}

// Route mirrors sdk.Route: a path prefix the plugin wants Gantry to proxy.
type Route struct {
\tPath    string   `json:"path"`
\tMethods []string `json:"methods,omitempty"`
}

// GantryPluginServer is the interface that plugin binaries implement.
type GantryPluginServer interface {
\tGetManifest() (Manifest, error)
\tConfigure(config map[string]string) error
\tSync() (SyncResult, error)
\tGetPanelData(args PanelArgs) (json.RawMessage, error)
\tExecuteAction(args ActionArgs) (ActionResult, error)
}

// GantryPluginRPC is the client-side interface used inside Gantry.
type GantryPluginRPC interface {
\tGetManifest() (Manifest, error)
\tGetListenAddr() (string, error)
\tConfigure(config map[string]string) error
\tSync() (SyncResult, error)
\tGetPanelData(args PanelArgs) (json.RawMessage, error)
\tExecuteAction(args ActionArgs) (ActionResult, error)
}

type rpcClient struct {
\tclient *rpc.Client
}

func (c *rpcClient) GetManifest() (Manifest, error) {
\tvar reply Manifest
\treturn reply, c.client.Call("Plugin.GetManifest", new(struct{}), &reply)
}

func (c *rpcClient) GetListenAddr() (string, error) {
\tvar addr string
\treturn addr, c.client.Call("Plugin.GetListenAddr", struct{}{}, &addr)
}

func (c *rpcClient) Configure(config map[string]string) error {
\treturn c.client.Call("Plugin.Configure", &config, new(struct{}))
}

func (c *rpcClient) Sync() (SyncResult, error) {
\tvar reply SyncResult
\treturn reply, c.client.Call("Plugin.Sync", new(struct{}), &reply)
}

func (c *rpcClient) GetPanelData(args PanelArgs) (json.RawMessage, error) {
\tvar reply json.RawMessage
\treturn reply, c.client.Call("Plugin.GetPanelData", &args, &reply)
}

func (c *rpcClient) ExecuteAction(args ActionArgs) (ActionResult, error) {
\tvar reply ActionResult
\treturn reply, c.client.Call("Plugin.ExecuteAction", &args, &reply)
}

type rpcServer struct {
\timpl GantryPluginServer
}

func (s *rpcServer) GetManifest(_ *struct{}, reply *Manifest) error {
\tm, err := s.impl.GetManifest()
\tif err != nil {
\t\treturn err
\t}
\t*reply = m
\treturn nil
}

func (s *rpcServer) GetListenAddr(_ *struct{}, reply *string) error {
\t*reply = ""
\treturn nil
}

func (s *rpcServer) Configure(config *map[string]string, _ *struct{}) error {
\tif config == nil {
\t\treturn s.impl.Configure(nil)
\t}
\treturn s.impl.Configure(*config)
}

func (s *rpcServer) Sync(_ *struct{}, reply *SyncResult) error {
\tr, err := s.impl.Sync()
\tif err != nil {
\t\treturn err
\t}
\t*reply = r
\treturn nil
}

func (s *rpcServer) GetPanelData(args *PanelArgs, reply *json.RawMessage) error {
\tif args == nil {
\t\targs = &PanelArgs{}
\t}
\tdata, err := s.impl.GetPanelData(*args)
\tif err != nil {
\t\treturn err
\t}
\t*reply = data
\treturn nil
}

func (s *rpcServer) ExecuteAction(args *ActionArgs, reply *ActionResult) error {
\tif args == nil {
\t\targs = &ActionArgs{}
\t}
\tr, err := s.impl.ExecuteAction(*args)
\tif err != nil {
\t\treturn err
\t}
\t*reply = r
\treturn nil
}

// GantryPlugin is the go-plugin Plugin interface implementation.
type GantryPlugin struct {
\tImpl GantryPluginServer
}

func (p *GantryPlugin) Server(_ *plugin.MuxBroker) (interface{}, error) {
\treturn &rpcServer{impl: p.Impl}, nil
}

func (p *GantryPlugin) Client(_ *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
\treturn &rpcClient{client: c}, nil
}

// PluginMap must use "Plugin" as the key — net/rpc registers methods under
// the struct type name (Plugin.GetManifest, Plugin.Configure, etc.).
var PluginMap = map[string]plugin.Plugin{
\t"Plugin": &GantryPlugin{},
}
''')

create("internal/plugins/external/loader.go", '''\
package external

import (
\t"fmt"
\t"os"
\t"path/filepath"
\t"strings"
)

// ScanDir returns the absolute paths of all executables in dir whose base name
// starts with "gantry-plugin-". Returns an empty slice (no error) when dir is
// empty or does not exist.
func ScanDir(dir string) ([]string, error) {
\tif dir == "" {
\t\treturn nil, nil
\t}
\tentries, err := os.ReadDir(dir)
\tif os.IsNotExist(err) {
\t\treturn nil, nil
\t}
\tif err != nil {
\t\treturn nil, fmt.Errorf("scan plugin dir %s: %w", dir, err)
\t}

\tvar paths []string
\tfor _, e := range entries {
\t\tif e.IsDir() {
\t\t\tcontinue
\t\t}
\t\tif !strings.HasPrefix(e.Name(), "gantry-plugin-") {
\t\t\tcontinue
\t\t}
\t\tinfo, err := e.Info()
\t\tif err != nil {
\t\t\tcontinue
\t\t}
\t\tif info.Mode()&0o111 == 0 {
\t\t\tcontinue
\t\t}
\t\tpaths = append(paths, filepath.Join(dir, e.Name()))
\t}
\treturn paths, nil
}

// PluginNameFromPath derives the plugin name from a binary path.
// e.g. "/opt/plugins/gantry-plugin-vcluster" -> "vcluster"
func PluginNameFromPath(path string) string {
\tbase := filepath.Base(path)
\treturn strings.TrimPrefix(base, "gantry-plugin-")
}
''')

create("internal/plugins/external/client.go", '''\
package external

import (
\t"encoding/json"
\t"fmt"
\t"log"
\t"os/exec"
\t"sync"

\t"github.com/hashicorp/go-hclog"
\t"github.com/hashicorp/go-plugin"
)

// ExternalPlugin wraps a hashicorp/go-plugin client for a single external plugin binary.
type ExternalPlugin struct {
\tName    string
\tBinPath string

\tmu        sync.RWMutex
\tclient    *plugin.Client
\trpc       GantryPluginRPC
\tmanifest  *Manifest
\tavailable bool
}

// Start launches the plugin subprocess and calls GetManifest + Configure.
func (ep *ExternalPlugin) Start(config map[string]string) error {
\tlogger := hclog.New(&hclog.LoggerOptions{
\t\tName:   fmt.Sprintf("plugin.%s", ep.Name),
\t\tLevel:  hclog.Info,
\t\tOutput: hclog.DefaultOutput,
\t})

\tc := plugin.NewClient(&plugin.ClientConfig{
\t\tHandshakeConfig: Handshake,
\t\tPlugins:         PluginMap,
\t\tCmd:             exec.Command(ep.BinPath),
\t\tLogger:          logger,
\t\tManaged:         true,
\t})

\trpcClient, err := c.Client()
\tif err != nil {
\t\tc.Kill()
\t\treturn fmt.Errorf("plugin %s: connect: %w", ep.Name, err)
\t}

\traw, err := rpcClient.Dispense("Plugin")
\tif err != nil {
\t\tc.Kill()
\t\treturn fmt.Errorf("plugin %s: dispense: %w", ep.Name, err)
\t}

\trpc, ok := raw.(GantryPluginRPC)
\tif !ok {
\t\tc.Kill()
\t\treturn fmt.Errorf("plugin %s: unexpected RPC type %T", ep.Name, raw)
\t}

\tm, err := rpc.GetManifest()
\tif err != nil {
\t\tc.Kill()
\t\treturn fmt.Errorf("plugin %s: GetManifest: %w", ep.Name, err)
\t}

\tif err := rpc.Configure(config); err != nil {
\t\tc.Kill()
\t\treturn fmt.Errorf("plugin %s: Configure: %w", ep.Name, err)
\t}

\tep.mu.Lock()
\tep.client = c
\tep.rpc = rpc
\tep.manifest = &m
\tep.available = true
\tep.mu.Unlock()

\tlog.Printf("[external-plugin] %s started (version %s)", ep.Name, m.Version)
\treturn nil
}

// Kill terminates the plugin subprocess.
func (ep *ExternalPlugin) Kill() {
\tep.mu.Lock()
\tdefer ep.mu.Unlock()
\tif ep.client != nil {
\t\tep.client.Kill()
\t\tep.client = nil
\t\tep.rpc = nil
\t}
\tep.available = false
}

// Available reports whether the plugin subprocess is running and ready.
func (ep *ExternalPlugin) Available() bool {
\tep.mu.RLock()
\tdefer ep.mu.RUnlock()
\treturn ep.available
}

// GetManifest returns the cached manifest obtained at startup.
func (ep *ExternalPlugin) GetManifest() *Manifest {
\tep.mu.RLock()
\tdefer ep.mu.RUnlock()
\treturn ep.manifest
}

// GetListenAddr calls the plugin subprocess for its embedded HTTP server address.
func (ep *ExternalPlugin) GetListenAddr() string {
\tep.mu.RLock()
\trpc := ep.rpc
\tep.mu.RUnlock()
\tif rpc == nil {
\t\treturn ""
\t}
\taddr, _ := rpc.GetListenAddr()
\treturn addr
}

// Sync calls the plugin\'s Sync method.
func (ep *ExternalPlugin) Sync() (SyncResult, error) {
\tep.mu.RLock()
\trpc := ep.rpc
\tep.mu.RUnlock()
\tif rpc == nil {
\t\treturn SyncResult{}, fmt.Errorf("plugin %s is not available", ep.Name)
\t}
\treturn rpc.Sync()
}

// GetPanelData calls the plugin\'s GetPanelData method.
func (ep *ExternalPlugin) GetPanelData(args PanelArgs) (json.RawMessage, error) {
\tep.mu.RLock()
\trpc := ep.rpc
\tep.mu.RUnlock()
\tif rpc == nil {
\t\treturn nil, fmt.Errorf("plugin %s is not available", ep.Name)
\t}
\treturn rpc.GetPanelData(args)
}

// ExecuteAction calls the plugin\'s ExecuteAction method.
func (ep *ExternalPlugin) ExecuteAction(args ActionArgs) (ActionResult, error) {
\tep.mu.RLock()
\trpc := ep.rpc
\tep.mu.RUnlock()
\tif rpc == nil {
\t\treturn ActionResult{}, fmt.Errorf("plugin %s is not available", ep.Name)
\t}
\treturn rpc.ExecuteAction(args)
}

// Configure (re-)sends configuration to a running plugin subprocess.
func (ep *ExternalPlugin) Configure(config map[string]string) error {
\tep.mu.RLock()
\trpc := ep.rpc
\tep.mu.RUnlock()
\tif rpc == nil {
\t\treturn fmt.Errorf("plugin %s is not available", ep.Name)
\t}
\treturn rpc.Configure(config)
}

func (ep *ExternalPlugin) isDead() bool {
\tep.mu.RLock()
\tc := ep.client
\tep.mu.RUnlock()
\tif c == nil {
\t\treturn true
\t}
\treturn c.Exited()
}

func (ep *ExternalPlugin) markUnavailable() {
\tep.mu.Lock()
\tep.available = false
\tep.mu.Unlock()
}
''')

create("internal/plugins/external/manager.go", '''\
package external

import (
\t"context"
\t"log"
\t"sync"
\t"time"
)

const healthCheckInterval = 30 * time.Second

// ConfigLoader returns the stored config for a plugin by name.
type ConfigLoader func(ctx context.Context, name string) (map[string]string, error)

// Manager holds all running external plugin subprocesses.
type Manager struct {
\tmu         sync.RWMutex
\tplugins    map[string]*ExternalPlugin
\tconfigLoad ConfigLoader
\tcancel     context.CancelFunc
}

// NewManager creates a Manager. configLoad is called at startup and on restart.
func NewManager(configLoad ConfigLoader) *Manager {
\treturn &Manager{
\t\tplugins:    make(map[string]*ExternalPlugin),
\t\tconfigLoad: configLoad,
\t}
}

// StartAll scans pluginDir, launches each discovered binary, and registers it.
func (m *Manager) StartAll(ctx context.Context, pluginDir string) error {
\tpaths, err := ScanDir(pluginDir)
\tif err != nil {
\t\treturn err
\t}
\tif len(paths) == 0 {
\t\treturn nil
\t}

\tfor _, path := range paths {
\t\tname := PluginNameFromPath(path)
\t\tep := &ExternalPlugin{Name: name, BinPath: path}

\t\tconfig, err := m.loadConfig(ctx, name)
\t\tif err != nil {
\t\t\tlog.Printf("[external-plugin] %s: failed to load config: %v", name, err)
\t\t\tconfig = nil
\t\t}

\t\tif err := ep.Start(config); err != nil {
\t\t\tlog.Printf("[external-plugin] %s: failed to start: %v", name, err)
\t\t}

\t\tm.mu.Lock()
\t\tm.plugins[name] = ep
\t\tm.mu.Unlock()
\t}

\thctx, cancel := context.WithCancel(context.Background())
\tm.cancel = cancel
\tgo m.healthLoop(hctx)

\treturn nil
}

// StopAll kills all plugin subprocesses and stops the health check loop.
func (m *Manager) StopAll() {
\tif m.cancel != nil {
\t\tm.cancel()
\t}
\tm.mu.RLock()
\tdefer m.mu.RUnlock()
\tfor _, ep := range m.plugins {
\t\tep.Kill()
\t}
}

// Get returns the ExternalPlugin for the given name, or nil if not found.
func (m *Manager) Get(name string) *ExternalPlugin {
\tm.mu.RLock()
\tdefer m.mu.RUnlock()
\treturn m.plugins[name]
}

// All returns a snapshot of all registered external plugins.
func (m *Manager) All() []*ExternalPlugin {
\tm.mu.RLock()
\tdefer m.mu.RUnlock()
\tout := make([]*ExternalPlugin, 0, len(m.plugins))
\tfor _, ep := range m.plugins {
\t\tout = append(out, ep)
\t}
\treturn out
}

func (m *Manager) healthLoop(ctx context.Context) {
\tticker := time.NewTicker(healthCheckInterval)
\tdefer ticker.Stop()
\tfor {
\t\tselect {
\t\tcase <-ctx.Done():
\t\t\treturn
\t\tcase <-ticker.C:
\t\t\tm.checkHealth(ctx)
\t\t}
\t}
}

func (m *Manager) checkHealth(ctx context.Context) {
\tm.mu.RLock()
\teps := make([]*ExternalPlugin, 0, len(m.plugins))
\tfor _, ep := range m.plugins {
\t\teps = append(eps, ep)
\t}
\tm.mu.RUnlock()

\tfor _, ep := range eps {
\t\tif !ep.isDead() {
\t\t\tcontinue
\t\t}
\t\tlog.Printf("[external-plugin] %s: subprocess exited, attempting restart", ep.Name)
\t\tep.Kill()

\t\tconfig, err := m.loadConfig(ctx, ep.Name)
\t\tif err != nil {
\t\t\tlog.Printf("[external-plugin] %s: restart aborted: %v", ep.Name, err)
\t\t\tep.markUnavailable()
\t\t\tcontinue
\t\t}

\t\tif err := ep.Start(config); err != nil {
\t\t\tlog.Printf("[external-plugin] %s: restart failed: %v", ep.Name, err)
\t\t\tep.markUnavailable()
\t\t} else {
\t\t\tlog.Printf("[external-plugin] %s: restarted successfully", ep.Name)
\t\t}
\t}
}

func (m *Manager) loadConfig(ctx context.Context, name string) (map[string]string, error) {
\tif m.configLoad == nil {
\t\treturn nil, nil
\t}
\treturn m.configLoad(ctx, name)
}
''')

# ---------------------------------------------------------------------------
# Part 2 — New handler files
# ---------------------------------------------------------------------------

section("Part 2 — Creating new handler files")

create("internal/api/handlers/internal_entity.go", '''\
package handlers

import (
\t"encoding/json"
\t"errors"
\t"net/http"

\t"github.com/go2engle/gantry/internal/entity"
)

// InternalGetEntity handles GET /api/internal/entity.
// Authenticated via X-Gantry-Internal-Token.
// Query params: kind, namespace, name.
// When name is provided returns a single entity; when omitted returns an array.
func (h *Handlers) InternalGetEntity(w http.ResponseWriter, r *http.Request) {
\tif r.Header.Get("X-Gantry-Internal-Token") != h.InternalPluginToken {
\t\twriteError(w, http.StatusUnauthorized, "invalid internal token")
\t\treturn
\t}

\tkind := r.URL.Query().Get("kind")
\tnamespace := r.URL.Query().Get("namespace")
\tname := r.URL.Query().Get("name")
\tctx := r.Context()

\tif name != "" {
\t\te, err := h.DB.GetEntity(ctx, kind, namespace, name)
\t\tif errors.Is(err, entity.ErrEntityNotFound) || e == nil {
\t\t\twriteError(w, http.StatusNotFound, "entity not found")
\t\t\treturn
\t\t}
\t\tif err != nil {
\t\t\twriteError(w, http.StatusInternalServerError, "get entity: "+err.Error())
\t\t\treturn
\t\t}
\t\twriteJSON(w, http.StatusOK, e)
\t\treturn
\t}

\tentities, err := h.DB.ListEntities(ctx, kind, namespace)
\tif err != nil {
\t\twriteError(w, http.StatusInternalServerError, "list entities: "+err.Error())
\t\treturn
\t}
\tif entities == nil {
\t\tentities = []*entity.Entity{}
\t}
\twriteJSON(w, http.StatusOK, entities)
}

// InternalUpsertEntity handles POST /api/internal/entity-upsert.
// Creates the entity if it does not exist; updates it if it does.
func (h *Handlers) InternalUpsertEntity(w http.ResponseWriter, r *http.Request) {
\tif r.Header.Get("X-Gantry-Internal-Token") != h.InternalPluginToken {
\t\twriteError(w, http.StatusUnauthorized, "invalid internal token")
\t\treturn
\t}

\tvar e entity.Entity
\tif err := json.NewDecoder(r.Body).Decode(&e); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
\t\treturn
\t}
\te.SetDefaults()
\tif err := e.Validate(); err != nil {
\t\twriteError(w, http.StatusBadRequest, err.Error())
\t\treturn
\t}

\tctx := r.Context()
\texisting, err := h.DB.GetEntity(ctx, e.Kind, e.Metadata.Namespace, e.Metadata.Name)
\tif err != nil && !errors.Is(err, entity.ErrEntityNotFound) {
\t\twriteError(w, http.StatusInternalServerError, "get entity: "+err.Error())
\t\treturn
\t}

\tif existing == nil || errors.Is(err, entity.ErrEntityNotFound) {
\t\tif err := h.DB.CreateEntity(ctx, &e); err != nil {
\t\t\twriteError(w, http.StatusInternalServerError, "create entity: "+err.Error())
\t\t\treturn
\t\t}
\t\twriteJSON(w, http.StatusCreated, &e)
\t\treturn
\t}

\te.Metadata.CreatedAt = existing.Metadata.CreatedAt
\tif err := h.DB.UpdateEntity(ctx, &e); err != nil {
\t\twriteError(w, http.StatusInternalServerError, "update entity: "+err.Error())
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, &e)
}

// InternalDeleteEntity handles POST /api/internal/entity-delete.
// Returns 204 on success, 404 if not found (idempotent).
func (h *Handlers) InternalDeleteEntity(w http.ResponseWriter, r *http.Request) {
\tif r.Header.Get("X-Gantry-Internal-Token") != h.InternalPluginToken {
\t\twriteError(w, http.StatusUnauthorized, "invalid internal token")
\t\treturn
\t}

\tvar req struct {
\t\tKind      string `json:"kind"`
\t\tNamespace string `json:"namespace"`
\t\tName      string `json:"name"`
\t}
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
\t\treturn
\t}
\tif req.Kind == "" || req.Name == "" {
\t\twriteError(w, http.StatusBadRequest, "kind and name are required")
\t\treturn
\t}
\tif req.Namespace == "" {
\t\treq.Namespace = entity.DefaultNamespace
\t}

\tctx := r.Context()
\tif err := h.DB.DeleteEntity(ctx, req.Kind, req.Namespace, req.Name); err != nil {
\t\tif errors.Is(err, entity.ErrEntityNotFound) {
\t\t\tw.WriteHeader(http.StatusNotFound)
\t\t\treturn
\t\t}
\t\twriteError(w, http.StatusInternalServerError, "delete entity: "+err.Error())
\t\treturn
\t}
\tw.WriteHeader(http.StatusNoContent)
}
''')

create("internal/api/plugin_proxy.go", '''\
package api

import (
\t"net/http"
\t"net/http/httputil"
\t"net/url"
)

// newPluginReverseProxy creates a reverse proxy that forwards requests to the
// plugin\'s embedded HTTP server at upstream. The full request path is preserved
// so the plugin can route against its own declared paths.
func newPluginReverseProxy(upstream, pathPrefix string) http.Handler {
\ttarget, _ := url.Parse(upstream)
\tproxy := httputil.NewSingleHostReverseProxy(target)

\torig := proxy.Director
\tproxy.Director = func(req *http.Request) {
\t\toriginalHost := req.Host
\t\torig(req)

\t\tif originalHost != "" {
\t\t\treq.Header.Set("X-Forwarded-Host", originalHost)
\t\t}

\t\tproto := "http"
\t\tif req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
\t\t\tproto = "https"
\t\t}
\t\treq.Header.Set("X-Forwarded-Proto", proto)
\t}

\treturn proxy
}
''')

# ---------------------------------------------------------------------------
# Part 3 — go.mod: add hashicorp dependencies
# ---------------------------------------------------------------------------

section("Part 3 — go.mod: add hashicorp dependencies")

patch(
    "go.mod",
    "\tgithub.com/hashicorp/golang-lru/v2 v2.0.7 // indirect",
    "\tgithub.com/hashicorp/go-hclog v1.6.3 // indirect\n"
    "\tgithub.com/hashicorp/go-plugin v1.8.0 // indirect\n"
    "\tgithub.com/hashicorp/golang-lru/v2 v2.0.7 // indirect",
    "add go-hclog and go-plugin",
)

if "hashicorp/yamux" in read("go.mod"):
    print("  SKIP (already applied): yamux already present")

# ---------------------------------------------------------------------------
# Part 4 — internal/config/config.go: add PluginDir
# ---------------------------------------------------------------------------

section("Part 4 — internal/config/config.go: add PluginDir")

patch(
    "internal/config/config.go",
    "\tEncryptionKey string `yaml:\"encryptionKey\"`\n}",
    "\tEncryptionKey string `yaml:\"encryptionKey\"`\n"
    "\tPluginDir     string `yaml:\"pluginDir\"`\n}",
    "FileConfig.PluginDir field",
)

patch(
    "internal/config/config.go",
    "\tEncryptionKey string // key for AES-256-GCM encryption of DB secrets (GANTRY_ENCRYPTION_KEY)\n}",
    "\tEncryptionKey string // key for AES-256-GCM encryption of DB secrets (GANTRY_ENCRYPTION_KEY)\n"
    "\tPluginDir     string // directory to scan for external plugin binaries (GANTRY_PLUGIN_DIR)\n}",
    "Config.PluginDir field",
)

patch(
    "internal/config/config.go",
    "\tif fc.EncryptionKey != \"\" {\n\t\tcfg.EncryptionKey = fc.EncryptionKey\n\t}\n}",
    "\tif fc.EncryptionKey != \"\" {\n\t\tcfg.EncryptionKey = fc.EncryptionKey\n\t}\n"
    "\tif fc.PluginDir != \"\" {\n\t\tcfg.PluginDir = fc.PluginDir\n\t}\n}",
    "applyFileConfig PluginDir",
)

patch(
    "internal/config/config.go",
    "\tif v := os.Getenv(\"GANTRY_ENCRYPTION_KEY\"); v != \"\" {\n\t\tcfg.EncryptionKey = v\n\t}\n}",
    "\tif v := os.Getenv(\"GANTRY_ENCRYPTION_KEY\"); v != \"\" {\n\t\tcfg.EncryptionKey = v\n\t}\n\n"
    "\tif v := os.Getenv(\"GANTRY_PLUGIN_DIR\"); v != \"\" {\n\t\tcfg.PluginDir = v\n\t}\n}",
    "applyEnv GANTRY_PLUGIN_DIR",
)

# ---------------------------------------------------------------------------
# Part 5 — internal/plugins/manifest.go: add Source field
# ---------------------------------------------------------------------------

section("Part 5 — internal/plugins/manifest.go: add Source field")

patch(
    "internal/plugins/manifest.go",
    "\tCategory    string            `json:\"category\"` // integration | widget | entity-kind | action-type | auth-provider\n\tBundleURL",
    "\tCategory    string            `json:\"category\"` // integration | widget | entity-kind | action-type | auth-provider\n"
    "\tSource      string            `json:\"source,omitempty\"`\n\tBundleURL",
    "Manifest.Source field",
)

# Add PluginRequirement type before the Plugin struct
patch(
    "internal/plugins/manifest.go",
    "// Plugin represents an installed plugin record combining a manifest with runtime state.",
    "// PluginRequirement describes a prerequisite for a plugin.\n"
    "type PluginRequirement struct {\n"
    "\tName        string `json:\"name\"`\n"
    "\tDescription string `json:\"description\"`\n"
    "\tOptional    bool   `json:\"optional,omitempty\"`\n"
    "}\n\n"
    "// Plugin represents an installed plugin record combining a manifest with runtime state.",
    "PluginRequirement type",
)

# Add Requirements to Manifest (after ActionTypes, using comment as anchor to distinguish from RegistryEntry)
patch(
    "internal/plugins/manifest.go",
    "\t// ActionTypes lists action types this plugin contributes.\n"
    "\tActionTypes []string `json:\"actionTypes,omitempty\"`\n"
    "}",
    "\t// ActionTypes lists action types this plugin contributes.\n"
    "\tActionTypes  []string             `json:\"actionTypes,omitempty\"`\n"
    "\tRequirements []PluginRequirement  `json:\"requirements,omitempty\"`\n"
    "}",
    "Manifest.Requirements field",
)

# Add Requirements + Source to RegistryEntry (aligned fields, no comment prefix)
patch(
    "internal/plugins/manifest.go",
    "\tActionTypes     []string       `json:\"actionTypes,omitempty\"`\n"
    "}",
    "\tActionTypes     []string            `json:\"actionTypes,omitempty\"`\n"
    "\tRequirements    []PluginRequirement `json:\"requirements,omitempty\"`\n"
    "\tSource          string              `json:\"source,omitempty\"`\n"
    "}",
    "RegistryEntry.Requirements and Source fields",
)

# ---------------------------------------------------------------------------
# Part 6 — internal/db/queries.go: add EnsureExternalPlugin
# ---------------------------------------------------------------------------

section("Part 6 — internal/db/queries.go: add EnsureExternalPlugin")

patch(
    "internal/db/queries.go",
    "// UpdatePluginEnabled sets the enabled flag for a plugin.",
    '''\
// EnsureExternalPlugin creates a DB record for an external plugin on first
// discovery, or updates the version and manifest on re-discovery while
// preserving the user\'s enabled state and config.
func (d *DB) EnsureExternalPlugin(ctx context.Context, manifest *plugins.Manifest) error {
\t// If a bundled plugin previously occupied this name, remove it so the
\t// external plugin starts with a clean enabled=false state and empty config.
\texisting, err := d.GetPlugin(ctx, manifest.Name)
\tif err != nil {
\t\treturn fmt.Errorf("check existing plugin %s: %w", manifest.Name, err)
\t}
\tif existing != nil && (existing.Manifest == nil || existing.Manifest.Source != "external") {
\t\tif _, err := d.exec(ctx, `DELETE FROM plugins WHERE name = ?`, manifest.Name); err != nil {
\t\t\treturn fmt.Errorf("remove stale bundled plugin %s: %w", manifest.Name, err)
\t\t}
\t}

\tnow := time.Now().UTC().Format(time.RFC3339)
\tmanifest.Source = "external"
\tmanifestJSON, err := json.Marshal(manifest)
\tif err != nil {
\t\treturn fmt.Errorf("marshal manifest for %s: %w", manifest.Name, err)
\t}

\tidBytes := make([]byte, 8)
\tif _, err := rand.Read(idBytes); err != nil {
\t\treturn fmt.Errorf("generate id for %s: %w", manifest.Name, err)
\t}
\tid := fmt.Sprintf("%x", idBytes)

\temptyConfig, err := json.Marshal(map[string]any{})
\tif err != nil {
\t\treturn err
\t}
\tencryptedConfig, err := gantrycrypto.Encrypt(d.encKey, emptyConfig)
\tif err != nil {
\t\treturn fmt.Errorf("encrypting empty config for %s: %w", manifest.Name, err)
\t}

\t_, err = d.exec(ctx, `
\t\tINSERT INTO plugins (id, name, version, enabled, config, manifest, installed_at, updated_at)
\t\tVALUES (?, ?, ?, 0, ?, ?, ?, ?)
\t\tON CONFLICT(name) DO UPDATE SET
\t\t\tversion    = excluded.version,
\t\t\tmanifest   = excluded.manifest,
\t\t\tupdated_at = excluded.updated_at`,
\t\tid, manifest.Name, manifest.Version,
\t\tencryptedConfig,
\t\tstring(manifestJSON),
\t\tnow, now,
\t)
\treturn err
}

// UpdatePluginEnabled sets the enabled flag for a plugin.''',
    "EnsureExternalPlugin function",
)

# ---------------------------------------------------------------------------
# Part 7 — internal/api/handlers/handlers.go: add new fields
# ---------------------------------------------------------------------------

section("Part 7 — internal/api/handlers/handlers.go: add new fields")

patch(
    "internal/api/handlers/handlers.go",
    "\t\"github.com/go2engle/gantry/internal/gitops\"\n\t\"github.com/go2engle/gantry/internal/search\"",
    "\t\"github.com/go2engle/gantry/internal/gitops\"\n"
    "\t\"github.com/go2engle/gantry/internal/plugins/external\"\n"
    "\t\"github.com/go2engle/gantry/internal/search\"",
    "handlers.go: add external import",
)

patch(
    "internal/api/handlers/handlers.go",
    "\tDispatcher *dispatcher.Manager\n\tGitOps     *gitops.Service\n\tDataDir    string // root data directory, used for GitOps repo storage\n\tVersion    string // build-time version string",
    "\tDispatcher          *dispatcher.Manager\n"
    "\tGitOps              *gitops.Service\n"
    "\tExternalManager     *external.Manager\n"
    "\tInternalPluginToken string // shared secret for plugin->Gantry internal API calls\n"
    "\tGantryURL           string // local base URL (e.g. http://127.0.0.1:8080) injected into plugin config\n"
    "\tDataDir             string // root data directory, used for GitOps repo storage\n"
    "\tVersion             string // build-time version string",
    "handlers.go: ExternalManager, InternalPluginToken, GantryURL fields",
)

# ---------------------------------------------------------------------------
# Part 8 — internal/api/handlers/plugins.go: multiple changes
# ---------------------------------------------------------------------------

section("Part 8 — internal/api/handlers/plugins.go: multiple changes")

# 8a: add imports
patch(
    "internal/api/handlers/plugins.go",
    "import (\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"log\"\n\t\"net/http\"\n\t\"strings\"",
    "import (\n\t\"context\"\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"log\"\n\t\"net/http\"\n\t\"strings\"\n\t\"time\"",
    "plugins.go: add context/time imports",
)

patch(
    "internal/api/handlers/plugins.go",
    "\t\"github.com/go2engle/gantry/internal/plugins\"\n\targocd",
    "\t\"github.com/go2engle/gantry/internal/plugins\"\n"
    "\t\"github.com/go2engle/gantry/internal/plugins/external\"\n"
    "\targocd",
    "plugins.go: add external import",
)

# 8b: ListPlugins — add Available field and external plugin metadata
patch(
    "internal/api/handlers/plugins.go",
    "\ttype pluginListItem struct {\n\t\tplugins.RegistryEntry\n\t\tEnabled bool `json:\"enabled\"`\n\t}",
    "\ttype pluginListItem struct {\n\t\tplugins.RegistryEntry\n\t\tEnabled   bool  `json:\"enabled\"`\n"
    "\t\tAvailable *bool `json:\"available,omitempty\"`\n\t}",
    "plugins.go: pluginListItem Available field",
)

patch(
    "internal/api/handlers/plugins.go",
    "\t\t// Prefer registry metadata (always up to date with the binary).\n"
    "\t\tif entry, ok := registryMap[p.Name]; ok {\n"
    "\t\t\titem.RegistryEntry = *entry\n"
    "\t\t} else if p.Manifest != nil {\n"
    "\t\t\titem.RegistryEntry = plugins.RegistryEntry{\n"
    "\t\t\t\tName:        p.Manifest.Name,\n"
    "\t\t\t\tTitle:       p.Manifest.Title,\n"
    "\t\t\t\tDescription: p.Manifest.Description,\n"
    "\t\t\t\tVersion:     p.Manifest.Version,\n"
    "\t\t\t\tAuthor:      p.Manifest.Author,\n"
    "\t\t\t\tCategory:    p.Manifest.Category,\n"
    "\t\t\t}\n"
    "\t\t}",
    "\t\t// Bundled plugins: prefer registry metadata (always up to date with the binary).\n"
    "\t\t// External plugins: use the manifest stored in the DB (set by EnsureExternalPlugin).\n"
    "\t\tif entry, ok := registryMap[p.Name]; ok {\n"
    "\t\t\titem.RegistryEntry = *entry\n"
    "\t\t\titem.RegistryEntry.Source = \"bundled\"\n"
    "\t\t} else if p.Manifest != nil {\n"
    "\t\t\titem.RegistryEntry = plugins.RegistryEntry{\n"
    "\t\t\t\tName:         p.Manifest.Name,\n"
    "\t\t\t\tTitle:        p.Manifest.Title,\n"
    "\t\t\t\tDescription:  p.Manifest.Description,\n"
    "\t\t\t\tVersion:      p.Manifest.Version,\n"
    "\t\t\t\tAuthor:       p.Manifest.Author,\n"
    "\t\t\t\tCategory:     p.Manifest.Category,\n"
    "\t\t\t\tIconURL:      p.Manifest.IconURL,\n"
    "\t\t\t\tHomepage:     p.Manifest.Homepage,\n"
    "\t\t\t\tEntityPanels: p.Manifest.EntityPanels,\n"
    "\t\t\t\tActionTypes:  p.Manifest.ActionTypes,\n"
    "\t\t\t\tRequirements: p.Manifest.Requirements,\n"
    "\t\t\t\tSource:       p.Manifest.Source,\n"
    "\t\t\t}\n"
    "\t\t\tif h.ExternalManager != nil {\n"
    "\t\t\t\tep := h.ExternalManager.Get(p.Name)\n"
    "\t\t\t\tavail := ep != nil && ep.Available()\n"
    "\t\t\t\titem.Available = &avail\n"
    "\t\t\t}\n"
    "\t\t}",
    "plugins.go: ListPlugins external plugin metadata + availability",
)

# 8c: GetPlugin — wrap response with Available field
patch(
    "internal/api/handlers/plugins.go",
    "\tp.Config = redactSecretValues(p.Config).(map[string]any)\n\twriteJSON(w, http.StatusOK, p)\n}",
    "\tp.Config = redactSecretValues(p.Config).(map[string]any)\n\n"
    "\ttype pluginDetail struct {\n"
    "\t\t*plugins.Plugin\n"
    "\t\tAvailable *bool `json:\"available,omitempty\"`\n"
    "\t}\n"
    "\tresp := &pluginDetail{Plugin: p}\n"
    "\tif h.ExternalManager != nil {\n"
    "\t\tif ep := h.ExternalManager.Get(name); ep != nil {\n"
    "\t\t\tavail := ep.Available()\n"
    "\t\t\tresp.Available = &avail\n"
    "\t\t}\n"
    "\t}\n"
    "\twriteJSON(w, http.StatusOK, resp)\n}",
    "plugins.go: GetPlugin availability",
)

# 8d: EnablePlugin — call Configure on external plugin subprocess
patch(
    "internal/api/handlers/plugins.go",
    "\t// Dynamically initialize or shut down the GitOps service.\n"
    "\tif name == \"gitops\" {\n"
    "\t\tgo h.InitGitOps()\n"
    "\t}\n\n"
    "\tw.WriteHeader(http.StatusNoContent)\n"
    "}\n\n"
    "// GetPluginConfig",
    "\t// Dynamically initialize or shut down the GitOps service.\n"
    "\tif name == \"gitops\" {\n"
    "\t\tgo h.InitGitOps()\n"
    "\t}\n\n"
    "\t// Notify external plugin subprocess of the enabled-state change.\n"
    "\tif h.ExternalManager != nil {\n"
    "\t\tif ep := h.ExternalManager.Get(name); ep != nil {\n"
    "\t\t\tcfgMap := make(map[string]string)\n"
    "\t\t\tif p, _ := h.DB.GetPlugin(r.Context(), name); p != nil {\n"
    "\t\t\t\tfor k, v := range p.Config {\n"
    "\t\t\t\t\tif s, ok := v.(string); ok {\n"
    "\t\t\t\t\t\tcfgMap[k] = s\n"
    "\t\t\t\t\t}\n"
    "\t\t\t\t}\n"
    "\t\t\t}\n"
    "\t\t\tif h.InternalPluginToken != \"\" {\n"
    "\t\t\t\tcfgMap[\"gantryInternalToken\"] = h.InternalPluginToken\n"
    "\t\t\t}\n"
    "\t\t\tif h.GantryURL != \"\" {\n"
    "\t\t\t\tcfgMap[\"gantryUrl\"] = h.GantryURL\n"
    "\t\t\t}\n"
    "\t\t\tif err := ep.Configure(cfgMap); err != nil {\n"
    "\t\t\t\tlog.Printf(\"[external-plugin:%s] configure after enable toggle: %v\", name, err)\n"
    "\t\t\t}\n"
    "\t\t}\n"
    "\t}\n\n"
    "\tw.WriteHeader(http.StatusNoContent)\n"
    "}\n\n"
    "// GetPluginConfig",
    "plugins.go: EnablePlugin external Configure call",
)

# 8e: UpdatePluginConfig — call Configure on external plugin subprocess
patch(
    "internal/api/handlers/plugins.go",
    "\t// Reinitialize GitOps if its config changed while enabled.\n"
    "\tif name == \"gitops\" {\n"
    "\t\tgo h.InitGitOps()\n"
    "\t}\n\n"
    "\tw.WriteHeader(http.StatusNoContent)\n"
    "}\n\n"
    "func redactSecretValues",
    "\t// Reinitialize GitOps if its config changed while enabled.\n"
    "\tif name == \"gitops\" {\n"
    "\t\tgo h.InitGitOps()\n"
    "\t}\n\n"
    "\t// Push updated config to external plugin subprocess.\n"
    "\tif h.ExternalManager != nil {\n"
    "\t\tif ep := h.ExternalManager.Get(name); ep != nil {\n"
    "\t\t\tcfgMap := make(map[string]string, len(merged))\n"
    "\t\t\tfor k, v := range merged {\n"
    "\t\t\t\tif s, ok := v.(string); ok {\n"
    "\t\t\t\t\tcfgMap[k] = s\n"
    "\t\t\t\t}\n"
    "\t\t\t}\n"
    "\t\t\tif h.InternalPluginToken != \"\" {\n"
    "\t\t\t\tcfgMap[\"gantryInternalToken\"] = h.InternalPluginToken\n"
    "\t\t\t}\n"
    "\t\t\tif h.GantryURL != \"\" {\n"
    "\t\t\t\tcfgMap[\"gantryUrl\"] = h.GantryURL\n"
    "\t\t\t}\n"
    "\t\t\tif err := ep.Configure(cfgMap); err != nil {\n"
    "\t\t\t\tlog.Printf(\"[external-plugin:%s] configure after config update: %v\", name, err)\n"
    "\t\t\t}\n"
    "\t\t}\n"
    "\t}\n\n"
    "\tw.WriteHeader(http.StatusNoContent)\n"
    "}\n\n"
    "func redactSecretValues",
    "plugins.go: UpdatePluginConfig external Configure call",
)

# 8f: SyncPlugin default case — dispatch to external plugin
patch(
    "internal/api/handlers/plugins.go",
    "\tdefault:\n\t\twriteError(w, http.StatusNotImplemented, \"sync not supported for plugin: \"+name)\n\t}\n}",
    "\tdefault:\n"
    "\t\t// Dispatch to external plugin subprocess if available.\n"
    "\t\tif h.ExternalManager != nil {\n"
    "\t\t\tif ep := h.ExternalManager.Get(name); ep != nil && ep.Available() {\n"
    "\t\t\t\tresult, err := ep.Sync()\n"
    "\t\t\t\tif err != nil {\n"
    "\t\t\t\t\twriteError(w, http.StatusInternalServerError, \"external plugin sync failed: \"+err.Error())\n"
    "\t\t\t\t\treturn\n"
    "\t\t\t\t}\n"
    "\t\t\t\twriteJSON(w, http.StatusOK, result)\n"
    "\t\t\t\treturn\n"
    "\t\t\t}\n"
    "\t\t}\n"
    "\t\twriteError(w, http.StatusNotImplemented, \"sync not supported for plugin: \"+name)\n"
    "\t}\n}",
    "plugins.go: SyncPlugin external dispatch",
)

# 8g: GetEntityPanelData handler — append after last function in file
plugins_content = read("internal/api/handlers/plugins.go")
panel_fn = "func (h *Handlers) GetEntityPanelData"
if panel_fn not in plugins_content:
    plugins_content += '''
// GetEntityPanelData calls GetPanelData on every enabled external plugin that
// declares support for the requested entity kind. Results are keyed by plugin name.
// Each plugin gets a 500 ms timeout.
func (h *Handlers) GetEntityPanelData(w http.ResponseWriter, r *http.Request) {
\tkind := chi.URLParam(r, "kind")
\tname := chi.URLParam(r, "name")
\tnamespace := r.URL.Query().Get("namespace")

\tif h.ExternalManager == nil {
\t\twriteJSON(w, http.StatusOK, map[string]any{})
\t\treturn
\t}

\tresults := make(map[string]json.RawMessage)
\tfor _, ep := range h.ExternalManager.All() {
\t\tif !ep.Available() {
\t\t\tcontinue
\t\t}
\t\tm := ep.GetManifest()
\t\tif m == nil || !entityKindInPanels(kind, m.EntityPanels) {
\t\t\tcontinue
\t\t}
\t\tdbPlugin, err := h.DB.GetPlugin(r.Context(), ep.Name)
\t\tif err != nil || dbPlugin == nil || !dbPlugin.Enabled {
\t\t\tcontinue
\t\t}

\t\ttype panelResult struct {
\t\t\tdata json.RawMessage
\t\t\terr  error
\t\t}
\t\tch := make(chan panelResult, 1)
\t\tepName := ep.Name
\t\targs := external.PanelArgs{Kind: kind, Name: name, Namespace: namespace}
\t\tgo func() {
\t\t\td, e := ep.GetPanelData(args)
\t\t\tch <- panelResult{d, e}
\t\t}()
\t\tctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
\t\tvar pr panelResult
\t\tselect {
\t\tcase pr = <-ch:
\t\tcase <-ctx.Done():
\t\t\tpr.err = ctx.Err()
\t\t}
\t\tcancel()
\t\tif pr.err != nil {
\t\t\tlog.Printf("[external-plugin:%s] GetPanelData: %v", epName, pr.err)
\t\t\tcontinue
\t\t}
\t\tresults[epName] = pr.data
\t}
\twriteJSON(w, http.StatusOK, results)
}

func entityKindInPanels(kind string, panels []string) bool {
\tfor _, p := range panels {
\t\tif p == kind {
\t\t\treturn true
\t\t}
\t}
\treturn false
}
'''
    write("internal/api/handlers/plugins.go", plugins_content)
    print("  OK: GetEntityPanelData + entityKindInPanels appended")
else:
    print("  SKIP (already applied): GetEntityPanelData")

# ---------------------------------------------------------------------------
# Part 9 — internal/api/server.go: router field, routes, MountPluginProxy
# ---------------------------------------------------------------------------

section("Part 9 — internal/api/server.go: router, routes, MountPluginProxy")

# 9a: change handler field to router
patch(
    "internal/api/server.go",
    "\tconfig   *config.Config\n\thandler  http.Handler\n\tport     int",
    "\tconfig   *config.Config\n\trouter   chi.Router\n\tport     int",
    "server.go: Server.handler -> router",
)

# 9b: internal plugin routes after healthz/readyz
patch(
    "internal/api/server.go",
    "\tr.Get(\"/healthz\", h.Healthz)\n\tr.Get(\"/readyz\", h.Readyz)\n\n\t// Prometheus metrics",
    "\tr.Get(\"/healthz\", h.Healthz)\n"
    "\tr.Get(\"/readyz\", h.Readyz)\n\n"
    "\t// Internal plugin API — authenticated via X-Gantry-Internal-Token header, not JWT.\n"
    "\tr.Get(\"/api/internal/entity\", h.InternalGetEntity)\n"
    "\tr.Post(\"/api/internal/entity-upsert\", h.InternalUpsertEntity)\n"
    "\tr.Post(\"/api/internal/entity-delete\", h.InternalDeleteEntity)\n\n"
    "\t// Prometheus metrics",
    "server.go: internal plugin routes",
)

# 9c: panels route before /entities/{kind}/{name}
patch(
    "internal/api/server.go",
    "\t\t\tprotected.Get(\"/entities/{kind}/{name}/documentation\", h.ListEntityDocumentation)\n"
    "\t\t\tprotected.Get(\"/entities/{kind}/{name}\", h.GetEntity)",
    "\t\t\tprotected.Get(\"/entities/{kind}/{name}/documentation\", h.ListEntityDocumentation)\n"
    "\t\t\tprotected.Get(\"/entities/{kind}/{name}/panels\", h.GetEntityPanelData)\n"
    "\t\t\tprotected.Get(\"/entities/{kind}/{name}\", h.GetEntity)",
    "server.go: panels route",
)

# 9d: update return statement to use router
patch(
    "internal/api/server.go",
    "\treturn &Server{\n\t\tconfig:   cfg,\n\t\thandler:  r,\n\t\tport:     cfg.Port,\n\t\tHandlers: h,\n\t}",
    "\treturn &Server{\n\t\tconfig:   cfg,\n\t\trouter:   r,\n\t\tport:     cfg.Port,\n\t\tHandlers: h,\n\t}",
    "server.go: return router",
)

# 9e: MountPluginProxy method + update Router() and Start() to use s.router
patch(
    "internal/api/server.go",
    "// Router returns the HTTP handler for use with a custom http.Server.\nfunc (s *Server) Router() http.Handler {\n\treturn s.handler\n}\n\n// Start begins listening for HTTP requests on the configured port.\nfunc (s *Server) Start() error {\n\taddr := fmt.Sprintf(\":%d\", s.port)\n\treturn http.ListenAndServe(addr, s.handler)\n}",
    "// MountPluginProxy registers a reverse-proxy handler at the given path prefix,\n"
    "// forwarding requests to upstream (e.g. \"http://127.0.0.1:54321\").\n"
    "// Called after NewServer so external plugins can declare routes dynamically.\n"
    "func (s *Server) MountPluginProxy(pathPrefix, upstream string) {\n"
    "\tproxy := newPluginReverseProxy(upstream, pathPrefix)\n"
    "\ts.router.Handle(pathPrefix, proxy)\n"
    "\ts.router.Handle(pathPrefix+\"/*\", proxy)\n"
    "}\n\n"
    "// Router returns the HTTP handler for use with a custom http.Server.\n"
    "func (s *Server) Router() http.Handler {\n"
    "\treturn s.router\n"
    "}\n\n"
    "// Start begins listening for HTTP requests on the configured port.\n"
    "func (s *Server) Start() error {\n"
    "\taddr := fmt.Sprintf(\":%d\", s.port)\n"
    "\treturn http.ListenAndServe(addr, s.router)\n"
    "}",
    "server.go: MountPluginProxy + router refs",
)

# ---------------------------------------------------------------------------
# Part 10 — cmd/gantry/serve.go: plugin-dir flag, token, manager, wiring
# ---------------------------------------------------------------------------

section("Part 10 — cmd/gantry/serve.go: plugin-dir, token, manager")

# 10a: add imports
patch(
    "cmd/gantry/serve.go",
    "import (\n\t\"context\"\n\t\"fmt\"",
    "import (\n\t\"context\"\n\t\"crypto/hmac\"\n\t\"crypto/sha256\"\n\t\"encoding/hex\"\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"log\"",
    "serve.go: crypto/hmac, sha256, hex, json imports",
)

patch(
    "cmd/gantry/serve.go",
    "\t\"github.com/go2engle/gantry/internal/plugins\"\n"
    "\t\"github.com/go2engle/gantry/internal/search\"",
    "\t\"github.com/go2engle/gantry/internal/plugins\"\n"
    "\t\"github.com/go2engle/gantry/internal/plugins/external\"\n"
    "\t\"github.com/go2engle/gantry/internal/search\"",
    "serve.go: external import",
)

# 10b: --plugin-dir flag
patch(
    "cmd/gantry/serve.go",
    "\tcmd.Flags().String(\"tls-cert\", \"\", \"Path to TLS certificate file (enables HTTPS)\")\n"
    "\tcmd.Flags().String(\"tls-key\", \"\", \"Path to TLS private key file (enables HTTPS)\")\n\n\treturn cmd",
    "\tcmd.Flags().String(\"tls-cert\", \"\", \"Path to TLS certificate file (enables HTTPS)\")\n"
    "\tcmd.Flags().String(\"tls-key\", \"\", \"Path to TLS private key file (enables HTTPS)\")\n"
    "\tcmd.Flags().String(\"plugin-dir\", \"\", \"Directory to scan for external plugin binaries (gantry-plugin-*)\")\n\n\treturn cmd",
    "serve.go: --plugin-dir flag",
)

# 10c: apply --plugin-dir flag to config
patch(
    "cmd/gantry/serve.go",
    "\ttlsCert, _ := cmd.Flags().GetString(\"tls-cert\")",
    "\tif pluginDir, _ := cmd.Flags().GetString(\"plugin-dir\"); pluginDir != \"\" {\n"
    "\t\tcfg.PluginDir = pluginDir\n"
    "\t}\n\n"
    "\ttlsCert, _ := cmd.Flags().GetString(\"tls-cert\")",
    "serve.go: apply plugin-dir flag to cfg",
)

# 10d: derive internalPluginToken after authService
patch(
    "cmd/gantry/serve.go",
    "\tauthService := auth.NewService(cfg.JWTSecret)\n\n\tdatabase",
    "\tauthService := auth.NewService(cfg.JWTSecret)\n\n"
    "\t// Derive a stable internal token for plugin->Gantry API calls.\n"
    "\t// HMAC-SHA256(jwtSecret, \"gantry-plugin-internal\") — changes only if JWTSecret rotates.\n"
    "\tmac := hmac.New(sha256.New, []byte(cfg.JWTSecret))\n"
    "\tmac.Write([]byte(\"gantry-plugin-internal\"))\n"
    "\tinternalPluginToken := hex.EncodeToString(mac.Sum(nil))\n\n"
    "\tdatabase",
    "serve.go: internalPluginToken derivation",
)

# 10e: create external manager + StartAll + DB registration, after bundled plugin registration
patch(
    "cmd/gantry/serve.go",
    "\tif err := database.EnsureBundledPlugins(context.Background(), registry); err != nil {\n"
    "\t\treturn fmt.Errorf(\"registering bundled plugins: %w\", err)\n"
    "\t}\n\n\t// Create core services.",
    "\tif err := database.EnsureBundledPlugins(context.Background(), registry); err != nil {\n"
    "\t\treturn fmt.Errorf(\"registering bundled plugins: %w\", err)\n"
    "\t}\n\n"
    "\t// Create and start the external plugin manager.\n"
    "\tgantryURL := fmt.Sprintf(\"http://127.0.0.1:%d\", cfg.Port)\n"
    "\textManager := external.NewManager(func(ctx context.Context, name string) (map[string]string, error) {\n"
    "\t\tp, err := database.GetPlugin(ctx, name)\n"
    "\t\tif err != nil || p == nil {\n"
    "\t\t\treturn nil, err\n"
    "\t\t}\n"
    "\t\tout := make(map[string]string, len(p.Config)+2)\n"
    "\t\tfor k, v := range p.Config {\n"
    "\t\t\tif s, ok := v.(string); ok {\n"
    "\t\t\t\tout[k] = s\n"
    "\t\t\t}\n"
    "\t\t}\n"
    "\t\tout[\"gantryInternalToken\"] = internalPluginToken\n"
    "\t\tout[\"gantryUrl\"] = gantryURL\n"
    "\t\treturn out, nil\n"
    "\t})\n"
    "\tif cfg.PluginDir != \"\" {\n"
    "\t\tif err := extManager.StartAll(context.Background(), cfg.PluginDir); err != nil {\n"
    "\t\t\tlog.Printf(\"[external-plugins] warning: %v\", err)\n"
    "\t\t}\n"
    "\t\tfor _, ep := range extManager.All() {\n"
    "\t\t\tm := ep.GetManifest()\n"
    "\t\t\tif m == nil {\n"
    "\t\t\t\tcontinue\n"
    "\t\t\t}\n"
    "\t\t\tdbManifest := &plugins.Manifest{\n"
    "\t\t\t\tName:         m.Name,\n"
    "\t\t\t\tTitle:        m.Title,\n"
    "\t\t\t\tDescription:  m.Description,\n"
    "\t\t\t\tVersion:      m.Version,\n"
    "\t\t\t\tAuthor:       m.Author,\n"
    "\t\t\t\tCategory:     m.Category,\n"
    "\t\t\t\tIconURL:      m.IconURL,\n"
    "\t\t\t\tHomepage:     m.Homepage,\n"
    "\t\t\t\tEntityPanels: m.EntityPanels,\n"
    "\t\t\t\tActionTypes:  m.ActionTypes,\n"
    "\t\t\t}\n"
    "\t\t\tif m.ConfigSchemaJSON != \"\" {\n"
    "\t\t\t\t_ = json.Unmarshal([]byte(m.ConfigSchemaJSON), &dbManifest.ConfigSchema)\n"
    "\t\t\t}\n"
    "\t\t\tfor _, r := range m.Requirements {\n"
    "\t\t\t\tdbManifest.Requirements = append(dbManifest.Requirements, plugins.PluginRequirement{\n"
    "\t\t\t\t\tName:        r.Name,\n"
    "\t\t\t\t\tDescription: r.Description,\n"
    "\t\t\t\t\tOptional:    r.Optional,\n"
    "\t\t\t\t})\n"
    "\t\t\t}\n"
    "\t\t\tif err := database.EnsureExternalPlugin(context.Background(), dbManifest); err != nil {\n"
    "\t\t\t\tlog.Printf(\"[external-plugins] %s: failed to register in DB: %v\", m.Name, err)\n"
    "\t\t\t}\n"
    "\t\t}\n"
    "\t}\n\n"
    "\t// Create core services.",
    "serve.go: external manager creation and startup",
)

# 10f: inject manager + HTTP proxy routes into server, after NewServer call
patch(
    "cmd/gantry/serve.go",
    "\tsrv := api.NewServer(cfg, database, authService, eventBus, validator, searchService, wsHub, Version)\n\n"
    "\t// Initialize GitOps service if the plugin is installed and enabled.\n"
    "\tsrv.Handlers.InitGitOps()",
    "\tsrv := api.NewServer(cfg, database, authService, eventBus, validator, searchService, wsHub, Version)\n\n"
    "\t// Inject external plugin manager and internal token into handlers.\n"
    "\tsrv.Handlers.ExternalManager = extManager\n"
    "\tsrv.Handlers.InternalPluginToken = internalPluginToken\n"
    "\tsrv.Handlers.GantryURL = gantryURL\n\n"
    "\t// Mount reverse-proxy routes for HTTP-capable external plugins.\n"
    "\tif cfg.PluginDir != \"\" {\n"
    "\t\tfor _, ep := range extManager.All() {\n"
    "\t\t\tm := ep.GetManifest()\n"
    "\t\t\tif m == nil || !m.SupportsHTTP {\n"
    "\t\t\t\tcontinue\n"
    "\t\t\t}\n"
    "\t\t\taddr := ep.GetListenAddr()\n"
    "\t\t\tif addr == \"\" {\n"
    "\t\t\t\tcontinue\n"
    "\t\t\t}\n"
    "\t\t\tvar routes []external.Route\n"
    "\t\t\tif m.HTTPRoutesJSON != \"\" {\n"
    "\t\t\t\t_ = json.Unmarshal([]byte(m.HTTPRoutesJSON), &routes)\n"
    "\t\t\t}\n"
    "\t\t\tfor _, route := range routes {\n"
    "\t\t\t\tsrv.MountPluginProxy(route.Path, \"http://\"+addr)\n"
    "\t\t\t\tlog.Printf(\"[external-plugins] %s: proxying %s -> http://%s\", m.Name, route.Path, addr)\n"
    "\t\t\t}\n"
    "\t\t}\n"
    "\t}\n\n"
    "\t// Initialize GitOps service if the plugin is installed and enabled.\n"
    "\tsrv.Handlers.InitGitOps()",
    "serve.go: inject manager fields + HTTP proxy routes",
)

# 10g: StopAll in shutdown block
patch(
    "cmd/gantry/serve.go",
    "\t\tctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)\n\t\tdefer cancel()",
    "\t\textManager.StopAll()\n\n"
    "\t\tctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)\n\t\tdefer cancel()",
    "serve.go: extManager.StopAll() on shutdown",
)

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------

section("Running go mod tidy")
ret = os.system("go mod tidy 2>&1")
if ret != 0:
    print("  WARNING: go mod tidy exited with non-zero status — check output above")
else:
    print("  OK: go mod tidy completed")

section("Verifying build")
os.makedirs("web/dist", exist_ok=True)
if not os.listdir("web/dist"):
    with open("web/dist/.keep", "w") as f:
        f.write("")  # satisfy embed.go; real dist is built by npm
ret = os.system("go build -buildvcs=false ./... 2>&1")
if ret != 0:
    print("\n  ERROR: go build ./... failed — review errors above")
    sys.exit(1)
else:
    print("  OK: go build ./... passed")

print("""
============================================================
  Done! External plugin system applied successfully.

  Next steps:
    cd web && npx tsc --noEmit   # verify frontend types
    go test ./internal/plugins/external/... -v

  To use:
    bin/gantry serve --plugin-dir /path/to/plugins/
============================================================
""")

