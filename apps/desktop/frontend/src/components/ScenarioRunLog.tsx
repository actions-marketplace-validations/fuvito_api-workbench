import { useState, useEffect, useCallback } from 'react'
import { CheckCircle2, XCircle, MinusCircle, ChevronDown, ChevronRight, RefreshCw, ScrollText, Copy } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScenarioRunEntry, ScenarioResult, StepResult } from '@/lib/bridge'
import { getScenarioRunHistory, loadScenarioResult } from '@/lib/bridge'

type Props = {
  workspacePath: string
  refreshTrigger?: number
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function statusColor(code: number): string {
  if (code < 300) return 'text-emerald-400'
  if (code < 400) return 'text-blue-400'
  if (code < 500) return 'text-amber-400'
  return 'text-red-400'
}

function tryPrettyJSON(s: string): string {
  try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s }
}

// ── Full step result (from cached ScenarioResult) ─────────────────────────────

function FullStepRow({ step, index }: { step: StepResult; index: number }) {
  const [bodyOpen, setBodyOpen] = useState(false)
  const [headersOpen, setHeadersOpen] = useState(false)
  const failedAssertions = step.assertions.filter(a => !a.passed)

  return (
    <div className={cn(
      'border-t border-[var(--color-border)] pl-4 pr-3',
      !step.passed && !step.skipped && 'bg-red-500/5',
    )}>
      {/* Step header */}
      <div className="flex items-center gap-2 py-1.5">
        <span className="text-[9px] text-[var(--color-muted-foreground)] w-4 text-right shrink-0">{index + 1}</span>
        {step.skipped
          ? <MinusCircle size={11} className="text-[var(--color-muted-foreground)] shrink-0" />
          : step.passed
            ? <CheckCircle2 size={11} className="text-emerald-400 shrink-0" />
            : <XCircle size={11} className="text-red-400 shrink-0" />}
        <span className={cn(
          'text-[11px] flex-1 truncate font-medium',
          step.skipped ? 'text-[var(--color-muted-foreground)] italic' : 'text-[var(--color-foreground)]',
        )}>
          {step.label}
        </span>
        {step.statusCode > 0 && (
          <span className={cn('text-[10px] font-mono tabular-nums shrink-0', statusColor(step.statusCode))}>
            {step.statusCode}
          </span>
        )}
        {step.durationMs > 0 && (
          <span className="text-[10px] text-[var(--color-muted-foreground)] tabular-nums shrink-0">
            {formatDuration(step.durationMs)}
          </span>
        )}
      </div>

      {/* Error */}
      {step.error && (
        <p className="text-[10px] text-red-400 pb-1.5 pl-7 break-all">{step.error}</p>
      )}

      {/* Skipped */}
      {step.skipped && (
        <p className="text-[10px] text-[var(--color-muted-foreground)] italic pb-1.5 pl-7">
          Skipped — previous step failed.
        </p>
      )}

      {/* Assertion results */}
      {step.assertions.length > 0 && (
        <div className="pl-7 pb-1.5 flex flex-col gap-1">
          {step.assertions.map((a, i) => (
            <div key={i} className="flex items-start gap-1.5">
              {a.passed
                ? <CheckCircle2 size={10} className="shrink-0 text-emerald-400 mt-0.5" />
                : <XCircle size={10} className="shrink-0 text-red-400 mt-0.5" />}
              <span className={cn(
                'text-[10px] break-all',
                a.passed ? 'text-[var(--color-muted-foreground)]' : 'text-red-300',
              )}>
                {a.message}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Extracted variables */}
      {Object.keys(step.extracted ?? {}).length > 0 && (
        <div className="pl-7 pb-1.5 flex flex-col gap-0.5">
          {Object.entries(step.extracted).map(([k, v]) => (
            <div key={k} className="flex items-center gap-1 text-[10px] group/ext">
              <span className="text-violet-400 shrink-0">→ {k}:</span>
              <span className="text-[var(--color-foreground)] font-mono break-all flex-1 truncate">{v}</span>
              <button
                type="button"
                onClick={() => navigator.clipboard.writeText(v)}
                className="shrink-0 opacity-0 group-hover/ext:opacity-100 p-0.5 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
              >
                <Copy size={9} />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Response body & headers toggles */}
      {(step.responseBody || (step.responseHeaders?.length ?? 0) > 0) && (
        <div className="pl-7 pb-1.5 flex gap-3">
          {step.responseBody && (
            <button
              type="button"
              onClick={() => setBodyOpen(o => !o)}
              className="flex items-center gap-0.5 text-[10px] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
            >
              {bodyOpen ? <ChevronDown size={9} /> : <ChevronRight size={9} />}
              Response body
            </button>
          )}
          {(step.responseHeaders?.length ?? 0) > 0 && (
            <button
              type="button"
              onClick={() => setHeadersOpen(o => !o)}
              className="flex items-center gap-0.5 text-[10px] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
            >
              {headersOpen ? <ChevronDown size={9} /> : <ChevronRight size={9} />}
              Headers ({step.responseHeaders!.length})
            </button>
          )}
        </div>
      )}

      {bodyOpen && step.responseBody && (
        <pre className="ml-7 mb-1.5 text-[10px] leading-tight font-mono text-[var(--color-foreground)] bg-[var(--color-background)] border border-[var(--color-border)] rounded px-2 py-1.5 max-h-40 overflow-auto whitespace-pre-wrap break-all">
          {tryPrettyJSON(step.responseBody)}
        </pre>
      )}

      {headersOpen && step.responseHeaders && (
        <div className="ml-7 mb-1.5 border border-[var(--color-border)] rounded overflow-hidden">
          {step.responseHeaders.map((h, i) => (
            <div key={i} className="flex gap-1.5 px-2 py-0.5 text-[10px] font-mono border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-muted)]">
              <span className="text-[var(--color-muted-foreground)] shrink-0">{h.name}:</span>
              <span className="text-[var(--color-foreground)] truncate flex-1" title={h.value}>{h.value}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ── Fallback summary step (when no cached result) ─────────────────────────────

function SummaryStepRow({ step, index }: { step: { label: string; passed: boolean; skipped: boolean; statusCode?: number; durationMs?: number; error?: string }; index: number }) {
  return (
    <div className="flex items-center gap-2 py-1 pl-4 pr-3 border-t border-[var(--color-border)]">
      <span className="text-[9px] text-[var(--color-muted-foreground)] w-4 text-right shrink-0">{index + 1}</span>
      {step.skipped
        ? <MinusCircle size={11} className="text-[var(--color-muted-foreground)] shrink-0" />
        : step.passed
          ? <CheckCircle2 size={11} className="text-emerald-400 shrink-0" />
          : <XCircle size={11} className="text-red-400 shrink-0" />}
      <span className={cn(
        'text-[11px] flex-1 truncate',
        step.skipped ? 'text-[var(--color-muted-foreground)] italic' : 'text-[var(--color-foreground)]',
      )}>
        {step.label}
      </span>
      {step.statusCode != null && step.statusCode > 0 && (
        <span className={cn('text-[10px] font-mono tabular-nums shrink-0', statusColor(step.statusCode))}>
          {step.statusCode}
        </span>
      )}
      {step.durationMs != null && step.durationMs > 0 && (
        <span className="text-[10px] text-[var(--color-muted-foreground)] tabular-nums shrink-0">
          {formatDuration(step.durationMs)}
        </span>
      )}
      {step.error && (
        <span className="text-[10px] text-red-400 truncate max-w-[120px]" title={step.error}>
          {step.error}
        </span>
      )}
    </div>
  )
}

// ── Run row ───────────────────────────────────────────────────────────────────

function RunRow({ run }: { run: ScenarioRunEntry }) {
  const [open, setOpen] = useState(false)
  const [fullResult, setFullResult] = useState<ScenarioResult | null | 'loading'>(null)
  const name = run.title || run.scenarioPath.split(/[\\/]/).pop()?.replace(/\.(yaml|yml|json)$/i, '') || 'scenario'

  async function handleToggle() {
    if (!open && fullResult === null) {
      setFullResult('loading')
      try {
        const r = await loadScenarioResult(run.scenarioPath)
        setFullResult(r)
      } catch {
        setFullResult(null)
      }
    }
    setOpen(o => !o)
  }

  const steps = fullResult !== null && fullResult !== 'loading' ? fullResult.steps : null

  return (
    <div className="border-b border-[var(--color-border)] last:border-0">
      <button
        type="button"
        onClick={handleToggle}
        className="w-full text-left px-3 py-2 hover:bg-[var(--color-muted)] transition-colors"
      >
        <div className="flex items-center gap-2">
          {run.passed
            ? <CheckCircle2 size={13} className="text-emerald-400 shrink-0" />
            : <XCircle size={13} className="text-red-400 shrink-0" />
          }
          <span className="text-xs font-medium text-[var(--color-foreground)] flex-1 truncate">{name}</span>
          <span className="text-[10px] text-[var(--color-muted-foreground)] tabular-nums shrink-0">
            {formatDuration(run.durationMs)}
          </span>
          <span className="text-[var(--color-muted-foreground)] shrink-0">
            {open ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
          </span>
        </div>
        <div className="flex items-center gap-2 mt-0.5 pl-5">
          <span className="text-[10px] text-[var(--color-muted-foreground)]">
            {formatTime(run.executedAt)}
          </span>
          <span className="text-[10px] text-emerald-400">{run.stepPassed} passed</span>
          {run.stepFailed > 0 && (
            <span className="text-[10px] text-red-400">{run.stepFailed} failed</span>
          )}
          {run.stepSkipped > 0 && (
            <span className="text-[10px] text-[var(--color-muted-foreground)]">{run.stepSkipped} skipped</span>
          )}
        </div>
      </button>

      {open && (
        <div className="bg-[var(--color-card)] pb-1">
          {fullResult === 'loading' && (
            <p className="pl-4 py-2 text-[10px] text-[var(--color-muted-foreground)]">Loading…</p>
          )}
          {steps !== null ? (
            steps.length > 0
              ? steps.map((step, i) => <FullStepRow key={i} step={step} index={i} />)
              : <p className="pl-4 py-2 text-[10px] text-[var(--color-muted-foreground)]">No step detail available.</p>
          ) : fullResult === null && (
            run.steps.map((step, i) => <SummaryStepRow key={i} step={step} index={i} />)
          )}
        </div>
      )}
    </div>
  )
}

// ── Main export ───────────────────────────────────────────────────────────────

export function ScenarioRunLog({ workspacePath, refreshTrigger }: Props) {
  const [runs, setRuns] = useState<ScenarioRunEntry[]>([])
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    if (!workspacePath) { setRuns([]); return }
    setLoading(true)
    try {
      const data = await getScenarioRunHistory(200)
      setRuns(data)
    } finally {
      setLoading(false)
    }
  }, [workspacePath])

  useEffect(() => { refresh() }, [refresh, refreshTrigger])

  if (!workspacePath) {
    return (
      <div className="flex flex-col items-center gap-2 px-4 py-8 text-center">
        <ScrollText size={20} className="text-[var(--color-muted-foreground)] opacity-30" />
        <p className="text-xs text-[var(--color-muted-foreground)]">No workspace open</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="shrink-0 flex items-center justify-between px-3 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-background)]">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          Run Log
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
        {runs.length === 0 && !loading ? (
          <div className="flex flex-col items-center gap-2 px-4 py-8 text-center">
            <ScrollText size={20} className="text-[var(--color-muted-foreground)] opacity-30" />
            <p className="text-xs text-[var(--color-muted-foreground)]">No runs yet</p>
            <p className="text-[11px] text-[var(--color-muted-foreground)] opacity-60">
              Run a scenario to see it here
            </p>
          </div>
        ) : (
          runs.map(run => <RunRow key={run.id} run={run} />)
        )}
      </div>
    </div>
  )
}
