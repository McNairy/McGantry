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

const pluginStartTimeout = 30 * time.Second
const maxConsecutiveFails = 5

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

		startCtx, cancel := context.WithTimeout(ctx, pluginStartTimeout)
		if err := ep.Start(startCtx, config); err != nil {
			log.Printf("[external-plugin] %s: failed to start: %v", name, err)
		}
		cancel()

		m.mu.Lock()
		m.plugins[name] = ep
		m.mu.Unlock()
	}

	hctx, cancel := context.WithCancel(ctx)
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

func (m *Manager) checkHealth(hctx context.Context) {
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

		ep.mu.RLock()
		fails, lastFail := ep.consecutiveFails, ep.lastFailAt
		ep.mu.RUnlock()

		if fails >= maxConsecutiveFails {
			// Exponential backoff capped at 10× the health check interval.
			shift := fails - maxConsecutiveFails
			if shift > 4 {
				shift = 4
			}
			backoff := healthCheckInterval * (1 << uint(shift))
			if time.Since(lastFail) < backoff {
				continue
			}
		}

		ep := ep
		go func() {
			log.Printf("[external-plugin] %s: subprocess exited, attempting restart", ep.Name)
			ep.Kill()

			config, err := m.loadConfig(hctx, ep.Name)
			if err != nil {
				log.Printf("[external-plugin] %s: restart aborted: %v", ep.Name, err)
				ep.markUnavailable()
				ep.markFailed()
				return
			}

			startCtx, cancel := context.WithTimeout(hctx, pluginStartTimeout)
			defer cancel()
			if err := ep.Start(startCtx, config); err != nil {
				log.Printf("[external-plugin] %s: restart failed: %v", ep.Name, err)
				ep.markUnavailable()
				ep.markFailed()
			} else {
				log.Printf("[external-plugin] %s: restarted successfully", ep.Name)
				ep.markRestartSucceeded()
			}
		}()
	}
}

func (m *Manager) loadConfig(ctx context.Context, name string) (map[string]string, error) {
	if m.configLoad == nil {
		return nil, nil
	}
	return m.configLoad(ctx, name)
}
