import { useState, useEffect, useCallback } from 'react'
import { RotateCcw, RefreshCw, BarChart2, ChevronDown, ChevronRight } from 'lucide-react'
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

function OpBadge({ op, covered }: { op: string; covered: boolean }) {
  const [method, ...rest] = op.split(' ')
  const path = rest.join(' ')
  const methodColor: Record<string, string> = {
    GET: 'text-sky-400', POST: 'text-emerald-400', PUT: 'text-amber-400',
    PATCH: 'text-orange-400', DELETE: 'text-red-400', HEAD: 'text-purple-400',
    OPTIONS: 'text-pink-400',
  }
  return (
    <div className={cn('flex items-center gap-1.5 py-0.5', covered ? '' : 'opacity-50')}>
      <span className={cn('text-[10px] font-mono font-bold w-12 shrink-0', methodColor[method] ?? 'text-[var(--color-foreground)]')}>
        {method}
      </span>
      <span className="text-[10px] font-mono text-[var(--color-muted-foreground)] truncate">
        {path}
      </span>
      {covered && (
        <span className="ml-auto shrink-0 text-[9px] text-emerald-400">✓</span>
      )}
    </div>
  )
}

export function CoverageDashboard({ workspacePath }: Props) {
  const [entries, setEntries] = useState<AccumulatedCoverage[]>([])
  const [loading, setLoading] = useState(false)
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

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
    // eslint-disable-next-line react/set-state-in-effect
    if (workspacePath) refresh()
    // eslint-disable-next-line react/set-state-in-effect
    else setEntries([])
  }, [workspacePath, refresh])

  async function handleReset(specFile: string) {
    await resetAccumulatedCoverage(specFile)
    setEntries(prev => prev.filter(e => e.specFile !== specFile))
    setExpanded(prev => { const s = new Set(prev); s.delete(specFile); return s })
  }

  function toggleExpanded(specFile: string) {
    setExpanded(prev => {
      const s = new Set(prev)
      if (s.has(specFile)) s.delete(specFile)
      else s.add(specFile)
      return s
    })
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
            const isExpanded = expanded.has(entry.specFile)
            const allOps = [
              ...(entry.exercised ?? []).map(op => ({ op, covered: true })),
              ...(entry.missed ?? []).map(op => ({ op, covered: false })),
            ].sort((a, b) => a.op.localeCompare(b.op))

            return (
              <div key={entry.specFile} className="border-b border-[var(--color-border)] last:border-0">
                {/* Summary row — click to expand */}
                <button
                  type="button"
                  onClick={() => toggleExpanded(entry.specFile)}
                  className="w-full text-left px-3 pt-2.5 pb-2 hover:bg-[var(--color-muted)] transition-colors"
                >
                  <div className="flex items-center gap-1 mb-1">
                    <span className="text-[var(--color-muted-foreground)] shrink-0">
                      {isExpanded
                        ? <ChevronDown size={11} />
                        : <ChevronRight size={11} />
                      }
                    </span>
                    <span
                      className="text-xs font-medium text-[var(--color-foreground)] truncate flex-1"
                      title={entry.specFile}
                    >
                      {specLabel(entry.specFile)}
                    </span>
                    <span className={cn('text-xs font-semibold tabular-nums shrink-0', pctColor)}>
                      {pct}%
                    </span>
                  </div>

                  <div className="ml-4">
                    <div className="h-1.5 rounded-full bg-[var(--color-muted)] overflow-hidden mb-1.5">
                      <div
                        className={cn('h-full rounded-full transition-all', barColor)}
                        style={{ width: `${Math.min(entry.percent, 100)}%` }}
                      />
                    </div>
                    <div className="flex flex-wrap gap-x-2.5 gap-y-0.5">
                      <span className={cn('text-[10px] font-semibold tabular-nums', pctColor)}>
                        {entry.covered}/{entry.total} ops
                      </span>
                      <span className="text-[10px] text-[var(--color-muted-foreground)]">
                        {entry.totalRuns} run{entry.totalRuns === 1 ? '' : 's'}
                      </span>
                      <span className="text-[10px] text-emerald-400">{entry.passedRuns} passed</span>
                      {entry.failedRuns > 0 && (
                        <span className="text-[10px] text-red-400">{entry.failedRuns} failed</span>
                      )}
                      {entry.lastRunAt && (
                        <span className="text-[10px] text-[var(--color-muted-foreground)]">
                          {new Date(entry.lastRunAt).toLocaleDateString()}
                        </span>
                      )}
                    </div>
                  </div>
                </button>

                {/* Detail panel */}
                {isExpanded && (
                  <div className="px-3 pb-2.5 bg-[var(--color-card)]">
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="text-[9px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
                        Operations
                      </span>
                      <button
                        type="button"
                        onClick={(e) => { e.stopPropagation(); handleReset(entry.specFile) }}
                        title="Reset coverage"
                        className="flex items-center gap-1 text-[10px] text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors"
                      >
                        <RotateCcw size={9} /> Reset
                      </button>
                    </div>

                    {allOps.length === 0 ? (
                      <p className="text-[10px] text-[var(--color-muted-foreground)] italic">
                        No operation details available — run a scenario to populate.
                      </p>
                    ) : (
                      <div className="divide-y divide-[var(--color-border)]">
                        {allOps.map(({ op, covered }) => (
                          <OpBadge key={op} op={op} covered={covered} />
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
