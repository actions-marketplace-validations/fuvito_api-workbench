import type { RefObject } from 'react'
import { Plus } from 'lucide-react'
import type { ScenarioResult, StepDetail } from '@/lib/bridge'
import { StepListRow } from './ScenarioStepList'

type Props = {
  widthPct: number
  listRef: RefObject<HTMLDivElement | null>
  loadError: string | null
  detailLoaded: boolean
  steps: StepDetail[]
  selectedStepIndex: number | null
  onSelectStep: (index: number) => void
  result: ScenarioResult | null
  running: boolean
  onMoveStep: (from: number, dir: -1 | 1) => void
  onRemoveStep: (index: number) => void
  onDuplicateStep: (index: number) => void
  onToggleDisabled: (index: number) => void
  onAddStep: () => void
}

export function StepListPanel({
  widthPct, listRef, loadError, detailLoaded, steps,
  selectedStepIndex, onSelectStep, result, running,
  onMoveStep, onRemoveStep, onDuplicateStep, onToggleDisabled, onAddStep,
}: Props) {
  return (
    <div className="shrink-0 flex flex-col overflow-hidden" style={{ width: `${widthPct}%` }}>

      <div className="shrink-0 px-3 py-1 border-b border-[var(--color-border)] bg-[var(--color-background)]">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          Steps
        </span>
      </div>

      <div
        ref={listRef}
        role="listbox"
        aria-label="Scenario steps"
        className="flex-1 overflow-y-auto py-1 outline-none"
        onKeyDown={e => {
          if (!steps.length) return
          if (e.key === 'ArrowDown') {
            e.preventDefault()
            onSelectStep(Math.min((selectedStepIndex ?? -1) + 1, steps.length - 1))
          } else if (e.key === 'ArrowUp') {
            e.preventDefault()
            onSelectStep(Math.max((selectedStepIndex ?? 1) - 1, 0))
          }
        }}
      >
        {loadError ? (
          <p className="px-3 py-2 text-xs text-red-400">{loadError}</p>
        ) : !detailLoaded ? (
          <div className="flex flex-col py-1">
            {[0.7, 0.9, 0.6].map((w, i) => (
              <div key={i} className="flex items-center gap-2 px-3 py-2">
                <div className="w-4 h-4 rounded-full bg-[var(--color-muted)] animate-pulse shrink-0" />
                <div className="h-2 rounded bg-[var(--color-muted)] animate-pulse" style={{ width: `${w * 100}%` }} />
              </div>
            ))}
          </div>
        ) : steps.length === 0 ? (
          <p className="px-3 py-4 text-xs text-center text-[var(--color-muted-foreground)]">
            No steps yet.
          </p>
        ) : (
          steps.map((step, i) => (
            <StepListRow
              key={i}
              step={step}
              index={i}
              result={result?.steps?.[i] ?? null}
              selected={selectedStepIndex === i}
              onSelect={onSelectStep}
              editMode
              isFirst={i === 0}
              isLast={i === steps.length - 1}
              onMoveUp={() => onMoveStep(i, -1)}
              onMoveDown={() => onMoveStep(i, 1)}
              onRemove={() => onRemoveStep(i)}
              onDuplicate={() => onDuplicateStep(i)}
              onToggleDisabled={() => onToggleDisabled(i)}
              running={running}
            />
          ))
        )}
      </div>

      <div className="shrink-0 px-3 py-2 border-t border-[var(--color-border)]">
        <button
          type="button"
          onClick={onAddStep}
          className="text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors flex items-center gap-1"
        >
          <Plus size={11} /> Add step
        </button>
      </div>
    </div>
  )
}
