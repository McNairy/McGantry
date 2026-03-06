import { useState, useEffect } from 'react';
import { useParams, useNavigate, useSearchParams, Link } from 'react-router-dom';
import { ChevronRight, Pencil, Trash2, X, ExternalLink, LayoutDashboard, BookOpen, FileText, Github, MessageSquare, Bell, Activity, Cpu, CircleHelp } from 'lucide-react';
import { api } from '../lib/api';
import { useAuth } from '../hooks/useAuth';
import type { Entity, JsonSchema, AuditEntry, GraphData, EntityLink } from '../lib/types';
import SchemaForm from '../components/SchemaForm';
import EntityGraph from '../components/EntityGraph';
import KubernetesTab from '../components/KubernetesTab';
import GitHubTab from '../components/GitHubTab';

const ACTION_COLORS: Record<string, string> = {
  'entity.created': 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  'entity.updated': 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  'entity.deleted': 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
};

function formatTime(ts: string): string {
  const date = new Date(ts);
  const diff = Date.now() - date.getTime();
  if (diff < 60_000) return 'just now';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function toYaml(obj: unknown, indent = 0): string {
  const pad = '  '.repeat(indent);
  if (obj === null || obj === undefined) return 'null';
  if (typeof obj === 'string') {
    if (obj === '') return "''";
    if (/[\n:#\[\]{},&*?|<>=!%@`]/.test(obj) || /^\s|\s$/.test(obj)) {
      return `"${obj.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/\n/g, '\\n')}"`;
    }
    return obj;
  }
  if (typeof obj === 'number' || typeof obj === 'boolean') return String(obj);
  if (Array.isArray(obj)) {
    if (obj.length === 0) return '[]';
    return '\n' + obj.map((item) => `${pad}- ${toYaml(item, indent + 1).trimStart()}`).join('\n');
  }
  if (typeof obj === 'object') {
    const entries = Object.entries(obj as Record<string, unknown>).filter(
      ([, v]) => v !== undefined && v !== null && v !== ''
    );
    if (entries.length === 0) return '{}';
    return (
      '\n' +
      entries
        .map(([k, v]) => {
          const val = toYaml(v, indent + 1);
          return val.startsWith('\n') ? `${pad}${k}:${val}` : `${pad}${k}: ${val}`;
        })
        .join('\n')
    );
  }
  return String(obj);
}

function entityToYaml(entity: Entity): string {
  const ordered: Record<string, unknown> = {
    kind: entity.kind,
    apiVersion: entity.apiVersion,
    metadata: entity.metadata,
    ...(entity.spec && Object.keys(entity.spec).length > 0 ? { spec: entity.spec } : {}),
  };
  return Object.entries(ordered)
    .map(([k, v]) => {
      const val = toYaml(v, 1);
      return val.startsWith('\n') ? `${k}:${val}` : `${k}: ${val}`;
    })
    .join('\n');
}

const LINK_ICONS: Record<string, React.ReactNode> = {
  dashboard: <LayoutDashboard className="h-3.5 w-3.5" />,
  docs:      <BookOpen className="h-3.5 w-3.5" />,
  runbook:   <FileText className="h-3.5 w-3.5" />,
  github:    <Github className="h-3.5 w-3.5" />,
  slack:     <MessageSquare className="h-3.5 w-3.5" />,
  alert:     <Bell className="h-3.5 w-3.5" />,
  monitor:   <Activity className="h-3.5 w-3.5" />,
  ci:        <Cpu className="h-3.5 w-3.5" />,
  other:     <CircleHelp className="h-3.5 w-3.5" />,
};

type Tab = 'overview' | 'yaml' | 'relationships' | 'activity' | 'kubernetes' | 'github';

export default function EntityDetail() {
  const { kind, name } = useParams<{ kind: string; name: string }>();
  const [searchParams] = useSearchParams();
  const namespace = searchParams.get('namespace') ?? 'default';
  const navigate = useNavigate();
  const { user } = useAuth();
  const canWrite = user?.role !== 'viewer';
  const [entity, setEntity] = useState<Entity | null>(null);
  const [schema, setSchema] = useState<JsonSchema | null>(null);
  const [activity, setActivity] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<Tab>('overview');
  const [editing, setEditing] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [graphData, setGraphData] = useState<GraphData | null>(null);
  const [graphLoading, setGraphLoading] = useState(false);

  useEffect(() => {
    if (!kind || !name) return;
    setLoading(true);
    Promise.all([
      api.getEntity(kind, name, namespace),
      api.getSchema(kind).catch(() => null),
      api.listAuditEntries(100, 0).catch(() => [] as AuditEntry[]),
    ])
      .then(([e, s, audit]) => {
        setEntity(e);
        setSchema(s);
        const entries = (audit ?? []).filter(
          (a) => a.resourceName === name && a.resourceType === kind
        );
        setActivity(entries);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [kind, name, namespace]);

  useEffect(() => {
    if (tab !== 'relationships' || !kind || !name || graphData) return;
    setGraphLoading(true);
    api.getEntityGraph(kind, name, namespace)
      .then(setGraphData)
      .catch(() => {}) // Graph errors are non-fatal; graph stays null
      .finally(() => setGraphLoading(false));
  }, [tab, kind, name, graphData]);

  const handleUpdate = async (spec: Record<string, any>) => {
    if (!entity || !kind || !name) return;
    try {
      const updated = await api.updateEntity(kind, name, {
        ...entity,
        spec,
      }, namespace);
      setEntity(updated);
      setEditing(false);
    } catch (e: any) {
      setError(e.message);
    }
  };

  const handleDelete = async () => {
    if (!kind || !name) return;
    try {
      await api.deleteEntity(kind, name, namespace);
      navigate('/catalog');
    } catch (e: any) {
      setError(e.message);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--gantry-accent)] border-t-transparent" />
      </div>
    );
  }

  if (error || !entity) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
        {error || 'Entity not found.'}
      </div>
    );
  }


  return (
    <div>
      {/* Breadcrumb */}
      <nav className="flex items-center gap-1 text-sm text-[var(--gantry-text-secondary)]">
        <Link to="/catalog" className="hover:text-[var(--gantry-accent)]">Catalog</Link>
        <ChevronRight className="h-4 w-4" />
        <Link to={`/catalog/${entity.kind}`} className="hover:text-[var(--gantry-accent)]">{entity.kind}</Link>
        <ChevronRight className="h-4 w-4" />
        <span className="text-[var(--gantry-text-primary)]">{entity.metadata.name}</span>
      </nav>

      {/* Header */}
      <div className="mt-4 flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <span className="rounded-md bg-[var(--gantry-bg-tertiary)] px-2.5 py-1 text-xs font-medium text-[var(--gantry-text-secondary)]">
              {entity.kind}
            </span>
            <h1 className="text-2xl font-bold text-[var(--gantry-text-primary)]">{entity.metadata.name}</h1>
          </div>
          {entity.metadata.title && entity.metadata.title !== entity.metadata.name && (
            <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">{entity.metadata.title}</p>
          )}
          {entity.metadata.description && (
            <p className="mt-2 text-sm text-[var(--gantry-text-secondary)]">{entity.metadata.description}</p>
          )}
          <div className="mt-3 flex flex-wrap items-center gap-3">
            {entity.metadata.owner && (
              <span className="text-sm text-[var(--gantry-text-secondary)]">
                Owner: <span className="font-medium text-[var(--gantry-text-primary)]">{entity.metadata.owner}</span>
              </span>
            )}
            {entity.spec?.lifecycle && (
              <span className="rounded-full bg-[var(--gantry-bg-tertiary)] px-2.5 py-0.5 text-xs text-[var(--gantry-text-secondary)]">
                {String(entity.spec.lifecycle)}
              </span>
            )}
            {entity.spec?.type && (
              <span className="rounded-full bg-[var(--gantry-bg-tertiary)] px-2.5 py-0.5 text-xs text-[var(--gantry-text-secondary)]">
                {String(entity.spec.type)}
              </span>
            )}
          </div>
          {entity.metadata.tags && entity.metadata.tags.length > 0 && (
            <div className="mt-3 flex flex-wrap gap-1.5">
              {entity.metadata.tags.map((tag) => (
                <span key={tag} className="rounded-full bg-[var(--gantry-bg-tertiary)] px-2.5 py-0.5 text-xs text-[var(--gantry-text-secondary)]">
                  {tag}
                </span>
              ))}
            </div>
          )}
        </div>
        {canWrite && (
          <div className="flex items-center gap-2">
            <button
              onClick={() => setEditing(true)}
              className="flex items-center gap-1.5 rounded-lg border border-[var(--gantry-border)] px-3 py-2 text-sm text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-tertiary)]"
            >
              <Pencil className="h-4 w-4" /> Edit
            </button>
            <button
              onClick={() => setShowDelete(true)}
              className="flex items-center gap-1.5 rounded-lg border border-red-200 px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20"
            >
              <Trash2 className="h-4 w-4" /> Delete
            </button>
          </div>
        )}
      </div>

      {/* Tabs */}
      {(() => {
        const isK8sEntity = !!(entity.metadata.annotations?.['kubernetes.io/kind']);
        const hasGitHub = !!(
          (entity.spec?.repoUrl as string | undefined)?.includes('github.com') ||
          entity.metadata.annotations?.['github.com/repo']
        );
        const tabs: Tab[] = ['overview', 'relationships', 'yaml', 'activity'];
        if (isK8sEntity && (entity.kind === 'Service' || entity.kind === 'Infrastructure')) tabs.splice(1, 0, 'kubernetes');
        if (hasGitHub) tabs.splice(1, 0, 'github');
        return (
          <div className="mt-6 flex gap-1 border-b border-[var(--gantry-border)]">
            {tabs.map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={`border-b-2 px-4 py-2 text-sm font-medium capitalize transition-colors ${
                  tab === t
                    ? 'border-[var(--gantry-accent)] text-[var(--gantry-accent)]'
                    : 'border-transparent text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]'
                }`}
              >
                {t === 'relationships' ? 'Dependencies' : t === 'kubernetes' ? 'Kubernetes' : t === 'github' ? 'GitHub' : t}
                {t === 'activity' && activity.length > 0 && (
                  <span className="ml-1.5 rounded-full bg-[var(--gantry-bg-tertiary)] px-1.5 py-0.5 text-xs">
                    {activity.length}
                  </span>
                )}
              </button>
            ))}
          </div>
        );
      })()}

      {/* Tab Content */}
      <div className="mt-6">
        {tab === 'overview' && (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            {/* Spec */}
            <div className="lg:col-span-2">
              <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
                <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Spec</h3>
                {entity.spec && Object.keys(entity.spec).length > 0 ? (
                  <dl className="mt-4 space-y-3">
                    {Object.entries(entity.spec).map(([key, value]) => {
                      // Skip repoUrl here — it's shown in the Links sidebar card.
                      if (key === 'repoUrl') return null;

                      // Render links array as clickable links.
                      const isLinkArray =
                        key === 'links' &&
                        Array.isArray(value) &&
                        (value as any[]).every((v) => v && typeof v === 'object' && 'url' in v && 'title' in v);

                      // Render arrays of entity references ({kind, name}) as chips instead of raw JSON.
                      const isRefArray =
                        Array.isArray(value) &&
                        value.length > 0 &&
                        (value as any[]).every((v) => v && typeof v === 'object' && 'kind' in v && 'name' in v);

                      return (
                        <div key={key}>
                          <dt className="text-xs font-medium text-[var(--gantry-text-secondary)]">{key}</dt>
                          <dd className="mt-1.5 text-sm text-[var(--gantry-text-primary)]">
                            {isLinkArray ? (
                              <ul className="space-y-1">
                                {(value as EntityLink[]).map((link, i) => (
                                  <li key={i}>
                                    <a
                                      href={link.url}
                                      target="_blank"
                                      rel="noopener noreferrer"
                                      className="flex items-center gap-1.5 text-sm text-[var(--gantry-accent)] hover:underline"
                                    >
                                      <span className="shrink-0 text-[var(--gantry-text-secondary)]">
                                        {LINK_ICONS[link.icon ?? 'other'] ?? <ExternalLink className="h-3.5 w-3.5" />}
                                      </span>
                                      {link.title}
                                    </a>
                                  </li>
                                ))}
                              </ul>
                            ) : isRefArray ? (
                              <div className="flex flex-wrap gap-1.5">
                                {(value as { kind: string; name: string }[]).map((ref) => (
                                  <span
                                    key={`${ref.kind}/${ref.name}`}
                                    className="inline-flex items-center gap-1 rounded-md border border-[var(--gantry-border)] bg-[var(--gantry-bg-tertiary)] px-2 py-0.5 text-xs"
                                  >
                                    <span className="text-[var(--gantry-text-secondary)]">{ref.kind}</span>
                                    <span className="text-[var(--gantry-text-primary)] font-medium">{ref.name}</span>
                                  </span>
                                ))}
                              </div>
                            ) : typeof value === 'object' ? (
                              <pre className="mt-1 overflow-auto rounded-md bg-[var(--gantry-bg-tertiary)] p-2 text-xs">
                                {JSON.stringify(value, null, 2)}
                              </pre>
                            ) : (
                              String(value)
                            )}
                          </dd>
                        </div>
                      );
                    })}
                  </dl>
                ) : (
                  <p className="mt-4 text-sm text-[var(--gantry-text-secondary)]">No spec fields defined.</p>
                )}
              </div>
            </div>
            {/* Metadata sidebar */}
            <div className="space-y-4">
              <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
                <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Metadata</h3>
                <dl className="mt-4 space-y-3">
                  {entity.metadata.namespace && entity.metadata.namespace !== 'default' && (
                    <div>
                      <dt className="text-xs font-medium text-[var(--gantry-text-secondary)]">Namespace</dt>
                      <dd className="mt-0.5 text-sm text-[var(--gantry-text-primary)]">{entity.metadata.namespace}</dd>
                    </div>
                  )}
                  <div>
                    <dt className="text-xs font-medium text-[var(--gantry-text-secondary)]">API Version</dt>
                    <dd className="mt-0.5 text-sm text-[var(--gantry-text-primary)]">{entity.apiVersion}</dd>
                  </div>
                  {entity.metadata.createdAt && (
                    <div>
                      <dt className="text-xs font-medium text-[var(--gantry-text-secondary)]">Created</dt>
                      <dd className="mt-0.5 text-sm text-[var(--gantry-text-primary)]">
                        {new Date(entity.metadata.createdAt).toLocaleString()}
                      </dd>
                    </div>
                  )}
                  {entity.metadata.updatedAt && (
                    <div>
                      <dt className="text-xs font-medium text-[var(--gantry-text-secondary)]">Updated</dt>
                      <dd className="mt-0.5 text-sm text-[var(--gantry-text-primary)]">
                        {new Date(entity.metadata.updatedAt).toLocaleString()}
                      </dd>
                    </div>
                  )}
                </dl>
              </div>

              {/* Kubernetes source info — shown only for k8s-synced entities */}
              {(() => {
                const anno = entity.metadata.annotations ?? {};
                const k8sKind = anno['kubernetes.io/kind'];
                if (!k8sKind) return null;

                // Clusters: comma-separated from kubernetes.io/clusters, or single clusterName.
                const clustersRaw = anno['kubernetes.io/clusters'] || anno['kubernetes.io/clusterName'];
                const clusters = clustersRaw ? clustersRaw.split(',').map((s: string) => s.trim()).filter(Boolean) : [];

                // Namespaces: from spec.deployedIn (accumulated across all syncs).
                const deployedIn: { kind: string; name: string }[] =
                  Array.isArray(entity.spec?.deployedIn)
                    ? (entity.spec.deployedIn as any[]).filter(
                        (d: any) => d && typeof d === 'object' && d.kind === 'Environment'
                      )
                    : [];

                return (
                  <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
                    <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Kubernetes</h3>
                    <dl className="mt-3 space-y-3">
                      <div className="flex items-start justify-between gap-3">
                        <dt className="text-xs font-medium text-[var(--gantry-text-secondary)] shrink-0">Resource</dt>
                        <dd className="text-xs text-[var(--gantry-text-primary)] text-right font-mono">{k8sKind}</dd>
                      </div>
                      {clusters.length > 0 && (
                        <div className="flex items-start justify-between gap-3">
                          <dt className="text-xs font-medium text-[var(--gantry-text-secondary)] shrink-0">
                            {clusters.length === 1 ? 'Cluster' : 'Clusters'}
                          </dt>
                          <dd className="text-xs text-[var(--gantry-text-primary)] text-right font-mono">
                            {clusters.join(', ')}
                          </dd>
                        </div>
                      )}
                      {deployedIn.length > 0 && (
                        <div>
                          <dt className="text-xs font-medium text-[var(--gantry-text-secondary)]">
                            {deployedIn.length === 1 ? 'Namespace' : 'Namespaces'}
                          </dt>
                          <dd className="mt-1.5 flex flex-wrap gap-1">
                            {deployedIn.map((d) => (
                              <span
                                key={d.name}
                                className="rounded-md border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-1.5 py-0.5 text-xs font-mono text-[var(--gantry-text-primary)]"
                              >
                                {d.name}
                              </span>
                            ))}
                          </dd>
                        </div>
                      )}
                      {anno['kubernetes.io/namespace'] && deployedIn.length === 0 && (
                        <div className="flex items-start justify-between gap-3">
                          <dt className="text-xs font-medium text-[var(--gantry-text-secondary)] shrink-0">Namespace</dt>
                          <dd className="text-xs text-[var(--gantry-text-primary)] text-right font-mono">{anno['kubernetes.io/namespace']}</dd>
                        </div>
                      )}
                      {anno['kubernetes.io/serviceType'] && (
                        <div className="flex items-start justify-between gap-3">
                          <dt className="text-xs font-medium text-[var(--gantry-text-secondary)] shrink-0">Service Type</dt>
                          <dd className="text-xs text-[var(--gantry-text-primary)] text-right font-mono">{anno['kubernetes.io/serviceType']}</dd>
                        </div>
                      )}
                      {anno['kubernetes.io/clusterIP'] && (
                        <div className="flex items-start justify-between gap-3">
                          <dt className="text-xs font-medium text-[var(--gantry-text-secondary)] shrink-0">Cluster IP</dt>
                          <dd className="text-xs text-[var(--gantry-text-primary)] text-right font-mono">{anno['kubernetes.io/clusterIP']}</dd>
                        </div>
                      )}
                      {anno['kubernetes.io/phase'] && (
                        <div className="flex items-start justify-between gap-3">
                          <dt className="text-xs font-medium text-[var(--gantry-text-secondary)] shrink-0">Phase</dt>
                          <dd className="text-xs text-[var(--gantry-text-primary)] text-right font-mono">{anno['kubernetes.io/phase']}</dd>
                        </div>
                      )}
                    </dl>
                  </div>
                );
              })()}

              {/* Links card — shown when entity has spec.links or spec.repoUrl */}
              {(() => {
                const links = (entity.spec?.links as EntityLink[] | undefined) ?? [];
                const repoUrl = entity.spec?.repoUrl as string | undefined;
                if (links.length === 0 && !repoUrl) return null;

                const allLinks: EntityLink[] = [];
                if (repoUrl) {
                  const isGitHub = repoUrl.includes('github.com');
                  allLinks.push({ title: isGitHub ? 'GitHub Repository' : 'Repository', url: repoUrl, icon: isGitHub ? 'github' : 'other' });
                }
                allLinks.push(...links);

                return (
                  <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
                    <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Links</h3>
                    <ul className="mt-3 space-y-2">
                      {allLinks.map((link, i) => (
                        <li key={i}>
                          <a
                            href={link.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="flex items-center gap-2 text-sm text-[var(--gantry-accent)] hover:underline"
                          >
                            <span className="shrink-0 text-[var(--gantry-text-secondary)]">
                              {LINK_ICONS[link.icon ?? 'other'] ?? <ExternalLink className="h-3.5 w-3.5" />}
                            </span>
                            <span className="truncate">{link.title}</span>
                            <ExternalLink className="ml-auto h-3 w-3 shrink-0 text-[var(--gantry-text-secondary)]" />
                          </a>
                        </li>
                      ))}
                    </ul>
                  </div>
                );
              })()}

              {/* User-defined annotations (kubernetes.io/* and github.com/* filtered out) */}
              {(() => {
                const userAnnotations = Object.entries(entity.metadata.annotations ?? {}).filter(
                  ([k]) =>
                    !k.startsWith('kubernetes.io/') &&
                    !k.startsWith('deployment.kubernetes.io/') &&
                    !k.startsWith('github.com/')
                );
                if (userAnnotations.length === 0) return null;
                return (
                  <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
                    <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Annotations</h3>
                    <dl className="mt-3 space-y-2">
                      {userAnnotations.map(([k, v]) => (
                        <div key={k}>
                          <dt className="truncate text-xs font-medium text-[var(--gantry-text-secondary)]">{k}</dt>
                          <dd className="mt-0.5 truncate text-xs text-[var(--gantry-text-primary)]">{v}</dd>
                        </div>
                      ))}
                    </dl>
                  </div>
                );
              })()}
            </div>
          </div>
        )}

        {tab === 'yaml' && (
          <pre className="overflow-auto rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6 text-sm text-[var(--gantry-text-primary)]">
            {entityToYaml(entity)}
          </pre>
        )}

        {tab === 'activity' && (
          <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)]">
            {activity.length === 0 ? (
              <p className="px-6 py-8 text-center text-sm text-[var(--gantry-text-secondary)]">
                No activity recorded for this entity yet.
              </p>
            ) : (
              <table className="min-w-full divide-y divide-[var(--gantry-border)]">
                <thead>
                  <tr className="bg-[var(--gantry-bg-secondary)]">
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Time</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">User</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Action</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Source</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--gantry-border)]">
                  {activity.map((entry) => (
                    <tr key={entry.id} className="hover:bg-[var(--gantry-bg-secondary)]">
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-[var(--gantry-text-secondary)]">
                        {formatTime(entry.timestamp)}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-[var(--gantry-text-primary)]">
                        {entry.userName || 'system'}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3">
                        <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${ACTION_COLORS[entry.action] ?? 'bg-[var(--gantry-bg-tertiary)] text-[var(--gantry-text-secondary)]'}`}>
                          {entry.action}
                        </span>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-[var(--gantry-text-secondary)]">
                        {entry.source || '—'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}

        {tab === 'kubernetes' && (
          <KubernetesTab entity={entity} />
        )}

        {tab === 'github' && (
          <GitHubTab entity={entity} />
        )}

        {tab === 'relationships' && (
          graphLoading ? (
            <div className="flex items-center justify-center py-16">
              <div className="h-7 w-7 animate-spin rounded-full border-2 border-[var(--gantry-accent)] border-t-transparent" />
            </div>
          ) : graphData && kind && name ? (
            <EntityGraph data={graphData} rootKind={kind} rootName={name} />
          ) : (
            <p className="py-8 text-center text-sm text-[var(--gantry-text-secondary)]">
              Could not load dependency graph.
            </p>
          )
        )}
      </div>

      {/* Edit Modal */}
      {editing && schema && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="mx-4 w-full max-w-lg rounded-xl bg-[var(--gantry-bg-primary)] p-6 shadow-xl">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-[var(--gantry-text-primary)]">Edit {entity.metadata.name}</h2>
              <button onClick={() => setEditing(false)} className="text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]">
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="mt-4 max-h-96 overflow-y-auto">
              <SchemaForm
                schema={schema}
                initialValues={entity.spec}
                onSubmit={handleUpdate}
                onCancel={() => setEditing(false)}
                submitLabel="Save"
              />
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      {showDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="mx-4 w-full max-w-sm rounded-xl bg-[var(--gantry-bg-primary)] p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-[var(--gantry-text-primary)]">Delete Entity</h2>
            <p className="mt-2 text-sm text-[var(--gantry-text-secondary)]">
              Are you sure you want to delete <strong>{entity.metadata.name}</strong>? This action cannot be undone.
            </p>
            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => setShowDelete(false)}
                className="rounded-lg border border-[var(--gantry-border)] px-4 py-2 text-sm text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-tertiary)]"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                className="rounded-lg bg-[var(--gantry-danger)] px-4 py-2 text-sm font-medium text-white hover:opacity-90"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
