import { useState, useEffect } from 'react';
import { Zap, Play, X, CheckCircle2, XCircle, Clock, Loader2 } from 'lucide-react';
import { api } from '../lib/api';
import type { Entity, ActionRun, JsonSchema } from '../lib/types';
import SchemaForm from '../components/SchemaForm';

const statusIcons: Record<string, React.ReactNode> = {
  pending: <Clock className="h-4 w-4 text-[var(--gantry-warning)]" />,
  running: <Loader2 className="h-4 w-4 animate-spin text-[var(--gantry-accent)]" />,
  success: <CheckCircle2 className="h-4 w-4 text-[var(--gantry-success)]" />,
  failed: <XCircle className="h-4 w-4 text-[var(--gantry-danger)]" />,
};

export default function Actions() {
  const [actions, setActions] = useState<Entity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [executing, setExecuting] = useState<Entity | null>(null);
  const [lastRun, setLastRun] = useState<ActionRun | null>(null);
  useEffect(() => {
    setLoading(true);
    api.listActions().then((data) => setActions(data || [])).catch((e) => setError(e.message)).finally(() => setLoading(false));
  }, []);

  const handleExecute = async (inputs: Record<string, any>) => {
    if (!executing) return;
    try {
      const run = await api.executeAction(executing.metadata.name, inputs);
      setLastRun(run);
      setExecuting(null);
    } catch (e: any) {
      setError(e.message);
    }
  };

  const getInputSchema = (action: Entity): JsonSchema => {
    const inputs = action.spec?.inputs;
    if (!inputs || !Array.isArray(inputs)) {
      return { type: 'object', properties: {} };
    }
    const properties: Record<string, JsonSchema> = {};
    const required: string[] = [];
    for (const input of inputs) {
      properties[input.name] = {
        type: input.type === 'entity-picker' ? 'string' : input.type === 'select' ? 'string' : input.type || 'string',
        title: input.title || input.name,
        description: input.description,
        enum: input.options?.map((o: any) => o.value || o),
        default: input.default,
      };
      if (input.required) required.push(input.name);
    }
    return { type: 'object', properties, required };
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--gantry-accent)] border-t-transparent" />
      </div>
    );
  }

  return (
    <div>
      <div>
        <h1 className="text-2xl font-bold text-[var(--gantry-text-primary)]">Actions</h1>
        <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">
          Self-service workflows you can trigger
        </p>
      </div>

      {error && (
        <div className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
          {error}
        </div>
      )}

      {lastRun && (
        <div className="mt-4 flex items-center gap-3 rounded-lg border border-green-200 bg-green-50 px-4 py-3 text-sm dark:border-green-800 dark:bg-green-900/20">
          {statusIcons[lastRun.status]}
          <span className="text-green-700 dark:text-green-400">
            Action <strong>{lastRun.actionName}</strong> triggered (status: {lastRun.status}, ID: {lastRun.id.slice(0, 8)})
          </span>
          <button onClick={() => setLastRun(null)} className="ml-auto text-green-600 hover:text-green-800 dark:text-green-400">
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {actions.map((action) => (
          <div
            key={action.metadata.name}
            className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-5"
          >
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-2">
                <div className="rounded-lg bg-[var(--gantry-accent)]/10 p-2">
                  <Zap className="h-5 w-5 text-[var(--gantry-accent)]" />
                </div>
                <div>
                  <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">{action.metadata.name}</h3>
                  {action.spec?.type && (
                    <span className="text-xs text-[var(--gantry-text-secondary)]">{action.spec.type}</span>
                  )}
                </div>
              </div>
            </div>
            {action.metadata.description && (
              <p className="mt-3 text-xs text-[var(--gantry-text-secondary)]">{action.metadata.description}</p>
            )}
            <button
              onClick={() => setExecuting(action)}
              className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg bg-[var(--gantry-accent)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--gantry-accent-hover)]"
            >
              <Play className="h-4 w-4" /> Execute
            </button>
          </div>
        ))}
        {actions.length === 0 && (
          <div className="col-span-full flex flex-col items-center justify-center rounded-lg border border-dashed border-[var(--gantry-border)] py-16">
            <Zap className="h-8 w-8 text-[var(--gantry-text-secondary)]" />
            <p className="mt-3 text-sm text-[var(--gantry-text-secondary)]">
              No actions defined yet. Create Action entities to see them here.
            </p>
          </div>
        )}
      </div>

      {/* Execute Modal */}
      {executing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="mx-4 w-full max-w-lg rounded-xl bg-[var(--gantry-bg-primary)] p-6 shadow-xl">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold text-[var(--gantry-text-primary)]">
                  Execute: {executing.metadata.title || executing.metadata.name}
                </h2>
                {executing.metadata.description && (
                  <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">{executing.metadata.description}</p>
                )}
              </div>
              <button onClick={() => setExecuting(null)} className="text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]">
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="mt-4 max-h-96 overflow-y-auto">
              <SchemaForm
                schema={getInputSchema(executing)}
                onSubmit={handleExecute}
                onCancel={() => setExecuting(null)}
                submitLabel="Execute"
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
