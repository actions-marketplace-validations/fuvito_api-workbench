import { AlertTriangle, Info, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { EnvInfo, WorkspaceOverview } from '@/lib/bridge'
import { ENV_SOURCE_LABEL } from '@/components/WorkspaceTab'

type Props = {
  overview: WorkspaceOverview | null
  error: string | null
  loading: boolean
  onRefresh: () => void
  environments: EnvInfo[]
  activeEnvPath: string
  onEnvChange: (path: string) => void
  // Show the Workspace page scrolled to the attention card with this index.
  onShowAttention: (index: number) => void
  onShowIcon: (icon: 'explorer' | 'scenarios') => void
}

const heading = 'px-3 pt-3 pb-1 text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]'

// A compact summary that drives the Workspace page; the page keeps the detail.
export function WorkspaceSidebarPanel({ overview, error, loading, onRefresh, environments, activeEnvPath, onEnvChange, onShowAttention, onShowIcon }: Props) {
  return (
    <div data-testid="sidebar-workspace" className="flex flex-col flex-1 overflow-auto text-xs">
      <div className="shrink-0 flex items-center gap-2 px-3 py-2 border-b border-[var(--color-border)]">
        <span className="flex-1 min-w-0 truncate font-semibold" title={overview?.path}>
          {overview?.name || 'Workspace'}
          {overview?.temporary && (
            <span className="ml-1.5 rounded border border-amber-500/40 bg-amber-500/10 px-1 py-px text-[9px] font-medium uppercase text-amber-400">temporary</span>
          )}
        </span>
        <button type="button" aria-label="Refresh workspace" title="Refresh" onClick={onRefresh} disabled={loading} className="shrink-0 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] disabled:opacity-50">
          <RefreshCw size={12} className={cn(loading && 'animate-spin')} aria-hidden />
        </button>
      </div>

      {error && <p role="alert" className="px-3 py-2 text-red-400">{error}</p>}
      {!overview && !error && <p className="px-3 py-3 text-[var(--color-muted-foreground)]">Reading the workspace…</p>}

      {overview && (
        <>
          <h3 className={heading}>Environment</h3>
          <div className="px-3 pb-2 flex flex-col gap-1">
            <select
              aria-label="Environment"
              value={activeEnvPath}
              onChange={e => onEnvChange(e.target.value)}
              className="w-full rounded border border-[var(--color-border)] bg-[var(--color-background)] px-1.5 py-1"
            >
              <option value="">No environment</option>
              {environments.map(env => <option key={env.path} value={env.path}>{env.name}</option>)}
            </select>
            <span data-testid="sidebar-env-source" className="text-[var(--color-muted-foreground)]">{ENV_SOURCE_LABEL[overview.environmentSource]}</span>
          </div>

          <h3 className={heading}>
            Needs attention{overview.attention.length > 0 && <span className="ml-1.5 tabular-nums">{overview.attention.length}</span>}
          </h3>
          {overview.attention.length === 0 ? (
            <p data-testid="sidebar-attention-empty" className="px-3 pb-2 text-emerald-400">Nothing needs attention.</p>
          ) : (
            <ul aria-label="Needs attention" className="pb-2">
              {overview.attention.map((a, i) => (
                <li key={`${a.id}-${i}`}>
                  <button
                    type="button"
                    onClick={() => onShowAttention(i)}
                    title={`${a.what} Fix: ${a.fix}`}
                    className="w-full flex items-start gap-1.5 px-3 py-1.5 text-left hover:bg-[var(--color-muted)]"
                  >
                    {a.severity === 'warning'
                      ? <AlertTriangle size={12} className="shrink-0 mt-0.5 text-amber-400" aria-label="warning" />
                      : <Info size={12} className="shrink-0 mt-0.5 text-sky-400" aria-label="info" />}
                    <span className="line-clamp-2">{a.what}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}

          <h3 className={heading}>Contents</h3>
          <div className="px-3 pb-3 flex flex-col gap-1">
            <button type="button" onClick={() => onShowIcon('explorer')} className="text-left hover:underline">{overview.requests} {overview.requests === 1 ? 'request' : 'requests'}</button>
            <button type="button" onClick={() => onShowIcon('scenarios')} className="text-left hover:underline">{overview.scenarios} {overview.scenarios === 1 ? 'scenario' : 'scenarios'}</button>
            <span>{overview.specs.length} {overview.specs.length === 1 ? 'spec' : 'specs'}</span>
          </div>
        </>
      )}
    </div>
  )
}
