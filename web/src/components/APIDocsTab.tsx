import { useState, useEffect, useCallback } from 'react';
import { RefreshCw, CheckCircle2, XCircle, Clock, ExternalLink, FileText } from 'lucide-react';
import { api } from '../lib/api';
import type { Entity } from '../lib/types';

interface HealthResult {
  reachable: boolean;
  statusCode?: number;
  latencyMs: number;
  body?: string;
  error?: string;
}

interface Props {
  entity: Entity;
}

export default function APIDocsTab({ entity }: Props) {
  const apiDocsUrl = (entity.spec?.apiDocsUrl as string) || '';
  const definitionUrl = (entity.spec?.definition as string) || '';
  const healthCheckUrl = (entity.spec?.healthCheckUrl as string) || '';

  const [health, setHealth] = useState<HealthResult | null>(null);
  const [healthLoading, setHealthLoading] = useState(false);
  const [healthHistory, setHealthHistory] = useState<HealthResult[]>([]);
  const [autoRefresh, setAutoRefresh] = useState(true);

  const checkHealth = useCallback(async () => {
    if (!healthCheckUrl) return;
    setHealthLoading(true);
    try {
      const result = await api.checkHealth(healthCheckUrl);
      setHealth(result);
      setHealthHistory((prev) => [result, ...prev].slice(0, 10));
    } catch (e: any) {
      const errResult: HealthResult = { reachable: false, latencyMs: 0, error: e.message };
      setHealth(errResult);
      setHealthHistory((prev) => [errResult, ...prev].slice(0, 10));
    } finally {
      setHealthLoading(false);
    }
  }, [healthCheckUrl]);

  // Initial check + auto-refresh every 30s
  useEffect(() => {
    if (!healthCheckUrl) return;
    checkHealth();
    if (!autoRefresh) return;
    const interval = setInterval(checkHealth, 30_000);
    return () => clearInterval(interval);
  }, [healthCheckUrl, autoRefresh, checkHealth]);

  const docsUrl = apiDocsUrl || definitionUrl;

  return (
    <div className="space-y-6">
      {/* Health Check Section */}
      {healthCheckUrl && (
        <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)]">
          <div className="flex items-center justify-between border-b border-[var(--gantry-border)] px-6 py-4">
            <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Health Status</h3>
            <div className="flex items-center gap-3">
              <label className="flex items-center gap-1.5 text-xs text-[var(--gantry-text-secondary)]">
                <input
                  type="checkbox"
                  checked={autoRefresh}
                  onChange={(e) => setAutoRefresh(e.target.checked)}
                  className="rounded"
                />
                Auto-refresh
              </label>
              <button
                onClick={checkHealth}
                disabled={healthLoading}
                className="flex items-center gap-1.5 rounded-lg border border-[var(--gantry-border)] px-2.5 py-1.5 text-xs text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-tertiary)] disabled:opacity-50"
              >
                <RefreshCw className={`h-3.5 w-3.5 ${healthLoading ? 'animate-spin' : ''}`} />
                Check Now
              </button>
            </div>
          </div>

          <div className="p-6">
            {/* Current Status */}
            {health ? (
              <div className="space-y-4">
                <div className="flex items-center gap-6">
                  {/* Status Badge */}
                  <div className={`flex items-center gap-2 rounded-lg px-4 py-3 ${
                    health.reachable
                      ? 'bg-green-50 dark:bg-green-900/20'
                      : 'bg-red-50 dark:bg-red-900/20'
                  }`}>
                    {health.reachable ? (
                      <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
                    ) : (
                      <XCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
                    )}
                    <div>
                      <div className={`text-sm font-semibold ${
                        health.reachable
                          ? 'text-green-700 dark:text-green-300'
                          : 'text-red-700 dark:text-red-300'
                      }`}>
                        {health.reachable ? 'Healthy' : 'Unhealthy'}
                      </div>
                      <div className="text-xs text-[var(--gantry-text-secondary)]">
                        {health.statusCode ? `HTTP ${health.statusCode}` : 'Unreachable'}
                      </div>
                    </div>
                  </div>

                  {/* Latency */}
                  <div className="flex items-center gap-2">
                    <Clock className="h-4 w-4 text-[var(--gantry-text-secondary)]" />
                    <div>
                      <div className="text-sm font-medium text-[var(--gantry-text-primary)]">{health.latencyMs}ms</div>
                      <div className="text-xs text-[var(--gantry-text-secondary)]">Response time</div>
                    </div>
                  </div>

                  {/* Endpoint */}
                  <div className="ml-auto text-right">
                    <div className="text-xs text-[var(--gantry-text-secondary)]">Endpoint</div>
                    <a
                      href={healthCheckUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1 text-xs text-[var(--gantry-accent)] hover:underline"
                    >
                      {healthCheckUrl}
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  </div>
                </div>

                {/* Error message */}
                {health.error && (
                  <div className="rounded-md bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-400">
                    {health.error}
                  </div>
                )}

                {/* Response body preview */}
                {health.body && (
                  <details className="group">
                    <summary className="cursor-pointer text-xs font-medium text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]">
                      Response Body
                    </summary>
                    <pre className="mt-2 overflow-auto rounded-md bg-[var(--gantry-bg-tertiary)] p-3 text-xs text-[var(--gantry-text-primary)]">
                      {(() => {
                        try { return JSON.stringify(JSON.parse(health.body), null, 2); } catch { return health.body; }
                      })()}
                    </pre>
                  </details>
                )}

                {/* History */}
                {healthHistory.length > 1 && (
                  <div>
                    <h4 className="mb-2 text-xs font-medium text-[var(--gantry-text-secondary)]">Recent Checks</h4>
                    <div className="flex items-center gap-1">
                      {healthHistory.map((h, i) => (
                        <div
                          key={i}
                          title={`${h.reachable ? 'OK' : 'FAIL'} — ${h.latencyMs}ms${h.statusCode ? ` (${h.statusCode})` : ''}`}
                          className={`h-6 w-3 rounded-sm ${
                            h.reachable
                              ? 'bg-green-500 dark:bg-green-400'
                              : 'bg-red-500 dark:bg-red-400'
                          } ${i === 0 ? 'opacity-100' : 'opacity-60'}`}
                        />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ) : healthLoading ? (
              <div className="flex items-center justify-center py-8">
                <div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--gantry-accent)] border-t-transparent" />
              </div>
            ) : null}
          </div>
        </div>
      )}

      {/* API Documentation Section */}
      {docsUrl ? (
        <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)]">
          <div className="flex items-center justify-between border-b border-[var(--gantry-border)] px-6 py-4">
            <div className="flex items-center gap-2">
              <FileText className="h-4 w-4 text-[var(--gantry-text-secondary)]" />
              <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">API Documentation</h3>
            </div>
            <a
              href={docsUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 rounded-lg border border-[var(--gantry-border)] px-2.5 py-1.5 text-xs text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-tertiary)]"
            >
              Open in new tab
              <ExternalLink className="h-3.5 w-3.5" />
            </a>
          </div>
          <div className="relative">
            <iframe
              src={docsUrl}
              title="API Documentation"
              className="h-[700px] w-full border-0"
              sandbox="allow-scripts allow-same-origin allow-popups allow-forms"
            />
          </div>
        </div>
      ) : (
        !healthCheckUrl && (
          <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] px-6 py-12 text-center">
            <FileText className="mx-auto h-10 w-10 text-[var(--gantry-text-secondary)] opacity-40" />
            <p className="mt-3 text-sm text-[var(--gantry-text-secondary)]">
              No API documentation or health check URL configured.
            </p>
            <p className="mt-1 text-xs text-[var(--gantry-text-secondary)]">
              Set <code className="rounded bg-[var(--gantry-bg-tertiary)] px-1.5 py-0.5">apiDocsUrl</code> or{' '}
              <code className="rounded bg-[var(--gantry-bg-tertiary)] px-1.5 py-0.5">healthCheckUrl</code> in the entity spec.
            </p>
          </div>
        )
      )}
    </div>
  );
}
