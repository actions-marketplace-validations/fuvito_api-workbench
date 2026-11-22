import { useEffect, useState } from 'react'
import { ShieldCheck, Terminal, Play, Square, AlertTriangle, Download } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  getConformContext, inspectConformSpec, runConformance, cancelConformance,
  exportConformReport, saveContentToFile, openInSystemViewer, onConformanceProgress,
} from '@/lib/bridge'
import { isLocalUrl, parseVars, splitList, cliValues } from '@/lib/conformHelpers'
import { FindingView } from '@/components/FindingView'
import type { ConformSummary } from '@/components/ConformanceSidebarPanel'
import type { ConformProgress, ConformContext, ConformSpecInfo, ConformRunOutput, ConformOperation, ConformReportFormat } from '@/lib/bridge'

const inputClass =
  'w-full rounded border border-[var(--color-border)] bg-[var(--color-background)] px-2 py-1 text-xs text-[var(--color-foreground)] outline-none focus:border-[var(--color-accent)]'

function Tile({ label, value, total, tone }: { label: string; value: number; total: number; tone: 'neutral' | 'good' | 'bad' | 'warn' }) {
  const active = value > 0
  const color = tone === 'neutral' || !active
    ? 'text-[var(--color-foreground)]'
    : tone === 'good' ? 'text-emerald-400' : tone === 'bad' ? 'text-red-400' : 'text-amber-400'
  const share = total > 0 ? Math.round((value / total) * 100) : 0
  return (
    <div className="rounded border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2">
      <div className="text-[11px] text-[var(--color-muted-foreground)]">{label}</div>
      <div className={cn('text-xl font-semibold tabular-nums', color)}>{value}</div>
      <div className="text-[11px] text-[var(--color-muted-foreground)]">{share}% of {total}</div>
    </div>
  )
}

const KIND_LABEL: Record<string, string> = {
  'wrong-type': 'wrong type',
  'missing-field': 'missing field',
  'missing-value': 'missing value',
  'undeclared-status': 'undeclared status',
  'wrong-content-type': 'wrong content type',
  'missing-header': 'missing header',
  'bad-header': 'bad header',
  'request': 'invalid request',
  'no-response': 'no answer',
  'other': 'other difference',
}

function isWrite(method: string): boolean {
  const m = method.toUpperCase()
  return m !== 'GET' && m !== 'HEAD'
}

function ProblemCard({ op }: { op: ConformOperation }) {
  const findings = op.findings ?? []
  return (
    <article aria-label={`${op.method} ${op.path} fails`} className="rounded border border-red-500/40 bg-red-500/5 p-3 flex flex-col gap-2">
      <header className="flex flex-wrap items-baseline gap-x-2 text-xs">
        <span className="rounded bg-red-500/20 px-1.5 py-0.5 text-[10px] font-bold text-red-400">
          {op.status === 'error' ? 'NO ANSWER' : op.status === 'blocked' ? 'NOT SENT' : 'FAILED'}
        </span>
        <span className="font-mono font-semibold">{op.method}</span>
        <span className="font-mono">{op.path}</span>
        {op.statusCode > 0 && <span className="text-[var(--color-muted-foreground)]">answered {op.statusCode} in {op.durationMs} ms</span>}
      </header>
      {findings.map((f, i) => <FindingView key={i} f={f} />)}
      {findings.length === 0 && (
        <ul className="list-disc pl-5 font-mono text-[11px] text-red-400">
          {op.error && <li className="break-all">{op.error}</li>}
          {op.violations.map((v, i) => <li key={i} className="break-all">{v}</li>)}
        </ul>
      )}
    </article>
  )
}

const EXPORTS: { format: ConformReportFormat; label: string; file: string }[] = [
  { format: 'html', label: 'HTML report', file: '' },
  { format: 'junit', label: 'JUnit XML', file: 'conformance.xml' },
  { format: 'json', label: 'JSON', file: 'conformance.json' },
  { format: 'markdown', label: 'Markdown', file: 'conformance.md' },
]

function ExportMenu({ disabled }: { disabled: boolean }) {
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
        disabled={disabled}
        onClick={() => setOpen(o => !o)}
        className="flex items-center gap-1.5 rounded border border-[var(--color-border)] px-2 py-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] disabled:opacity-40 disabled:hover:bg-transparent"
      >
        <Download size={12} aria-hidden /> Export report
      </button>
      {open && !disabled && (
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

// A choice made in the Check Conformance sidebar; `id` changes with every click so the same choice can be made again.
export type ConformRequest = { id: number; specPath?: string; envPath?: string }

type Props = {
  onOpenCliBuilder?: (values: Record<string, string>) => void
  request?: ConformRequest | null
  // The spec and environment now chosen on the page, and the totals of the last run, for the sidebar.
  onSelectionChange?: (sel: { specPath: string; envPath: string }) => void
  onResult?: (summary: ConformSummary) => void
}

export function ConformanceTab({ onOpenCliBuilder, request, onSelectionChange, onResult }: Props) {
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
  const [maxResponseText, setMaxResponseText] = useState('')
  const [varsText, setVarsText] = useState('')
  const [optionsOpen, setOptionsOpen] = useState(false)
  const [negative, setNegative] = useState(false)
  const [failFast, setFailFast] = useState(false)
  const [recordCoverage, setRecordCoverage] = useState(false)
  const [running, setRunning] = useState(false)
  const [progress, setProgress] = useState<ConformProgress | null>(null)
  const [output, setOutput] = useState<ConformRunOutput | null>(null)

  const [appliedRequest, setAppliedRequest] = useState<ConformRequest | null>(null)
  if ((request ?? null) !== appliedRequest) {
    setAppliedRequest(request ?? null)
    if (request) {
      if (request.specPath !== undefined) setSpecPath(request.specPath)
      if (request.envPath !== undefined) setEnvPath(request.envPath)
    }
  }

  useEffect(() => { onSelectionChange?.({ specPath, envPath }) }, [specPath, envPath]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!output || output.error) return
    onResult?.({ specPath, specTitle: output.specTitle, total: output.total, covered: output.covered, passing: output.passing, failing: output.failing, failed: !!output.failed })
  }, [output]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    getConformContext().then(setCtx).catch(e => setLoadError(String(e)))
  }, [])

  useEffect(() => {
    if (!running) return
    return onConformanceProgress(setProgress)
  }, [running])

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
  const maxResponseMs = Number.parseInt(maxResponseText, 10)
  const maxResponseBad = maxResponseText !== '' && !(maxResponseMs > 0)
  const parsedVars = parseVars(varsText)
  const canRun = !timeoutBad && !maxResponseBad && parsedVars.bad.length === 0 && !!specPath && !info?.error && baseUrl.trim() !== '' && (!remote || confirmedRemote) && !running

  async function run() {
    setRunning(true)
    setProgress(null)
    setOutput(null)
    try {
      setOutput(await runConformance({
        specPath, baseUrl, envPath, allowWrites, confirmedRemote,
        tags: splitList(tagsText), paths: splitList(pathsText), negative, failFast,
        timeoutMs: timeoutBad || !timeoutText ? 0 : timeoutMs,
        maxResponseTimeMs: maxResponseBad || !maxResponseText ? 0 : maxResponseMs,
        recordCoverage: recordCoverage && !filtered,
        vars: parsedVars.vars,
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
            onClick={() => onOpenCliBuilder(cliValues({
              specPath, baseUrl, envPath, activeEnvironment: ctx.activeEnvironment ?? '',
              tags: splitList(tagsText), paths: splitList(pathsText), negative, failFast, timeoutText, maxResponseText, allowWrites,
            }))}
            title="Open the CLI Command Builder with the choices on this page"
            className="shrink-0 flex items-center gap-1.5 rounded border border-[var(--color-border)] px-2 py-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]"
          >
            <Terminal size={12} aria-hidden /> Get CLI command
          </button>
        )}
      </div>

      <div className="flex flex-1 min-h-0 overflow-hidden">
      <div className="flex-1 min-w-0 overflow-y-auto p-5 flex flex-col gap-5">
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
            <label htmlFor="conf-base" className="text-xs font-medium">Base URL{info && !info.serverUrl && <span className="text-red-400"> *</span>}</label>
            <input id="conf-base" value={baseUrl} onChange={e => { setBaseUrl(e.target.value); setConfirmedRemote(false) }} placeholder="http://localhost:8120" className={cn(inputClass, 'font-mono')} />
            <p className="text-[11px] text-[var(--color-muted-foreground)]">
              {!info ? 'Choose a spec first; the address starts as the first server it declares.'
                : info.serverUrl ? 'Starts as the first server in the spec; change it to test another deployment.'
                : 'The spec declares no server, so enter the address to test.'}
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

        <details
          open={optionsOpen}
          onToggle={e => setOptionsOpen((e.currentTarget as HTMLDetailsElement).open)}
          className="rounded border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2"
        >
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
            <div className="flex flex-col gap-1">
              <label htmlFor="conf-max-response" className="text-xs font-medium">Slowest acceptable response (ms)</label>
              <input id="conf-max-response" value={maxResponseText} onChange={e => setMaxResponseText(e.target.value)} placeholder="no limit" inputMode="numeric" className={cn(inputClass, 'font-mono')} />
              {maxResponseBad
                ? <p role="alert" className="text-[11px] text-red-400">Enter a number above 0, or leave empty.</p>
                : <p className="text-[11px] text-[var(--color-muted-foreground)]">A slower operation fails; its response is still compared with the spec. Leave a margin.</p>}
            </div>
          </div>
          <div className="mt-3 flex flex-col gap-1">
            <label htmlFor="conf-vars" className="text-xs font-medium">Variable overrides</label>
            <textarea
              id="conf-vars"
              value={varsText}
              onChange={e => setVarsText(e.target.value)}
              rows={3}
              placeholder={'id=1\nname=Lamp'}
              spellCheck={false}
              className={cn(inputClass, 'font-mono resize-y')}
            />
            <p className="text-[11px] text-[var(--color-muted-foreground)]">
              One name=value per line. They replace or add environment variables for this run only; nothing is saved. They are not part of Get CLI command (add <span className="font-mono">--var name=value</span> yourself). Do not put secrets here.
            </p>
            {parsedVars.bad.length > 0 && (
              <p role="alert" className="text-[11px] text-red-400">Each line must look like name=value: {parsedVars.bad[0]}</p>
            )}
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
          {running && (
            <div className="flex items-center gap-2 min-w-0 text-xs text-[var(--color-muted-foreground)]">
              <progress
                aria-label="Conformance run progress"
                value={progress?.done ?? 0}
                max={progress?.total || 1}
                className="w-32 h-1.5 shrink-0 accent-[var(--color-accent)]"
              />
              <span data-testid="conform-progress" className="truncate">
                {progress ? `${progress.done} of ${progress.total} · ${progress.operation}` : 'Starting…'}
              </span>
            </div>
          )}
        </div>

        {output?.error && <p role="alert" className="text-xs text-red-400">{output.error}</p>}

        {output && output.warnings?.length > 0 && (
          <ul aria-label="Environment warnings" className="rounded border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-400">
            {output.warnings.map((w, i) => <li key={i}>{w}</li>)}
          </ul>
        )}

        {output && !output.error && (
          <section aria-label="Conformance results" className="flex flex-col gap-4">
            {(() => {
              const problems = output.operations.filter(o => o.status === 'failed' || o.status === 'error' || o.status === 'blocked')
              const blocked = problems.filter(o => o.status === 'blocked').length
              const missing = output.missingValues ?? []
              const passed = output.operations.filter(o => o.status === 'passed')
              const notRun = output.operations.filter(o => o.status === 'skipped')
              const kinds = new Map<string, number>()
              for (const op of problems) {
                for (const kind of new Set((op.findings ?? []).map(f => f.kind))) kinds.set(kind, (kinds.get(kind) ?? 0) + 1)
              }
              const noAnswer = problems.filter(o => o.status === 'error').length
              const broken = problems.length - blocked - noAnswer
              let headline: string
              if (problems.length === 0) {
                headline = `All ${passed.length} checked operations match the spec`
              } else if (output.covered === 0 && noAnswer > 0) {
                headline = `No operation was reached: is the server running at ${output.baseUrl}?`
              } else if (broken === 0 && noAnswer === 0) {
                headline = `${blocked} of ${output.total} operations could not be checked`
              } else {
                const parts: string[] = []
                if (broken > 0) parts.push(`${broken} of ${output.total} operations break the spec`)
                if (noAnswer > 0) parts.push(`${noAnswer} got no answer`)
                if (blocked > 0) parts.push(`${blocked} could not be checked`)
                headline = parts.join(', ')
              }
              return (
                <>
                  <div>
                    <h2 data-testid="conform-headline" className={cn('text-sm font-semibold', problems.length > 0 || output.failed ? 'text-red-400' : 'text-emerald-400')}>
                      {headline}
                    </h2>
                    {notRun.length > 0 && (
                      <p data-testid="conform-skipped" className="mt-1 text-[11px] text-[var(--color-muted-foreground)]">
                        {notRun.length} skipped (not sent: read-only mode or a filter left them out)
                      </p>
                    )}
                    {kinds.size > 0 && (
                      <p className="mt-1 flex flex-wrap gap-1.5 text-[11px]">
                        <span className="text-[var(--color-muted-foreground)]">Why they fail:</span>
                        {[...kinds].map(([kind, n]) => (
                          <span key={kind} className="rounded border border-[var(--color-border)] px-1.5 py-0.5">{KIND_LABEL[kind] ?? kind} × {n}</span>
                        ))}
                      </p>
                    )}
                  </div>

                  {missing.length > 0 && (
                    <div role="region" aria-label="Missing values" className="rounded border border-amber-500/40 bg-amber-500/10 p-3 text-xs">
                      <p className="font-semibold text-amber-400">
                        {output.blockedOperations ?? blocked} {(output.blockedOperations ?? blocked) === 1 ? 'operation was' : 'operations were'} not sent: no value for {missing.map(m => `{{${m}}}`).join(', ')}
                      </p>
                      <p className="mt-1">Nothing was compared for them, so they are not counted as passing. Add the values to the environment, or give them under Options → Variable overrides.</p>
                      <button
                        type="button"
                        onClick={() => {
                          const have = new Set(parseVars(varsText).vars.map(v => v.slice(0, v.indexOf('=')).trim()))
                          const lines = missing.filter(m => !have.has(m)).map(m => `${m}=`)
                          if (lines.length) setVarsText(t => (t.trim() ? t.replace(/\n*$/, '\n') : '') + lines.join('\n'))
                          setOptionsOpen(true)
                        }}
                        className="mt-2 rounded border border-amber-500/50 px-2 py-1 text-xs hover:bg-amber-500/20"
                      >
                        Add these to Variable overrides
                      </button>
                    </div>
                  )}

                  {problems.length > 0 && (
                    <div role="region" aria-label="Problems" className="flex flex-col gap-2.5">
                      {problems.map(op => <ProblemCard key={`${op.method} ${op.path}`} op={op} />)}
                    </div>
                  )}

                  {passed.length > 0 && (
                    <details open role="region" aria-label="Passing operations" className="rounded border border-[var(--color-border)] px-3 py-2">
                      <summary className="cursor-pointer text-xs font-semibold text-emerald-400">Passing ({passed.length})</summary>
                      <ul className="mt-2 text-xs">
                        {passed.map(op => (
                          <li key={`${op.method} ${op.path}`} className="flex gap-2 border-b border-[var(--color-border)] py-1 last:border-0">
                            <span className="font-mono font-semibold w-14 shrink-0">{op.method}</span>
                            <span className="font-mono flex-1 min-w-0 break-all">{op.path}</span>
                            <span className="font-mono text-[var(--color-muted-foreground)]">{op.statusCode} · {op.durationMs} ms</span>
                          </li>
                        ))}
                      </ul>
                    </details>
                  )}

                  {notRun.length > 0 && (
                    <details role="region" aria-label="Not run" className="rounded border border-[var(--color-border)] px-3 py-2">
                      <summary className="cursor-pointer text-xs font-semibold text-[var(--color-muted-foreground)]">Not run ({notRun.length})</summary>
                      <ul className="mt-2 text-xs">
                        {notRun.map(op => (
                          <li key={`${op.method} ${op.path}`} className="flex gap-2 border-b border-[var(--color-border)] py-1 last:border-0">
                            <span className="font-mono font-semibold w-14 shrink-0">{op.method}</span>
                            <span className="font-mono flex-1 min-w-0 break-all">{op.path}</span>
                            <span className="text-[var(--color-muted-foreground)]">
                              {isWrite(op.method) && !output.writes ? 'write request, not sent' : 'left out by the tag or path filter'}
                            </span>
                          </li>
                        ))}
                      </ul>
                    </details>
                  )}
                </>
              )
            })()}

            {output.coverageNote && (
              <p data-testid="conform-coverage-note" className="text-[11px] text-[var(--color-muted-foreground)]">{output.coverageNote}</p>
            )}

            {output.probes.length > 0 && (
              <div role="region" aria-label="Negative probes" className="flex flex-col gap-1.5">
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

      <aside role="region" aria-label="Summary" className="w-80 shrink-0 overflow-y-auto border-l border-[var(--color-border)] bg-[var(--color-card)] p-4 flex flex-col gap-3">
        <div className="flex justify-end"><ExportMenu disabled={!output || !!output.error} /></div>
        {output && !output.error ? (
          <>
            <div className="grid grid-cols-2 gap-2">
              <Tile label="Covered" value={output.covered} total={output.total} tone="neutral" />
              <Tile label="Not covered" value={output.notCovered} total={output.total} tone="warn" />
              <Tile label="Passing" value={output.passing} total={output.total} tone="good" />
              <Tile label="Failing" value={output.failing} total={output.total} tone="bad" />
            </div>
            <p className="text-[11px] text-[var(--color-muted-foreground)]">
              {output.specTitle} at {output.baseUrl} · {output.writes ? 'reads and writes' : 'read requests only'} · {output.durationMs} ms
            </p>
          </>
        ) : (
          <p className="text-xs text-[var(--color-muted-foreground)]">
            After a run, the four totals appear here: how many operations were reached (Covered), how many were not (Not covered), and how many of the reached ones match the spec (Passing) or do not (Failing).
          </p>
        )}
      </aside>
      </div>
    </div>
  )
}
