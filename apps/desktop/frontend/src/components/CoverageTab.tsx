import { useState, useEffect, useCallback, useRef } from 'react'
import { RotateCcw, RefreshCw, BarChart2, AlertCircle, ChevronDown, ChevronRight, Download, ChevronDown as ChevronDownSm } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AccumulatedCoverage } from '@/lib/bridge'
import { getAccumulatedCoverage, resetAccumulatedCoverage, saveContentToFile, openInSystemViewer } from '@/lib/bridge'
import { QualityPanel } from './QualityPanel'
import { parseOp, specLabel, buildCsvRows, generateMarkdownReport } from '@/lib/coverage-report'

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

// ── Report generators ─────────────────────────────────────────────────────────

function generateMarkdown(specFile: string, entry: AccumulatedCoverage): string {
  return generateMarkdownReport(specFile, entry)
}

const METHOD_HEX: Record<string, string> = {
  GET: '#0ea5e9', POST: '#10b981', PUT: '#f59e0b',
  PATCH: '#f97316', DELETE: '#ef4444', HEAD: '#a855f7', OPTIONS: '#ec4899',
}

function generateHTMLReport(specFile: string, entry: AccumulatedCoverage): string {
  const label = specLabel(specFile)
  const pct = Math.round(entry.percent)
  const barColor = pct >= 80 ? '#10b981' : pct >= 40 ? '#f59e0b' : '#ef4444'
  const now = new Date().toLocaleString()

  function methodBadge(method: string) {
    const color = METHOD_HEX[method] ?? '#6b7280'
    return `<span style="font-family:monospace;font-weight:700;font-size:11px;color:${color};background:${color}18;padding:1px 5px;border-radius:3px">${method}</span>`
  }

  const allOps = [...(entry.exercised ?? []), ...(entry.missed ?? [])].sort((a, b) => {
    const pa = parseOp(a), pb = parseOp(b)
    return pa.path !== pb.path ? pa.path.localeCompare(pb.path) : pa.method.localeCompare(pb.method)
  })

  const opRows = allOps.map(op => {
    const { method, path } = parseOp(op)
    const covered = (entry.exercised ?? []).includes(op)
    const r = entry.operationResults?.[op]
    const result = r
      ? r.lastPassed
        ? `<span style="color:#059669;font-weight:600">✓ Pass</span>`
        : `<span style="color:#dc2626;font-weight:600">✗ Fail</span>`
      : `<span style="color:#9ca3af">—</span>`
    const runAt = r?.lastRunAt ? relTime(r.lastRunAt) : '—'
    return `<tr>
      <td>${methodBadge(method)}</td>
      <td style="font-family:monospace;font-size:12px">${path}</td>
      <td>${covered ? '<span style="color:#059669;font-weight:600">✓ Covered</span>' : '<span style="color:#9ca3af">—</span>'}</td>
      <td>${result}</td>
      <td style="color:#6b7280;font-size:12px">${runAt}</td>
    </tr>`
  }).join('\n')

  const q = entry.quality
  let qualityHTML = ''
  if (q) {
    const gaps: string[] = []

    if (q.happyPathOnly.length > 0) {
      gaps.push(`<h3 style="margin:1.5rem 0 0.4rem;font-size:0.95rem;color:#b45309">Happy-path only (${q.happyPathOnly.length})</h3>
<p style="margin:0 0 0.5rem;font-size:0.8rem;color:#6b7280">Only 2xx responses observed — declared error paths untested.</p>
<ul style="margin:0;padding-left:1.5rem">${q.happyPathOnly.map(op => `<li style="font-family:monospace;font-size:12px;padding:1px 0">${op}</li>`).join('')}</ul>`)
    }

    if (q.errorsNeverTriggered.length > 0) {
      const rows = q.errorsNeverTriggered.map(gap =>
        `<tr><td style="font-family:monospace;font-size:12px">${gap.operation}</td><td>${gap.missing.map(c => `<span style="background:#fff3cd;padding:1px 4px;border-radius:3px;font-size:11px;font-weight:600">${c}</span>`).join(' ')}</td></tr>`
      ).join('')
      gaps.push(`<h3 style="margin:1.5rem 0 0.4rem;font-size:0.95rem;color:#b45309">Error responses never triggered (${q.errorsNeverTriggered.length})</h3>
<table style="width:100%;border-collapse:collapse"><thead><tr><th style="text-align:left;padding:4px 8px;border-bottom:1px solid #e5e7eb;font-size:11px;text-transform:uppercase;letter-spacing:.04em;color:#6b7280">Operation</th><th style="text-align:left;padding:4px 8px;border-bottom:1px solid #e5e7eb;font-size:11px;text-transform:uppercase;letter-spacing:.04em;color:#6b7280">Missing codes</th></tr></thead><tbody>${rows}</tbody></table>`)
    }

    if (q.securedNeverAuthed.length > 0) {
      gaps.push(`<h3 style="margin:1.5rem 0 0.4rem;font-size:0.95rem;color:#1d4ed8">Secured — never tested with auth (${q.securedNeverAuthed.length})</h3>
<p style="margin:0 0 0.5rem;font-size:0.8rem;color:#6b7280">Operations requiring credentials but always called without.</p>
<ul style="margin:0;padding-left:1.5rem">${q.securedNeverAuthed.map(op => `<li style="font-family:monospace;font-size:12px;padding:1px 0">${op}</li>`).join('')}</ul>`)
    }

    qualityHTML = `<h2 style="font-size:1.1rem;border-bottom:1px solid #e5e7eb;padding-bottom:0.3rem;margin-top:2rem">Quality Gaps</h2>
${gaps.length > 0 ? gaps.join('\n') : '<p style="color:#059669;font-size:0.9rem">All quality signals clear.</p>'}`
  }

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Contracts Report — ${label}</title>
<style>
*{box-sizing:border-box}
body{font-family:system-ui,-apple-system,sans-serif;max-width:960px;margin:0 auto;padding:2rem;color:#111;font-size:14px;line-height:1.5}
h1{font-size:1.4rem;margin:0 0 0.25rem}
h2{font-size:1.1rem;border-bottom:1px solid #e5e7eb;padding-bottom:0.3rem;margin-top:2rem}
table{width:100%;border-collapse:collapse;margin-top:0.5rem}
th{text-align:left;padding:5px 8px;border-bottom:2px solid #e5e7eb;font-size:11px;text-transform:uppercase;letter-spacing:.05em;color:#6b7280;white-space:nowrap}
td{padding:5px 8px;border-bottom:1px solid #f3f4f6;vertical-align:middle}
tr:last-child td{border-bottom:none}
.stat{display:inline-block;margin-right:1.5rem;font-size:0.85rem;color:#6b7280}
.stat strong{color:#111}
@media print{body{padding:0}@page{margin:1.5cm}}
</style>
</head>
<body>
<h1>Contracts Report — ${label}</h1>
<p style="margin:0.25rem 0 1rem;font-size:0.85rem;color:#6b7280">Generated ${now}</p>

<div style="margin-bottom:1rem">
  <span style="font-size:2rem;font-weight:700;color:${barColor}">${pct}%</span>
  <span style="margin-left:0.75rem;font-size:0.95rem;color:#374151">${entry.covered} / ${entry.total} operations covered</span>
  <div style="margin-top:0.5rem;height:8px;background:#e5e7eb;border-radius:999px;overflow:hidden;max-width:400px">
    <div style="height:100%;background:${barColor};border-radius:999px;width:${Math.min(pct, 100)}%"></div>
  </div>
</div>
<div>
  <span class="stat"><strong>${entry.totalRuns}</strong> run${entry.totalRuns !== 1 ? 's' : ''}</span>
  <span class="stat"><strong style="color:#059669">${entry.passedRuns}</strong> passed</span>
  ${entry.failedRuns > 0 ? `<span class="stat"><strong style="color:#dc2626">${entry.failedRuns}</strong> failed</span>` : ''}
  ${entry.lastRunAt ? `<span class="stat">Last run <strong>${new Date(entry.lastRunAt).toLocaleString()}</strong></span>` : ''}
</div>

<h2>Operations</h2>
<table>
  <thead><tr>
    <th>Method</th><th>Path</th><th>Coverage</th><th>Last Result</th><th>Last Run</th>
  </tr></thead>
  <tbody>${opRows}</tbody>
</table>

${qualityHTML}
</body>
</html>`
}

// ── Export menu ───────────────────────────────────────────────────────────────

function ExportMenu({ specFile, entry }: { specFile: string; entry: AccumulatedCoverage }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

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
    exportCSV(specFile, entry)
  }

  async function handleMarkdown() {
    setOpen(false)
    const md = generateMarkdown(specFile, entry)
    const name = `${specLabel(specFile).replace(/\.(yaml|yml|json)$/i, '')}-contracts.md`
    await saveContentToFile(md, name)
  }

  async function handleHTML() {
    setOpen(false)
    const html = generateHTMLReport(specFile, entry)
    const err = await openInSystemViewer(html, 'html')
    if (err) console.error('open report:', err)
  }

  return (
    <div ref={ref} className="relative">
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

function exportCSV(specFile: string, entry: AccumulatedCoverage) {
  const rows = buildCsvRows(entry)
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
              {specEntry && <ExportMenu specFile={specFile} entry={specEntry} />}
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
