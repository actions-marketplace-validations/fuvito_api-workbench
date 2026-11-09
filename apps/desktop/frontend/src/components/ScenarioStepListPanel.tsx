import type { RefObject } from 'react'
import { Plus } from 'lucide-react'
import type { ScenarioDetail, ScenarioResult, StepDetail } from '@/lib/bridge'
import { StepListRow } from './ScenarioStepList'

type Props = {
  width: number
  mode: 'view' | 'visual'
  listRef: RefObject<HTMLDivElement | null>
  loadError: string | null
  detailLoaded: boolean
  viewSteps: StepDetail[]
  editSteps: StepDetail[]
  visualDetail: ScenarioDetail | null
  onMetaChange: (patch: { title?: string; description?: string }) => void
  selectedStepIndex: number | null
  onSelectStep: (index: number) => void
  result: ScenarioResult | null
  running: boolean
  onRunFrom: (index: number) => void
  onMoveStep: (from: number, dir: -1 | 1) => void
  onRemoveStep: (index: number) => void
  onDuplicateStep: (index: number) => void
  onToggleDisabled: (index: number) => void
  onAddStep: () => void
}

export function StepListPanel({
  width, mode, listRef, loadError, detailLoaded,
  viewSteps, editSteps, visualDetail, onMetaChange,
  selectedStepIndex, onSelectStep, result, running,
  onRunFrom, onMoveStep, onRemoveStep, onDuplicateStep, onToggleDisabled, onAddStep,
}: Props) {
  const steps = mode === 'view' ? viewSteps : editSteps

  return (
    <div className="shrink-0 flex flex-col overflow-hidden" style={{ width }}>

      {/* Edit mode: scenario meta */}
      {mode === 'visual' && visualDetail && (
        <div className="shrink-0 px-3 py-2 border-b border-[var(--color-border)] bg-[var(--color-card)]">
          <input
            value={visualDetail.title}
            onChange={e => onMetaChange({ title: e.target.value })}
            placeholder="Scenario title"
            className="w-full bg-transparent border-b border-transparent focus:border-[var(--color-accent)] outline-none text-xs font-semibold text-[var(--color-foreground)] py-0.5 mb-0.5"
          />
          <input
            value={visualDetail.description}
            onChange={e => onMetaChange({ description: e.target.value })}
            placeholder="Description (optional)"
            className="w-full bg-transparent border-b border-transparent focus:border-[var(--color-accent)] outline-none text-xs text-[var(--color-muted-foreground)] py-0.5"
          />
        </div>
      )}

      {/* Panel column header */}
      <div className="shrink-0 px-3 py-1 border-b border-[var(--color-border)] bg-[var(--color-background)]">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          {mode === 'visual' ? 'Edit Steps' : 'Steps'}
        </span>
      </div>

      {/* Step list */}
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
        ) : mode === 'view' && !detailLoaded ? (
          <div className="flex flex-col py-1">
            {[0.7, 0.9, 0.6].map((w, i) => (
              <div key={i} className="flex items-center gap-2 px-3 py-2">
                <div className="w-4 h-4 rounded-full bg-[var(--color-muted)] animate-pulse shrink-0" />
                <div className="h-2 rounded bg-[var(--color-muted)] animate-pulse" style={{ width: `${w * 100}%` }} />
              </div>
            ))}
          </div>
        ) : mode === 'view' ? (
          viewSteps.length === 0 ? (
            <p className="px-3 py-4 text-xs text-center text-[var(--color-muted-foreground)]">
              No steps yet. Switch to <strong>Edit</strong> to add your first step.
            </p>
          ) : (
            viewSteps.map((step, i) => (
              <StepListRow
                key={i} step={step} index={i}
                result={result?.steps?.[i] ?? null}
                selected={selectedStepIndex === i}
                onSelect={onSelectStep}
                onRunFrom={() => onRunFrom(i)}
                running={running}
              />
            ))
          )
        ) : (
          // Visual edit mode
          editSteps.length === 0 ? (
            <p className="px-3 py-4 text-xs text-center text-[var(--color-muted-foreground)]">No steps yet.</p>
          ) : (
            editSteps.map((step, i) => (
              <StepListRow
                key={i} step={step} index={i}
                result={result?.steps?.[i] ?? null}
                selected={selectedStepIndex === i}
                onSelect={onSelectStep}
                editMode
                isFirst={i === 0}
                isLast={i === editSteps.length - 1}
                onMoveUp={() => onMoveStep(i, -1)}
                onMoveDown={() => onMoveStep(i, 1)}
                onRemove={() => onRemoveStep(i)}
                onDuplicate={() => onDuplicateStep(i)}
                onToggleDisabled={() => onToggleDisabled(i)}
              />
            ))
          )
        )}
      </div>

      {/* Edit mode: add step */}
      {mode === 'visual' && (
        <div className="shrink-0 px-3 py-2 border-t border-[var(--color-border)]">
          <button type="button" onClick={onAddStep}
            className="text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors flex items-center gap-1">
            <Plus size={11} /> Add Step
          </button>
        </div>
      )}
    </div>
  )
}
