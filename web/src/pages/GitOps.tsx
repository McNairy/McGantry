import { useState, useEffect, useCallback } from 'react';
import {
  RefreshCw,
  GitBranch,
  ArrowUpCircle,
  ArrowDownCircle,
  CheckCircle2,
  AlertCircle,
  Loader2,
  FolderGit2,
  Clock,
  FileText,
  XCircle,
  GitPullRequest,
  ChevronRight,
  ChevronDown,
  Folder,
  FolderOpen,
  FileCode,
  X,
} from 'lucide-react';
import { api } from '../lib/api';
import type { GitOpsStatus, GitOpsSyncEntry, GitOpsFileEntry } from '../lib/types';

function relativeTime(ts: string): string {
  const diff = Date.now() - new Date(ts).getTime();
  if (diff < 60_000) return 'just now';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

interface FileTreeNodeProps {
  kind: string;
  files: GitOpsFileEntry[];
  onSelectFile: (file: GitOpsFileEntry) => void;
  selectedPath: string | null;
}

function FileTreeNode({ kind, files, onSelectFile, selectedPath }: FileTreeNodeProps) {
  const [expanded, setExpanded] = useState(true);
  const hasErrors = files.some(f => f.error);

  return (
    <div>
      <button
        onClick={() => setExpanded(e => !e)}
        className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm font-medium transition-colors hover:bg-[var(--gantry-bg-tertiary)]"
      >
        {expanded ? (
          <ChevronDown className="h-3.5 w-3.5 shrink-0 text-[var(--gantry-text-secondary)]" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 shrink-0 text-[var(--gantry-text-secondary)]" />
        )}
        {expanded ? (
          <FolderOpen className="h-4 w-4 shrink-0 text-[var(--gantry-accent)]" />
        ) : (
          <Folder className="h-4 w-4 shrink-0 text-[var(--gantry-accent)]" />
        )}
        <span className="flex-1 text-[var(--gantry-text-primary)]">{kind}</span>
        <span className="text-xs text-[var(--gantry-text-secondary)]">{files.length}</span>
        {hasErrors && (
          <AlertCircle className="h-3.5 w-3.5 text-[var(--gantry-danger)]" />
        )}
      </button>
      {expanded && (
        <div className="ml-6 mt-0.5 space-y-0.5 border-l border-[var(--gantry-border)] pl-2">
          {files.map(file => (
            <button
              key={file.path}
              onClick={() => onSelectFile(file)}
              className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs transition-colors hover:bg-[var(--gantry-bg-tertiary)] ${
                selectedPath === file.path ? 'bg-[var(--gantry-accent)]/10 text-[var(--gantry-accent)]' : 'text-[var(--gantry-text-secondary)]'
              }`}
            >
              <FileCode className="h-3.5 w-3.5 shrink-0" />
              <span className="flex-1 truncate font-mono">{file.name}</span>
              {file.error && (
                <AlertCircle className="h-3 w-3 shrink-0 text-[var(--gantry-danger)]" />
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

export default function GitOps() {
  const [status, setStatus] = useState<GitOpsStatus | null>(null);
  const [history, setHistory] = useState<GitOpsSyncEntry[]>([]);
  const [files, setFiles] = useState<GitOpsFileEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refreshing, setRefreshing] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [tab, setTab] = useState<'history' | 'files'>('history');
  const [expandedErrorId, setExpandedErrorId] = useState<string | null>(null);
  const [selectedFile, setSelectedFile] = useState<GitOpsFileEntry | null>(null);
  const [fileContent, setFileContent] = useState<string>('');
  const [loadingContent, setLoadingContent] = useState(false);

  const fetchData = useCallback(async (showRefresh = false) => {
    if (showRefresh) setRefreshing(true);
    try {
      const [s, h, f] = await Promise.all([
        api.getGitOpsStatus(),
        api.getGitOpsHistory(),
        api.getGitOpsFiles(),
      ]);
      setStatus(s);
      setHistory(h || []);
      setFiles(f || []);
      setError('');
    } catch (err: any) {
      setError(err.message || 'Failed to fetch GitOps status');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(() => fetchData(), 30_000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const handleSync = async () => {
    setSyncing(true);
    try {
      await api.triggerGitOpsBidisync();
      setTimeout(() => fetchData(), 2000);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSyncing(false);
    }
  };

  const handleSelectFile = async (file: GitOpsFileEntry) => {
    setSelectedFile(file);
    setFileContent('');
    setLoadingContent(true);
    try {
      const result = await api.getGitOpsFileContent(file.path);
      setFileContent(result.content);
    } catch (err: any) {
      setFileContent(`# Error loading file\n# ${err.message}`);
    } finally {
      setLoadingContent(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--gantry-accent)]" />
      </div>
    );
  }

  if (error && !status) {
    return (
      <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-8 text-center">
        <AlertCircle className="mx-auto mb-3 h-10 w-10 text-[var(--gantry-danger)]" />
        <h2 className="text-lg font-semibold text-[var(--gantry-text-primary)]">Unable to load GitOps status</h2>
        <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">{error}</p>
        <button
          onClick={() => { setLoading(true); setError(''); fetchData(); }}
          className="mt-4 rounded-lg bg-[var(--gantry-accent)] px-4 py-2 text-sm font-medium text-[var(--gantry-bg-primary)] hover:bg-[var(--gantry-accent-hover)]"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-[var(--gantry-text-primary)]">GitOps</h1>
          <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">
            Bidirectional Git sync for your entity catalog
          </p>
        </div>
        <button
          onClick={() => fetchData(true)}
          disabled={refreshing}
          className="flex items-center gap-2 rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] px-4 py-2 text-sm font-medium text-[var(--gantry-text-primary)] transition-colors hover:border-[var(--gantry-accent)] hover:text-[var(--gantry-accent)] disabled:opacity-50"
        >
          <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {/* Connection status banner */}
      {status?.connected ? (
        <div className="flex items-center gap-3 rounded-xl border border-green-500/30 bg-green-500/10 px-5 py-4">
          <CheckCircle2 className="h-5 w-5 shrink-0 text-green-500" />
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold text-[var(--gantry-text-primary)]">Connected</p>
            <p className="truncate text-xs text-[var(--gantry-text-secondary)]">
              {status.repoUrl} &middot; {status.branch || 'main'}
            </p>
          </div>
          {status.lastCommit && (
            <span className="shrink-0 rounded bg-[var(--gantry-bg-tertiary)] px-2 py-1 font-mono text-xs text-[var(--gantry-text-secondary)]">
              {status.lastCommit}
            </span>
          )}
        </div>
      ) : (
        <div className="flex items-center gap-3 rounded-xl border border-[var(--gantry-danger)]/30 bg-[var(--gantry-danger)]/10 px-5 py-4">
          <XCircle className="h-5 w-5 shrink-0 text-[var(--gantry-danger)]" />
          <div>
            <p className="text-sm font-semibold text-[var(--gantry-text-primary)]">Disconnected</p>
            <p className="text-xs text-[var(--gantry-text-secondary)]">
              {status?.lastError || 'Unable to connect to Git repository. Check plugin configuration.'}
            </p>
          </div>
        </div>
      )}

      {/* Error banner */}
      {status?.lastError && status?.connected && (
        <div className="rounded-xl border border-yellow-400/30 bg-yellow-400/10 px-5 py-4">
          <div className="flex items-start gap-3">
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-yellow-500" />
            <div className="min-w-0">
              <p className="text-sm font-semibold text-[var(--gantry-text-primary)]">Latest pull error</p>
              <p className="mt-1 whitespace-pre-wrap break-words font-mono text-xs text-[var(--gantry-text-secondary)]">
                {status.lastError}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Status cards + actions */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {/* Last push */}
        <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-4">
          <div className="flex items-center gap-2 text-[var(--gantry-text-secondary)]">
            <ArrowUpCircle className="h-4 w-4" />
            <span className="text-xs font-medium">Last Push</span>
          </div>
          <p className="mt-2 text-sm font-semibold text-[var(--gantry-text-primary)]">
            {status?.lastPushAt ? relativeTime(status.lastPushAt) : 'Never'}
          </p>
        </div>

        {/* Last pull */}
        <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-4">
          <div className="flex items-center gap-2 text-[var(--gantry-text-secondary)]">
            <ArrowDownCircle className="h-4 w-4" />
            <span className="text-xs font-medium">Last Pull</span>
          </div>
          <p className="mt-2 text-sm font-semibold text-[var(--gantry-text-primary)]">
            {status?.lastPullAt ? relativeTime(status.lastPullAt) : 'Never'}
          </p>
        </div>

        {/* Pending */}
        <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-4">
          <div className="flex items-center gap-2 text-[var(--gantry-text-secondary)]">
            <Clock className="h-4 w-4" />
            <span className="text-xs font-medium">Pending</span>
          </div>
          <p className="mt-2 text-sm font-semibold text-[var(--gantry-text-primary)]">
            {status?.pendingFiles ?? 0} {status?.pendingFiles === 1 ? 'file' : 'files'}
          </p>
        </div>

        {/* Files tracked */}
        <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-4">
          <div className="flex items-center gap-2 text-[var(--gantry-text-secondary)]">
            <FileText className="h-4 w-4" />
            <span className="text-xs font-medium">Files Tracked</span>
          </div>
          <p className="mt-2 text-sm font-semibold text-[var(--gantry-text-primary)]">
            {files.length}
          </p>
        </div>
      </div>

      {/* Action buttons */}
      <div className="flex gap-3">
        <button
          onClick={handleSync}
          disabled={syncing || !status?.connected}
          className="flex items-center gap-2 rounded-lg bg-[var(--gantry-accent)] px-4 py-2 text-sm font-medium text-[var(--gantry-bg-primary)] transition-colors hover:bg-[var(--gantry-accent-hover)] disabled:opacity-50"
        >
          {syncing ? <Loader2 className="h-4 w-4 animate-spin" /> : <GitPullRequest className="h-4 w-4" />}
          Sync
        </button>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-[var(--gantry-border)]">
        <button
          onClick={() => setTab('history')}
          className={`px-4 py-2 text-sm font-medium transition-colors ${
            tab === 'history'
              ? 'border-b-2 border-[var(--gantry-accent)] text-[var(--gantry-accent)]'
              : 'text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]'
          }`}
        >
          <span className="flex items-center gap-2">
            <GitBranch className="h-4 w-4" />
            Sync History
          </span>
        </button>
        <button
          onClick={() => setTab('files')}
          className={`px-4 py-2 text-sm font-medium transition-colors ${
            tab === 'files'
              ? 'border-b-2 border-[var(--gantry-accent)] text-[var(--gantry-accent)]'
              : 'text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]'
          }`}
        >
          <span className="flex items-center gap-2">
            <FolderGit2 className="h-4 w-4" />
            Files ({files.length})
          </span>
        </button>
      </div>

      {/* Tab content */}
      {tab === 'history' && (
        <div className="space-y-2">
          {history.length === 0 ? (
            <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-8 text-center">
              <GitBranch className="mx-auto mb-3 h-8 w-8 text-[var(--gantry-text-secondary)]" />
              <p className="text-sm text-[var(--gantry-text-secondary)]">No sync operations yet.</p>
            </div>
          ) : (
            history.map((entry) => (
              <div
                key={entry.id}
                className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] px-4 py-3"
              >
                <div className="flex items-start gap-4">
                  {/* Direction icon */}
                  {entry.direction === 'push' ? (
                    <ArrowUpCircle className="mt-0.5 h-5 w-5 shrink-0 text-blue-500" />
                  ) : (
                    <ArrowDownCircle className="mt-0.5 h-5 w-5 shrink-0 text-purple-500" />
                  )}

                  {/* Details */}
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-[var(--gantry-text-primary)]">
                      {entry.message}
                    </p>
                    <div className="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-[var(--gantry-text-secondary)]">
                      <span>{relativeTime(entry.timestamp)}</span>
                      {entry.commit && (
                        <span className="rounded bg-[var(--gantry-bg-tertiary)] px-1.5 py-0.5 font-mono text-[11px]">
                          {entry.commit}
                        </span>
                      )}
                      <span>{entry.files} {entry.files === 1 ? 'file' : 'files'}</span>
                    </div>
                  </div>

                  {/* Direction badge */}
                  <span
                    className={`shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium ${
                      entry.direction === 'push'
                        ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                        : 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                    }`}
                  >
                    {entry.direction}
                  </span>

                  {/* Error indicator */}
                  {entry.error && (
                    <button
                      type="button"
                      onClick={() => setExpandedErrorId(expandedErrorId === entry.id ? null : entry.id)}
                      className="flex shrink-0 items-center gap-1 rounded-md border border-[var(--gantry-danger)]/30 bg-[var(--gantry-danger)]/10 px-2 py-1 text-[11px] font-medium text-[var(--gantry-danger)] transition-colors hover:border-[var(--gantry-danger)]"
                    >
                      <AlertCircle className="h-3.5 w-3.5" />
                      {expandedErrorId === entry.id ? 'Hide error' : 'View error'}
                    </button>
                  )}
                </div>

                {entry.error && expandedErrorId === entry.id && (
                  <div className="mt-3 rounded-lg border border-[var(--gantry-danger)]/20 bg-[var(--gantry-danger)]/10 px-3 py-2">
                    <p className="text-[11px] font-semibold uppercase tracking-wide text-[var(--gantry-danger)]">
                      Error details
                    </p>
                    <p className="mt-1 whitespace-pre-wrap break-words font-mono text-xs text-[var(--gantry-text-primary)]">
                      {entry.error}
                    </p>
                  </div>
                )}
              </div>
            ))
          )}
        </div>
      )}

      {tab === 'files' && (
        <div className="flex gap-4" style={{ minHeight: '400px' }}>
          {/* File tree */}
          <div className="w-64 shrink-0 rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] overflow-hidden flex flex-col">
            <div className="border-b border-[var(--gantry-border)] px-3 py-2">
              <p className="text-xs font-medium text-[var(--gantry-text-secondary)]">Repository Files</p>
            </div>
            {files.length === 0 ? (
              <div className="flex flex-1 flex-col items-center justify-center p-6 text-center">
                <FolderGit2 className="mb-2 h-7 w-7 text-[var(--gantry-text-secondary)]" />
                <p className="text-xs text-[var(--gantry-text-secondary)]">No entity files yet.</p>
              </div>
            ) : (
              <div className="flex-1 overflow-y-auto p-2 space-y-1">
                {(() => {
                  const byKind = files.reduce<Record<string, GitOpsFileEntry[]>>((acc, f) => {
                    (acc[f.kind] = acc[f.kind] || []).push(f);
                    return acc;
                  }, {});
                  return Object.entries(byKind).sort(([a], [b]) => a.localeCompare(b)).map(([kind, kindFiles]) => (
                    <FileTreeNode
                      key={kind}
                      kind={kind}
                      files={kindFiles}
                      onSelectFile={handleSelectFile}
                      selectedPath={selectedFile?.path ?? null}
                    />
                  ));
                })()}
              </div>
            )}
          </div>

          {/* YAML viewer */}
          <div className="flex-1 rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] overflow-hidden flex flex-col">
            {selectedFile ? (
              <>
                <div className="flex items-center gap-3 border-b border-[var(--gantry-border)] px-4 py-2">
                  <FileCode className="h-4 w-4 shrink-0 text-[var(--gantry-accent)]" />
                  <span className="flex-1 truncate font-mono text-xs text-[var(--gantry-text-secondary)]">{selectedFile.path}</span>
                  {selectedFile.error && (
                    <div className="flex items-center gap-1 rounded-md border border-[var(--gantry-danger)]/30 bg-[var(--gantry-danger)]/10 px-2 py-0.5">
                      <AlertCircle className="h-3 w-3 text-[var(--gantry-danger)]" />
                      <span className="text-[11px] font-medium text-[var(--gantry-danger)]">Parse error</span>
                    </div>
                  )}
                  <button
                    onClick={() => { setSelectedFile(null); setFileContent(''); }}
                    className="rounded p-0.5 text-[var(--gantry-text-secondary)] transition-colors hover:text-[var(--gantry-text-primary)]"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>

                {selectedFile.error && (
                  <div className="border-b border-[var(--gantry-border)] bg-[var(--gantry-danger)]/10 px-4 py-2">
                    <p className="font-mono text-xs text-[var(--gantry-danger)]">{selectedFile.error}</p>
                  </div>
                )}

                <div className="flex-1 overflow-auto p-4">
                  {loadingContent ? (
                    <div className="flex h-full items-center justify-center">
                      <Loader2 className="h-6 w-6 animate-spin text-[var(--gantry-accent)]" />
                    </div>
                  ) : (
                    <pre className="text-xs leading-relaxed text-[var(--gantry-text-primary)] font-mono whitespace-pre-wrap break-all">{fileContent}</pre>
                  )}
                </div>
              </>
            ) : (
              <div className="flex flex-1 flex-col items-center justify-center p-8 text-center">
                <FileCode className="mb-3 h-10 w-10 text-[var(--gantry-text-secondary)]" />
                <p className="text-sm font-medium text-[var(--gantry-text-primary)]">Select a file to view its YAML</p>
                <p className="mt-1 text-xs text-[var(--gantry-text-secondary)]">Click any file in the tree on the left</p>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
