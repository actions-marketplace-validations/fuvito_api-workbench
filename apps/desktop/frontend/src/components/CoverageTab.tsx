import { useState, useEffect, useCallback, useRef } from 'react'
import { RotateCcw, RefreshCw, BarChart2, AlertCircle, ChevronDown, ChevronRight, Download, ChevronDown as ChevronDownSm } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AccumulatedCoverage } from '@/lib/bridge'
import { getAccumulatedCoverage, resetAccumulatedCoverage, saveContentToFile, openInSystemViewer, exportCoverageReport } from '@/lib/bridge'
import { QualityPanel } from './QualityPanel'
import { parseOp, specLabel, operationOutcomes, share } from '@/lib/coverage-report'

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


const TILE_TONE = {
  neutral: { value: 'text-[var(--color-foreground)]', idle: '', edge: 'border-l-[var(--color-muted-foreground)]/40' },
  pass: { value: 'text-emerald-400', idle: 'text-[var(--color-muted-foreground)]', edge: 'border-l-emerald-500' },
  fail: { value: 'text-red-400', idle: 'text-[var(--color-muted-foreground)]', edge: 'border-l-red-500' },
  warn: { value: 'text-amber-400', idle: 'text-[var(--color-muted-foreground)]', edge: 'border-l-amber-500' },
} as const

// One of the equal totals tiles: how many operations, and what share of all of them.
function StatTile({ label, count, total, hint, tone }: {
  label: string
  count: number
  total: number
  hint: string
  tone: keyof typeof TILE_TONE
}) {
  const t = TILE_TONE[tone]
  const active = tone === 'neutral' || count > 0
  return (
    <div
      role="group"
      aria-label={label}
      className={cn('rounded border border-[var(--color-border)] border-l-4 bg-[var(--color-card)] px-4 py-3', active ? t.edge : 'border-l-[var(--color-border)]')}
    >
      <div className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">{label}</div>
      <div className="flex items-baseline gap-2 mt-1">
        <span className={cn('text-3xl font-bold tabular-nums', active ? t.value : t.idle)}>{count}</span>
        <span className="text-sm tabular-nums text-[var(--color-muted-foreground)]">{share(count, total)}%</span>
      </div>
      <div className="text-[11px] text-[var(--color-muted-foreground)] mt-0.5">
        {hint} <span className="opacity-70">· of {total}</span>
      </div>
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
  lastResult?: { lastPassed: boolean; lastRunAt: string; lastDurationMs?: number; slowestDurationMs?: number }
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
      <span
        data-testid="op-times"
        title="Response time of the last run, and the slowest ever recorded"
        className="w-28 shrink-0 text-right text-[var(--color-muted-foreground)] tabular-nums"
      >
        {lastResult?.lastDurationMs ? `${lastResult.lastDurationMs} / ${lastResult.slowestDurationMs ?? lastResult.lastDurationMs} ms` : '—'}
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
      <span className="w-28 shrink-0 text-right" title="Response time of the last run / the slowest ever recorded">Last / slowest</span>
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
  const out = operationOutcomes(entry)

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
      {/* Totals: four equal tiles. Covered only says an operation was called; passing and failing say
          what happened, so a fully covered spec can still show failures. */}
      <div className="px-6 py-5 border-b border-[var(--color-border)]">
        {/* Two pairs that each add up: Covered + Not covered = all operations; Passing + Failing = Covered. */}
        <div className="grid grid-cols-2 gap-8">
          <div className="grid grid-cols-2 gap-3">
            <StatTile label="Covered" count={out.covered} total={out.total} hint="called by a scenario" tone="neutral" />
            <StatTile label="Not covered" count={out.notCovered} total={out.total} hint="no scenario calls them" tone="warn" />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <StatTile label="Passing" count={out.passing} total={out.total} hint="covered, last result passed" tone="pass" />
            <StatTile label="Failing" count={out.failing} total={out.total} hint="covered, last result failed" tone="fail" />
          </div>
        </div>
        {out.noResult > 0 && (
          <p className="mt-2 text-[11px] text-[var(--color-muted-foreground)]">
            {out.noResult} covered operation{out.noResult === 1 ? ' has' : 's have'} no recorded result yet; run the scenarios again.
          </p>
        )}
        <div className="flex items-center gap-4 mt-3">
          <div className="flex-1 min-w-0 flex flex-wrap gap-x-4 gap-y-1 text-xs">
            <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">Scenario runs</span>
            <span className="text-[var(--color-muted-foreground)]">{entry.totalRuns} run{entry.totalRuns !== 1 ? 's' : ''}</span>
            <span className="text-emerald-400">{entry.passedRuns} passed</span>
            {entry.failedRuns > 0 && <span className="text-red-400">{entry.failedRuns} failed</span>}
            {entry.lastRunAt && (
              <span className="text-[var(--color-muted-foreground)]">
                Last run: {new Date(entry.lastRunAt).toLocaleString()}
              </span>
            )}
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

// ── Export menu ───────────────────────────────────────────────────────────────

// The reports are built by the Go core (the same renderers the CLI uses); this menu only saves or
// opens what comes back.
function ExportMenu({ specFile }: { specFile: string }) {
  const [open, setOpen] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const ref = useRef<HTMLDivElement>(null)

  async function fetchReport(format: 'csv' | 'markdown' | 'html'): Promise<string | null> {
    setError(null)
    try {
      const res = await exportCoverageReport(specFile, format)
      if (res.error) {
        setError(res.error)
        return null
      }
      return res.content
    } catch (e) {
      setError(String(e))
      return null
    }
  }

  useEffect(() => {
    if (!open) return
    function onDown(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [open])

  async function handleCSV() {
    setOpen(false)
    const csv = await fetchReport('csv')
    if (csv !== null) downloadText(csv, `${baseName(specFile)}-contracts.csv`, 'text/csv')
  }

  async function handleMarkdown() {
    setOpen(false)
    const md = await fetchReport('markdown')
    if (md !== null) await saveContentToFile(md, `${baseName(specFile)}-contracts.md`)
  }

  async function handleHTML() {
    setOpen(false)
    const html = await fetchReport('html')
    if (html === null) return
    const err = await openInSystemViewer(html, 'html')
    if (err) console.error('open report:', err)
  }

  return (
    <div ref={ref} className="relative flex items-center gap-2">
      {error && <span role="alert" className="text-[11px] text-red-400 max-w-[260px] truncate" title={error}>{error}</span>}
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        className="flex items-center gap-1.5 px-2.5 py-1.5 rounded border border-[var(--color-border)] text-xs font-medium text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
      >
        <Download size={12} />
        Export
        <ChevronDownSm size={9} />
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-50 min-w-[148px] rounded border border-[var(--color-border)] bg-[var(--color-card)] shadow-lg py-1 text-xs">
          <button type="button" onClick={handleCSV}
            className="w-full text-left px-3 py-1.5 hover:bg-[var(--color-muted)] text-[var(--color-foreground)] transition-colors">
            Export CSV
          </button>
          <button type="button" onClick={handleMarkdown}
            className="w-full text-left px-3 py-1.5 hover:bg-[var(--color-muted)] text-[var(--color-foreground)] transition-colors">
            Export Markdown
          </button>
          <button type="button" onClick={handleHTML}
            className="w-full text-left px-3 py-1.5 hover:bg-[var(--color-muted)] text-[var(--color-foreground)] transition-colors">
            Export HTML
          </button>
        </div>
      )}
    </div>
  )
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function baseName(specFile: string): string {
  return specLabel(specFile).replace(/\.(yaml|yml|json)$/i, '')
}

function downloadText(text: string, filename: string, type: string) {
  const blob = new Blob([text], { type })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
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

  // eslint-disable-next-line react/set-state-in-effect
  useEffect(() => { refresh() }, [refresh])

  // eslint-disable-next-line react/set-state-in-effect
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
            <div className="flex-1 min-w-0 flex items-baseline gap-1.5">
              <span className="text-sm font-semibold text-[var(--color-foreground)] shrink-0">{specLabel(specFile)}</span>
              {specFile !== specLabel(specFile) && (
                <span className="text-xs text-[var(--color-muted-foreground)] truncate" title={specFile}>
                  ({specFile})
                </span>
              )}
            </div>
            <div className="shrink-0 flex items-center gap-2">
              {specEntry && <ExportMenu specFile={specFile} />}
              <button
                type="button"
                onClick={refresh}
                title="Refresh"
                className="flex items-center gap-1.5 px-2.5 py-1.5 rounded border border-[var(--color-border)] text-xs font-medium text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
              >
                <RefreshCw size={12} className={loading ? 'animate-spin' : ''} />
                Refresh
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
