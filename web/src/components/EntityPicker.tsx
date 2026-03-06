import { useState, useEffect, useRef, useCallback } from 'react';
import { ChevronDown, X } from 'lucide-react';
import { api } from '../lib/api';
import type { Entity } from '../lib/types';

// Module-level cache so multiple pickers on the same form share one fetch per kind.
const entityCache = new Map<string, Entity[]>();
const inflight = new Map<string, Promise<Entity[]>>();

async function loadKind(kind: string): Promise<Entity[]> {
  if (entityCache.has(kind)) return entityCache.get(kind)!;
  if (!inflight.has(kind)) {
    const p = api.listEntities(kind).then((data) => {
      const list = data ?? [];
      entityCache.set(kind, list);
      inflight.delete(kind);
      return list;
    });
    inflight.set(kind, p);
  }
  return inflight.get(kind)!;
}

interface Props {
  entityKind: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

export default function EntityPicker({ entityKind, value, onChange, placeholder }: Props) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState(value);
  const [entities, setEntities] = useState<Entity[]>([]);
  const [loading, setLoading] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Keep query in sync when value is changed externally
  useEffect(() => { setQuery(value); }, [value]);

  const load = useCallback(async () => {
    if (entities.length > 0 || loading) return;
    setLoading(true);
    try {
      const data = await loadKind(entityKind);
      setEntities(data);
    } finally {
      setLoading(false);
    }
  }, [entityKind, entities.length, loading]);

  // Close on click outside
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (!containerRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  const filtered = entities.filter((e) => {
    if (!query) return true;
    const q = query.toLowerCase();
    return (
      e.metadata.name.toLowerCase().includes(q) ||
      (e.metadata.title ?? '').toLowerCase().includes(q)
    );
  });

  const select = (name: string) => {
    onChange(name);
    setQuery(name);
    setOpen(false);
    setActiveIndex(-1);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!open) {
      if (e.key === 'ArrowDown' || e.key === 'Enter') { setOpen(true); load(); }
      return;
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setActiveIndex((i) => Math.min(i + 1, filtered.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActiveIndex((i) => Math.max(i - 1, -1));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (activeIndex >= 0 && filtered[activeIndex]) {
        select(filtered[activeIndex].metadata.name);
      } else {
        onChange(query);
        setOpen(false);
      }
    } else if (e.key === 'Escape') {
      setOpen(false);
      setActiveIndex(-1);
    }
  };

  return (
    <div ref={containerRef} className="relative">
      <div className="relative">
        <input
          ref={inputRef}
          type="text"
          value={query}
          placeholder={placeholder ?? `Search ${entityKind}s…`}
          onChange={(e) => {
            const v = e.target.value;
            setQuery(v);
            onChange(v);
            setActiveIndex(-1);
            if (!open) { setOpen(true); load(); }
          }}
          onFocus={() => { setOpen(true); load(); }}
          onKeyDown={handleKeyDown}
          className="w-full rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] px-3 py-2 pr-8 text-sm text-[var(--gantry-text-primary)] focus:border-[var(--gantry-accent)] focus:outline-none focus:ring-1 focus:ring-[var(--gantry-accent)]"
        />
        {query ? (
          <button
            type="button"
            onClick={() => { onChange(''); setQuery(''); inputRef.current?.focus(); }}
            className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        ) : (
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[var(--gantry-text-secondary)]" />
        )}
      </div>

      {open && (
        <div className="absolute z-50 mt-1 w-full overflow-hidden rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] shadow-lg">
          {loading ? (
            <p className="px-3 py-2 text-xs text-[var(--gantry-text-secondary)]">Loading…</p>
          ) : filtered.length === 0 ? (
            <p className="px-3 py-4 text-center text-xs text-[var(--gantry-text-secondary)]">
              {query
                ? `No ${entityKind} entities match "${query}"`
                : `No ${entityKind} entities found`}
            </p>
          ) : (
            <ul className="max-h-52 overflow-y-auto py-1">
              {filtered.map((e, i) => (
                <li key={e.metadata.name}>
                  <button
                    type="button"
                    // Use mousedown so it fires before the input's blur
                    onMouseDown={(ev) => { ev.preventDefault(); select(e.metadata.name); }}
                    className={`flex w-full items-center gap-3 px-3 py-2 text-left transition-colors ${
                      i === activeIndex
                        ? 'bg-[var(--gantry-accent)]/10 text-[var(--gantry-accent)]'
                        : 'text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-secondary)]'
                    }`}
                  >
                    <span className="min-w-0 flex-1 truncate text-sm font-medium">
                      {e.metadata.name}
                    </span>
                    {e.metadata.title && (
                      <span className="shrink-0 truncate text-xs text-[var(--gantry-text-secondary)]">
                        {e.metadata.title}
                      </span>
                    )}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
