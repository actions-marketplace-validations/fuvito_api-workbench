import { Loader2, Play, Plus } from 'lucide-react'
import type { ScenarioDetail, ScenarioResult, StepDetail } from '@/lib/bridge'
import { StepEditorCard } from './ScenarioStepEditor'
import { StepDetailPanel } from './ScenarioStepResult'

type Props = {
  mode: 'view' | 'visual'
  running: boolean
  detail: ScenarioDetail | null
  visualDetail: ScenarioDetail | null
  result: ScenarioResult | null
  selectedStepIndex: number | null
  editStepCount: number
  onRunFrom: (index: number) => void
  onUpdateStep: (index: number, updates: Partial<StepDetail>) => void
  onRemoveStep: (index: number) => void
  onMoveStep: (from: number, dir: -1 | 1) => void
  onToggleDisabled: (index: number) => void
}

export function StepDetailPane({
  mode, running, detail, visualDetail, result,
  selectedStepIndex, editStepCount,
  onRunFrom, onUpdateStep, onRemoveStep, onMoveStep, onToggleDisabled,
}: Props) {
  const visualStep = selectedStepIndex !== null ? visualDetail?.steps[selectedStepIndex] : undefined

  return (
    <div className="flex-1 overflow-hidden min-w-0 flex flex-col">
      {/* Panel column header */}
      <div className="shrink-0 px-3 py-1 border-b border-[var(--color-border)] bg-[var(--color-background)]">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          {mode === 'visual' ? 'Step Editor' : 'Results'}
        </span>
      </div>
      <div className="flex-1 overflow-hidden" role="region" aria-label="Step details">
        {mode === 'view' ? (
          running ? (
            <div className="h-full flex flex-col items-center justify-center gap-3 text-[var(--color-muted-foreground)]">
              <Loader2 size={20} className="animate-spin" />
              <p className="text-sm">Running scenario…</p>
            </div>
          ) : selectedStepIndex !== null && detail?.steps[selectedStepIndex] ? (
            <StepDetailPanel
              step={detail.steps[selectedStepIndex]}
              result={result?.steps[selectedStepIndex] ?? null}
              onRunFrom={!running ? () => onRunFrom(selectedStepIndex) : undefined}
            />
          ) : (
            <div className="h-full flex flex-col items-center justify-center text-xs text-[var(--color-muted-foreground)]">
              <Play size={24} className="mb-2 opacity-30" />
              {detail && detail.steps.length > 0
                ? 'Select a step to see its details.'
                : <span>Click <strong>Edit</strong> to add steps, then <strong>Run</strong>.</span>}
            </div>
          )
        ) : (
          // Visual edit mode
          selectedStepIndex !== null && visualStep ? (
            <div className="h-full overflow-y-auto p-3">
              <StepEditorCard
                step={visualStep}
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
          ) : (
            <div className="h-full flex flex-col items-center justify-center text-xs text-[var(--color-muted-foreground)]">
              <Plus size={24} className="mb-2 opacity-30" />
              Select a step to edit, or add a new one.
            </div>
          )
        )}
      </div>
    </div>
  )
}
