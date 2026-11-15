import { useState, useEffect, useCallback } from 'react'
import { CheckCircle2, XCircle, MinusCircle, ChevronDown, ChevronRight, RefreshCw, ScrollText } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScenarioRunEntry, StepRunEntry } from '@/lib/bridge'
import { getScenarioRunHistory } from '@/lib/bridge'

type Props = {
  workspacePath: string
  /** Bump this to force a refresh after a run completes */
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

function StepRow({ step, index }: { step: StepRunEntry; index: number }) {
  const icon = step.skipped
    ? <MinusCircle size={11} className="text-[var(--color-muted-foreground)] shrink-0" />
    : step.passed
      ? <CheckCircle2 size={11} className="text-emerald-400 shrink-0" />
      : <XCircle size={11} className="text-red-400 shrink-0" />

  return (
    <div className="flex items-center gap-2 py-1 pl-6 border-t border-[var(--color-border)]">
      <span className="text-[9px] text-[var(--color-muted-foreground)] w-4 text-right shrink-0">{index + 1}</span>
      {icon}
      <span className={cn(
        'text-[11px] flex-1 truncate',
        step.skipped ? 'text-[var(--color-muted-foreground)] italic' : 'text-[var(--color-foreground)]',
      )}>
        {step.label}
      </span>
      {step.statusCode != null && step.statusCode > 0 && (
        <span className={cn(
          'text-[10px] font-mono tabular-nums shrink-0',
          step.statusCode < 300 ? 'text-emerald-400' : step.statusCode < 500 ? 'text-amber-400' : 'text-red-400',
        )}>
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

function RunRow({ run }: { run: ScenarioRunEntry }) {
  const [open, setOpen] = useState(false)
  const name = run.title || run.scenarioPath.split(/[\\/]/).pop()?.replace(/\.(yaml|yml|json)$/i, '') || 'scenario'

  return (
    <div className="border-b border-[var(--color-border)] last:border-0">
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
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

      {open && run.steps.length > 0 && (
        <div className="bg-[var(--color-card)] pb-1">
          {run.steps.map((step, i) => (
            <StepRow key={i} step={step} index={i} />
          ))}
        </div>
      )}
    </div>
  )
}

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
