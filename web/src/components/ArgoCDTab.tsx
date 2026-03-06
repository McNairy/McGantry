import { useState, useEffect } from 'react';
import {
  RefreshCw, GitBranch, GitCommit, Server, Package,
  CheckCircle2, XCircle, Clock, AlertTriangle, HelpCircle,
  Pause, ExternalLink, RotateCcw,
} from 'lucide-react';
import { api } from '../lib/api';
import { useAuth } from '../hooks/useAuth';
import type { Entity, ArgoCDAppStatus, ArgoCDResourceStatus } from '../lib/types';

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

function StatusCard({ data, argoURL }: { data: ArgoCDAppStatus; argoURL: string }) {
  return (
    <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h3 className="text-base font-semibold text-[var(--gantry-text-primary)]">{data.appName}</h3>
          {data.project && (
            <p className="mt-0.5 text-xs text-[var(--gantry-text-secondary)]">Project: {data.project}</p>
          )}
        </div>
        {argoURL && (
          <a
            href={`${argoURL}/applications/${data.appName}`}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1.5 shrink-0 rounded-lg border border-[var(--gantry-border)] px-3 py-1.5 text-sm text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]"
          >
            <ExternalLink className="h-4 w-4" /> Open in ArgoCD
          </a>
        )}
      </div>

      <div className="mt-4 flex flex-wrap gap-3">
        <div className="flex items-center gap-1.5">
          <SyncIcon status={data.syncStatus} />
          <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${SYNC_STYLES[data.syncStatus] ?? SYNC_STYLES.Unknown}`}>
            {data.syncStatus}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <HealthIcon status={data.healthStatus} />
          <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${HEALTH_STYLES[data.healthStatus] ?? HEALTH_STYLES.Unknown}`}>
            {data.healthStatus}
          </span>
        </div>
        {data.operationPhase && (
          <span className="rounded-full bg-[var(--gantry-bg-tertiary)] px-2.5 py-0.5 text-xs text-[var(--gantry-text-secondary)]">
            Op: {data.operationPhase}
          </span>
        )}
      </div>

      {(data.healthMessage || data.operationMsg) && (
        <p className="mt-3 rounded-md bg-[var(--gantry-bg-secondary)] px-3 py-2 text-xs text-[var(--gantry-text-secondary)]">
          {data.healthMessage || data.operationMsg}
        </p>
      )}

      {/* Source info */}
      <dl className="mt-4 grid grid-cols-1 gap-y-2 sm:grid-cols-2 text-xs">
        {data.repoURL && (
          <div className="flex items-center gap-1.5 col-span-2">
            <GitBranch className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)] shrink-0" />
            <span className="text-[var(--gantry-text-secondary)] shrink-0">Repo:</span>
            <span className="truncate font-mono text-[var(--gantry-text-primary)]">{data.repoURL}</span>
          </div>
        )}
        {data.targetRevision && (
          <div className="flex items-center gap-1.5">
            <GitCommit className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
            <span className="text-[var(--gantry-text-secondary)]">Target:</span>
            <span className="font-mono text-[var(--gantry-text-primary)]">{data.targetRevision}</span>
          </div>
        )}
        {data.syncRevision && (
          <div className="flex items-center gap-1.5">
            <GitCommit className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
            <span className="text-[var(--gantry-text-secondary)]">Synced at:</span>
            <span className="font-mono text-[var(--gantry-text-primary)]">{data.syncRevision.slice(0, 8)}</span>
          </div>
        )}
        {data.path && (
          <div className="flex items-center gap-1.5">
            <Package className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
            <span className="text-[var(--gantry-text-secondary)]">Path:</span>
            <span className="font-mono text-[var(--gantry-text-primary)]">{data.path}</span>
          </div>
        )}
        {data.chart && (
          <div className="flex items-center gap-1.5">
            <Package className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
            <span className="text-[var(--gantry-text-secondary)]">Chart:</span>
            <span className="font-mono text-[var(--gantry-text-primary)]">{data.chart}</span>
          </div>
        )}
        {data.destNamespace && (
          <div className="flex items-center gap-1.5">
            <Server className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
            <span className="text-[var(--gantry-text-secondary)]">Namespace:</span>
            <span className="font-mono text-[var(--gantry-text-primary)]">{data.destNamespace}</span>
          </div>
        )}
      </dl>
    </div>
  );
}

function ResourcesTable({ resources }: { resources: ArgoCDResourceStatus[] }) {
  // Group by kind, only show the most interesting ones (skip CRDs, etc.)
  const visible = resources.filter((r) => r.kind && r.name);
  if (visible.length === 0) return null;

  return (
    <div>
      <h3 className="mb-3 text-sm font-semibold text-[var(--gantry-text-primary)]">
        Managed Resources
        <span className="ml-2 text-xs font-normal text-[var(--gantry-text-secondary)]">({visible.length})</span>
      </h3>
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
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function ArgoCDTab({ entity }: { entity: Entity }) {
  const [data, setData] = useState<ArgoCDAppStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState('');
  const { user } = useAuth();
  const canSync = user?.role !== 'viewer';

  // The ArgoCD app name is stored in the argocd.io/appName annotation.
  const appName = entity.metadata.annotations?.['argocd.io/appName'] ?? entity.metadata.name;
  // The ArgoCD server URL annotation gives us a deep-link base.
  const argoURLAnno = entity.metadata.annotations?.['argocd.io/appURL'] ?? '';
  const argoURL = argoURLAnno
    ? argoURLAnno.replace(/\/applications\/.*$/, '')
    : '';

  function load() {
    setLoading(true);
    setError('');
    api
      .getArgoCDApp(appName)
      .then(setData)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }

  useEffect(() => { load(); }, [appName]);

  async function handleSync(hard: boolean) {
    setSyncing(true);
    setError('');
    try {
      const updated = await api.syncArgoCDApp(appName, hard);
      if (updated) setData(updated);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSyncing(false);
    }
  }

  async function handleRefresh() {
    setSyncing(true);
    setError('');
    try {
      const updated = await api.refreshArgoCDApp(appName);
      if (updated) setData(updated);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSyncing(false);
    }
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

  if (!data) return null;

  return (
    <div className="space-y-6">
      {/* Action bar */}
      <div className="flex items-center gap-2">
        <button
          onClick={load}
          disabled={syncing}
          title="Reload status from Gantry"
          className="flex items-center gap-1.5 rounded-lg border border-[var(--gantry-border)] px-3 py-1.5 text-sm text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)] disabled:opacity-50"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${syncing ? 'animate-spin' : ''}`} /> Reload
        </button>
        {canSync && (
          <>
            <button
              onClick={() => handleRefresh()}
              disabled={syncing}
              title="Refresh from Git without syncing"
              className="flex items-center gap-1.5 rounded-lg border border-[var(--gantry-border)] px-3 py-1.5 text-sm text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)] disabled:opacity-50"
            >
              <RotateCcw className={`h-3.5 w-3.5 ${syncing ? 'animate-spin' : ''}`} /> Refresh
            </button>
            <button
              onClick={() => handleSync(false)}
              disabled={syncing}
              className="flex items-center gap-1.5 rounded-lg bg-[var(--gantry-accent)] px-3 py-1.5 text-sm font-medium text-white hover:bg-[var(--gantry-accent-hover)] disabled:opacity-50"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${syncing ? 'animate-spin' : ''}`} />
              {syncing ? 'Syncing…' : 'Sync'}
            </button>
            <button
              onClick={() => handleSync(true)}
              disabled={syncing}
              title="Hard sync — prune resources and force re-apply"
              className="flex items-center gap-1.5 rounded-lg border border-[var(--gantry-border)] px-3 py-1.5 text-sm text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)] disabled:opacity-50"
            >
              Hard Sync
            </button>
          </>
        )}
      </div>

      {/* Status card */}
      <StatusCard data={data} argoURL={argoURL} />

      {/* Images */}
      {data.images && data.images.length > 0 && (
        <div>
          <h3 className="mb-3 text-sm font-semibold text-[var(--gantry-text-primary)]">Images</h3>
          <div className="flex flex-wrap gap-2">
            {data.images.map((img) => (
              <span
                key={img}
                className="rounded-md border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-2.5 py-1 font-mono text-xs text-[var(--gantry-text-primary)]"
              >
                {img}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Managed resources table */}
      {data.resources && <ResourcesTable resources={data.resources} />}
    </div>
  );
}
