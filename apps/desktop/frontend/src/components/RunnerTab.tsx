import { CheckCircle2, XCircle, Loader2, Play } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { RunAllState } from '@/types'

type RunnerTabProps = {
  runAllState: RunAllState | null
  onSelectScenario?: (path: string, name: string) => void
  onCancel?: () => void
  onDismiss?: () => void
  onRunAll?: () => void
}

export function RunnerTab({ runAllState, onSelectScenario, onCancel, onDismiss, onRunAll }: RunnerTabProps) {
  if (!runAllState) {
    return (
      <div className="flex flex-col items-center justify-center flex-1 gap-4 p-8 text-center">
        <Play size={32} className="text-[var(--color-muted-foreground)] opacity-40" />
        <p className="text-sm text-[var(--color-muted-foreground)]">No run in progress.</p>
        {onRunAll && (
          <button
            type="button"
            onClick={onRunAll}
            className="px-4 py-1.5 bg-[var(--color-accent)] text-white text-xs font-semibold rounded hover:opacity-90 transition-opacity"
          >
            Run All Scenarios
          </button>
        )}
      </div>
    )
  }

  const { running, done, total, passed, results } = runAllState
  const failed = results.filter(r => !r.passed).length
  const totalDurationMs = results.reduce((s, r) => s + (r.durationMs ?? 0), 0)

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="shrink-0 flex items-center justify-between px-6 py-3 border-b border-[var(--color-border)] bg-[var(--color-muted)]">
        <div className="flex items-center gap-4">
          {running ? (
            <>
              <Loader2 size={16} className="animate-spin text-[var(--color-accent)]" />
              <div>
                <p className="text-sm font-semibold text-[var(--color-foreground)]">
                  Running {done + 1} / {total}…
                </p>
                <p className="text-xs text-[var(--color-muted-foreground)]">{passed} passed so far</p>
              </div>
            </>
          ) : (
            <>
              {failed === 0
                ? <CheckCircle2 size={18} className="text-emerald-400 shrink-0" />
                : <XCircle size={18} className="text-red-400 shrink-0" />
              }
              <div>
                <p className={cn('text-sm font-semibold', failed === 0 ? 'text-emerald-400' : 'text-red-400')}>
                  {failed === 0 ? 'All passed' : `${failed} failed`}
                </p>
                <p className="text-xs text-[var(--color-muted-foreground)]">
                  {passed}/{total} scenarios · {totalDurationMs}ms total
                </p>
              </div>
            </>
          )}
        </div>
        <div className="flex items-center gap-2">
          {running && onCancel && (
            <button
              type="button"
              onClick={onCancel}
              className="px-3 py-1 text-xs font-medium text-amber-400 hover:text-amber-300 border border-amber-400/30 rounded transition-colors"
            >
              Stop
            </button>
          )}
          {!running && onRunAll && (
            <button
              type="button"
              onClick={onRunAll}
              className="px-3 py-1 text-xs font-medium bg-[var(--color-accent)] text-white rounded hover:opacity-90 transition-opacity"
            >
              Run Again
            </button>
          )}
          {!running && onDismiss && (
            <button
              type="button"
              onClick={onDismiss}
              className="text-[10px] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] px-2 py-1 rounded transition-colors"
            >
              Clear
            </button>
          )}
        </div>
      </div>

      {/* Progress bar */}
      {running && (
        <div className="shrink-0 h-1 bg-[var(--color-border)]">
          <div
            className="h-full bg-[var(--color-accent)] transition-all duration-300"
            style={{ width: total > 0 ? `${(done / total) * 100}%` : '0%' }}
          />
        </div>
      )}

      {/* Results list */}
      <div className="flex-1 overflow-auto">
        {results.length === 0 && running && (
          <div className="flex items-center justify-center h-32">
            <p className="text-xs text-[var(--color-muted-foreground)]">Waiting for first result…</p>
          </div>
        )}
        <table className="w-full text-xs">
          <thead className="sticky top-0 bg-[var(--color-card)] border-b border-[var(--color-border)]">
            <tr>
              <th className="text-left px-6 py-2 font-medium text-[var(--color-muted-foreground)] w-8"></th>
              <th className="text-left px-2 py-2 font-medium text-[var(--color-muted-foreground)]">Scenario</th>
              <th className="text-right px-6 py-2 font-medium text-[var(--color-muted-foreground)] w-20">Duration</th>
            </tr>
          </thead>
          <tbody>
            {results.map((r, i) => (
              <tr
                key={r.path || i}
                onClick={() => onSelectScenario?.(r.path, r.name)}
                className="border-b border-[var(--color-border)]/50 hover:bg-[var(--color-muted)] cursor-pointer transition-colors"
                title={r.error ?? 'Click to open scenario'}
              >
                <td className="px-6 py-2.5">
                  {r.error
                    ? <XCircle size={13} className="text-red-400" />
                    : r.passed
                      ? <CheckCircle2 size={13} className="text-emerald-400" />
                      : <XCircle size={13} className="text-amber-400" />}
                </td>
                <td className="px-2 py-2.5">
                  <span className={cn('font-medium', r.passed ? 'text-[var(--color-foreground)]' : 'text-[var(--color-foreground)]')}>
                    {r.name}
                  </span>
                  {r.error && (
                    <span className="ml-2 text-red-400 text-[10px]">{r.error}</span>
                  )}
                </td>
                <td className="px-6 py-2.5 text-right text-[var(--color-muted-foreground)] tabular-nums">
                  {r.durationMs != null ? `${r.durationMs}ms` : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
