import { useEffect, useState } from 'react'
import { ShieldCheck, Terminal, Play, Square, AlertTriangle, Download } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  getConformContext, inspectConformSpec, runConformance, cancelConformance,
  exportConformReport, saveContentToFile, openInSystemViewer,
} from '@/lib/bridge'
import type { ConformContext, ConformSpecInfo, ConformRunOutput, ConformOperation, ConformReportFormat } from '@/lib/bridge'

const inputClass =
  'w-full rounded border border-[var(--color-border)] bg-[var(--color-background)] px-2 py-1 text-xs text-[var(--color-foreground)] outline-none focus:border-[var(--color-accent)]'

// Display only: the Go service makes the real decision and refuses an unconfirmed remote URL.
export function isLocalUrl(raw: string): boolean {
  try {
    const host = new URL(raw).hostname.toLowerCase()
    return host === 'localhost' || host === '127.0.0.1' || host === '[::1]' || host === '::1'
  } catch {
    return false
  }
}

const STATUS_STYLE: Record<ConformOperation['status'], string> = {
  passed: 'text-emerald-400',
  failed: 'text-red-400',
  error: 'text-red-400',
  skipped: 'text-[var(--color-muted-foreground)]',
}

function Tile({ label, value, total, tone }: { label: string; value: number; total: number; tone: 'neutral' | 'good' | 'bad' | 'warn' }) {
  const active = value > 0
  const color = tone === 'neutral' || !active
    ? 'text-[var(--color-foreground)]'
    : tone === 'good' ? 'text-emerald-400' : tone === 'bad' ? 'text-red-400' : 'text-amber-400'
  const share = total > 0 ? Math.round((value / total) * 100) : 0
  return (
    <div className="flex-1 min-w-[7rem] rounded border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2">
      <div className="text-[11px] text-[var(--color-muted-foreground)]">{label}</div>
      <div className={cn('text-xl font-semibold tabular-nums', color)}>{value}</div>
      <div className="text-[11px] text-[var(--color-muted-foreground)]">{share}% of {total}</div>
    </div>
  )
}

export function splitList(text: string): string[] {
  return text.split(',').map(t => t.trim()).filter(Boolean)
}

const EXPORTS: { format: ConformReportFormat; label: string; file: string }[] = [
  { format: 'html', label: 'HTML report', file: '' },
  { format: 'junit', label: 'JUnit XML', file: 'conformance.xml' },
  { format: 'json', label: 'JSON', file: 'conformance.json' },
  { format: 'markdown', label: 'Markdown', file: 'conformance.md' },
]

function ExportMenu() {
  const [open, setOpen] = useState(false)
  const [error, setError] = useState('')

  async function run(item: (typeof EXPORTS)[number]) {
    setOpen(false)
    setError('')
    try {
      const res = await exportConformReport(item.format)
      if (res.error) { setError(res.error); return }
      if (item.format === 'html') {
        const err = await openInSystemViewer(res.content, 'html')
        if (err) setError(err)
      } else {
        await saveContentToFile(res.content, item.file)
      }
    } catch (e) {
      setError(String(e))
    }
  }

  return (
    <div className="relative flex items-center gap-2">
      {error && <span role="alert" className="text-[11px] text-red-400 truncate max-w-[16rem]" title={error}>{error}</span>}
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        className="flex items-center gap-1.5 rounded border border-[var(--color-border)] px-2 py-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]"
      >
        <Download size={12} aria-hidden /> Export report
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-50 min-w-[9rem] rounded border border-[var(--color-border)] bg-[var(--color-card)] shadow-lg py-1 text-xs">
          {EXPORTS.map(item => (
            <button key={item.format} type="button" onClick={() => run(item)} className="w-full text-left px-3 py-1.5 hover:bg-[var(--color-muted)]">
              {item.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

type Props = { onOpenCliBuilder?: () => void }

export function ConformanceTab({ onOpenCliBuilder }: Props) {
  const [ctx, setCtx] = useState<ConformContext | null>(null)
  const [loadError, setLoadError] = useState('')
  const [specPath, setSpecPath] = useState('')
  const [info, setInfo] = useState<ConformSpecInfo | null>(null)
  const [baseUrl, setBaseUrl] = useState('')
  const [envPath, setEnvPath] = useState('')
  const [allowWrites, setAllowWrites] = useState(false)
  const [confirmedRemote, setConfirmedRemote] = useState(false)
  const [tagsText, setTagsText] = useState('')
  const [pathsText, setPathsText] = useState('')
  const [timeoutText, setTimeoutText] = useState('')
  const [negative, setNegative] = useState(false)
  const [failFast, setFailFast] = useState(false)
  const [recordCoverage, setRecordCoverage] = useState(false)
  const [running, setRunning] = useState(false)
  const [output, setOutput] = useState<ConformRunOutput | null>(null)

  useEffect(() => {
    getConformContext().then(setCtx).catch(e => setLoadError(String(e)))
  }, [])

  useEffect(() => {
    setOutput(null)
    setConfirmedRemote(false)
    if (!specPath) { setInfo(null); setBaseUrl(''); return }
    let current = true
    inspectConformSpec(specPath).then(i => {
      if (!current) return
      setInfo(i)
      setBaseUrl(i.serverUrl)
    }).catch(e => current && setInfo({ title: '', version: '', serverUrl: '', reads: 0, writes: 0, error: String(e) }))
    return () => { current = false }
  }, [specPath])

  if (loadError) return <div className="p-6 text-sm text-red-400" role="alert">Could not load the page: {loadError}</div>
  if (!ctx) return <div className="p-6 text-sm text-[var(--color-muted-foreground)]">Loading…</div>

  const remote = baseUrl !== '' && !isLocalUrl(baseUrl)
  const writes = info?.writes ?? 0
  const reads = info?.reads ?? 0
  const filtered = splitList(tagsText).length > 0 || splitList(pathsText).length > 0
  const timeoutMs = Number.parseInt(timeoutText, 10)
  const timeoutBad = timeoutText !== '' && !(timeoutMs > 0)
  const canRun = !timeoutBad && !!specPath && !info?.error && baseUrl.trim() !== '' && (!remote || confirmedRemote) && !running

  async function run() {
    setRunning(true)
    setOutput(null)
    try {
      setOutput(await runConformance({
        specPath, baseUrl, envPath, allowWrites, confirmedRemote,
        tags: splitList(tagsText), paths: splitList(pathsText), negative, failFast,
        timeoutMs: timeoutBad || !timeoutText ? 0 : timeoutMs,
        recordCoverage: recordCoverage && !filtered,
      }))
    } catch (e) {
      setOutput({ error: String(e), specTitle: '', baseUrl, writes: allowWrites, durationMs: 0, operations: [], total: 0, covered: 0, notCovered: 0, passing: 0, failing: 0, probes: [], warnings: [] })
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="flex flex-col h-full w-full overflow-hidden bg-[var(--color-background)]">
      <div className="shrink-0 px-5 py-3 border-b border-[var(--color-border)] bg-[var(--color-card)] flex items-start gap-3">
        <div className="flex-1 min-w-0">
          <h1 className="flex items-center gap-2 text-sm font-semibold text-[var(--color-foreground)]">
            <ShieldCheck size={15} className="text-emerald-400" aria-hidden />
            Check Conformance
          </h1>
          <p className="mt-0.5 text-xs text-[var(--color-muted-foreground)]">
            Spec Conformance Testing: call every operation in an OpenAPI spec and check that each response matches what the spec declares.
          </p>
        </div>
        {onOpenCliBuilder && (
          <button
            type="button"
            onClick={onOpenCliBuilder}
            className="shrink-0 flex items-center gap-1.5 rounded border border-[var(--color-border)] px-2 py-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]"
          >
            <Terminal size={12} aria-hidden /> Get CLI command
          </button>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-5 flex flex-col gap-5 max-w-4xl">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="flex flex-col gap-1">
            <label htmlFor="conf-spec" className="text-xs font-medium">Spec <span className="text-red-400">*</span></label>
            <select id="conf-spec" value={specPath} onChange={e => setSpecPath(e.target.value)} className={inputClass}>
              <option value="">Choose a spec…</option>
              {ctx.specs.map(f => <option key={f.path} value={f.path}>{f.label}</option>)}
            </select>
            {ctx.specs.length === 0 && (
              <p className="text-[11px] text-[var(--color-muted-foreground)]">No OpenAPI spec was found in this workspace. Import one from the Requests panel.</p>
            )}
            {info?.error && <p role="alert" className="text-[11px] text-red-400">{info.error}</p>}
          </div>

          <div className="flex flex-col gap-1">
            <label htmlFor="conf-env" className="text-xs font-medium">Environment</label>
            <select id="conf-env" value={envPath} onChange={e => setEnvPath(e.target.value)} className={inputClass}>
              <option value="">Active environment</option>
              {ctx.environments.map(f => <option key={f.path} value={f.path}>{f.label}</option>)}
            </select>
          </div>

          <div className="flex flex-col gap-1 sm:col-span-2">
            <label htmlFor="conf-base" className="text-xs font-medium">Base URL <span className="text-red-400">*</span></label>
            <input id="conf-base" value={baseUrl} onChange={e => { setBaseUrl(e.target.value); setConfirmedRemote(false) }} placeholder="http://localhost:8120" className={cn(inputClass, 'font-mono')} />
            <p className="text-[11px] text-[var(--color-muted-foreground)]">
              {info?.serverUrl ? 'Starts as the first server in the spec; change it to test another deployment.' : 'The spec declares no server, so enter the address to test.'}
            </p>
          </div>
        </div>

        {remote && (
          <label className="flex items-start gap-2 rounded border border-amber-500/40 bg-amber-500/10 p-3 text-xs cursor-pointer">
            <input type="checkbox" checked={confirmedRemote} onChange={e => setConfirmedRemote(e.target.checked)} className="mt-0.5" />
            <span>
              <span className="font-semibold text-amber-400"><AlertTriangle size={12} className="inline -mt-0.5 mr-1" aria-hidden />This address is not on your computer.</span>{' '}
              Requests will go to <span className="font-mono">{baseUrl}</span>. I confirm I may test it.
            </span>
          </label>
        )}

        <div className="flex flex-col gap-1">
          <label className="flex items-center gap-2 text-xs cursor-pointer">
            <input type="checkbox" checked={allowWrites} onChange={e => setAllowWrites(e.target.checked)} />
            <span className="font-medium">Also send write requests (POST, PUT, PATCH, DELETE)</span>
          </label>
          <p data-testid="conform-send-note" className={cn('text-[11px]', allowWrites && writes > 0 ? 'text-amber-400' : 'text-[var(--color-muted-foreground)]')}>
            {!specPath
              ? 'By default only read requests (GET, HEAD) are sent.'
              : allowWrites
                ? `Will send ${reads} read and ${writes} write request${writes === 1 ? '' : 's'}. Writes can create, change or delete data.`
                : `Will send ${reads} read request${reads === 1 ? '' : 's'}. ${writes} write operation${writes === 1 ? ' is' : 's are'} skipped.`}
          </p>
        </div>

        <details className="rounded border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2">
          <summary className="cursor-pointer text-xs font-medium">Options</summary>
          <div className="mt-3 grid gap-3 sm:grid-cols-3">
            <div className="flex flex-col gap-1">
              <label htmlFor="conf-tags" className="text-xs font-medium">Tags</label>
              <input id="conf-tags" value={tagsText} onChange={e => setTagsText(e.target.value)} placeholder="users, orders" className={inputClass} />
              <p className="text-[11px] text-[var(--color-muted-foreground)]">Only operations with one of these OpenAPI tags.</p>
            </div>
            <div className="flex flex-col gap-1">
              <label htmlFor="conf-paths" className="text-xs font-medium">Path prefixes</label>
              <input id="conf-paths" value={pathsText} onChange={e => setPathsText(e.target.value)} placeholder="/users, /orders" className={cn(inputClass, 'font-mono')} />
              <p className="text-[11px] text-[var(--color-muted-foreground)]">Only paths that start with one of these.</p>
            </div>
            <div className="flex flex-col gap-1">
              <label htmlFor="conf-timeout" className="text-xs font-medium">Request timeout (ms)</label>
              <input id="conf-timeout" value={timeoutText} onChange={e => setTimeoutText(e.target.value)} placeholder="10000" inputMode="numeric" className={cn(inputClass, 'font-mono')} />
              {timeoutBad && <p role="alert" className="text-[11px] text-red-400">Enter a number above 0, or leave empty.</p>}
            </div>
          </div>
          <div className="mt-3 flex flex-col gap-2">
            <label className="flex items-start gap-2 text-xs cursor-pointer">
              <input type="checkbox" checked={negative} onChange={e => setNegative(e.target.checked)} className="mt-0.5" />
              <span><span className="font-medium">Also run negative probes</span> — send malformed or unauthorized requests that the server should reject.</span>
            </label>
            <label className="flex items-center gap-2 text-xs cursor-pointer">
              <input type="checkbox" checked={failFast} onChange={e => setFailFast(e.target.checked)} />
              <span className="font-medium">Stop after the first failure</span>
            </label>
            <label className={cn('flex items-start gap-2 text-xs', filtered ? 'opacity-50' : 'cursor-pointer')}>
              <input type="checkbox" checked={recordCoverage && !filtered} disabled={filtered} onChange={e => setRecordCoverage(e.target.checked)} className="mt-0.5" />
              <span>
                <span className="font-medium">Add this run to Contracts coverage</span> — operations that got a response are counted.
                {filtered && ' Not available while tags or path prefixes are set.'}
              </span>
            </label>
          </div>
        </details>

        <div className="flex items-center gap-2">
          {!running ? (
            <button
              type="button"
              disabled={!canRun}
              onClick={run}
              className="flex items-center gap-1.5 rounded bg-[var(--color-accent)] px-3 py-1.5 text-xs font-medium text-white disabled:opacity-40"
            >
              <Play size={12} aria-hidden /> Run conformance check
            </button>
          ) : (
            <button
              type="button"
              onClick={() => cancelConformance()}
              className="flex items-center gap-1.5 rounded border border-[var(--color-border)] px-3 py-1.5 text-xs font-medium hover:bg-[var(--color-muted)]"
            >
              <Square size={12} aria-hidden /> Cancel
            </button>
          )}
          {running && <span className="text-xs text-[var(--color-muted-foreground)]">Running…</span>}
        </div>

        {output?.error && <p role="alert" className="text-xs text-red-400">{output.error}</p>}

        {output && output.warnings?.length > 0 && (
          <ul aria-label="Environment warnings" className="rounded border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-400">
            {output.warnings.map((w, i) => <li key={i}>{w}</li>)}
          </ul>
        )}

        {output && !output.error && (
          <section aria-label="Conformance results" className="flex flex-col gap-3">
            <div className="flex items-center justify-end"><ExportMenu /></div>
            <div className="flex flex-wrap gap-2">
              <div className="flex gap-2 flex-1 min-w-[15rem]">
                <Tile label="Covered" value={output.covered} total={output.total} tone="neutral" />
                <Tile label="Not covered" value={output.notCovered} total={output.total} tone="warn" />
              </div>
              <div className="flex gap-2 flex-1 min-w-[15rem]">
                <Tile label="Passing" value={output.passing} total={output.total} tone="good" />
                <Tile label="Failing" value={output.failing} total={output.total} tone="bad" />
              </div>
            </div>
            <p className="text-[11px] text-[var(--color-muted-foreground)]">
              {output.specTitle} at {output.baseUrl} · {output.writes ? 'reads and writes' : 'read requests only'} · {output.durationMs} ms
            </p>

            <table className="w-full text-xs border-collapse">
              <thead>
                <tr className="text-left text-[11px] text-[var(--color-muted-foreground)] border-b border-[var(--color-border)]">
                  <th className="py-1 pr-2 font-medium">Operation</th>
                  <th className="py-1 pr-2 font-medium">Result</th>
                  <th className="py-1 pr-2 font-medium">Status</th>
                  <th className="py-1 font-medium text-right">Time</th>
                </tr>
              </thead>
              <tbody>
                {output.operations.map(op => (
                  <tr key={`${op.method} ${op.path}`} className="border-b border-[var(--color-border)] align-top">
                    <td className="py-1.5 pr-2">
                      <span className="font-mono font-semibold">{op.method}</span>{' '}
                      <span className="font-mono">{op.path}</span>
                      {(op.violations.length > 0 || op.error) && (
                        <ul className="mt-1 list-disc pl-5 font-mono text-[11px] text-red-400">
                          {op.error && <li className="break-all">{op.error}</li>}
                          {op.violations.map((v, i) => <li key={i} className="break-all">{v}</li>)}
                        </ul>
                      )}
                    </td>
                    <td className={cn('py-1.5 pr-2 font-medium', STATUS_STYLE[op.status])}>
                      {op.status === 'skipped' ? 'skipped' : op.status}
                    </td>
                    <td className="py-1.5 pr-2 font-mono">{op.statusCode > 0 ? op.statusCode : '—'}</td>
                    <td className="py-1.5 text-right font-mono">{op.status === 'skipped' ? '—' : `${op.durationMs} ms`}</td>
                  </tr>
                ))}
              </tbody>
            </table>

            {output.coverageNote && (
              <p data-testid="conform-coverage-note" className="text-[11px] text-[var(--color-muted-foreground)]">{output.coverageNote}</p>
            )}

            {output.probes.length > 0 && (
              <div aria-label="Negative probes" className="flex flex-col gap-1.5">
                <h2 className="text-xs font-semibold">Negative probes</h2>
                <p className="text-[11px] text-[var(--color-muted-foreground)]">
                  {output.probes.filter(p => p.status === 'rejected').length} rejected as they should be,{' '}
                  <span className={output.probes.some(p => p.status === 'accepted') ? 'text-red-400' : ''}>
                    {output.probes.filter(p => p.status === 'accepted').length} accepted by the server
                  </span>{' '}
                  (it should have refused them).
                </p>
                <ul className="text-xs">
                  {output.probes.map((p, i) => (
                    <li key={i} className="flex gap-2 border-b border-[var(--color-border)] py-1">
                      <span className={cn('w-16 shrink-0 font-medium', p.status === 'accepted' || p.status === 'error' ? 'text-red-400' : p.status === 'rejected' ? 'text-emerald-400' : 'text-[var(--color-muted-foreground)]')}>{p.status}</span>
                      <span className="font-mono shrink-0">{p.method} {p.path}</span>
                      <span className="text-[var(--color-muted-foreground)] min-w-0">{p.description}{p.statusCode > 0 ? ` (${p.statusCode})` : ''}{p.error ? `: ${p.error}` : ''}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </section>
        )}
      </div>
    </div>
  )
}
