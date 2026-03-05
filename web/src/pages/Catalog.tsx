import { useState, useEffect, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Search, LayoutGrid, List, Plus, X } from 'lucide-react';
import { api } from '../lib/api';
import { ENTITY_KINDS } from '../lib/types';
import type { Entity, JsonSchema } from '../lib/types';
import EntityCard from '../components/EntityCard';
import EntityTable from '../components/EntityTable';
import SchemaForm from '../components/SchemaForm';

export default function Catalog() {
  const { kind } = useParams<{ kind?: string }>();
  const navigate = useNavigate();
  const [entities, setEntities] = useState<Entity[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [view, setView] = useState<'grid' | 'table'>('table');
  const [showCreate, setShowCreate] = useState(false);
  const [createKind, setCreateKind] = useState('Service');
  const [schemas, setSchemas] = useState<Record<string, JsonSchema>>({});
  const [error, setError] = useState('');

  useEffect(() => {
    setLoading(true);
    setError('');
    api.listEntities(kind).then((data) => setEntities(data || [])).catch((e) => setError(e.message)).finally(() => setLoading(false));
  }, [kind]);

  useEffect(() => {
    api.listSchemas().then((data) => setSchemas(data || {})).catch(() => {});
  }, []);

  const filtered = useMemo(() => {
    if (!searchQuery) return entities;
    const q = searchQuery.toLowerCase();
    return entities.filter(
      (e) =>
        e.metadata.name.toLowerCase().includes(q) ||
        (e.metadata.title || '').toLowerCase().includes(q) ||
        (e.metadata.owner || '').toLowerCase().includes(q) ||
        (e.metadata.tags || []).some((t) => t.toLowerCase().includes(q)),
    );
  }, [entities, searchQuery]);

  const handleCreate = async (spec: Record<string, any>) => {
    try {
      const name = (spec._name as string) || '';
      const title = (spec._title as string) || '';
      const owner = (spec._owner as string) || '';
      const description = (spec._description as string) || '';
      delete spec._name;
      delete spec._title;
      delete spec._owner;
      delete spec._description;

      const newEntity: Entity = {
        kind: createKind,
        apiVersion: 'gantry.io/v1',
        metadata: { name, title, owner, description },
        spec,
      };
      const created = await api.createEntity(newEntity);
      setEntities((prev) => [...prev, created]);
      setShowCreate(false);
    } catch (e: any) {
      setError(e.message);
    }
  };

  const createSchema: JsonSchema = useMemo(() => {
    const kindSchema = schemas[createKind.toLowerCase()] || { type: 'object', properties: {} };
    return {
      type: 'object',
      properties: {
        _name: { type: 'string', title: 'Name', description: 'Unique identifier' },
        _title: { type: 'string', title: 'Title', description: 'Display name' },
        _owner: { type: 'string', title: 'Owner', description: 'Team or user that owns this entity' },
        _description: { type: 'string', title: 'Description' },
        ...(kindSchema as any).properties,
      },
      required: ['_name'],
    };
  }, [schemas, createKind]);

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-[var(--gantry-text-primary)]">
            {kind ? `${kind}s` : 'Catalog'}
          </h1>
          <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">
            {kind ? `All ${kind} entities` : 'Browse all entities in the catalog'}
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 rounded-lg bg-[var(--gantry-accent)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--gantry-accent-hover)]"
        >
          <Plus className="h-4 w-4" />
          Create Entity
        </button>
      </div>

      {/* Filters */}
      <div className="mt-6 flex items-center gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--gantry-text-secondary)]" />
          <input
            type="text"
            placeholder="Search entities..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] py-2 pl-10 pr-4 text-sm text-[var(--gantry-text-primary)] placeholder:text-[var(--gantry-text-secondary)] focus:border-[var(--gantry-accent)] focus:outline-none focus:ring-1 focus:ring-[var(--gantry-accent)]"
          />
        </div>
        <div className="flex items-center rounded-lg border border-[var(--gantry-border)]">
          <button
            onClick={() => setView('table')}
            className={`rounded-l-lg p-2 ${view === 'table' ? 'bg-[var(--gantry-bg-tertiary)] text-[var(--gantry-accent)]' : 'text-[var(--gantry-text-secondary)]'}`}
          >
            <List className="h-4 w-4" />
          </button>
          <button
            onClick={() => setView('grid')}
            className={`rounded-r-lg p-2 ${view === 'grid' ? 'bg-[var(--gantry-bg-tertiary)] text-[var(--gantry-accent)]' : 'text-[var(--gantry-text-secondary)]'}`}
          >
            <LayoutGrid className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Kind tabs */}
      <div className="mt-4 flex gap-1 overflow-x-auto border-b border-[var(--gantry-border)]">
        <button
          onClick={() => navigate('/catalog')}
          className={`whitespace-nowrap border-b-2 px-4 py-2 text-sm font-medium transition-colors ${
            !kind
              ? 'border-[var(--gantry-accent)] text-[var(--gantry-accent)]'
              : 'border-transparent text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]'
          }`}
        >
          All
        </button>
        {ENTITY_KINDS.map((k) => (
          <button
            key={k.name}
            onClick={() => navigate(`/catalog/${k.name}`)}
            className={`whitespace-nowrap border-b-2 px-4 py-2 text-sm font-medium transition-colors ${
              kind === k.name
                ? 'border-[var(--gantry-accent)] text-[var(--gantry-accent)]'
                : 'border-transparent text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]'
            }`}
          >
            {k.name}
          </button>
        ))}
      </div>

      {/* Error */}
      {error && (
        <div className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Content */}
      <div className="mt-6">
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--gantry-accent)] border-t-transparent" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-[var(--gantry-border)] py-16">
            <p className="text-sm text-[var(--gantry-text-secondary)]">
              {searchQuery ? 'No entities match your search.' : 'No entities yet.'}
            </p>
            {!searchQuery && (
              <button
                onClick={() => setShowCreate(true)}
                className="mt-3 flex items-center gap-2 rounded-lg border border-[var(--gantry-border)] px-4 py-2 text-sm text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-tertiary)]"
              >
                <Plus className="h-4 w-4" />
                Create your first entity
              </button>
            )}
          </div>
        ) : view === 'table' ? (
          <EntityTable entities={filtered} />
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filtered.map((e) => (
              <EntityCard key={`${e.kind}-${e.metadata.name}`} entity={e} />
            ))}
          </div>
        )}
      </div>

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="mx-4 w-full max-w-lg rounded-xl bg-[var(--gantry-bg-primary)] p-6 shadow-xl">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-[var(--gantry-text-primary)]">Create Entity</h2>
              <button onClick={() => setShowCreate(false)} className="text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]">
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="mt-4">
              <label className="block text-sm font-medium text-[var(--gantry-text-primary)]">Kind</label>
              <select
                value={createKind}
                onChange={(e) => setCreateKind(e.target.value)}
                className="mt-1 w-full rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] px-3 py-2 text-sm text-[var(--gantry-text-primary)] focus:border-[var(--gantry-accent)] focus:outline-none"
              >
                {ENTITY_KINDS.map((k) => (
                  <option key={k.name} value={k.name}>{k.name}</option>
                ))}
              </select>
            </div>
            <div className="mt-4 max-h-96 overflow-y-auto">
              <SchemaForm
                schema={createSchema}
                onSubmit={handleCreate}
                onCancel={() => setShowCreate(false)}
                submitLabel="Create"
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
