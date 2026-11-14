import { Pencil } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { EnvInfo } from '@/lib/bridge'

type StatusBarProps = {
  environments: EnvInfo[]
  activeEnv: EnvInfo | null
  workspacePath: string
  demoRunning: boolean
  demoPort: number
  onEnvChange: (path: string) => void
  onEditEnv: () => void
  onStopDemo: () => void
}

export function StatusBar({
  environments, activeEnv, workspacePath,
  demoRunning, demoPort,
  onEnvChange, onEditEnv, onStopDemo,
}: StatusBarProps) {
  const workspaceName = workspacePath
    ? workspacePath.replace(/\\/g, '/').split('/').filter(Boolean).pop() ?? null
    : null

  return (
    <footer className="flex items-center gap-2 px-2 h-6 shrink-0 border-t border-[var(--color-border)] bg-[var(--color-card)]">
      <div className="flex items-center gap-1.5 min-w-0 flex-1 text-[10px]">
        {environments.length > 0 && (
          <>
            <select
              value={activeEnv?.path ?? ''}
              onChange={e => onEnvChange(e.target.value)}
              className={cn(
                'bg-transparent border-none outline-none cursor-pointer text-[10px] max-w-[140px]',
                activeEnv ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)]',
              )}
            >
              <option value="">No environment</option>
              {environments.map(env => (
                <option key={env.path} value={env.path}>{env.name}</option>
              ))}
            </select>
            {workspacePath && (
              <button
                type="button"
                onClick={onEditEnv}
                title={activeEnv ? `Edit environment: ${activeEnv.name}` : 'Create environment'}
                className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors leading-none"
              >
                <Pencil size={10} />
              </button>
            )}
            {demoRunning && <span className="w-px h-3 bg-[var(--color-border)]" />}
          </>
        )}

        {demoRunning && (
          <>
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shrink-0" />
            <button
              type="button"
              onClick={() => navigator.clipboard.writeText(`http://localhost:${demoPort}`).catch(() => {})}
              title={`Copy base URL · http://localhost:${demoPort}`}
              className="text-emerald-400 font-mono hover:text-emerald-300 transition-colors"
            >
              Demo :{demoPort}
            </button>
            <button
              type="button"
              onClick={onStopDemo}
              title="Stop demo server"
              className="text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors leading-none"
            >
              ✕
            </button>
          </>
        )}
      </div>

      {workspaceName && (
        <span
          className="ml-auto text-[10px] text-[var(--color-muted-foreground)] truncate max-w-[200px] select-none"
          title={workspacePath}
        >
          {workspaceName}
        </span>
      )}
    </footer>
  )
}
