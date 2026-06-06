package external

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// ExternalPlugin wraps a hashicorp/go-plugin client for a single external plugin binary.
type ExternalPlugin struct {
	Name    string
	BinPath string

	mu        sync.RWMutex
	client    *plugin.Client
	rpc       GantryPluginRPC
	manifest  *Manifest
	available bool
}

// Start launches the plugin subprocess and calls GetManifest + Configure.
func (ep *ExternalPlugin) Start(config map[string]string) error {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   fmt.Sprintf("plugin.%s", ep.Name),
		Level:  hclog.Info,
		Output: hclog.DefaultOutput,
	})

	c := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins:         PluginMap,
		Cmd:             exec.Command(ep.BinPath),
		Logger:          logger,
		Managed:         true,
	})

	rpcClient, err := c.Client()
	if err != nil {
		c.Kill()
		return fmt.Errorf("plugin %s: connect: %w", ep.Name, err)
	}

	raw, err := rpcClient.Dispense("Plugin")
	if err != nil {
		c.Kill()
		return fmt.Errorf("plugin %s: dispense: %w", ep.Name, err)
	}

	rpc, ok := raw.(GantryPluginRPC)
	if !ok {
		c.Kill()
		return fmt.Errorf("plugin %s: unexpected RPC type %T", ep.Name, raw)
	}

	m, err := rpc.GetManifest()
	if err != nil {
		c.Kill()
		return fmt.Errorf("plugin %s: GetManifest: %w", ep.Name, err)
	}

	if err := rpc.Configure(config); err != nil {
		c.Kill()
		return fmt.Errorf("plugin %s: Configure: %w", ep.Name, err)
	}

	ep.mu.Lock()
	ep.client = c
	ep.rpc = rpc
	ep.manifest = &m
	ep.available = true
	ep.mu.Unlock()

	log.Printf("[external-plugin] %s started (version %s)", ep.Name, m.Version)
	return nil
}

// Kill terminates the plugin subprocess.
func (ep *ExternalPlugin) Kill() {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	if ep.client != nil {
		ep.client.Kill()
		ep.client = nil
		ep.rpc = nil
	}
	ep.available = false
}

// Available reports whether the plugin subprocess is running and ready.
func (ep *ExternalPlugin) Available() bool {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.available
}

// GetManifest returns the cached manifest obtained at startup.
func (ep *ExternalPlugin) GetManifest() *Manifest {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.manifest
}

// GetListenAddr calls the plugin subprocess for its embedded HTTP server address.
func (ep *ExternalPlugin) GetListenAddr() string {
	ep.mu.RLock()
	rpc := ep.rpc
	ep.mu.RUnlock()
	if rpc == nil {
		return ""
	}
	addr, _ := rpc.GetListenAddr()
	return addr
}

// Sync calls the plugin's Sync method.
func (ep *ExternalPlugin) Sync() (SyncResult, error) {
	ep.mu.RLock()
	rpc := ep.rpc
	ep.mu.RUnlock()
	if rpc == nil {
		return SyncResult{}, fmt.Errorf("plugin %s is not available", ep.Name)
	}
	return rpc.Sync()
}

// GetPanelData calls the plugin's GetPanelData method.
func (ep *ExternalPlugin) GetPanelData(args PanelArgs) (json.RawMessage, error) {
	ep.mu.RLock()
	rpc := ep.rpc
	ep.mu.RUnlock()
	if rpc == nil {
		return nil, fmt.Errorf("plugin %s is not available", ep.Name)
	}
	return rpc.GetPanelData(args)
}

// ExecuteAction calls the plugin's ExecuteAction method.
func (ep *ExternalPlugin) ExecuteAction(args ActionArgs) (ActionResult, error) {
	ep.mu.RLock()
	rpc := ep.rpc
	ep.mu.RUnlock()
	if rpc == nil {
		return ActionResult{}, fmt.Errorf("plugin %s is not available", ep.Name)
	}
	return rpc.ExecuteAction(args)
}

// Configure (re-)sends configuration to a running plugin subprocess.
func (ep *ExternalPlugin) Configure(config map[string]string) error {
	ep.mu.RLock()
	rpc := ep.rpc
	ep.mu.RUnlock()
	if rpc == nil {
		return fmt.Errorf("plugin %s is not available", ep.Name)
	}
	return rpc.Configure(config)
}

func (ep *ExternalPlugin) isDead() bool {
	ep.mu.RLock()
	c := ep.client
	ep.mu.RUnlock()
	if c == nil {
		return true
	}
	return c.Exited()
}

func (ep *ExternalPlugin) markUnavailable() {
	ep.mu.Lock()
	ep.available = false
	ep.mu.Unlock()
}
