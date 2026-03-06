import { useState, useEffect, useRef } from 'react';
import { ChevronDown, ChevronRight, Terminal, X, RefreshCw } from 'lucide-react';
import { api, getToken } from '../lib/api';
import type { Entity, K8sWorkloadInfo, K8sPodInfo, K8sContainerInfo } from '../lib/types';

interface LogTarget {
  namespace: string;
  pod: string;
  container: string;
}

const PHASE_COLORS: Record<string, string> = {
  Running: 'text-green-600 dark:text-green-400',
  Succeeded: 'text-blue-600 dark:text-blue-400',
  Pending: 'text-yellow-600 dark:text-yellow-400',
  Failed: 'text-red-600 dark:text-red-400',
  Unknown: 'text-[var(--gantry-text-secondary)]',
};

const STATE_BADGE: Record<string, string> = {
  running: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  waiting: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
  terminated: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
  unknown: 'bg-[var(--gantry-bg-tertiary)] text-[var(--gantry-text-secondary)]',
};

function formatAge(startTime?: string): string {
  if (!startTime) return '—';
  const ms = Date.now() - new Date(startTime).getTime();
  const m = Math.floor(ms / 60_000);
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h`;
  return `${Math.floor(h / 24)}d`;
}

function LogViewer({ target, onClose }: { target: LogTarget; onClose: () => void }) {
  const [lines, setLines] = useState<string[]>([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);
  const controllerRef = useRef<AbortController | null>(null);

  function connect() {
    if (controllerRef.current) controllerRef.current.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    setLines([]);
    setError('');
    setConnected(true);

    const token = getToken();
    const url = `/api/v1/plugins/kubernetes/pods/${target.namespace}/${target.pod}/containers/${target.container}/logs`;

    fetch(url, {
      headers: { Authorization: `Bearer ${token}` },
      signal: controller.signal,
    })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const reader = res.body!.getReader();
        const decoder = new TextDecoder();
        let buf = '';

        function read() {
          reader
            .read()
            .then(({ done, value }) => {
              if (done) { setConnected(false); return; }
              buf += decoder.decode(value, { stream: true });
              const parts = buf.split('\n');
              buf = parts.pop() ?? '';
              const newLines = parts.filter((l) => l !== '');
              if (newLines.length > 0) {
                setLines((prev) => [...prev, ...newLines].slice(-1000));
              }
              read();
            })
            .catch(() => setConnected(false));
        }
        read();
      })
      .catch((e) => {
        if (e.name !== 'AbortError') setError(e.message);
        setConnected(false);
      });
  }

  useEffect(() => {
    connect();
    return () => controllerRef.current?.abort();
  }, [target.namespace, target.pod, target.container]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [lines]);

  return (
    <div className="mt-4 rounded-lg border border-[var(--gantry-border)] bg-gray-950 dark:bg-black overflow-hidden">
      {/* Log header */}
      <div className="flex items-center justify-between border-b border-gray-800 px-4 py-2">
        <div className="flex items-center gap-2">
          <Terminal className="h-4 w-4 text-gray-400" />
          <span className="text-xs font-mono text-gray-300">
            {target.pod} / {target.container}
          </span>
          <span className={`h-2 w-2 rounded-full ${connected ? 'bg-green-500 animate-pulse' : 'bg-gray-500'}`} />
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={connect}
            title="Reconnect"
            className="rounded p-1 text-gray-400 hover:text-gray-200"
          >
            <RefreshCw className="h-3.5 w-3.5" />
          </button>
          <button onClick={onClose} className="rounded p-1 text-gray-400 hover:text-gray-200">
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
      {/* Log body */}
      <div className="h-72 overflow-y-auto p-4 font-mono text-xs text-gray-300 leading-5">
        {error && <p className="text-red-400">{error}</p>}
        {lines.length === 0 && !error && (
          <p className="text-gray-500">{connected ? 'Waiting for logs…' : 'No logs.'}</p>
        )}
        {lines.map((line, i) => (
          <div key={i} className="whitespace-pre-wrap break-all">
            {line}
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}

function PodRow({ pod }: { pod: K8sPodInfo }) {
  const [expanded, setExpanded] = useState(false);
  const [logTarget, setLogTarget] = useState<LogTarget | null>(null);

  return (
    <>
      <tr
        className="cursor-pointer hover:bg-[var(--gantry-bg-secondary)]"
        onClick={() => setExpanded((v) => !v)}
      >
        <td className="px-4 py-3">
          {expanded ? (
            <ChevronDown className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
          )}
        </td>
        <td className="px-4 py-3 font-mono text-xs text-[var(--gantry-text-primary)]">{pod.name}</td>
        <td className="px-4 py-3 text-xs text-[var(--gantry-text-secondary)]">{pod.namespace}</td>
        <td className="px-4 py-3">
          <span className={`text-xs font-medium ${PHASE_COLORS[pod.phase] ?? PHASE_COLORS.Unknown}`}>
            {pod.phase}
          </span>
        </td>
        <td className="px-4 py-3 text-xs text-[var(--gantry-text-secondary)]">
          {pod.containers.filter((c) => c.ready).length}/{pod.containers.length}
        </td>
        <td className="px-4 py-3 text-xs text-[var(--gantry-text-secondary)]">{pod.totalRestarts}</td>
        <td className="px-4 py-3 text-xs text-[var(--gantry-text-secondary)]">{formatAge(pod.startTime)}</td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={7} className="bg-[var(--gantry-bg-secondary)] px-8 pb-4 pt-2">
            <div className="space-y-2">
              {pod.nodeName && (
                <p className="text-xs text-[var(--gantry-text-secondary)]">
                  Node: <span className="font-mono text-[var(--gantry-text-primary)]">{pod.nodeName}</span>
                </p>
              )}
              <div className="space-y-1.5">
                {pod.containers.map((c: K8sContainerInfo) => (
                  <div
                    key={c.name}
                    className="flex items-center gap-3 rounded-md border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] px-3 py-2"
                  >
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATE_BADGE[c.state] ?? STATE_BADGE.unknown}`}
                    >
                      {c.state}
                      {c.reason ? ` (${c.reason})` : ''}
                    </span>
                    <span className="text-xs font-medium text-[var(--gantry-text-primary)]">{c.name}</span>
                    <span className="truncate text-xs text-[var(--gantry-text-secondary)]">{c.image}</span>
                    {c.restarts > 0 && (
                      <span className="ml-auto text-xs text-yellow-600 dark:text-yellow-400">
                        {c.restarts} restart{c.restarts !== 1 ? 's' : ''}
                      </span>
                    )}
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        setLogTarget(
                          logTarget?.container === c.name ? null : { namespace: pod.namespace, pod: pod.name, container: c.name }
                        );
                      }}
                      className="ml-auto flex items-center gap-1 rounded border border-[var(--gantry-border)] px-2 py-0.5 text-xs text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]"
                    >
                      <Terminal className="h-3 w-3" /> Logs
                    </button>
                  </div>
                ))}
              </div>
              {logTarget && (
                <LogViewer target={logTarget} onClose={() => setLogTarget(null)} />
              )}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

export default function KubernetesTab({ entity }: { entity: Entity }) {
  const [workload, setWorkload] = useState<K8sWorkloadInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const namespaces: string[] = ((entity.spec?.deployedIn as any[]) ?? [])
    .filter((d: any) => d?.kind === 'Environment')
    .map((d: any) => d.name as string);

  useEffect(() => {
    setLoading(true);
    api
      .getKubernetesWorkload(entity.metadata.name, namespaces)
      .then(setWorkload)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [entity.metadata.name]);

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
      </div>
    );
  }

  if (!workload) return null;

  const hasPods = workload.pods.length > 0;

  return (
    <div className="space-y-6">
      {/* Deployments summary */}
      {workload.deployments.length > 0 && (
        <div>
          <h3 className="mb-3 text-sm font-semibold text-[var(--gantry-text-primary)]">Deployments</h3>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {workload.deployments.map((dep) => (
              <div
                key={`${dep.namespace}/${dep.name}`}
                className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-4"
              >
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-[var(--gantry-text-primary)]">{dep.name}</span>
                  <span className="rounded-full bg-[var(--gantry-bg-tertiary)] px-2 py-0.5 text-xs text-[var(--gantry-text-secondary)]">
                    {dep.namespace}
                  </span>
                </div>
                <div className="mt-3 flex items-end gap-1">
                  <span
                    className={`text-2xl font-bold ${
                      dep.readyReplicas === dep.desiredReplicas && dep.desiredReplicas > 0
                        ? 'text-green-600 dark:text-green-400'
                        : 'text-yellow-600 dark:text-yellow-400'
                    }`}
                  >
                    {dep.readyReplicas}
                  </span>
                  <span className="mb-1 text-sm text-[var(--gantry-text-secondary)]">
                    / {dep.desiredReplicas} ready
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Pods table */}
      <div>
        <h3 className="mb-3 text-sm font-semibold text-[var(--gantry-text-primary)]">
          Pods
          {hasPods && (
            <span className="ml-2 text-xs font-normal text-[var(--gantry-text-secondary)]">
              — click a row to expand containers and view logs
            </span>
          )}
        </h3>
        {!hasPods ? (
          <p className="text-sm text-[var(--gantry-text-secondary)]">No pods found.</p>
        ) : (
          <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] overflow-hidden">
            <table className="min-w-full divide-y divide-[var(--gantry-border)]">
              <thead>
                <tr className="bg-[var(--gantry-bg-secondary)]">
                  <th className="w-8 px-4 py-3" />
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Pod</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Namespace</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Status</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Ready</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Restarts</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">Age</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--gantry-border)]">
                {workload.pods.map((pod) => (
                  <PodRow key={`${pod.namespace}/${pod.name}`} pod={pod} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
