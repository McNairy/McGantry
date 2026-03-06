import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import {
  Server,
  Globe,
  Database,
  Users,
  Cloud,
  FileText,
  Box,
  ArrowRight,
  BookOpen,
  Zap,
  Search,
  ClipboardList,
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
} from 'lucide-react';
import { api } from '../lib/api';
import { useAuth } from '../hooks/useAuth';
import type { Entity, AuditEntry, ActionRun } from '../lib/types';
import { ENTITY_KINDS } from '../lib/types';

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  Server,
  Globe,
  Database,
  Users,
  Cloud,
  FileText,
};

interface KindCount {
  name: string;
  plural: string;
  icon: string;
  count: number;
}

const ACTION_DOT: Record<string, string> = {
  'entity.created': 'bg-green-500',
  'entity.updated': 'bg-blue-500',
  'entity.deleted': 'bg-red-500',
};

const RUN_STATUS_ICON: Record<string, React.ReactNode> = {
  pending: <Clock className="h-4 w-4 text-[var(--gantry-warning)]" />,
  running: <Loader2 className="h-4 w-4 animate-spin text-[var(--gantry-accent)]" />,
  success: <CheckCircle2 className="h-4 w-4 text-[var(--gantry-success)]" />,
  failed: <XCircle className="h-4 w-4 text-[var(--gantry-danger)]" />,
};

function relativeTime(ts: string): string {
  const diff = Date.now() - new Date(ts).getTime();
  if (diff < 60_000) return 'just now';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

export default function Dashboard() {
  const { user } = useAuth();
  const [entities, setEntities] = useState<Entity[]>([]);
  const [activity, setActivity] = useState<AuditEntry[]>([]);
  const [actionRuns, setActionRuns] = useState<ActionRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([
      api.listEntities(),
      api.listAuditEntries(5, 0).catch(() => [] as AuditEntry[]),
      api.listAllActionRuns(5).catch(() => [] as ActionRun[]),
    ])
      .then(([ents, audit, runs]) => {
        setEntities(ents || []);
        setActivity(audit || []);
        setActionRuns(runs || []);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const kindCounts: KindCount[] = ENTITY_KINDS.map((k) => ({
    ...k,
    count: entities.filter((e) => e.kind === k.name).length,
  }));

  const totalEntities = entities.length;

  const recentEntities = [...entities]
    .sort((a, b) => {
      const aDate = a.metadata.updatedAt || a.metadata.createdAt || '';
      const bDate = b.metadata.updatedAt || b.metadata.createdAt || '';
      return bDate.localeCompare(aDate);
    })
    .slice(0, 5);

  const myEntities = user
    ? entities.filter((e) => e.metadata.owner === user.username).slice(0, 5)
    : [];

  function formatDate(dateStr?: string): string {
    if (!dateStr) return '';
    try {
      return new Date(dateStr).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return dateStr;
    }
  }

  return (
    <div className="space-y-8">
      {/* Welcome Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-[var(--gantry-text-primary)]">
            Welcome to Gantry{user?.displayName ? `, ${user.displayName}` : ''}
          </h1>
          <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">
            Your internal developer platform overview
          </p>
        </div>
        <button
          onClick={() => document.dispatchEvent(new Event('gantry:open-search'))}
          className="flex items-center gap-2 rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] px-4 py-2 text-sm text-[var(--gantry-text-secondary)] transition-colors hover:border-[var(--gantry-accent)] hover:text-[var(--gantry-text-primary)] sm:w-64"
        >
          <Search className="h-4 w-4 shrink-0" />
          <span className="flex-1 text-left">Search entities...</span>
          <kbd className="hidden rounded border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-1.5 py-0.5 text-xs sm:inline-block">
            ⌘K
          </kbd>
        </button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-[var(--gantry-danger)] dark:border-red-800 dark:bg-red-900/20">
          {error}
        </div>
      )}

      {loading && (
        <div className="flex items-center justify-center py-16">
          <div className="spinner h-8 w-8 text-[var(--gantry-accent)]" />
        </div>
      )}

      {!loading && (
        <>
          {/* Stats Cards */}
          <div className="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-6">
            {kindCounts.map((kind) => {
              const Icon = iconMap[kind.icon] || Box;
              return (
                <Link
                  key={kind.name}
                  to={`/catalog/${kind.name}`}
                  className="group rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-4 transition-all hover:shadow-md"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-[var(--gantry-accent)]/10">
                      <Icon className="h-5 w-5 text-[var(--gantry-accent)]" />
                    </div>
                    <div>
                      <p className="text-2xl font-bold text-[var(--gantry-text-primary)]">
                        {kind.count}
                      </p>
                      <p className="text-xs text-[var(--gantry-text-secondary)]">{kind.name}s</p>
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>

          {/* Empty state / Getting started */}
          {totalEntities === 0 && (
            <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-8">
              <h2 className="text-lg font-semibold text-[var(--gantry-text-primary)]">
                Getting Started
              </h2>
              <p className="mt-2 text-sm text-[var(--gantry-text-secondary)]">
                Your software catalog is empty. Here are some things you can do to get started:
              </p>
              <div className="mt-6 grid gap-4 md:grid-cols-3">
                <Link
                  to="/catalog"
                  className="flex items-start gap-3 rounded-lg border border-[var(--gantry-border)] p-4 transition-colors hover:bg-[var(--gantry-bg-secondary)]"
                >
                  <BookOpen className="mt-0.5 h-5 w-5 shrink-0 text-[var(--gantry-accent)]" />
                  <div>
                    <p className="text-sm font-medium text-[var(--gantry-text-primary)]">Register a Service</p>
                    <p className="mt-1 text-xs text-[var(--gantry-text-secondary)]">
                      Add your first service to the catalog to start tracking your software.
                    </p>
                  </div>
                </Link>
                <Link
                  to="/actions"
                  className="flex items-start gap-3 rounded-lg border border-[var(--gantry-border)] p-4 transition-colors hover:bg-[var(--gantry-bg-secondary)]"
                >
                  <Zap className="mt-0.5 h-5 w-5 shrink-0 text-[var(--gantry-accent)]" />
                  <div>
                    <p className="text-sm font-medium text-[var(--gantry-text-primary)]">Explore Actions</p>
                    <p className="mt-1 text-xs text-[var(--gantry-text-secondary)]">
                      Browse and execute self-service actions to automate common tasks.
                    </p>
                  </div>
                </Link>
                <div className="flex items-start gap-3 rounded-lg border border-[var(--gantry-border)] p-4">
                  <Search className="mt-0.5 h-5 w-5 shrink-0 text-[var(--gantry-accent)]" />
                  <div>
                    <p className="text-sm font-medium text-[var(--gantry-text-primary)]">Search with Cmd+K</p>
                    <p className="mt-1 text-xs text-[var(--gantry-text-secondary)]">
                      Use the command palette to quickly find entities across your catalog.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Two-column: Recent Activity + Action Runs */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            {/* Recent Activity */}
            <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)]">
              <div className="flex items-center justify-between border-b border-[var(--gantry-border)] px-6 py-4">
                <div className="flex items-center gap-2">
                  <ClipboardList className="h-4 w-4 text-[var(--gantry-text-secondary)]" />
                  <h2 className="text-base font-semibold text-[var(--gantry-text-primary)]">Recent Activity</h2>
                </div>
                <Link to="/audit" className="flex items-center gap-1 text-sm text-[var(--gantry-accent)] hover:opacity-75">
                  View all <ArrowRight className="h-4 w-4" />
                </Link>
              </div>
              {activity.length === 0 ? (
                <p className="px-6 py-4 text-sm text-[var(--gantry-text-secondary)]">No recent activity.</p>
              ) : (
                <div className="divide-y divide-[var(--gantry-border)]">
                  {activity.map((entry) => (
                    <div key={entry.id} className="flex items-center gap-3 px-6 py-3">
                      <div className={`h-2 w-2 shrink-0 rounded-full ${ACTION_DOT[entry.action] ?? 'bg-[var(--gantry-text-secondary)]'}`} />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm text-[var(--gantry-text-primary)]">
                          {entry.resourceType && entry.resourceName
                            ? `${entry.resourceType}/${entry.resourceName}`
                            : entry.action}
                        </p>
                        <p className="text-xs text-[var(--gantry-text-secondary)]">
                          {entry.action}{entry.userName ? ` by ${entry.userName}` : ''}
                        </p>
                      </div>
                      <span className="shrink-0 text-xs text-[var(--gantry-text-secondary)]">
                        {relativeTime(entry.timestamp)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Action Runs */}
            <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)]">
              <div className="flex items-center justify-between border-b border-[var(--gantry-border)] px-6 py-4">
                <div className="flex items-center gap-2">
                  <Zap className="h-4 w-4 text-[var(--gantry-text-secondary)]" />
                  <h2 className="text-base font-semibold text-[var(--gantry-text-primary)]">Action Runs</h2>
                </div>
                <Link to="/actions" className="flex items-center gap-1 text-sm text-[var(--gantry-accent)] hover:opacity-75">
                  Go to Actions <ArrowRight className="h-4 w-4" />
                </Link>
              </div>
              {actionRuns.length === 0 ? (
                <p className="px-6 py-4 text-sm text-[var(--gantry-text-secondary)]">No action runs yet.</p>
              ) : (
                <div className="divide-y divide-[var(--gantry-border)]">
                  {actionRuns.map((run) => (
                    <div key={run.id} className="flex items-center gap-3 px-6 py-3">
                      {RUN_STATUS_ICON[run.status] ?? RUN_STATUS_ICON.pending}
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium text-[var(--gantry-text-primary)]">
                          {run.actionName}
                        </p>
                        <p className="truncate text-xs text-[var(--gantry-text-secondary)]">
                          {run.triggeredBy ? `by ${run.triggeredBy}` : 'system'}
                          {run.error ? ` — ${run.error}` : ''}
                        </p>
                      </div>
                      <span className={`shrink-0 rounded-full px-2 py-0.5 text-xs font-medium ${
                        run.status === 'success' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                        run.status === 'failed'  ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' :
                        run.status === 'running' ? 'bg-[var(--gantry-accent)]/10 text-[var(--gantry-accent)]' :
                        'bg-[var(--gantry-bg-tertiary)] text-[var(--gantry-text-secondary)]'
                      }`}>
                        {run.status}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* My Entities + Recently Updated */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            {myEntities.length > 0 && (
              <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)]">
                <div className="border-b border-[var(--gantry-border)] px-6 py-4">
                  <h2 className="text-base font-semibold text-[var(--gantry-text-primary)]">My Entities</h2>
                  <p className="mt-0.5 text-xs text-[var(--gantry-text-secondary)]">Owned by {user?.username}</p>
                </div>
                <div className="divide-y divide-[var(--gantry-border)]">
                  {myEntities.map((entity) => {
                    const Icon = iconMap[ENTITY_KINDS.find((k) => k.name === entity.kind)?.icon || ''] || Box;
                    return (
                      <Link
                        key={`${entity.kind}-${entity.metadata.name}`}
                        to={`/catalog/${entity.kind}/${entity.metadata.name}${entity.metadata.namespace && entity.metadata.namespace !== 'default' ? `?namespace=${encodeURIComponent(entity.metadata.namespace)}` : ''}`}
                        className="flex items-center gap-4 px-6 py-3 transition-colors hover:bg-[var(--gantry-bg-secondary)]"
                      >
                        <Icon className="h-4 w-4 shrink-0 text-[var(--gantry-text-secondary)]" />
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-sm font-medium text-[var(--gantry-text-primary)]">
                            {entity.metadata.title || entity.metadata.name}
                          </p>
                          <p className="text-xs text-[var(--gantry-text-secondary)]">{entity.kind}</p>
                        </div>
                      </Link>
                    );
                  })}
                </div>
              </div>
            )}

            {recentEntities.length > 0 && (
              <div className={`rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] ${myEntities.length === 0 ? 'lg:col-span-2' : ''}`}>
                <div className="flex items-center justify-between border-b border-[var(--gantry-border)] px-6 py-4">
                  <h2 className="text-base font-semibold text-[var(--gantry-text-primary)]">Recently Updated</h2>
                  <Link to="/catalog" className="flex items-center gap-1 text-sm text-[var(--gantry-accent)] hover:opacity-75">
                    View all <ArrowRight className="h-4 w-4" />
                  </Link>
                </div>
                <div className="divide-y divide-[var(--gantry-border)]">
                  {recentEntities.map((entity) => {
                    const Icon = iconMap[ENTITY_KINDS.find((k) => k.name === entity.kind)?.icon || ''] || Box;
                    return (
                      <Link
                        key={`${entity.kind}-${entity.metadata.name}`}
                        to={`/catalog/${entity.kind}/${entity.metadata.name}${entity.metadata.namespace && entity.metadata.namespace !== 'default' ? `?namespace=${encodeURIComponent(entity.metadata.namespace)}` : ''}`}
                        className="flex items-center gap-4 px-6 py-3 transition-colors hover:bg-[var(--gantry-bg-secondary)]"
                      >
                        <Icon className="h-5 w-5 shrink-0 text-[var(--gantry-text-secondary)]" />
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-sm font-medium text-[var(--gantry-text-primary)]">
                            {entity.metadata.name}
                          </p>
                          <p className="text-xs text-[var(--gantry-text-secondary)]">
                            {entity.kind}{entity.metadata.owner ? ` / ${entity.metadata.owner}` : ''}
                          </p>
                        </div>
                        <span className="shrink-0 text-xs text-[var(--gantry-text-secondary)]">
                          {formatDate(entity.metadata.updatedAt || entity.metadata.createdAt)}
                        </span>
                      </Link>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
