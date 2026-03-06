import { useState, useEffect, useRef } from 'react';
import { Zap, X, CheckCircle2, XCircle, Clock, Loader2 } from 'lucide-react';
import { api } from '../lib/api';
import type { Entity, ActionRun, JsonSchema } from '../lib/types';
import SchemaForm from '../components/SchemaForm';

const statusIcon: Record<string, React.ReactNode> = {
  pending: <Clock className="h-8 w-8 text-[var(--gantry-warning)]" />,
  running: <Loader2 className="h-8 w-8 animate-spin text-[var(--gantry-accent)]" />,
  success: <CheckCircle2 className="h-8 w-8 text-[var(--gantry-success)]" />,
  failed: <XCircle className="h-8 w-8 text-[var(--gantry-danger)]" />,
};

const statusLabel: Record<string, string> = {
  pending: 'Queued',
  running: 'Running…',
  success: 'Completed',
  failed: 'Failed',
};

export default function Actions() {
  const [actions, setActions] = useState<Entity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [executing, setExecuting] = useState<Entity | null>(null);
  const [activeRun, setActiveRun] = useState<ActionRun | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    setLoading(true);
    api.listActions()
      .then((data) => setActions(data || []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  // Poll active run until terminal state.
  useEffect(() => {
    if (!activeRun || activeRun.status === 'success' || activeRun.status === 'failed') {
      if (pollRef.current) clearInterval(pollRef.current);
      return;
    }
    pollRef.current = setInterval(async () => {
      try {
        const updated = await api.getActionRun(activeRun.actionName, activeRun.id);
        setActiveRun(updated);
        if (updated.status === 'success' || updated.status === 'failed') {
          clearInterval(pollRef.current!);
        }
      } catch {
        clearInterval(pollRef.current!);
      }
    }, 1500);
    return () => { if (pollRef.current) clearInterval(pollRef.current); };
  }, [activeRun?.id, activeRun?.status]);

  const handleExecute = async (inputs: Record<string, any>) => {
    if (!executing) return;
    try {
      const run = await api.executeAction(executing.metadata.name, inputs);
      setExecuting(null);
      setActiveRun(run);
    } catch (e: any) {
      setError(e.message);
    }
  };

  const closeRunModal = () => {
    if (pollRef.current) clearInterval(pollRef.current);
    setActiveRun(null);
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
        enum: input.options?.map((o: any) => String(o.value ?? o)),
        default: input.default,
      };
      if (input.required) required.push(input.name);
    }
    return { type: 'object', properties, required };
  };

  const parseOutputs = (outputs: string) => {
    try { return JSON.parse(outputs); } catch { return null; }
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

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {actions.map((action) => (
          <div
            key={action.metadata.name}
            className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-5"
          >
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
            {action.metadata.description && (
              <p className="mt-3 text-xs text-[var(--gantry-text-secondary)]">{action.metadata.description}</p>
            )}
            <button
              onClick={() => setExecuting(action)}
              className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg bg-[var(--gantry-accent)] px-4 py-2 text-sm font-medium text-[var(--gantry-bg-primary)] hover:bg-[var(--gantry-accent-hover)]"
            >
              <Zap className="h-4 w-4" /> Execute
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

      {/* Execute input modal */}
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

      {/* Run status modal */}
      {activeRun && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="mx-4 w-full max-w-lg rounded-xl bg-[var(--gantry-bg-primary)] p-6 shadow-xl">
            <div className="flex items-start justify-between">
              <h2 className="text-lg font-semibold text-[var(--gantry-text-primary)]">
                {activeRun.actionName}
              </h2>
              {(activeRun.status === 'success' || activeRun.status === 'failed') && (
                <button onClick={closeRunModal} className="text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]">
                  <X className="h-5 w-5" />
                </button>
              )}
            </div>

            {/* Status */}
            <div className="mt-6 flex flex-col items-center gap-3 py-4">
              {statusIcon[activeRun.status] ?? statusIcon.pending}
              <p className="text-base font-medium text-[var(--gantry-text-primary)]">
                {statusLabel[activeRun.status] ?? activeRun.status}
              </p>
              <p className="text-xs text-[var(--gantry-text-secondary)]">Run ID: {activeRun.id.slice(0, 8)}…</p>
            </div>

            {/* Error */}
            {activeRun.error && (
              <div className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
                {activeRun.error}
              </div>
            )}

            {/* Outputs */}
            {activeRun.outputs && (
              <div className="mt-4">
                <p className="mb-1 text-xs font-medium text-[var(--gantry-text-secondary)]">Output</p>
                <pre className="overflow-auto rounded-lg bg-[var(--gantry-bg-tertiary)] p-3 text-xs text-[var(--gantry-text-primary)]">
                  {parseOutputs(activeRun.outputs)
                    ? JSON.stringify(parseOutputs(activeRun.outputs), null, 2)
                    : activeRun.outputs}
                </pre>
              </div>
            )}

            {/* Timing */}
            {activeRun.completedAt && (
              <p className="mt-4 text-center text-xs text-[var(--gantry-text-secondary)]">
                Completed {new Date(activeRun.completedAt).toLocaleTimeString()}
              </p>
            )}

            {(activeRun.status === 'success' || activeRun.status === 'failed') && (
              <button
                onClick={closeRunModal}
                className="mt-6 w-full rounded-lg border border-[var(--gantry-border)] py-2 text-sm font-medium text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-tertiary)]"
              >
                Close
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
