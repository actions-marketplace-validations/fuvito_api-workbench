import { useState, useEffect, useCallback } from 'react'
import { RotateCcw, RefreshCw, BarChart2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AccumulatedCoverage } from '@/lib/bridge'
import { listAccumulatedCoverage, resetAccumulatedCoverage } from '@/lib/bridge'

type Props = {
  workspacePath: string
}

function specLabel(specFile: string): string {
  const parts = specFile.replace(/\\/g, '/').split('/')
  return parts[parts.length - 1] || specFile
}

export function CoverageDashboard({ workspacePath }: Props) {
  const [entries, setEntries] = useState<AccumulatedCoverage[]>([])
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const data = await listAccumulatedCoverage()
      setEntries(data)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (workspacePath) refresh()
    else setEntries([])
  }, [workspacePath, refresh])

  async function handleReset(specFile: string) {
    await resetAccumulatedCoverage(specFile)
    setEntries(prev => prev.filter(e => e.specFile !== specFile))
  }

  if (!workspacePath) {
    return (
      <div className="flex flex-col items-center gap-2 px-4 py-8 text-center">
        <BarChart2 size={20} className="text-[var(--color-muted-foreground)] opacity-30" />
        <p className="text-xs text-[var(--color-muted-foreground)]">No workspace open</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="shrink-0 flex items-center justify-between px-3 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-background)]">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          Spec Coverage
        </span>
        <button
          type="button"
          onClick={refresh}
          title="Refresh"
          className="p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
        >
          <RefreshCw size={11} className={loading ? 'animate-spin' : ''} />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto">
        {entries.length === 0 && !loading ? (
          <div className="flex flex-col items-center gap-2 px-4 py-8 text-center">
            <BarChart2 size={20} className="text-[var(--color-muted-foreground)] opacity-30" />
            <p className="text-xs text-[var(--color-muted-foreground)]">No coverage data yet</p>
            <p className="text-[11px] text-[var(--color-muted-foreground)] opacity-60">
              Run a scenario with a specFile to track coverage
            </p>
          </div>
        ) : (
          entries.map(entry => {
            const pct = Math.round(entry.percent)
            const barColor = pct >= 100 ? 'bg-emerald-500' : pct >= 50 ? 'bg-amber-500' : 'bg-red-500'
            const pctColor = pct >= 100 ? 'text-emerald-400' : pct >= 50 ? 'text-amber-400' : 'text-red-400'
            return (
              <div key={entry.specFile} className="px-3 py-2.5 border-b border-[var(--color-border)] last:border-0">
                <div className="flex items-start justify-between gap-1 mb-1">
                  <span
                    className="text-xs font-medium text-[var(--color-foreground)] truncate"
                    title={entry.specFile}
                  >
                    {specLabel(entry.specFile)}
                  </span>
                  <button
                    type="button"
                    onClick={() => handleReset(entry.specFile)}
                    title="Reset coverage"
                    className="shrink-0 p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-red-400 hover:bg-[var(--color-muted)] transition-colors"
                  >
                    <RotateCcw size={10} />
                  </button>
                </div>

                <div className="flex items-center gap-2 mb-1">
                  <span className={cn('text-xs font-semibold tabular-nums', pctColor)}>
                    {entry.covered} / {entry.total} ops ({pct}%)
                  </span>
                </div>

                <div className="h-1.5 rounded-full bg-[var(--color-muted)] overflow-hidden mb-1.5">
                  <div
                    className={cn('h-full rounded-full transition-all', barColor)}
                    style={{ width: `${Math.min(entry.percent, 100)}%` }}
                  />
                </div>

                <div className="flex flex-wrap gap-x-2.5 gap-y-0.5">
                  <span className="text-[10px] text-[var(--color-muted-foreground)]">
                    {entry.totalRuns} run{entry.totalRuns === 1 ? '' : 's'}
                  </span>
                  <span className="text-[10px] text-emerald-400">
                    {entry.passedRuns} passed
                  </span>
                  {entry.failedRuns > 0 && (
                    <span className="text-[10px] text-red-400">
                      {entry.failedRuns} failed
                    </span>
                  )}
                  {entry.lastRunAt && (
                    <span className="text-[10px] text-[var(--color-muted-foreground)]">
                      {new Date(entry.lastRunAt).toLocaleDateString()}
                    </span>
                  )}
                </div>
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
