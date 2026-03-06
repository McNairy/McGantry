import { useState, useEffect } from 'react';
import { Package, CheckCircle, Circle, Settings, ExternalLink, Search, Puzzle, RefreshCw } from 'lucide-react';
import { api } from '../lib/api';
import type { PluginRegistryEntry, PluginConfig, PluginSyncResult } from '../lib/types';

// Plugins that expose a server-side sync operation.
const SYNCABLE_PLUGINS = new Set(['kubernetes']);

const CATEGORIES = [
  { id: 'all', label: 'All' },
  { id: 'integration', label: 'Integrations' },
  { id: 'widget', label: 'Widgets' },
  { id: 'action-type', label: 'Action Types' },
  { id: 'entity-kind', label: 'Entity Kinds' },
  { id: 'auth-provider', label: 'Auth Providers' },
];

const CATEGORY_COLORS: Record<string, string> = {
  integration: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  widget: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  'action-type': 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  'entity-kind': 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  'auth-provider': 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
};

// ConfigModal — shown when user clicks Configure on an installed plugin.
function ConfigModal({
  plugin,
  onClose,
}: {
  plugin: PluginRegistryEntry;
  onClose: () => void;
}) {
  const [config, setConfig] = useState<PluginConfig | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.getPluginConfig(plugin.name).then((c) => {
      setConfig(c);
      setValues((c.values as Record<string, string>) ?? {});
    }).catch(() => {});
  }, [plugin.name]);

  async function handleSave() {
    setSaving(true);
    setError(null);
    try {
      await api.updatePluginConfig(plugin.name, values);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  }

  // Extract field definitions from the JSON Schema.
  const fields = config?.schema?.properties
    ? Object.entries(config.schema.properties as Record<string, any>)
    : [];

  const required: string[] = config?.schema?.required ?? [];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-[var(--gantry-surface)] rounded-xl shadow-2xl w-full max-w-lg mx-4 overflow-hidden">
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--gantry-border)]">
          <div className="flex items-center gap-2">
            <Settings size={18} className="text-[var(--gantry-accent)]" />
            <h2 className="font-semibold text-[var(--gantry-text)]">Configure {plugin.title}</h2>
          </div>
          <button
            onClick={onClose}
            className="text-[var(--gantry-text-muted)] hover:text-[var(--gantry-text)] transition-colors"
          >
            ✕
          </button>
        </div>

        <div className="px-6 py-5 space-y-4">
          {fields.length === 0 && (
            <p className="text-sm text-[var(--gantry-text-muted)]">This plugin has no configuration options.</p>
          )}
          {fields.map(([key, schema]) => (
            <div key={key}>
              <label className="block text-sm font-medium text-[var(--gantry-text)] mb-1">
                {schema.title ?? key}
                {required.includes(key) && <span className="text-red-500 ml-1">*</span>}
              </label>
              {schema.description && (
                <p className="text-xs text-[var(--gantry-text-muted)] mb-1">{schema.description}</p>
              )}
              <input
                type={key.toLowerCase().includes('key') || key.toLowerCase().includes('token') || key.toLowerCase().includes('secret') ? 'password' : 'text'}
                className="w-full rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg)] px-3 py-2 text-sm text-[var(--gantry-text)] focus:outline-none focus:ring-2 focus:ring-[var(--gantry-accent)]"
                value={values[key] ?? ''}
                placeholder={schema.default ?? ''}
                onChange={(e) => setValues((v) => ({ ...v, [key]: e.target.value }))}
              />
            </div>
          ))}

          {error && <p className="text-sm text-red-500">{error}</p>}
        </div>

        <div className="flex justify-end gap-3 px-6 py-4 border-t border-[var(--gantry-border)]">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm rounded-lg border border-[var(--gantry-border)] text-[var(--gantry-text)] hover:bg-[var(--gantry-hover)] transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={saving || fields.length === 0}
            className="px-4 py-2 text-sm rounded-lg bg-[var(--gantry-accent)] text-[var(--gantry-bg-primary)] hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {saved ? 'Saved!' : saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
}

// PluginCard — single card in the grid.
function PluginCard({
  plugin,
  syncing,
  syncResult,
  onAction,
  onSync,
}: {
  plugin: PluginRegistryEntry;
  syncing: boolean;
  syncResult: PluginSyncResult | null;
  onAction: (action: 'install' | 'enable' | 'disable' | 'configure') => void;
  onSync: () => void;
}) {
  const categoryColor = CATEGORY_COLORS[plugin.category] ?? 'bg-gray-100 text-gray-700';
  const canSync = plugin.installed && plugin.enabled && SYNCABLE_PLUGINS.has(plugin.name);

  return (
    <div className="bg-[var(--gantry-surface)] rounded-xl border border-[var(--gantry-border)] p-5 flex flex-col gap-4 hover:border-[var(--gantry-accent)] transition-colors">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-[var(--gantry-accent)]/10 flex items-center justify-center flex-shrink-0">
            <Puzzle size={20} className="text-[var(--gantry-accent)]" />
          </div>
          <div>
            <h3 className="font-semibold text-[var(--gantry-text)] text-sm leading-tight">{plugin.title}</h3>
            <p className="text-xs text-[var(--gantry-text-muted)]">by {plugin.author} · v{plugin.version}</p>
          </div>
        </div>
        {plugin.installed && (
          <span className="flex-shrink-0 flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded-full bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
            <CheckCircle size={10} />
            {plugin.enabled ? 'Active' : 'Installed'}
          </span>
        )}
      </div>

      <p className="text-sm text-[var(--gantry-text-muted)] line-clamp-2 flex-1">{plugin.description}</p>

      <div className="flex items-center gap-2 flex-wrap">
        <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${categoryColor}`}>
          {CATEGORIES.find((c) => c.id === plugin.category)?.label ?? plugin.category}
        </span>
        {plugin.homepage && (
          <a
            href={plugin.homepage}
            target="_blank"
            rel="noopener noreferrer"
            className="ml-auto text-xs text-[var(--gantry-text-muted)] hover:text-[var(--gantry-accent)] flex items-center gap-1 transition-colors"
          >
            <ExternalLink size={10} />
            Docs
          </a>
        )}
      </div>

      {/* Sync result summary */}
      {syncResult && (
        <div className="text-xs rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 px-3 py-2 text-green-700 dark:text-green-400">
          Synced: {syncResult.created} created, {syncResult.updated} updated
          {(syncResult.errors?.length ?? 0) > 0 && (
            <span className="ml-2 text-amber-600 dark:text-amber-400">· {syncResult.errors!.length} error(s)</span>
          )}
        </div>
      )}

      <div className="flex items-center gap-2 pt-1 border-t border-[var(--gantry-border)]">
        {!plugin.installed ? (
          <button
            onClick={() => onAction('install')}
            className="flex-1 py-1.5 text-xs font-semibold rounded-lg bg-[var(--gantry-accent)] text-[var(--gantry-bg-primary)] hover:opacity-90 transition-opacity"
          >
            Install
          </button>
        ) : (
          <>
            <button
              onClick={() => onAction(plugin.enabled ? 'disable' : 'enable')}
              className={`flex-1 py-1.5 text-xs font-medium rounded-lg border transition-colors flex items-center justify-center gap-1 ${
                plugin.enabled
                  ? 'border-[var(--gantry-border)] text-[var(--gantry-text)] hover:bg-[var(--gantry-hover)]'
                  : 'border-[var(--gantry-accent)] text-[var(--gantry-accent)] hover:bg-[var(--gantry-accent)]/10'
              }`}
            >
              {plugin.enabled ? (
                <><Circle size={10} /> Disable</>
              ) : (
                <><CheckCircle size={10} /> Enable</>
              )}
            </button>
            {canSync && (
              <button
                onClick={onSync}
                disabled={syncing}
                title="Sync now — discover resources from your cluster"
                className="py-1.5 px-3 text-xs font-medium rounded-lg border border-[var(--gantry-border)] text-[var(--gantry-text)] hover:bg-[var(--gantry-hover)] transition-colors disabled:opacity-50"
              >
                <RefreshCw size={12} className={syncing ? 'animate-spin' : ''} />
              </button>
            )}
            <button
              onClick={() => onAction('configure')}
              className="py-1.5 px-3 text-xs font-medium rounded-lg border border-[var(--gantry-border)] text-[var(--gantry-text)] hover:bg-[var(--gantry-hover)] transition-colors"
            >
              <Settings size={12} />
            </button>
          </>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main Plugins page
// ---------------------------------------------------------------------------
export default function Plugins() {
  const [plugins, setPlugins] = useState<PluginRegistryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState('all');
  const [tab, setTab] = useState<'browse' | 'installed'>('browse');
  const [configPlugin, setConfigPlugin] = useState<PluginRegistryEntry | null>(null);
  const [syncingPlugins, setSyncingPlugins] = useState<Set<string>>(new Set());
  const [syncResults, setSyncResults] = useState<Record<string, PluginSyncResult>>({});

  useEffect(() => {
    api.listPlugins()
      .then(setPlugins)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  async function handleAction(plugin: PluginRegistryEntry, action: 'install' | 'enable' | 'disable' | 'configure') {
    if (action === 'configure') {
      setConfigPlugin(plugin);
      return;
    }

    try {
      if (action === 'install') {
        await api.installPlugin(plugin.name);
        setPlugins((prev) =>
          prev.map((p) => (p.name === plugin.name ? { ...p, installed: true } : p))
        );
      } else if (action === 'enable') {
        await api.enablePlugin(plugin.name, true);
        setPlugins((prev) =>
          prev.map((p) => (p.name === plugin.name ? { ...p, enabled: true } : p))
        );
      } else if (action === 'disable') {
        await api.enablePlugin(plugin.name, false);
        setPlugins((prev) =>
          prev.map((p) => (p.name === plugin.name ? { ...p, enabled: false } : p))
        );
      }
    } catch (e: any) {
      setError(e.message);
    }
  }

  async function handleSync(pluginName: string) {
    setSyncingPlugins((s) => new Set(s).add(pluginName));
    setSyncResults((r) => { const n = { ...r }; delete n[pluginName]; return n; });
    try {
      const result = await api.syncPlugin(pluginName);
      setSyncResults((r) => ({ ...r, [pluginName]: result }));
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSyncingPlugins((s) => { const n = new Set(s); n.delete(pluginName); return n; });
    }
  }

  const filtered = plugins.filter((p) => {
    const matchSearch =
      !search ||
      p.title.toLowerCase().includes(search.toLowerCase()) ||
      p.description.toLowerCase().includes(search.toLowerCase());
    const matchCategory = category === 'all' || p.category === category;
    const matchTab = tab === 'browse' || p.installed;
    return matchSearch && matchCategory && matchTab;
  });

  const installedCount = plugins.filter((p) => p.installed).length;
  const enabledCount = plugins.filter((p) => p.enabled).length;

  return (
    <div className="flex-1 flex flex-col overflow-auto bg-[var(--gantry-bg)]">
      {/* Header */}
      <div className="px-8 pt-8 pb-4">
        <div className="flex items-center justify-between mb-1">
          <h1 className="text-2xl font-bold text-[var(--gantry-text)]">Plugins</h1>
          <div className="flex items-center gap-4 text-sm text-[var(--gantry-text-muted)]">
            <span>{installedCount} installed</span>
            <span>{enabledCount} enabled</span>
          </div>
        </div>
        <p className="text-sm text-[var(--gantry-text-muted)]">
          Browse and manage plugins to extend Gantry with integrations, widgets, and more.
        </p>
      </div>

      {/* Tabs */}
      <div className="px-8 border-b border-[var(--gantry-border)]">
        <div className="flex gap-6">
          {(['browse', 'installed'] as const).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`pb-3 text-sm font-medium border-b-2 transition-colors ${
                tab === t
                  ? 'border-[var(--gantry-accent)] text-[var(--gantry-accent)]'
                  : 'border-transparent text-[var(--gantry-text-muted)] hover:text-[var(--gantry-text)]'
              }`}
            >
              {t === 'browse' ? 'Browse' : `Installed${installedCount > 0 ? ` (${installedCount})` : ''}`}
            </button>
          ))}
        </div>
      </div>

      {/* Filters */}
      <div className="px-8 py-4 flex items-center gap-3 flex-wrap">
        <div className="relative flex-shrink-0">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--gantry-text-muted)]" />
          <input
            type="text"
            placeholder="Search plugins…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-8 pr-4 py-2 text-sm rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-surface)] text-[var(--gantry-text)] focus:outline-none focus:ring-2 focus:ring-[var(--gantry-accent)] w-52"
          />
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          {CATEGORIES.map((cat) => (
            <button
              key={cat.id}
              onClick={() => setCategory(cat.id)}
              className={`px-3 py-1 text-xs rounded-full font-medium transition-colors ${
                category === cat.id
                  ? 'bg-[var(--gantry-accent)] text-[var(--gantry-bg-primary)]'
                  : 'bg-[var(--gantry-surface)] border border-[var(--gantry-border)] text-[var(--gantry-text-muted)] hover:text-[var(--gantry-text)]'
              }`}
            >
              {cat.label}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 px-8 pb-8">
        {loading ? (
          <div className="flex items-center justify-center py-24 text-[var(--gantry-text-muted)]">
            <Package size={24} className="animate-pulse mr-3" />
            Loading plugins…
          </div>
        ) : error ? (
          <div className="flex items-center justify-center py-24">
            <p className="text-sm text-red-500">{error}</p>
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 gap-3 text-[var(--gantry-text-muted)]">
            <Package size={32} />
            <p className="text-sm">No plugins found.</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {filtered.map((plugin) => (
              <PluginCard
                key={plugin.name}
                plugin={plugin}
                syncing={syncingPlugins.has(plugin.name)}
                syncResult={syncResults[plugin.name] ?? null}
                onAction={(action) => handleAction(plugin, action)}
                onSync={() => handleSync(plugin.name)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Config modal */}
      {configPlugin && (
        <ConfigModal plugin={configPlugin} onClose={() => setConfigPlugin(null)} />
      )}
    </div>
  );
}
