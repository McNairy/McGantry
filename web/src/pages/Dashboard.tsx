import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import {
  Server,
  Globe,
  Database,
  Users,
  Cloud,
  FileText,
  Box,
  ArrowRight,
  BookOpen,
  Zap,
  Search,
} from 'lucide-react';
import { api } from '../lib/api';
import { useAuth } from '../hooks/useAuth';
import type { Entity } from '../lib/types';
import { ENTITY_KINDS } from '../lib/types';

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  Server,
  Globe,
  Database,
  Users,
  Cloud,
  FileText,
};

interface KindCount {
  name: string;
  plural: string;
  icon: string;
  count: number;
}

export default function Dashboard() {
  const { user } = useAuth();
  const [entities, setEntities] = useState<Entity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api
      .listEntities()
      .then((data) => setEntities(data || []))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const kindCounts: KindCount[] = ENTITY_KINDS.map((k) => ({
    ...k,
    count: entities.filter((e) => e.kind === k.name).length,
  }));

  const totalEntities = entities.length;

  const recentEntities = [...entities]
    .sort((a, b) => {
      const aDate = a.metadata.updatedAt || a.metadata.createdAt || '';
      const bDate = b.metadata.updatedAt || b.metadata.createdAt || '';
      return bDate.localeCompare(aDate);
    })
    .slice(0, 5);

  function formatDate(dateStr?: string): string {
    if (!dateStr) return '';
    try {
      return new Date(dateStr).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return dateStr;
    }
  }

  return (
    <div className="space-y-8">
      {/* Welcome Header */}
      <div>
        <h1 className="text-2xl font-bold text-[var(--gantry-text-primary)]">
          Welcome to Gantry{user?.displayName ? `, ${user.displayName}` : ''}
        </h1>
        <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">
          Your internal developer platform overview
        </p>
      </div>

      {/* Error state */}
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-[var(--gantry-danger)] dark:border-red-800 dark:bg-red-900/20">
          {error}
        </div>
      )}

      {/* Loading state */}
      {loading && (
        <div className="flex items-center justify-center py-16">
          <div className="spinner h-8 w-8 text-[var(--gantry-accent)]" />
        </div>
      )}

      {!loading && (
        <>
          {/* Stats Cards */}
          <div className="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-6">
            {kindCounts.map((kind) => {
              const Icon = iconMap[kind.icon] || Box;
              return (
                <Link
                  key={kind.name}
                  to={`/catalog/${kind.name}`}
                  className="group rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-4 transition-all hover:shadow-md"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-[var(--gantry-accent)] bg-opacity-10">
                      <Icon className="h-5 w-5 text-[var(--gantry-accent)]" />
                    </div>
                    <div>
                      <p className="text-2xl font-bold text-[var(--gantry-text-primary)]">
                        {kind.count}
                      </p>
                      <p className="text-xs text-[var(--gantry-text-secondary)]">{kind.name}s</p>
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>

          {/* Empty state / Getting started */}
          {totalEntities === 0 && (
            <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-8">
              <h2 className="text-lg font-semibold text-[var(--gantry-text-primary)]">
                Getting Started
              </h2>
              <p className="mt-2 text-sm text-[var(--gantry-text-secondary)]">
                Your software catalog is empty. Here are some things you can do to get started:
              </p>
              <div className="mt-6 grid gap-4 md:grid-cols-3">
                <Link
                  to="/catalog"
                  className="flex items-start gap-3 rounded-lg border border-[var(--gantry-border)] p-4 transition-colors hover:bg-[var(--gantry-bg-secondary)]"
                >
                  <BookOpen className="mt-0.5 h-5 w-5 shrink-0 text-[var(--gantry-accent)]" />
                  <div>
                    <p className="text-sm font-medium text-[var(--gantry-text-primary)]">
                      Register a Service
                    </p>
                    <p className="mt-1 text-xs text-[var(--gantry-text-secondary)]">
                      Add your first service to the catalog to start tracking your software.
                    </p>
                  </div>
                </Link>
                <Link
                  to="/actions"
                  className="flex items-start gap-3 rounded-lg border border-[var(--gantry-border)] p-4 transition-colors hover:bg-[var(--gantry-bg-secondary)]"
                >
                  <Zap className="mt-0.5 h-5 w-5 shrink-0 text-[var(--gantry-accent)]" />
                  <div>
                    <p className="text-sm font-medium text-[var(--gantry-text-primary)]">
                      Explore Actions
                    </p>
                    <p className="mt-1 text-xs text-[var(--gantry-text-secondary)]">
                      Browse and execute self-service actions to automate common tasks.
                    </p>
                  </div>
                </Link>
                <div className="flex items-start gap-3 rounded-lg border border-[var(--gantry-border)] p-4">
                  <Search className="mt-0.5 h-5 w-5 shrink-0 text-[var(--gantry-accent)]" />
                  <div>
                    <p className="text-sm font-medium text-[var(--gantry-text-primary)]">
                      Search with Cmd+K
                    </p>
                    <p className="mt-1 text-xs text-[var(--gantry-text-secondary)]">
                      Use the command palette to quickly find entities across your catalog.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Recent Entities */}
          {recentEntities.length > 0 && (
            <div className="rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)]">
              <div className="flex items-center justify-between border-b border-[var(--gantry-border)] px-6 py-4">
                <h2 className="text-base font-semibold text-[var(--gantry-text-primary)]">
                  Recent Entities
                </h2>
                <Link
                  to="/catalog"
                  className="flex items-center gap-1 text-sm text-[var(--gantry-accent)] hover:text-[var(--gantry-accent-hover)]"
                >
                  View all <ArrowRight className="h-4 w-4" />
                </Link>
              </div>
              <div className="divide-y divide-[var(--gantry-border)]">
                {recentEntities.map((entity) => {
                  const Icon = iconMap[
                    ENTITY_KINDS.find((k) => k.name === entity.kind)?.icon || ''
                  ] || Box;
                  return (
                    <Link
                      key={`${entity.kind}-${entity.metadata.name}`}
                      to={`/catalog/${entity.kind}/${entity.metadata.name}`}
                      className="flex items-center gap-4 px-6 py-3 transition-colors hover:bg-[var(--gantry-bg-secondary)]"
                    >
                      <Icon className="h-5 w-5 shrink-0 text-[var(--gantry-text-secondary)]" />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium text-[var(--gantry-text-primary)]">
                          {entity.metadata.name}
                        </p>
                        <p className="text-xs text-[var(--gantry-text-secondary)]">
                          {entity.kind}
                          {entity.metadata.owner ? ` / ${entity.metadata.owner}` : ''}
                        </p>
                      </div>
                      <span className="shrink-0 text-xs text-[var(--gantry-text-secondary)]">
                        {formatDate(entity.metadata.updatedAt || entity.metadata.createdAt)}
                      </span>
                    </Link>
                  );
                })}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
