import { useState, useEffect, useCallback } from 'react'
import { RefreshCw, BarChart2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AccumulatedCoverage } from '@/lib/bridge'
import { listAccumulatedCoverage } from '@/lib/bridge'

type Props = {
  workspacePath: string
  selectedSpecFile: string | null
  onSelectSpec: (specFile: string) => void
}

function specLabel(specFile: string): string {
  return specFile.replace(/\\/g, '/').split('/').pop() ?? specFile
}

export function CoverageSidebarPanel({ workspacePath, selectedSpecFile, onSelectSpec }: Props) {
  const [entries, setEntries] = useState<AccumulatedCoverage[]>([])
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    if (!workspacePath) { setEntries([]); return }
    setLoading(true)
    try {
      const data = await listAccumulatedCoverage()
      setEntries(data)
      if (!selectedSpecFile && data.length > 0) {
        onSelectSpec(data[0].specFile)
      }
    } finally {
      setLoading(false)
    }
  }, [workspacePath, selectedSpecFile, onSelectSpec])

  useEffect(() => { refresh() }, [refresh])

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
          Contracts
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
        ) : entries.map(entry => {
          const pct = Math.round(entry.percent)
          const barColor = pct >= 80 ? 'bg-emerald-500' : pct >= 40 ? 'bg-amber-500' : 'bg-red-500'
          const pctColor = pct >= 80 ? 'text-emerald-400' : pct >= 40 ? 'text-amber-400' : 'text-red-400'
          const isSelected = entry.specFile === selectedSpecFile

          return (
            <button
              key={entry.specFile}
              type="button"
              onClick={() => onSelectSpec(entry.specFile)}
              className={cn(
                'relative w-full text-left px-3 py-2.5 border-b border-[var(--color-border)] transition-colors',
                isSelected ? 'bg-[var(--color-muted)]' : 'hover:bg-[var(--color-muted)]',
              )}
            >
              {isSelected && (
                <span className="absolute left-0 inset-y-0 w-[3px] rounded-r-full bg-emerald-400" />
              )}
              <div className="flex items-center justify-between gap-2 mb-1.5">
                <span className="text-xs font-medium text-[var(--color-foreground)] truncate" title={entry.specFile}>
                  {specLabel(entry.specFile)}
                </span>
                <span className={cn('text-xs font-bold tabular-nums shrink-0', pctColor)}>{pct}%</span>
              </div>
              <div className="h-1 rounded-full bg-[var(--color-border)] overflow-hidden mb-1.5">
                <div className={cn('h-full rounded-full transition-all', barColor)} style={{ width: `${Math.min(pct, 100)}%` }} />
              </div>
              <div className="flex flex-wrap items-center gap-x-2.5 gap-y-0.5 text-[10px] text-[var(--color-muted-foreground)]">
                <span className={cn('font-medium', pctColor)}>{entry.covered}/{entry.total} ops</span>
                <span>{entry.totalRuns} run{entry.totalRuns !== 1 ? 's' : ''}</span>
                <span className="text-emerald-400">{entry.passedRuns} passed</span>
                {entry.failedRuns > 0 && <span className="text-red-400">{entry.failedRuns} failed</span>}
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
