import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { ChevronRight, Pencil, Trash2, X, ExternalLink } from 'lucide-react';
import { api } from '../lib/api';
import type { Entity, JsonSchema } from '../lib/types';
import SchemaForm from '../components/SchemaForm';

type Tab = 'overview' | 'yaml' | 'relationships';

export default function EntityDetail() {
  const { kind, name } = useParams<{ kind: string; name: string }>();
  const navigate = useNavigate();
  const [entity, setEntity] = useState<Entity | null>(null);
  const [schema, setSchema] = useState<JsonSchema | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<Tab>('overview');
  const [editing, setEditing] = useState(false);
  const [showDelete, setShowDelete] = useState(false);

  useEffect(() => {
    if (!kind || !name) return;
    setLoading(true);
    Promise.all([
      api.getEntity(kind, name),
      api.getSchema(kind).catch(() => null),
    ])
      .then(([e, s]) => {
        setEntity(e);
        setSchema(s);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [kind, name]);

  const handleUpdate = async (spec: Record<string, any>) => {
    if (!entity || !kind || !name) return;
    try {
      const updated = await api.updateEntity(kind, name, {
        ...entity,
        spec,
      });
      setEntity(updated);
      setEditing(false);
    } catch (e: any) {
      setError(e.message);
    }
  };

  const handleDelete = async () => {
    if (!kind || !name) return;
    try {
      await api.deleteEntity(kind, name);
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

  const deps = (entity.spec?.dependsOn as Array<{ kind: string; name: string }>) || [];
  const providesApis = (entity.spec?.providesApis as string[]) || [];
  const consumesApis = (entity.spec?.consumesApis as string[]) || [];
  const hasRelationships = deps.length > 0 || providesApis.length > 0 || consumesApis.length > 0;

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
      </div>

      {/* Tabs */}
      <div className="mt-6 flex gap-1 border-b border-[var(--gantry-border)]">
        {(['overview', 'yaml', ...(hasRelationships ? ['relationships'] : [])] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`border-b-2 px-4 py-2 text-sm font-medium capitalize transition-colors ${
              tab === t
                ? 'border-[var(--gantry-accent)] text-[var(--gantry-accent)]'
                : 'border-transparent text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]'
            }`}
          >
            {t}
          </button>
        ))}
      </div>

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
                    {Object.entries(entity.spec).map(([key, value]) => (
                      <div key={key}>
                        <dt className="text-xs font-medium text-[var(--gantry-text-secondary)]">{key}</dt>
                        <dd className="mt-0.5 text-sm text-[var(--gantry-text-primary)]">
                          {typeof value === 'object' ? (
                            <pre className="mt-1 overflow-auto rounded-md bg-[var(--gantry-bg-tertiary)] p-2 text-xs">
                              {JSON.stringify(value, null, 2)}
                            </pre>
                          ) : (
                            String(value)
                          )}
                        </dd>
                      </div>
                    ))}
                  </dl>
                ) : (
                  <p className="mt-4 text-sm text-[var(--gantry-text-secondary)]">No spec fields defined.</p>
                )}
              </div>
            </div>
            {/* Metadata sidebar */}
            <div>
              <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
                <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Metadata</h3>
                <dl className="mt-4 space-y-3">
                  <div>
                    <dt className="text-xs font-medium text-[var(--gantry-text-secondary)]">Namespace</dt>
                    <dd className="mt-0.5 text-sm text-[var(--gantry-text-primary)]">{entity.metadata.namespace || 'default'}</dd>
                  </div>
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
                {entity.metadata.annotations && Object.keys(entity.metadata.annotations).length > 0 && (
                  <>
                    <h3 className="mt-6 text-sm font-semibold text-[var(--gantry-text-primary)]">Annotations</h3>
                    <dl className="mt-2 space-y-2">
                      {Object.entries(entity.metadata.annotations).map(([k, v]) => (
                        <div key={k}>
                          <dt className="truncate text-xs font-medium text-[var(--gantry-text-secondary)]">{k}</dt>
                          <dd className="mt-0.5 truncate text-xs text-[var(--gantry-text-primary)]">{v}</dd>
                        </div>
                      ))}
                    </dl>
                  </>
                )}
              </div>
            </div>
          </div>
        )}

        {tab === 'yaml' && (
          <pre className="overflow-auto rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6 text-sm text-[var(--gantry-text-primary)]">
            {JSON.stringify(entity, null, 2)}
          </pre>
        )}

        {tab === 'relationships' && (
          <div className="space-y-6">
            {deps.length > 0 && (
              <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
                <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Depends On</h3>
                <ul className="mt-3 space-y-2">
                  {deps.map((dep, i) => (
                    <li key={i}>
                      <Link
                        to={`/catalog/${dep.kind}/${dep.name}`}
                        className="flex items-center gap-2 text-sm text-[var(--gantry-accent)] hover:underline"
                      >
                        <ExternalLink className="h-3.5 w-3.5" />
                        {dep.kind}/{dep.name}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {providesApis.length > 0 && (
              <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
                <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Provides APIs</h3>
                <ul className="mt-3 space-y-2">
                  {providesApis.map((a) => (
                    <li key={a}>
                      <Link to={`/catalog/API/${a}`} className="flex items-center gap-2 text-sm text-[var(--gantry-accent)] hover:underline">
                        <ExternalLink className="h-3.5 w-3.5" />{a}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {consumesApis.length > 0 && (
              <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
                <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Consumes APIs</h3>
                <ul className="mt-3 space-y-2">
                  {consumesApis.map((a) => (
                    <li key={a}>
                      <Link to={`/catalog/API/${a}`} className="flex items-center gap-2 text-sm text-[var(--gantry-accent)] hover:underline">
                        <ExternalLink className="h-3.5 w-3.5" />{a}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
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
