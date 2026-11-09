import { X, Loader2, CheckCircle2, XCircle } from 'lucide-react'
import type { FolderRunState } from './WorkspacePanel'
import type { RunAllState } from '@/components/Sidebar'
import { MethodBadge } from './WorkspaceFileTree'

export function FolderRunResults({ state, onSelectFile, onDismiss }: {
  state: FolderRunState
  onSelectFile: (path: string) => void
  onDismiss?: () => void
}) {
  return (
    <div className="mx-2 mt-2 mb-1 rounded border border-[var(--color-border)] bg-[var(--color-muted)] text-xs overflow-hidden">
      <div className="flex items-center justify-between px-2 py-1.5 border-b border-[var(--color-border)]">
        <span className="font-semibold text-[var(--color-foreground)] truncate flex-1 min-w-0 mr-2" title={state.folderName}>
          {state.running
            ? <span className="flex items-center gap-1.5"><Loader2 size={10} className="animate-spin" /> Running {state.folderName}…</span>
            : <span className="flex items-center gap-1"><span className="truncate">{state.folderName}</span> <span className="shrink-0 text-[var(--color-muted-foreground)]">— {state.results.filter(r => r.passed).length}/{state.results.length} ok</span></span>}
        </span>
        {!state.running && onDismiss && (
          <button type="button" onClick={onDismiss} className="shrink-0 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]">
            <X size={10} />
          </button>
        )}
      </div>
      {state.results.map(r => (
        <button
          key={r.path}
          type="button"
          onClick={() => onSelectFile(r.path)}
          className="w-full flex items-center gap-1.5 px-2 py-1 text-left hover:bg-[var(--color-border)] transition-colors"
          title={r.error || undefined}
        >
          {r.error
            ? <XCircle size={10} className="shrink-0 text-red-400" />
            : r.passed
              ? <CheckCircle2 size={10} className="shrink-0 text-emerald-400" />
              : <XCircle size={10} className="shrink-0 text-amber-400" />}
          {r.method && <MethodBadge method={r.method} />}
          <span className="flex-1 truncate text-[var(--color-foreground)]">{r.name}</span>
          {r.error
            ? <span className="shrink-0 text-[10px] text-red-400 truncate max-w-[80px]" title={r.error}>err</span>
            : <span className="shrink-0 text-[var(--color-muted-foreground)] text-[10px]">{r.statusCode} · {r.durationMs}ms</span>}
        </button>
      ))}
    </div>
  )
}

export function RunAllResults({ state, onSelectScenario, onDismiss }: {
  state: RunAllState
  onSelectScenario?: (path: string, name: string) => void
  onDismiss?: () => void
}) {
  return (
    <div className="mx-2 mt-2 mb-1 rounded border border-[var(--color-border)] bg-[var(--color-muted)] text-xs overflow-hidden">
      <div className="flex items-center justify-between px-2 py-1.5 border-b border-[var(--color-border)]">
        <span className="font-semibold text-[var(--color-foreground)]">
          {state.running
            ? <span className="flex items-center gap-1.5"><Loader2 size={10} className="animate-spin" /> Running {state.done + 1}/{state.total}…</span>
            : `${state.passed}/${state.total} passed`}
        </span>
        {!state.running && onDismiss && (
          <button type="button" onClick={onDismiss} className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]">
            <X size={10} />
          </button>
        )}
      </div>
      {state.results.map(r => (
        <button
          key={r.path}
          type="button"
          onClick={() => onSelectScenario?.(r.path, r.name)}
          className="w-full flex items-center gap-1.5 px-2 py-1 text-left hover:bg-[var(--color-border)] transition-colors"
        >
          {r.passed
            ? <CheckCircle2 size={10} className="shrink-0 text-emerald-400" />
            : <XCircle size={10} className="shrink-0 text-red-400" />}
          <span className="flex-1 truncate text-[var(--color-foreground)]">{r.name}</span>
          <span className="shrink-0 text-[var(--color-muted-foreground)] text-[10px]">{r.durationMs}ms</span>
        </button>
      ))}
    </div>
  )
}
