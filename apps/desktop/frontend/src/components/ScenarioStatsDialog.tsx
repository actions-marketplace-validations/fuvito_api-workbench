import { CheckCircle2, XCircle, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScenarioResult } from '@/lib/bridge'
import { statusColor } from './ScenarioStepResult'

type Props = {
  result: ScenarioResult
  onClose: () => void
}

export function ScenarioStatsDialog({ result, onClose }: Props) {
  const steps = result.steps ?? []
  const passed = steps.filter(s => s.passed && !s.skipped).length
  const failed = steps.filter(s => !s.passed && !s.skipped).length
  const skipped = steps.filter(s => s.skipped).length
  const maxMs = Math.max(...steps.map(s => s.durationMs ?? 0), 1)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={onClose}>
      <div className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg shadow-xl w-[540px] max-h-[70vh] flex flex-col overflow-hidden"
        onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border)]">
          <h3 className="text-sm font-semibold text-[var(--color-foreground)]">Run Statistics</h3>
          <button type="button" onClick={onClose}
            className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
            <X size={14} />
          </button>
        </div>

        {/* Summary */}
        <div className="shrink-0 flex items-center gap-3 px-4 py-3 border-b border-[var(--color-border)] bg-[var(--color-muted)]/40">
          <span className={cn('text-xs font-semibold px-2 py-0.5 rounded',
            result.passed ? 'bg-emerald-500/15 text-emerald-400' : 'bg-red-500/15 text-red-400')}>
            {result.passed ? 'PASSED' : 'FAILED'}
          </span>
          <span className="text-xs text-[var(--color-muted-foreground)]">
            {passed} passed · {failed} failed · {skipped} skipped
          </span>
          <span className="ml-auto text-xs font-mono text-[var(--color-foreground)]">
            {result.durationMs} ms total
          </span>
        </div>

        {/* Per-step table */}
        <div className="overflow-y-auto flex-1">
          <table className="w-full text-xs">
            <thead className="sticky top-0 bg-[var(--color-card)] border-b border-[var(--color-border)]">
              <tr className="text-[var(--color-muted-foreground)]">
                <th className="text-left px-4 py-2 font-medium w-6">#</th>
                <th className="text-left px-2 py-2 font-medium">Step</th>
                <th className="text-right px-2 py-2 font-medium w-14">Status</th>
                <th className="text-right px-4 py-2 font-medium w-24">Duration</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--color-border)]">
              {steps.map((sr, i) => (
                <tr key={i} className="hover:bg-[var(--color-muted)]/30 transition-colors">
                  <td className="px-4 py-2.5">
                    <span className={cn('w-5 h-5 rounded-full flex items-center justify-center font-bold text-[11px]',
                      sr.skipped ? 'bg-[var(--color-muted)] text-[var(--color-muted-foreground)]'
                      : sr.passed ? 'bg-emerald-500/20 text-emerald-400'
                      : 'bg-red-500/20 text-red-400')}>
                      {i + 1}
                    </span>
                  </td>
                  <td className="px-2 py-2.5">
                    <div className="font-medium text-[var(--color-foreground)] truncate max-w-[220px]">
                      {sr.title || `Step ${i + 1}`}
                    </div>
                    <div className="flex items-center gap-1.5 mt-1">
                      {sr.skipped
                        ? <span className="text-[var(--color-muted-foreground)]">skipped</span>
                        : sr.passed
                          ? <CheckCircle2 size={11} className="text-emerald-400" />
                          : <XCircle size={11} className="text-red-400" />}
                      <span className="text-[var(--color-muted-foreground)]">
                        {(sr.assertions ?? []).filter(a => a.passed).length}/{(sr.assertions ?? []).length} assertions
                      </span>
                    </div>
                  </td>
                  <td className="px-2 py-2.5 text-right">
                    {sr.statusCode > 0 && (
                      <span className={cn('font-mono font-bold', statusColor(sr.statusCode))}>
                        {sr.statusCode}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-2.5">
                    <div className="flex flex-col items-end gap-1">
                      <span className="font-mono text-[var(--color-foreground)]">
                        {sr.durationMs > 0 ? `${sr.durationMs} ms` : '—'}
                      </span>
                      {sr.durationMs > 0 && (
                        <div className="w-full h-1 bg-[var(--color-border)] rounded-full overflow-hidden">
                          <div
                            className={cn('h-full rounded-full', sr.passed ? 'bg-emerald-500' : 'bg-red-500')}
                            style={{width: `${Math.round((sr.durationMs / maxMs) * 100)}%`}}
                          />
                        </div>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
