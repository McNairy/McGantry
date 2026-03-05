import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, Server, Globe, Database, Users, Cloud, FileText, Box } from 'lucide-react';
import { api } from '../lib/api';
import type { SearchResult } from '../lib/types';

const kindIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  Service: Server,
  API: Globe,
  Infrastructure: Database,
  Team: Users,
  Environment: Cloud,
  Documentation: FileText,
};

export default function CommandPalette() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();
  const debounceRef = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 50);
    } else {
      setQuery('');
      setResults([]);
      setSelectedIndex(0);
    }
  }, [open]);

  const doSearch = useCallback((q: string) => {
    if (!q.trim()) {
      setResults([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    api
      .search(q)
      .then((data) => {
        setResults(data || []);
        setSelectedIndex(0);
      })
      .catch(() => setResults([]))
      .finally(() => setLoading(false));
  }, []);

  const handleInputChange = (value: string) => {
    setQuery(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => doSearch(value), 250);
  };

  const selectResult = (result: SearchResult) => {
    setOpen(false);
    navigate(`/catalog/${result.kind}/${result.name}`);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIndex((prev) => Math.min(prev + 1, results.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIndex((prev) => Math.max(prev - 1, 0));
    } else if (e.key === 'Enter' && results[selectedIndex]) {
      e.preventDefault();
      selectResult(results[selectedIndex]);
    } else if (e.key === 'Escape') {
      setOpen(false);
    }
  };

  const grouped = results.reduce<Record<string, SearchResult[]>>((acc, r) => {
    if (!acc[r.kind]) acc[r.kind] = [];
    acc[r.kind].push(r);
    return acc;
  }, {});

  let flatIndex = -1;

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[20vh]">
      <div className="fixed inset-0 bg-black/50" onClick={() => setOpen(false)} />
      <div className="relative w-full max-w-lg rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] shadow-2xl">
        {/* Search input */}
        <div className="flex items-center gap-3 border-b border-[var(--gantry-border)] px-4 py-3">
          <Search className="h-5 w-5 shrink-0 text-[var(--gantry-text-secondary)]" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => handleInputChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search entities..."
            className="flex-1 bg-transparent text-sm text-[var(--gantry-text-primary)] outline-none placeholder:text-[var(--gantry-text-secondary)]"
          />
          <kbd className="hidden rounded border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-1.5 py-0.5 text-xs text-[var(--gantry-text-secondary)] sm:inline-block">
            Esc
          </kbd>
        </div>

        {/* Results */}
        <div className="max-h-80 overflow-y-auto p-2">
          {loading && (
            <div className="flex items-center justify-center py-8">
              <div className="spinner text-[var(--gantry-accent)]" />
            </div>
          )}

          {!loading && query && results.length === 0 && (
            <div className="py-8 text-center text-sm text-[var(--gantry-text-secondary)]">
              No results found for &quot;{query}&quot;
            </div>
          )}

          {!loading && !query && (
            <div className="py-8 text-center text-sm text-[var(--gantry-text-secondary)]">
              Start typing to search entities...
            </div>
          )}

          {!loading &&
            Object.entries(grouped).map(([kind, items]) => {
              const KindIcon = kindIcons[kind] || Box;
              return (
                <div key={kind}>
                  <div className="flex items-center gap-2 px-3 py-1.5">
                    <KindIcon className="h-3.5 w-3.5 text-[var(--gantry-text-secondary)]" />
                    <span className="text-xs font-medium uppercase tracking-wide text-[var(--gantry-text-secondary)]">
                      {kind}
                    </span>
                  </div>
                  {items.map((result) => {
                    flatIndex++;
                    const idx = flatIndex;
                    return (
                      <button
                        key={`${result.kind}-${result.name}`}
                        className={`flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left text-sm transition-colors ${
                          idx === selectedIndex
                            ? 'bg-[var(--gantry-accent)] bg-opacity-10 text-[var(--gantry-accent)]'
                            : 'text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-tertiary)]'
                        }`}
                        onClick={() => selectResult(result)}
                        onMouseEnter={() => setSelectedIndex(idx)}
                      >
                        <span className="font-medium">{result.name}</span>
                        {result.title && result.title !== result.name && (
                          <span className="truncate text-[var(--gantry-text-secondary)]">
                            {result.title}
                          </span>
                        )}
                      </button>
                    );
                  })}
                </div>
              );
            })}
        </div>

        {/* Footer */}
        <div className="flex items-center gap-4 border-t border-[var(--gantry-border)] px-4 py-2 text-xs text-[var(--gantry-text-secondary)]">
          <span className="flex items-center gap-1">
            <kbd className="rounded border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-1 py-0.5">
              &uarr;&darr;
            </kbd>
            Navigate
          </span>
          <span className="flex items-center gap-1">
            <kbd className="rounded border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-1 py-0.5">
              &crarr;
            </kbd>
            Select
          </span>
          <span className="flex items-center gap-1">
            <kbd className="rounded border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-1 py-0.5">
              Esc
            </kbd>
            Close
          </span>
        </div>
      </div>
    </div>
  );
}
