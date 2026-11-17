import { useState, useEffect, useCallback } from 'react'
import { RotateCcw, RefreshCw, BarChart2, AlertCircle, ChevronDown, ChevronRight, Download } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AccumulatedCoverage } from '@/lib/bridge'
import { getAccumulatedCoverage, resetAccumulatedCoverage } from '@/lib/bridge'
import { QualityPanel } from './QualityPanel'

type Props = {
  workspacePath: string
  specFile?: string | null
  refreshTrigger?: number
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

function relTime(iso: string): string {
  const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (s < 60) return 'just now'
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

function OpRow({ op, covered, lastResult }: {
  op: string
  covered: boolean
  lastResult?: { lastPassed: boolean; lastRunAt: string }
}) {
  const { method, path } = parseOp(op)
  return (
    <div className={cn(
      'flex items-center gap-3 px-4 py-2 border-b border-[var(--color-border)] last:border-0 text-xs',
      !covered && 'opacity-60',
    )}>
      <span className={cn(
        'font-mono font-bold w-16 shrink-0 px-1.5 py-0.5 rounded text-center text-[11px]',
        METHOD_COLOR[method] ?? 'text-[var(--color-foreground)]',
        METHOD_BG[method] ?? '',
      )}>
        {method}
      </span>
      <span className="font-mono text-[var(--color-foreground)] flex-1 min-w-0 truncate" title={path}>{path}</span>
      <span className="w-24 shrink-0 text-right">
        {covered
          ? <span className="text-emerald-400 font-medium">✓ Covered</span>
          : <span className="text-[var(--color-muted-foreground)]">—</span>}
      </span>
      <span className="w-24 shrink-0 text-right">
        {lastResult
          ? lastResult.lastPassed
            ? <span className="text-emerald-400 font-semibold">✓ Pass</span>
            : <span className="text-red-400 font-semibold">✗ Fail</span>
          : <span className="text-[var(--color-muted-foreground)]">—</span>}
      </span>
      <span className="w-24 shrink-0 text-right text-[var(--color-muted-foreground)] tabular-nums">
        {lastResult?.lastRunAt ? relTime(lastResult.lastRunAt) : '—'}
      </span>
    </div>
  )
}

function TableHeader() {
  return (
    <div className="flex items-center gap-3 px-4 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-background)] sticky top-0 z-10 text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
      <span className="w-16 shrink-0">Method</span>
      <span className="flex-1">Path</span>
      <span className="w-24 shrink-0 text-right">Coverage</span>
      <span className="w-24 shrink-0 text-right">Last Result</span>
      <span className="w-24 shrink-0 text-right">Last Run</span>
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
  const [confirmingReset, setConfirmingReset] = useState(false)
  const [coveredOpen, setCoveredOpen] = useState(true)
  const [missedOpen, setMissedOpen] = useState(true)
  const pct = Math.round(entry.percent)
  const pctColor = pct >= 80 ? 'text-emerald-400' : pct >= 40 ? 'text-amber-400' : 'text-red-400'

  function sortOps(ops: string[]) {
    return [...ops].sort((a, b) => {
      const pa = parseOp(a), pb = parseOp(b)
      return pa.path !== pb.path ? pa.path.localeCompare(pb.path) : pa.method.localeCompare(pb.method)
    })
  }
  const covered = sortOps(entry.exercised ?? [])
  const missed  = sortOps(entry.missed   ?? [])

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
          {confirmingReset ? (
            <div className="flex items-center gap-2 shrink-0">
              <span className="text-xs text-[var(--color-muted-foreground)]">Clear all data?</span>
              <button
                type="button"
                onClick={() => { setConfirmingReset(false); onReset() }}
                className="text-xs font-medium text-red-400 hover:text-red-300 transition-colors"
              >
                Yes, clear
              </button>
              <button
                type="button"
                onClick={() => setConfirmingReset(false)}
                className="text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                Cancel
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => setConfirmingReset(true)}
              className="flex items-center gap-1.5 text-xs text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors shrink-0"
            >
              <RotateCcw size={12} /> Reset data
            </button>
          )}
        </div>
      </div>

      {(covered.length > 0 || missed.length > 0) && <TableHeader />}

      <div className="divide-y divide-[var(--color-border)]">
        {/* Covered */}
        {covered.length > 0 && (
          <section>
            <button
              type="button"
              onClick={() => setCoveredOpen(o => !o)}
              className="w-full px-4 py-2 bg-emerald-500/5 border-b border-[var(--color-border)] flex items-center gap-2 hover:bg-emerald-500/10 transition-colors"
            >
              {coveredOpen ? <ChevronDown size={11} className="text-emerald-400 shrink-0" /> : <ChevronRight size={11} className="text-emerald-400 shrink-0" />}
              <span className="text-[10px] font-semibold uppercase tracking-widest text-emerald-400">Covered</span>
              <span className="text-[10px] text-emerald-400/60 tabular-nums">{covered.length}</span>
            </button>
            {coveredOpen && covered.map(op => (
              <OpRow key={op} op={op} covered={true} lastResult={entry.operationResults?.[op]} />
            ))}
          </section>
        )}

        {/* Not covered */}
        {missed.length > 0 && (
          <section>
            <button
              type="button"
              onClick={() => setMissedOpen(o => !o)}
              className="w-full px-4 py-2 bg-amber-500/5 border-b border-[var(--color-border)] flex items-center gap-2 hover:bg-amber-500/10 transition-colors"
            >
              {missedOpen ? <ChevronDown size={11} className="text-amber-400 shrink-0" /> : <ChevronRight size={11} className="text-amber-400 shrink-0" />}
              <AlertCircle size={11} className="text-amber-400 shrink-0" />
              <span className="text-[10px] font-semibold uppercase tracking-widest text-amber-400">Not covered</span>
              <span className="text-[10px] text-amber-400/60 tabular-nums">{missed.length}</span>
              {missedOpen && (
                <span className="ml-auto text-[10px] text-[var(--color-muted-foreground)] italic">
                  Add these to your scenarios
                </span>
              )}
            </button>
            {missedOpen && missed.map(op => <OpRow key={op} op={op} covered={false} />)}
          </section>
        )}

        {covered.length === 0 && missed.length === 0 && (
          <div className="px-6 py-10 text-center text-sm text-[var(--color-muted-foreground)]">
            Run a scenario with this spec to populate operation details.
          </div>
        )}
      </div>

      {entry.quality && <QualityPanel quality={entry.quality} />}
    </div>
  )
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function csvCell(v: string): string {
  if (v.includes(',') || v.includes('"') || v.includes('\n')) {
    return '"' + v.replace(/"/g, '""') + '"'
  }
  return v
}

function exportCSV(specFile: string, entry: AccumulatedCoverage) {
  const rows = ['Method,Path,Coverage,Last Result,Last Run']
  for (const op of [...(entry.exercised ?? [])].sort()) {
    const { method, path } = parseOp(op)
    const r = entry.operationResults?.[op]
    const result = r ? (r.lastPassed ? 'Pass' : 'Fail') : ''
    const runAt = r?.lastRunAt ? new Date(r.lastRunAt).toISOString() : ''
    rows.push([method, path, 'Covered', result, runAt].map(csvCell).join(','))
  }
  for (const op of [...(entry.missed ?? [])].sort()) {
    const { method, path } = parseOp(op)
    rows.push([method, path, 'Not Covered', '', ''].map(csvCell).join(','))
  }
  const blob = new Blob([rows.join('\n')], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${specLabel(specFile).replace(/\.(yaml|yml|json)$/i, '')}-contracts.csv`
  a.click()
  URL.revokeObjectURL(url)
}

// ── Root component ────────────────────────────────────────────────────────────

export function CoverageTab({ workspacePath, specFile, refreshTrigger }: Props) {
  const [specEntry, setSpecEntry] = useState<AccumulatedCoverage | null>(null)
  const [loading, setLoading] = useState(false)

  const [loadError, setLoadError] = useState<string | null>(null)
  const refresh = useCallback(async () => {
    if (!workspacePath || !specFile) { setSpecEntry(null); return }
    setLoading(true)
    setLoadError(null)
    try {
      const data = await getAccumulatedCoverage(specFile)
      setSpecEntry(data)
    } catch (e) {
      setLoadError(String(e))
    } finally {
      setLoading(false)
    }
  }, [workspacePath, specFile])

  useEffect(() => { refresh() }, [refresh])

  useEffect(() => { if (refreshTrigger) refresh() }, [refreshTrigger, refresh])

  async function handleReset() {
    if (!specFile) return
    await resetAccumulatedCoverage(specFile).catch(() => {})
    setSpecEntry(null)
  }

  if (!workspacePath) {
    return (
      <div className="flex flex-col items-center justify-center flex-1 gap-3 h-full">
        <BarChart2 size={36} className="text-[var(--color-muted-foreground)] opacity-20" />
        <p className="text-sm text-[var(--color-muted-foreground)]">No workspace open</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="shrink-0 flex items-center gap-3 px-4 py-2.5 border-b border-[var(--color-border)] bg-[var(--color-background)]">
        <BarChart2 size={15} className="text-emerald-400 shrink-0" />
        {specFile ? (
          <>
            <span className="text-sm font-semibold text-[var(--color-foreground)] truncate flex-1" title={specFile}>
              {specLabel(specFile)}
            </span>
            {specFile !== specLabel(specFile) && (
              <span className="text-xs text-[var(--color-muted-foreground)] truncate max-w-xs hidden sm:block" title={specFile}>
                {specFile}
              </span>
            )}
            <div className="ml-auto flex items-center gap-1">
              {specEntry && (
                <button
                  type="button"
                  onClick={() => exportCSV(specFile, specEntry)}
                  title="Export CSV"
                  className="p-1.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
                >
                  <Download size={13} />
                </button>
              )}
              <button
                type="button"
                onClick={refresh}
                title="Refresh"
                className="p-1.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
              >
                <RefreshCw size={13} className={loading ? 'animate-spin' : ''} />
              </button>
            </div>
          </>
        ) : (
          <span className="text-sm font-semibold text-[var(--color-foreground)]">Contracts</span>
        )}
      </div>

      {/* Body */}
      {!specFile ? (
        <div className="flex flex-col items-center justify-center flex-1 gap-3">
          <BarChart2 size={36} className="text-[var(--color-muted-foreground)] opacity-20" />
          <p className="text-sm text-[var(--color-muted-foreground)]">Select a spec from the sidebar</p>
          <p className="text-xs text-[var(--color-muted-foreground)] opacity-60">
            Click a spec in the Contracts panel to view its details
          </p>
        </div>
      ) : specEntry ? (
        <SpecDetailPanel entry={specEntry} onReset={handleReset} />
      ) : loading ? (
        <div className="flex flex-col items-center justify-center flex-1 gap-3">
          <RefreshCw size={20} className="animate-spin text-[var(--color-muted-foreground)]" />
          <p className="text-sm text-[var(--color-muted-foreground)]">Loading…</p>
        </div>
      ) : loadError ? (
        <div className="flex flex-col items-center justify-center flex-1 gap-3">
          <p className="text-sm text-red-400">{loadError}</p>
          <button type="button" onClick={refresh} className="text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">Retry</button>
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center flex-1 gap-3">
          <BarChart2 size={36} className="text-[var(--color-muted-foreground)] opacity-20" />
          <p className="text-sm text-[var(--color-muted-foreground)]">No data for this spec yet</p>
          <p className="text-xs text-[var(--color-muted-foreground)] opacity-60">
            Run a scenario with <code className="font-mono">specFile: {specFile}</code>
          </p>
        </div>
      )}
    </div>
  )
}
