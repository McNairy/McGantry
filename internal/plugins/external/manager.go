package external

import (
	"context"
	"log"
	"sync"
	"time"
)

const healthCheckInterval = 30 * time.Second

// ConfigLoader returns the stored config for a plugin by name.
type ConfigLoader func(ctx context.Context, name string) (map[string]string, error)

// Manager holds all running external plugin subprocesses.
type Manager struct {
	mu         sync.RWMutex
	plugins    map[string]*ExternalPlugin
	configLoad ConfigLoader
	cancel     context.CancelFunc
}

// NewManager creates a Manager. configLoad is called at startup and on restart.
func NewManager(configLoad ConfigLoader) *Manager {
	return &Manager{
		plugins:    make(map[string]*ExternalPlugin),
		configLoad: configLoad,
	}
}

// StartAll scans pluginDir, launches each discovered binary, and registers it.
func (m *Manager) StartAll(ctx context.Context, pluginDir string) error {
	paths, err := ScanDir(pluginDir)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}

	for _, path := range paths {
		name := PluginNameFromPath(path)
		ep := &ExternalPlugin{Name: name, BinPath: path}

		config, err := m.loadConfig(ctx, name)
		if err != nil {
			log.Printf("[external-plugin] %s: failed to load config: %v", name, err)
			config = nil
		}

		if err := ep.Start(config); err != nil {
			log.Printf("[external-plugin] %s: failed to start: %v", name, err)
		}

		m.mu.Lock()
		m.plugins[name] = ep
		m.mu.Unlock()
	}

	hctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	go m.healthLoop(hctx)

	return nil
}

// StopAll kills all plugin subprocesses and stops the health check loop.
func (m *Manager) StopAll() {
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ep := range m.plugins {
		ep.Kill()
	}
}

// Get returns the ExternalPlugin for the given name, or nil if not found.
func (m *Manager) Get(name string) *ExternalPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.plugins[name]
}

// All returns a snapshot of all registered external plugins.
func (m *Manager) All() []*ExternalPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*ExternalPlugin, 0, len(m.plugins))
	for _, ep := range m.plugins {
		out = append(out, ep)
	}
	return out
}

func (m *Manager) healthLoop(ctx context.Context) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkHealth(ctx)
		}
	}
}

func (m *Manager) checkHealth(ctx context.Context) {
	m.mu.RLock()
	eps := make([]*ExternalPlugin, 0, len(m.plugins))
	for _, ep := range m.plugins {
		eps = append(eps, ep)
	}
	m.mu.RUnlock()

	for _, ep := range eps {
		if !ep.isDead() {
			continue
		}
		log.Printf("[external-plugin] %s: subprocess exited, attempting restart", ep.Name)
		ep.Kill()

		config, err := m.loadConfig(ctx, ep.Name)
		if err != nil {
			log.Printf("[external-plugin] %s: restart aborted: %v", ep.Name, err)
			ep.markUnavailable()
			continue
		}

		if err := ep.Start(config); err != nil {
			log.Printf("[external-plugin] %s: restart failed: %v", ep.Name, err)
			ep.markUnavailable()
		} else {
			log.Printf("[external-plugin] %s: restarted successfully", ep.Name)
		}
	}
}

func (m *Manager) loadConfig(ctx context.Context, name string) (map[string]string, error) {
	if m.configLoad == nil {
		return nil, nil
	}
	return m.configLoad(ctx, name)
}
