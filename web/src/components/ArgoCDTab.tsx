import { useState, useEffect, useCallback } from 'react';
import {
  RefreshCw, GitBranch, GitCommit, Server, Package,
  CheckCircle2, XCircle, Clock, AlertTriangle, HelpCircle,
  Pause, ExternalLink, RotateCcw, ChevronDown, ChevronRight,
} from 'lucide-react';
import { api } from '../lib/api';
import { useAuth } from '../hooks/useAuth';
import type { Entity, ArgoCDAppWithInstance, ArgoCDResourceStatus } from '../lib/types';

// ---------------------------------------------------------------------------
// Status helpers
// ---------------------------------------------------------------------------

const SYNC_STYLES: Record<string, string> = {
  Synced:    'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  OutOfSync: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
  Unknown:   'bg-[var(--gantry-bg-tertiary)] text-[var(--gantry-text-secondary)]',
};

const HEALTH_STYLES: Record<string, string> = {
  Healthy:     'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  Degraded:    'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
  Progressing: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  Missing:     'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400',
  Suspended:   'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
  Unknown:     'bg-[var(--gantry-bg-tertiary)] text-[var(--gantry-text-secondary)]',
};

function SyncIcon({ status }: { status: string }) {
  switch (status) {
    case 'Synced':    return <CheckCircle2 className="h-4 w-4 text-green-500" />;
    case 'OutOfSync': return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
    default:          return <HelpCircle className="h-4 w-4 text-[var(--gantry-text-secondary)]" />;
  }
}

function HealthIcon({ status }: { status: string }) {
  switch (status) {
    case 'Healthy':     return <CheckCircle2 className="h-4 w-4 text-green-500" />;
    case 'Degraded':    return <XCircle className="h-4 w-4 text-red-500" />;
    case 'Progressing': return <Clock className="h-4 w-4 text-blue-500 animate-spin" />;
    case 'Missing':     return <AlertTriangle className="h-4 w-4 text-orange-500" />;
    case 'Suspended':   return <Pause className="h-4 w-4 text-purple-500" />;
    default:            return <HelpCircle className="h-4 w-4 text-[var(--gantry-text-secondary)]" />;
  }
}

function ResourceHealthIcon({ health }: { health?: { status: string } }) {
  if (!health) return null;
  switch (health.status) {
    case 'Healthy':     return <CheckCircle2 className="h-3 w-3 text-green-500" />;
    case 'Degraded':    return <XCircle className="h-3 w-3 text-red-500" />;
    case 'Progressing': return <Clock className="h-3 w-3 text-blue-500" />;
    default:            return <HelpCircle className="h-3 w-3 text-[var(--gantry-text-secondary)]" />;
  }
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function ResourcesTable({ resources }: { resources: ArgoCDResourceStatus[] }) {
  const visible = resources.filter((r) => r.kind && r.name);
  if (visible.length === 0) return null;
  return (
    <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] overflow-hidden">
      <table className="min-w-full divide-y divide-[var(--gantry-border)]">
        <thead>
          <tr className="bg-[var(--gantry-bg-secondary)]">
            <th className="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Kind</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Name</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Namespace</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Status</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Health</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[var(--gantry-border)]">
          {visible.map((r, i) => (
            <tr key={i} className="hover:bg-[var(--gantry-bg-secondary)]">
              <td className="px-4 py-2.5 text-xs text-[var(--gantry-text-secondary)]">{r.kind}</td>
              <td className="px-4 py-2.5 font-mono text-xs text-[var(--gantry-text-primary)]">{r.name}</td>
              <td className="px-4 py-2.5 text-xs text-[var(--gantry-text-secondary)]">{r.namespace || '—'}</td>
              <td className="px-4 py-2.5 text-xs text-[var(--gantry-text-secondary)]">{r.status || '—'}</td>
              <td className="px-4 py-2.5">
                {r.health ? (
                  <div className="flex items-center gap-1">
                    <ResourceHealthIcon health={r.health} />
                    <span className="text-xs text-[var(--gantry-text-secondary)]">{r.health.status}</span>
                  </div>
                ) : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// AppCard renders one ArgoCD application with its own Sync/Refresh/Hard Sync buttons.
function AppCard({
  app,
  canSync,
  onUpdated,
}: {
  app: ArgoCDAppWithInstance;
  canSync: boolean;
  onUpdated: (updated: ArgoCDAppWithInstance) => void;
}) {
  const [expanded, setExpanded] = useState(true);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');

  async function handleSync(hard: boolean) {
    setBusy(true);
    setErr('');
    try {
      const updated = await api.syncArgoCDApp(app.appName, hard, app.instance);
      if (updated) onUpdated({ ...updated, instance: app.instance });
    } catch (e: any) {
      setErr(e.message);
    } finally {
      setBusy(false);
    }
  }

  async function handleRefresh() {
    setBusy(true);
    setErr('');
    try {
      const updated = await api.refreshArgoCDApp(app.appName, app.instance);
      if (updated) onUpdated({ ...updated, instance: app.instance });
    } catch (e: any) {
      setErr(e.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] overflow-hidden">
      {/* Header row */}
      <div
        className="flex items-center gap-3 px-4 py-3 cursor-pointer select-none hover:bg-[var(--gantry-bg-secondary)]"
        onClick={() => setExpanded((e) => !e)}
      >
        {expanded ? <ChevronDown className="h-4 w-4 text-[var(--gantry-text-secondary)] shrink-0" /> : <ChevronRight className="h-4 w-4 text-[var(--gantry-text-secondary)] shrink-0" />}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-medium text-sm text-[var(--gantry-text-primary)] font-mono">{app.appName}</span>
            <span className="rounded-md bg-[var(--gantry-bg-tertiary)] px-1.5 py-0.5 text-xs text-[var(--gantry-text-secondary)]">{app.instance}</span>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0" onClick={(e) => e.stopPropagation()}>
          <div className="flex items-center gap-1.5">
            <SyncIcon status={app.syncStatus} />
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${SYNC_STYLES[app.syncStatus] ?? SYNC_STYLES.Unknown}`}>
              {app.syncStatus}
            </span>
          </div>
          <div className="flex items-center gap-1.5">
            <HealthIcon status={app.healthStatus} />
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${HEALTH_STYLES[app.healthStatus] ?? HEALTH_STYLES.Unknown}`}>
              {app.healthStatus}
            </span>
          </div>
          {canSync && (
            <>
              <button
                onClick={handleRefresh}
                disabled={busy}
                title="Refresh from Git"
                className="flex items-center gap-1 rounded-md border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-2 py-1 text-xs font-medium text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-tertiary)] disabled:opacity-50"
              >
                <RotateCcw className={`h-3 w-3 ${busy ? 'animate-spin' : ''}`} /> Refresh
              </button>
              <button
                onClick={() => handleSync(false)}
                disabled={busy}
                className="flex items-center gap-1 rounded-md bg-[var(--gantry-accent)] px-2 py-1 text-xs font-medium text-white hover:bg-[var(--gantry-accent-hover)] disabled:opacity-50"
              >
                <RefreshCw className={`h-3 w-3 ${busy ? 'animate-spin' : ''}`} />
                {busy ? 'Syncing…' : 'Sync'}
              </button>
              <button
                onClick={() => handleSync(true)}
                disabled={busy}
                title="Hard sync — prune and force re-apply"
                className="flex items-center gap-1 rounded-md border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-2 py-1 text-xs font-medium text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-tertiary)] disabled:opacity-50"
              >
                Hard Sync
              </button>
            </>
          )}
          {app.destServer && (
            <a
              href={`${app.destServer.replace(/\/$/, '')}/applications/${app.appName}`}
              target="_blank"
              rel="noopener noreferrer"
              title="Open in ArgoCD"
              className="p-1 text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]"
            >
              <ExternalLink className="h-3.5 w-3.5" />
            </a>
          )}
        </div>
      </div>

      {/* Expanded body */}
      {expanded && (
        <div className="border-t border-[var(--gantry-border)] px-4 py-4 space-y-3">
          {err && (
            <p className="rounded-md bg-red-50 dark:bg-red-900/20 px-3 py-2 text-xs text-red-700 dark:text-red-400">{err}</p>
          )}
          {(app.healthMessage || app.operationMsg) && (
            <p className="rounded-md bg-[var(--gantry-bg-secondary)] px-3 py-2 text-xs text-[var(--gantry-text-secondary)]">
              {app.healthMessage || app.operationMsg}
            </p>
          )}
          <dl className="grid grid-cols-1 gap-y-2 sm:grid-cols-2 text-xs">
            {app.repoURL && (
              <div className="flex items-center gap-1.5 col-span-2">
                <GitBranch className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)] shrink-0" />
                <span className="text-[var(--gantry-text-secondary)] shrink-0">Repo:</span>
                <span className="truncate font-mono text-[var(--gantry-text-primary)]">{app.repoURL}</span>
              </div>
            )}
            {app.targetRevision && (
              <div className="flex items-center gap-1.5">
                <GitCommit className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
                <span className="text-[var(--gantry-text-secondary)]">Target:</span>
                <span className="font-mono text-[var(--gantry-text-primary)]">{app.targetRevision}</span>
              </div>
            )}
            {app.syncRevision && (
              <div className="flex items-center gap-1.5">
                <GitCommit className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
                <span className="text-[var(--gantry-text-secondary)]">Synced at:</span>
                <span className="font-mono text-[var(--gantry-text-primary)]">{app.syncRevision.slice(0, 8)}</span>
              </div>
            )}
            {app.path && (
              <div className="flex items-center gap-1.5">
                <Package className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
                <span className="text-[var(--gantry-text-secondary)]">Path:</span>
                <span className="font-mono text-[var(--gantry-text-primary)]">{app.path}</span>
              </div>
            )}
            {app.chart && (
              <div className="flex items-center gap-1.5">
                <Package className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
                <span className="text-[var(--gantry-text-secondary)]">Chart:</span>
                <span className="font-mono text-[var(--gantry-text-primary)]">{app.chart}</span>
              </div>
            )}
            {app.destNamespace && (
              <div className="flex items-center gap-1.5">
                <Server className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
                <span className="text-[var(--gantry-text-secondary)]">Namespace:</span>
                <span className="font-mono text-[var(--gantry-text-primary)]">{app.destNamespace}</span>
              </div>
            )}
            {app.project && (
              <div className="flex items-center gap-1.5">
                <span className="text-[var(--gantry-text-secondary)]">Project:</span>
                <span className="text-[var(--gantry-text-primary)]">{app.project}</span>
              </div>
            )}
          </dl>
          {app.images && app.images.length > 0 && (
            <div className="flex flex-wrap gap-1.5 pt-1">
              {app.images.map((img) => (
                <span
                  key={img}
                  className="rounded-md border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-2 py-0.5 font-mono text-xs text-[var(--gantry-text-primary)]"
                >
                  {img}
                </span>
              ))}
            </div>
          )}
          {app.resources && app.resources.length > 0 && (
            <div>
              <p className="mb-2 text-xs font-semibold text-[var(--gantry-text-secondary)] uppercase tracking-wide">
                Managed Resources ({app.resources.filter((r) => r.kind && r.name).length})
              </p>
              <ResourcesTable resources={app.resources} />
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function ArgoCDTab({ entity }: { entity: Entity }) {
  const [apps, setApps] = useState<ArgoCDAppWithInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { user } = useAuth();
  const canSync = user?.role !== 'viewer';

  // Parse the argocd.io/appNames annotation — CSV of "instance:appName" pairs.
  // Fall back to the old singular argocd.io/appName for backward compat.
  const appNamesRaw =
    entity.metadata.annotations?.['argocd.io/appNames'] ||
    entity.metadata.annotations?.['argocd.io/appName'] ||
    '';
  const appNames = appNamesRaw.split(',').map((s) => s.trim()).filter(Boolean);

  const load = useCallback(() => {
    if (appNames.length === 0) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError('');
    api
      .getArgoCDEntityApps(appNames)
      .then(setApps)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [appNamesRaw]);

  useEffect(() => { load(); }, [load]);

  function handleUpdated(updated: ArgoCDAppWithInstance) {
    setApps((prev) =>
      prev.map((a) =>
        a.appName === updated.appName && a.instance === updated.instance ? updated : a
      )
    );
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="h-7 w-7 animate-spin rounded-full border-2 border-[var(--gantry-accent)] border-t-transparent" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
        {error}
        <button onClick={load} className="ml-3 underline">Retry</button>
      </div>
    );
  }

  if (apps.length === 0) {
    return (
      <p className="text-sm text-[var(--gantry-text-secondary)]">No ArgoCD applications found.</p>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-xs text-[var(--gantry-text-secondary)]">{apps.length} application{apps.length !== 1 ? 's' : ''}</p>
        <button
          onClick={load}
          className="flex items-center gap-1.5 rounded-lg border border-[var(--gantry-border)] px-3 py-1.5 text-sm text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]"
        >
          <RefreshCw className="h-3.5 w-3.5" /> Reload all
        </button>
      </div>
      {apps.map((app) => (
        <AppCard
          key={`${app.instance}:${app.appName}`}
          app={app}
          canSync={canSync}
          onUpdated={handleUpdated}
        />
      ))}
    </div>
  );
}
