import { useEffect, useRef, useState } from 'react'
import { AlertTriangle, Info, RefreshCw, FolderOpen, X, Lock } from 'lucide-react'
import { cn } from '@/lib/utils'
import { openInExplorer } from '@/lib/bridge'
import type { AttentionItem, DashSpec, EnvironmentSource, WorkspaceOverview } from '@/lib/bridge'

type Props = {
  // The snapshot is read by the app (useWorkspaceOverview) and shared with the Workspace sidebar.
  overview: WorkspaceOverview | null
  error: string | null
  loading: boolean
  onRefresh: () => void
  // Set when an item in the sidebar was clicked: the page scrolls to that attention card.
  focusAttention?: { index: number; seq: number }
  onOpenScenario: (path: string, name: string) => void
  onOpenContracts: (specRef: string) => void
  onShowPanel: (panel: 'explorer' | 'scenarios') => void
  onCloseWorkspace: () => void
}

export const ENV_SOURCE_LABEL: Record<EnvironmentSource, string> = {
  remembered: 'remembered for this workspace',
  chosen: 'chosen this session',
  temporary: 'temporary workspace, not remembered',
  none: 'none chosen',
}

const SPEC_STATE_LABEL: Record<DashSpec['state'], string> = {
  found: 'found',
  missing: 'missing',
  outside: 'outside the workspace',
  url: 'URL, not checked',
}

export function attentionCount(o: WorkspaceOverview): number {
  return o.attention.length
}

// The one-line summary shown when a workspace opens: what was restored and what needs attention.
export function openSummary(o: WorkspaceOverview): string {
  const env = o.environmentName
    ? `Environment: ${o.environmentName} (${o.environmentSource === 'none' ? 'none chosen' : o.environmentSource === 'temporary' ? 'temporary' : o.environmentSource})`
    : 'Environment: none chosen'
  const n = attentionCount(o)
  return `${env} · ${n === 0 ? 'nothing needs attention' : n === 1 ? '1 thing needs attention' : `${n} things need attention`}`
}

function baseName(p: string): string {
  const parts = p.replace(/\\/g, '/').split('/')
  const file = parts[parts.length - 1] ?? p
  return file.replace(/\.(ya?ml|json)$/i, '')
}

function formatTime(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

// Workspaces whose opening reminder was dismissed in this session. Kept outside the component so closing
// and reopening the Workspace tab does not bring the reminder back.
const dismissedReminders = new Set<string>()

export function WorkspaceTab({ overview, error, loading, onRefresh, focusAttention, onOpenScenario, onOpenContracts, onShowPanel, onCloseWorkspace }: Props) {
  const [, setDismissals] = useState(0)
  const page = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!focusAttention || !overview) return
    page.current?.querySelector(`[data-attention-index="${focusAttention.index}"]`)?.scrollIntoView?.({ block: 'center' })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focusAttention?.seq, overview !== null])

  if (!overview) {
    return (
      <div className="flex-1 p-6 text-sm text-[var(--color-muted-foreground)]">
        {error ? <p role="alert" className="text-red-400">{error}</p> : 'Reading the workspace…'}
      </div>
    )
  }

  const attention = overview.attention
  const showReminder = overview.open && !dismissedReminders.has(overview.path)

  return (
    <div ref={page} className="flex-1 overflow-auto" data-testid="workspace-page">
      <div className="p-5 flex flex-col gap-4">
        {showReminder && (
          <div role="status" data-testid="workspace-reminder" className="flex items-center gap-2 rounded border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-xs">
            <Info size={14} className="shrink-0 text-sky-400" aria-hidden />
            <span className="flex-1">{openSummary(overview)}</span>
            <button type="button" aria-label="Dismiss reminder" onClick={() => { dismissedReminders.add(overview.path); setDismissals(n => n + 1) }} className="shrink-0 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]">
              <X size={14} />
            </button>
          </div>
        )}

        <header className="flex items-start gap-3">
          <div className="flex-1 min-w-0">
            <h1 className="text-lg font-semibold truncate">
              {overview.name || 'Workspace'}
              {overview.temporary && (
                <span className="ml-2 align-middle rounded border border-amber-500/40 bg-amber-500/10 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-amber-400">temporary</span>
              )}
            </h1>
            <p className="text-xs text-[var(--color-muted-foreground)] truncate" title={overview.path}>{overview.path}</p>
          </div>
          <button type="button" onClick={onRefresh} disabled={loading} className="flex items-center gap-1.5 rounded border border-[var(--color-border)] px-2.5 py-1 text-xs hover:bg-[var(--color-muted)] disabled:opacity-50">
            <RefreshCw size={12} className={cn(loading && 'animate-spin')} aria-hidden /> Refresh
          </button>
          <button type="button" onClick={() => void openInExplorer(overview.path)} className="flex items-center gap-1.5 rounded border border-[var(--color-border)] px-2.5 py-1 text-xs hover:bg-[var(--color-muted)]">
            <FolderOpen size={12} aria-hidden /> Show in file manager
          </button>
          <button type="button" onClick={onCloseWorkspace} className="rounded border border-[var(--color-border)] px-2.5 py-1 text-xs hover:bg-[var(--color-muted)]">
            Close workspace
          </button>
        </header>

        {error && <p role="alert" className="text-xs text-red-400">{error}</p>}

        <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_minmax(360px,38%)]">
          <div className="flex flex-col gap-4 min-w-0">
            <section aria-label="Needs attention" className="rounded border border-[var(--color-border)] bg-[var(--color-card)]">
              <h2 className="px-3 py-2 text-xs font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)] border-b border-[var(--color-border)]">
                Needs attention{attention.length > 0 && <span className="ml-2 tabular-nums">{attention.length}</span>}
              </h2>
              {attention.length === 0 ? (
                <p data-testid="attention-empty" className="px-3 py-3 text-xs text-emerald-400">Nothing needs attention.</p>
              ) : (
                <ul className="divide-y divide-[var(--color-border)]">
                  {attention.map((a, i) => (
                    <AttentionCard key={`${a.id}-${i}`} index={i} item={a} onOpenScenario={onOpenScenario} onOpenContracts={onOpenContracts} />
                  ))}
                </ul>
              )}
            </section>

            <section aria-label="Specs" className="rounded border border-[var(--color-border)] bg-[var(--color-card)]">
              <h2 className="px-3 py-2 text-xs font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)] border-b border-[var(--color-border)]">Specs</h2>
              {overview.specs.length === 0 ? (
                <p className="px-3 py-3 text-xs text-[var(--color-muted-foreground)]">No scenario names a spec file yet.</p>
              ) : (
                <ul className="divide-y divide-[var(--color-border)]">
                  {overview.specs.map(sp => (
                    <li key={sp.path || sp.ref} className="px-3 py-2 text-xs flex flex-col gap-1">
                      <div className="flex items-center gap-2">
                        <span className="font-mono truncate flex-1" title={sp.ref}>{sp.ref}</span>
                        <span className={cn('shrink-0 rounded px-1.5 py-0.5 text-[10px]', sp.state === 'missing' ? 'bg-red-500/10 text-red-400' : sp.state === 'found' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-[var(--color-muted)] text-[var(--color-muted-foreground)]')}>
                          {SPEC_STATE_LABEL[sp.state]}
                        </span>
                      </div>
                      <div className="text-[var(--color-muted-foreground)]">
                        Used by {sp.scenarios.length} {sp.scenarios.length === 1 ? 'scenario' : 'scenarios'}
                        {sp.coverage.map(c => (
                          <span key={c.ref}>
                            {' · '}
                            <button type="button" onClick={() => onOpenContracts(c.ref)} className="underline hover:text-[var(--color-foreground)]">
                              Covered {c.covered}, Not covered {c.notCovered}, Passing {c.passing}, Failing {c.failing}
                            </button>
                          </span>
                        ))}
                        {sp.coverage.length === 0 && ' · no coverage recorded yet'}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <section aria-label="Last results" className="rounded border border-[var(--color-border)] bg-[var(--color-card)]">
              <h2 className="px-3 py-2 text-xs font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)] border-b border-[var(--color-border)]">Last results</h2>
              {overview.lastRun ? (
                <p className="px-3 py-3 text-xs" data-testid="last-run">
                  <span className={overview.lastRun.passed ? 'text-emerald-400' : 'text-red-400'}>{overview.lastRun.passed ? 'PASSED' : 'FAILED'}</span>
                  {' '}{overview.lastRun.scenario} · {formatTime(overview.lastRun.at)}
                  <span className="block mt-1 text-[var(--color-muted-foreground)]">
                    {overview.lastRun.scenarios} {overview.lastRun.scenarios === 1 ? 'scenario has' : 'scenarios have'} run here; {overview.lastRun.failingNow === 0 ? 'none failed' : `${overview.lastRun.failingNow} failed`} the last time. Kept on disk, so it survives a restart.
                  </span>
                </p>
              ) : (
                <p className="px-3 py-3 text-xs text-[var(--color-muted-foreground)]">No scenario has run in this workspace yet.</p>
              )}
            </section>
          </div>

          <aside className="flex flex-col gap-4">
            <section aria-label="Contents" className="rounded border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-3 text-xs flex flex-col gap-1.5">
              <h2 className="text-xs font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)] mb-1">Contents</h2>
              <button type="button" onClick={() => onShowPanel('explorer')} className="text-left hover:underline">{overview.requests} {overview.requests === 1 ? 'request' : 'requests'}</button>
              <button type="button" onClick={() => onShowPanel('scenarios')} className="text-left hover:underline">{overview.scenarios} {overview.scenarios === 1 ? 'scenario' : 'scenarios'}</button>
              <span>{overview.environments} {overview.environments === 1 ? 'environment' : 'environments'}</span>
              <span>{overview.specs.length} {overview.specs.length === 1 ? 'spec' : 'specs'}</span>
            </section>

            <section aria-label="Environment" className="rounded border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-3 text-xs">
              <h2 className="text-xs font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)] mb-1">Environment</h2>
              <p data-testid="env-source">
                <span className="font-medium">{overview.environmentName || 'None selected'}</span>
                <span className="block text-[var(--color-muted-foreground)]">{ENV_SOURCE_LABEL[overview.environmentSource]}</span>
              </p>
              {overview.envVars.length > 0 && (
                <ul aria-label="Environment variables" className="mt-2 flex flex-col gap-1">
                  {overview.envVars.map(v => (
                    <li key={v.name} className="flex items-center gap-1.5">
                      <span className="font-mono truncate flex-1">{v.name}</span>
                      {v.secret && <Lock size={11} aria-label="secret" className="shrink-0 text-[var(--color-muted-foreground)]" />}
                      <span className={cn('shrink-0', v.set ? 'text-emerald-400' : 'text-amber-400')}>
                        {v.set ? 'set' : v.unsetRef ? `unset \${${v.unsetRef}}` : 'empty'}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <p className="text-[10px] text-[var(--color-muted-foreground)]">Read {formatTime(overview.snapshotAt)}</p>
          </aside>
        </div>
      </div>
    </div>
  )
}

function AttentionCard({ item, index, onOpenScenario, onOpenContracts }: {
  item: AttentionItem
  index: number
  onOpenScenario: (path: string, name: string) => void
  onOpenContracts: (specRef: string) => void
}) {
  const warn = item.severity === 'warning'
  return (
    <li>
      <article aria-label={item.what} data-attention-index={index} className="px-3 py-3 text-xs flex flex-col gap-1">
        <div className="flex items-start gap-2">
          {warn ? <AlertTriangle size={14} className="shrink-0 mt-0.5 text-amber-400" aria-hidden /> : <Info size={14} className="shrink-0 mt-0.5 text-sky-400" aria-hidden />}
          <p className="font-medium">{item.what}</p>
        </div>
        <dl className="ml-6 grid grid-cols-[4.5rem_1fr] gap-x-2 gap-y-0.5">
          <dt className="text-[var(--color-muted-foreground)]">Where</dt><dd>{item.where}</dd>
          <dt className="text-[var(--color-muted-foreground)]">Why</dt><dd>{item.why}</dd>
          <dt className="text-[var(--color-muted-foreground)]">Fix</dt><dd>{item.fix}</dd>
        </dl>
        <div className="ml-6 mt-1">
          {item.target.kind === 'scenario' && item.target.path && (
            <button type="button" onClick={() => onOpenScenario(item.target.path, baseName(item.target.path))} className="rounded border border-[var(--color-border)] px-2 py-1 hover:bg-[var(--color-muted)]">Open scenario</button>
          )}
          {item.target.kind === 'contracts' && (
            <button type="button" onClick={() => onOpenContracts(item.target.path)} className="rounded border border-[var(--color-border)] px-2 py-1 hover:bg-[var(--color-muted)]">Open Contracts</button>
          )}
          {item.target.kind === 'environment' && item.target.path && (
            <button type="button" onClick={() => void openInExplorer(item.target.path)} className="rounded border border-[var(--color-border)] px-2 py-1 hover:bg-[var(--color-muted)]">Show environment file</button>
          )}
        </div>
      </article>
    </li>
  )
}
