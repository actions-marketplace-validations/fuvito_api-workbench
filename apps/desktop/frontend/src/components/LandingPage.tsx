import { FolderOpen, FolderPlus, Play, Clock, ChevronRight, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

type Props = {
  recentWorkspaces: string[]
  demoStarting: boolean
  demoRunning: boolean
  onNewWorkspace: () => void
  onOpenWorkspace: () => void
  onOpenRecentWorkspace: (path: string) => void
  onTryDemo: () => void
  onStopDemo: () => void
}

function workspaceName(p: string) {
  return p.replace(/\\/g, '/').split('/').filter(Boolean).pop() ?? p
}

function workspaceDir(p: string) {
  const parts = p.replace(/\\/g, '/').split('/').filter(Boolean)
  return parts.length > 1 ? parts.slice(0, -1).join('/') : p
}

export function LandingPage({
  recentWorkspaces, demoStarting, demoRunning,
  onNewWorkspace, onOpenWorkspace, onOpenRecentWorkspace, onTryDemo, onStopDemo,
}: Props) {
  return (
    <div className="flex-1 flex items-center justify-center bg-[var(--color-background)] overflow-auto">
      <div className="flex flex-col items-center w-full max-w-sm gap-8 px-6 py-10">

        {/* Brand */}
        <div className="flex flex-col items-center gap-2 text-center select-none">
          <img src="/appicon.png" alt="Clear Signal" className="w-16 h-16 rounded-2xl shadow-lg mb-1" draggable={false} />
          <h1 className="text-2xl font-bold text-[var(--color-foreground)] tracking-tight">Clear Signal</h1>
          <p className="text-xs text-[var(--color-muted-foreground)]">API Contract Validation &amp; Testing</p>
        </div>

        {/* Primary actions */}
        <div className="flex flex-col gap-2 w-full">
          <button
            type="button"
            onClick={onNewWorkspace}
            className={cn(
              'flex items-center gap-2.5 px-4 py-2.5 rounded-lg text-sm font-semibold',
              'bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity',
            )}
          >
            <FolderPlus size={15} className="shrink-0" />
            New Workspace
          </button>
          <button
            type="button"
            onClick={onOpenWorkspace}
            className={cn(
              'flex items-center gap-2.5 px-4 py-2.5 rounded-lg text-sm',
              'bg-[var(--color-muted)] text-[var(--color-foreground)] border border-[var(--color-border)]',
              'hover:bg-[var(--color-border)] transition-colors',
            )}
          >
            <FolderOpen size={15} className="shrink-0" />
            Open Workspace…
          </button>
        </div>

        {/* Demo */}
        <div className="w-full rounded-lg border border-[var(--color-accent)]/30 bg-[var(--color-accent)]/5 px-4 py-3.5">
          <p className="text-xs font-semibold text-[var(--color-foreground)] mb-1">Not sure where to start?</p>
          <p className="text-[11px] text-[var(--color-muted-foreground)] mb-3 leading-relaxed">
            Launch a local sample API server and explore Clear Signal in one click.
          </p>
          {demoRunning ? (
            <button
              type="button"
              onClick={onStopDemo}
              className="flex items-center gap-1.5 text-xs text-red-400 hover:underline"
            >
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
              Demo running — Stop demo
            </button>
          ) : (
            <button
              type="button"
              onClick={onTryDemo}
              disabled={demoStarting}
              className="flex items-center gap-1.5 text-xs font-semibold text-[var(--color-accent)] hover:underline disabled:opacity-50 transition-opacity"
            >
              {demoStarting
                ? <><Loader2 size={11} className="animate-spin" /> Starting demo…</>
                : <><Play size={11} /> Try the demo</>
              }
            </button>
          )}
        </div>

        {/* Recent workspaces */}
        {recentWorkspaces.length > 0 && (
          <div className="w-full">
            <p className="text-[10px] uppercase font-semibold text-[var(--color-muted-foreground)] tracking-wider mb-2 flex items-center gap-1.5">
              <Clock size={10} />
              Recent
            </p>
            <ul className="flex flex-col">
              {recentWorkspaces.slice(0, 6).map(p => (
                <li key={p}>
                  <button
                    type="button"
                    onClick={() => onOpenRecentWorkspace(p)}
                    className="w-full flex items-center gap-2 px-2 py-1.5 rounded text-left text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors group"
                  >
                    <FolderOpen size={12} className="shrink-0 opacity-50" />
                    <div className="flex-1 min-w-0">
                      <span className="font-medium text-[var(--color-foreground)] truncate block">{workspaceName(p)}</span>
                      <span className="text-[10px] opacity-50 truncate block">{workspaceDir(p)}</span>
                    </div>
                    <ChevronRight size={11} className="shrink-0 opacity-0 group-hover:opacity-60 transition-opacity" />
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  )
}
