import { ExternalLink, FileText } from 'lucide-react';
import type { Entity } from '../lib/types';

interface Props {
  entity: Entity;
}

export default function APIDocsTab({ entity }: Props) {
  const apiDocsUrl = (entity.spec?.apiDocsUrl as string) || '';

  return (
    <div className="space-y-6">
      {/* API Docs URL Section — existing UI page shown in iframe */}
      {apiDocsUrl ? (
        <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)]">
          <div className="flex items-center justify-between border-b border-[var(--gantry-border)] px-6 py-4">
            <div className="flex items-center gap-2">
              <FileText className="h-4 w-4 text-[var(--gantry-text-secondary)]" />
              <h3 className="text-sm font-semibold text-[var(--gantry-text-primary)]">API Documentation</h3>
            </div>
            <a
              href={apiDocsUrl}
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
              src={apiDocsUrl}
              title="API Documentation"
              className="h-[700px] w-full border-0"
              sandbox="allow-scripts allow-same-origin allow-popups allow-forms"
            />
          </div>
        </div>
      ) : (
        <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] px-6 py-12 text-center">
          <FileText className="mx-auto h-10 w-10 text-[var(--gantry-text-secondary)] opacity-40" />
          <p className="mt-3 text-sm text-[var(--gantry-text-secondary)]">No API documentation URL configured.</p>
          <p className="mt-1 text-xs text-[var(--gantry-text-secondary)]">
            Set <code className="rounded bg-[var(--gantry-bg-tertiary)] px-1.5 py-0.5">apiDocsUrl</code> in the entity spec.
          </p>
        </div>
      )}
    </div>
  );
}
