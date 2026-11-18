import { useState } from 'react'
import { Plus, Play, CheckCircle2, XCircle, MinusCircle, ChevronDown, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import { statusColor } from '@/lib/httpUtils'
import { StepEditorCard } from './ScenarioStepEditor'
import type { ScenarioDetail, ScenarioResult, StepDetail } from '@/lib/bridge'

type Props = {
  visualDetail: ScenarioDetail | null
  result: ScenarioResult | null
  running: boolean
  selectedStepIndex: number | null
  editStepCount: number
  onUpdateStep: (index: number, updates: Partial<StepDetail>) => void
  onRemoveStep: (index: number) => void
  onMoveStep: (from: number, dir: -1 | 1) => void
  onToggleDisabled: (index: number) => void
  onRunFrom: (index: number) => void
}

function StepResultStrip({ result, onRunFrom, running }: {
  result: NonNullable<ScenarioResult['steps']>[number]
  onRunFrom: () => void
  running: boolean
}) {
  const [open, setOpen] = useState(!result.passed && !result.skipped)
  const failedAssertions = (result.assertions ?? []).filter(a => !a.passed)

  return (
    <div className={cn(
      'shrink-0 border-b text-xs',
      result.skipped
        ? 'bg-[var(--color-muted)]/50 border-[var(--color-border)] text-[var(--color-muted-foreground)]'
        : result.passed
          ? 'bg-emerald-500/8 border-emerald-500/20'
          : 'bg-red-500/8 border-red-500/20',
    )}>
      <div className="flex items-center hover:bg-black/5 transition-colors">
        <button
          type="button"
          onClick={() => setOpen(o => !o)}
          className="flex-1 flex items-center gap-2 px-3 py-2 text-left"
        >
          {open
            ? <ChevronDown size={11} className="shrink-0 text-[var(--color-muted-foreground)]" />
            : <ChevronRight size={11} className="shrink-0 text-[var(--color-muted-foreground)]" />}
          {result.skipped
            ? <MinusCircle size={13} className="shrink-0 text-[var(--color-muted-foreground)]" />
            : result.passed
              ? <CheckCircle2 size={13} className="shrink-0 text-emerald-400" />
              : <XCircle size={13} className="shrink-0 text-red-400" />}
          <span className={cn(
            'font-semibold',
            result.passed ? 'text-emerald-400' : result.skipped ? 'text-[var(--color-muted-foreground)]' : 'text-red-400',
          )}>
            {result.passed ? 'PASSED' : result.skipped ? 'SKIPPED' : 'FAILED'}
          </span>
          {result.statusCode > 0 && (
            <span className={cn('font-mono font-bold', statusColor(result.statusCode))}>
              {result.statusCode}
            </span>
          )}
          {result.durationMs > 0 && (
            <span className="text-[var(--color-muted-foreground)]">{result.durationMs} ms</span>
          )}
          {(result.attempts ?? 0) > 1 && (
            <span className="text-amber-400">{result.attempts} attempts</span>
          )}
        </button>
        <button
          type="button"
          onClick={() => onRunFrom()}
          disabled={running}
          title="Run scenario from this step"
          className="shrink-0 mr-2 flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] text-[var(--color-muted-foreground)] hover:text-[var(--color-accent)] hover:bg-[var(--color-accent)]/10 transition-colors disabled:opacity-40"
        >
          <Play size={9} /> Run from here
        </button>
      </div>
      {open && (
        <div className="px-4 pb-2.5 flex flex-col gap-1">
          {result.error && (
            <p className="text-red-400 font-mono break-all">{result.error}</p>
          )}
          {result.skipped && (
            <p className="text-[var(--color-muted-foreground)]">Skipped — previous step failed.</p>
          )}
          {failedAssertions.length > 0 && failedAssertions.map((a, i) => (
            <div key={i} className="flex items-start gap-1.5">
              <XCircle size={12} className="shrink-0 text-red-400 mt-0.5" />
              <span className="text-red-300 font-mono break-all">{a.message}</span>
            </div>
          ))}
          {result.passed && (result.assertions ?? []).map((a, i) => (
            <div key={i} className="flex items-start gap-1.5">
              <CheckCircle2 size={12} className="shrink-0 text-emerald-400 mt-0.5" />
              <span className="text-[var(--color-muted-foreground)] font-mono">{a.message}</span>
            </div>
          ))}
          {Object.entries(result.extracted ?? {}).map(([k, v]) => (
            <div key={k} className="flex items-center gap-1.5">
              <span className="text-violet-400 font-mono shrink-0">→ {k}:</span>
              <span className="font-mono text-[var(--color-muted-foreground)] truncate">{v}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export function StepDetailPane({
  visualDetail, result, running, selectedStepIndex, editStepCount,
  onUpdateStep, onRemoveStep, onMoveStep, onToggleDisabled, onRunFrom,
}: Props) {
  const step = selectedStepIndex !== null ? visualDetail?.steps[selectedStepIndex] : undefined
  const stepResult = selectedStepIndex !== null ? result?.steps?.[selectedStepIndex] ?? null : null

  return (
    <div className="flex-1 overflow-hidden min-w-0 flex flex-col">
      <div className="shrink-0 px-3 py-1 border-b border-[var(--color-border)] bg-[var(--color-background)]">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          Step Editor
        </span>
      </div>

      <div className="flex-1 overflow-hidden flex flex-col" role="region" aria-label="Step details">
        {step && selectedStepIndex !== null ? (
          <>
            {stepResult && (
              <StepResultStrip
                result={stepResult}
                onRunFrom={() => onRunFrom(selectedStepIndex)}
                running={running}
              />
            )}
            <div className="flex-1 overflow-y-auto p-3">
              <StepEditorCard
                step={step}
                index={selectedStepIndex}
                isFirst={selectedStepIndex === 0}
                isLast={selectedStepIndex === editStepCount - 1}
                onChange={updates => onUpdateStep(selectedStepIndex, updates)}
                onRemove={() => onRemoveStep(selectedStepIndex)}
                onMoveUp={() => onMoveStep(selectedStepIndex, -1)}
                onMoveDown={() => onMoveStep(selectedStepIndex, 1)}
                onToggleDisabled={() => onToggleDisabled(selectedStepIndex)}
              />
            </div>
          </>
        ) : (
          <div className="h-full flex flex-col items-center justify-center text-xs text-[var(--color-muted-foreground)]">
            <Plus size={24} className="mb-2 opacity-30" />
            Select a step to edit, or add a new one.
          </div>
        )}
      </div>
    </div>
  )
}
