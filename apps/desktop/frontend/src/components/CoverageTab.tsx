import { useState, useEffect, useCallback } from 'react'
import { RotateCcw, RefreshCw, BarChart2, AlertCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AccumulatedCoverage } from '@/lib/bridge'
import { listAccumulatedCoverage, resetAccumulatedCoverage } from '@/lib/bridge'

type Props = {
  workspacePath: string
  specFile?: string | null
  onOpenSpecCoverage: (specFile: string) => void
}

const METHOD_COLOR: Record<string, string> = {
  GET: 'text-sky-400', POST: 'text-emerald-400', PUT: 'text-amber-400',
  PATCH: 'text-orange-400', DELETE: 'text-red-400', HEAD: 'text-purple-400',
  OPTIONS: 'text-pink-400',
}

const METHOD_BG: Record<string, string> = {
  GET: 'bg-sky-400/10', POST: 'bg-emerald-400/10', PUT: 'bg-amber-400/10',
  PATCH: 'bg-orange-400/10', DELETE: 'bg-red-400/10', HEAD: 'bg-purple-400/10',
  OPTIONS: 'bg-pink-400/10',
}

function parseOp(op: string): { method: string; path: string } {
  const idx = op.indexOf(' ')
  if (idx === -1) return { method: op, path: '' }
  return { method: op.slice(0, idx), path: op.slice(idx + 1) }
}

function specLabel(specFile: string): string {
  return specFile.replace(/\\/g, '/').split('/').pop() ?? specFile
}

function ProgressBar({ pct }: { pct: number }) {
  const color = pct >= 80 ? 'bg-emerald-500' : pct >= 40 ? 'bg-amber-500' : 'bg-red-500'
  const textColor = pct >= 80 ? 'text-emerald-400' : pct >= 40 ? 'text-amber-400' : 'text-red-400'
  return (
    <div className="flex items-center gap-3">
      <div className="flex-1 h-2 rounded-full bg-[var(--color-muted)] overflow-hidden">
        <div
          className={cn('h-full rounded-full transition-all', color)}
          style={{ width: `${Math.min(pct, 100)}%` }}
        />
      </div>
      <span className={cn('text-sm font-bold tabular-nums w-10 text-right shrink-0', textColor)}>
        {Math.round(pct)}%
      </span>
    </div>
  )
}

function OpRow({ op, covered }: { op: string; covered: boolean }) {
  const { method, path } = parseOp(op)
  return (
    <div className={cn(
      'flex items-center gap-3 px-4 py-2.5 border-b border-[var(--color-border)] last:border-0',
      covered ? '' : 'opacity-80',
    )}>
      <span className={cn(
        'text-[11px] font-mono font-bold w-14 shrink-0 px-1.5 py-0.5 rounded text-center',
        METHOD_COLOR[method] ?? 'text-[var(--color-foreground)]',
        METHOD_BG[method] ?? '',
      )}>
        {method}
      </span>
      <span className="text-sm font-mono text-[var(--color-foreground)] flex-1 truncate">{path}</span>
      {covered
        ? <span className="text-emerald-400 text-sm shrink-0">✓</span>
        : <span className="text-[10px] text-[var(--color-muted-foreground)] shrink-0 italic">not tested</span>
      }
    </div>
  )
}

// ── Overview ──────────────────────────────────────────────────────────────────

function OverviewPanel({
  entries,
  loading,
  onOpenSpec,
  onReset,
}: {
  entries: AccumulatedCoverage[]
  loading: boolean
  onOpenSpec: (specFile: string) => void
  onReset: (specFile: string) => void
}) {
  if (entries.length === 0 && !loading) {
    return (
      <div className="flex flex-col items-center justify-center flex-1 gap-3">
        <BarChart2 size={36} className="text-[var(--color-muted-foreground)] opacity-20" />
        <p className="text-sm text-[var(--color-muted-foreground)]">No coverage data yet</p>
        <p className="text-xs text-[var(--color-muted-foreground)] opacity-60">
          Run a scenario that has a <code className="font-mono">specFile:</code> set to start tracking
        </p>
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="grid grid-cols-1 gap-4" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))' }}>
        {entries.map(entry => {
          const pct = Math.round(entry.percent)
          const pctColor = pct >= 80 ? 'text-emerald-400' : pct >= 40 ? 'text-amber-400' : 'text-red-400'
          return (
            <div
              key={entry.specFile}
              className="rounded-lg border border-[var(--color-border)] bg-[var(--color-card)] overflow-hidden"
            >
              {/* Card header */}
              <div className="px-4 pt-4 pb-3 border-b border-[var(--color-border)]">
                <div className="flex items-start justify-between gap-2 mb-2">
                  <div className="min-w-0">
                    <p
                      className="text-sm font-semibold text-[var(--color-foreground)] truncate"
                      title={entry.specFile}
                    >
                      {specLabel(entry.specFile)}
                    </p>
                    {entry.specFile !== specLabel(entry.specFile) && (
                      <p className="text-[11px] text-[var(--color-muted-foreground)] truncate mt-0.5" title={entry.specFile}>
                        {entry.specFile}
                      </p>
                    )}
                  </div>
                  <span className={cn('text-2xl font-bold tabular-nums shrink-0', pctColor)}>
                    {pct}%
                  </span>
                </div>
                <ProgressBar pct={pct} />
              </div>

              {/* Stats row */}
              <div className="px-4 py-2.5 flex flex-wrap gap-x-4 gap-y-1 text-xs border-b border-[var(--color-border)]">
                <span className={cn('font-semibold tabular-nums', pctColor)}>
                  {entry.covered}/{entry.total} operations
                </span>
                <span className="text-[var(--color-muted-foreground)]">{entry.totalRuns} run{entry.totalRuns !== 1 ? 's' : ''}</span>
                <span className="text-emerald-400">{entry.passedRuns} passed</span>
                {entry.failedRuns > 0 && <span className="text-red-400">{entry.failedRuns} failed</span>}
                {entry.lastRunAt && (
                  <span className="text-[var(--color-muted-foreground)]">
                    Last: {new Date(entry.lastRunAt).toLocaleDateString()}
                  </span>
                )}
              </div>

              {/* Quick uncovered summary */}
              {(entry.missed ?? []).length > 0 && (
                <div className="px-4 py-2 flex items-center gap-1.5 text-xs text-amber-400 border-b border-[var(--color-border)]">
                  <AlertCircle size={11} className="shrink-0" />
                  <span>{entry.missed!.length} operation{entry.missed!.length !== 1 ? 's' : ''} not yet covered</span>
                </div>
              )}

              {/* Actions */}
              <div className="px-4 py-2.5 flex items-center justify-between">
                <button
                  type="button"
                  onClick={() => onOpenSpec(entry.specFile)}
                  className="text-xs font-medium text-[var(--color-accent)] hover:underline"
                >
                  View details →
                </button>
                <button
                  type="button"
                  onClick={() => onReset(entry.specFile)}
                  className="flex items-center gap-1 text-[11px] text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors"
                >
                  <RotateCcw size={10} /> Reset
                </button>
              </div>
            </div>
          )
        })}
      </div>

      {loading && (
        <div className="flex justify-center py-8">
          <RefreshCw size={16} className="animate-spin text-[var(--color-muted-foreground)]" />
        </div>
      )}
    </div>
  )
}

// ── Per-spec detail ───────────────────────────────────────────────────────────

function SpecDetailPanel({
  entry,
  onReset,
}: {
  entry: AccumulatedCoverage
  onReset: () => void
}) {
  const pct = Math.round(entry.percent)
  const pctColor = pct >= 80 ? 'text-emerald-400' : pct >= 40 ? 'text-amber-400' : 'text-red-400'

  const covered = [...(entry.exercised ?? [])].sort((a, b) => a.localeCompare(b))
  const missed  = [...(entry.missed   ?? [])].sort((a, b) => a.localeCompare(b))

  return (
    <div className="flex-1 overflow-y-auto">
      {/* Stats header */}
      <div className="px-6 py-5 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-4 mb-3">
          <span className={cn('text-4xl font-bold tabular-nums', pctColor)}>{pct}%</span>
          <div className="flex-1 min-w-0">
            <ProgressBar pct={pct} />
            <div className="flex flex-wrap gap-x-4 gap-y-1 mt-2 text-xs">
              <span className={cn('font-semibold', pctColor)}>{entry.covered}/{entry.total} operations covered</span>
              <span className="text-[var(--color-muted-foreground)]">{entry.totalRuns} run{entry.totalRuns !== 1 ? 's' : ''}</span>
              <span className="text-emerald-400">{entry.passedRuns} passed</span>
              {entry.failedRuns > 0 && <span className="text-red-400">{entry.failedRuns} failed</span>}
              {entry.lastRunAt && (
                <span className="text-[var(--color-muted-foreground)]">
                  Last run: {new Date(entry.lastRunAt).toLocaleString()}
                </span>
              )}
            </div>
          </div>
          <button
            type="button"
            onClick={onReset}
            className="flex items-center gap-1.5 text-xs text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors shrink-0"
          >
            <RotateCcw size={12} /> Reset data
          </button>
        </div>
      </div>

      <div className="divide-y divide-[var(--color-border)]">
        {/* Covered */}
        {covered.length > 0 && (
          <section>
            <div className="px-6 py-3 bg-emerald-500/5 border-b border-[var(--color-border)] flex items-center gap-2">
              <span className="text-xs font-semibold uppercase tracking-widest text-emerald-400">
                Covered
              </span>
              <span className="text-xs text-emerald-400/60 tabular-nums">{covered.length}</span>
            </div>
            {covered.map(op => <OpRow key={op} op={op} covered={true} />)}
          </section>
        )}

        {/* Not covered */}
        {missed.length > 0 && (
          <section>
            <div className="px-6 py-3 bg-amber-500/5 border-b border-[var(--color-border)] flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <AlertCircle size={12} className="text-amber-400 shrink-0" />
                <span className="text-xs font-semibold uppercase tracking-widest text-amber-400">
                  Not covered
                </span>
                <span className="text-xs text-amber-400/60 tabular-nums">{missed.length}</span>
              </div>
              <span className="text-[11px] text-[var(--color-muted-foreground)] italic">
                Add these operations to your scenarios
              </span>
            </div>
            {missed.map(op => <OpRow key={op} op={op} covered={false} />)}
          </section>
        )}

        {covered.length === 0 && missed.length === 0 && (
          <div className="px-6 py-10 text-center text-sm text-[var(--color-muted-foreground)]">
            Run a scenario with this spec to populate operation details.
          </div>
        )}
      </div>
    </div>
  )
}

// ── Root component ────────────────────────────────────────────────────────────

export function CoverageTab({ workspacePath, specFile, onOpenSpecCoverage }: Props) {
  const [entries, setEntries] = useState<AccumulatedCoverage[]>([])
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    if (!workspacePath) { setEntries([]); return }
    setLoading(true)
    try {
      const data = await listAccumulatedCoverage()
      setEntries(data)
    } finally {
      setLoading(false)
    }
  }, [workspacePath])

  useEffect(() => { refresh() }, [refresh])

  async function handleReset(sf: string) {
    await resetAccumulatedCoverage(sf).catch(() => {})
    setEntries(prev => prev.filter(e => e.specFile !== sf))
  }

  if (!workspacePath) {
    return (
      <div className="flex flex-col items-center justify-center flex-1 gap-3 h-full">
        <BarChart2 size={36} className="text-[var(--color-muted-foreground)] opacity-20" />
        <p className="text-sm text-[var(--color-muted-foreground)]">No workspace open</p>
      </div>
    )
  }

  const currentEntry = specFile ? entries.find(e => e.specFile === specFile) : null

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="shrink-0 flex items-center gap-3 px-4 py-2.5 border-b border-[var(--color-border)] bg-[var(--color-background)]">
        {specFile ? (
          <>
            <BarChart2 size={15} className="text-emerald-400 shrink-0" />
            <span className="text-sm font-semibold text-[var(--color-foreground)] truncate flex-1" title={specFile}>
              {specLabel(specFile)}
            </span>
            {specFile !== specLabel(specFile) && (
              <span className="text-xs text-[var(--color-muted-foreground)] truncate max-w-xs hidden sm:block" title={specFile}>
                {specFile}
              </span>
            )}
          </>
        ) : (
          <>
            <BarChart2 size={15} className="text-emerald-400 shrink-0" />
            <span className="text-sm font-semibold text-[var(--color-foreground)]">Spec Coverage</span>
            <span className="text-xs text-[var(--color-muted-foreground)]">
              {entries.length} spec{entries.length !== 1 ? 's' : ''} tracked
            </span>
          </>
        )}
        <button
          type="button"
          onClick={refresh}
          title="Refresh"
          className="ml-auto p-1.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
        >
          <RefreshCw size={13} className={loading ? 'animate-spin' : ''} />
        </button>
      </div>

      {/* Body */}
      {specFile ? (
        currentEntry ? (
          <SpecDetailPanel entry={currentEntry} onReset={() => handleReset(specFile)} />
        ) : loading ? (
          <div className="flex flex-col items-center justify-center flex-1 gap-3">
            <RefreshCw size={20} className="animate-spin text-[var(--color-muted-foreground)]" />
            <p className="text-sm text-[var(--color-muted-foreground)]">Loading…</p>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center flex-1 gap-3">
            <BarChart2 size={36} className="text-[var(--color-muted-foreground)] opacity-20" />
            <p className="text-sm text-[var(--color-muted-foreground)]">No data for this spec yet</p>
            <p className="text-xs text-[var(--color-muted-foreground)] opacity-60">
              Run a scenario with <code className="font-mono">specFile: {specFile}</code>
            </p>
          </div>
        )
      ) : (
        <OverviewPanel
          entries={entries}
          loading={loading}
          onOpenSpec={onOpenSpecCoverage}
          onReset={handleReset}
        />
      )}
    </div>
  )
}
