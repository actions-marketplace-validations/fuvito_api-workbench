import { Play, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

type Props = {
  onTryDemo?: () => void
  demoStarting?: boolean
  onNewWorkspace: () => void
  onOpenWorkspace: () => void
}

export function WorkspaceEmptyState({ onTryDemo, demoStarting, onNewWorkspace, onOpenWorkspace }: Props) {
  return (
    <div className="flex flex-col gap-2 p-3">
      {onTryDemo && (
        <div className="rounded border border-[var(--color-accent)]/40 bg-[var(--color-accent)]/5 p-3 mb-1">
          <p className="text-xs font-semibold text-[var(--color-foreground)] mb-1">Try the demo</p>
          <p className="text-[11px] text-[var(--color-muted-foreground)] mb-2.5 leading-relaxed">
            Start a local API server with sample data, open example requests, and explore Clear Signal in one click.
          </p>
          <button
            type="button"
            onClick={onTryDemo}
            disabled={demoStarting}
            className={cn(
              'w-full flex items-center justify-center gap-1.5 text-xs rounded px-3 py-1.5 font-semibold',
              'bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity disabled:opacity-50',
            )}
          >
            {demoStarting ? (
              <><Loader2 size={11} className="animate-spin" /> Starting…</>
            ) : (
              <><Play size={11} /> Try the Demo</>
            )}
          </button>
        </div>
      )}
      <p className="text-xs text-[var(--color-muted-foreground)]">
        Or open your own workspace folder.
      </p>
      <button
        type="button"
        onClick={onNewWorkspace}
        className={cn(
          'text-xs rounded px-3 py-1.5 font-semibold',
          'bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity',
        )}
      >
        New Workspace
      </button>
      <button
        type="button"
        onClick={onOpenWorkspace}
        className={cn(
          'text-xs rounded px-3 py-1.5',
          'bg-[var(--color-muted)] text-[var(--color-foreground)] border border-[var(--color-border)]',
          'hover:bg-[var(--color-border)] transition-colors',
        )}
      >
        Open Existing
      </button>
    </div>
  )
}
