import type { RefObject } from 'react'
import { Plus, SlidersHorizontal } from 'lucide-react'
import { cn } from '@/lib/utils'
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
  setupSelected: boolean
  onSelectSetup: () => void
  setupAlert?: 'error' | 'warning' | null
  variableCount?: number
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
  selectedStepIndex, onSelectStep, setupSelected, onSelectSetup, setupAlert = null, variableCount = 0, result, running,
  onMoveStep, onRemoveStep, onDuplicateStep, onToggleDisabled, onAddStep,
}: Props) {
  return (
    <div className="shrink-0 flex flex-col overflow-hidden" style={{ width: `${widthPct}%` }}>

      {detailLoaded && !loadError && (
        <div
          role="button"
          tabIndex={0}
          aria-pressed={setupSelected}
          aria-label="Setup"
          onClick={onSelectSetup}
          onKeyDown={e => {
            if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelectSetup() }
            else if (e.key === 'ArrowDown' && steps.length > 0) { e.preventDefault(); onSelectStep(0) }
          }}
          className={cn(
            'shrink-0 flex items-center gap-2 px-3 py-3 cursor-pointer border-l-4 border-b-2 border-b-[var(--color-border)] select-none transition-colors',
            setupSelected
              ? 'bg-[var(--color-accent)]/15 border-l-[var(--color-accent)]'
              : 'bg-[var(--color-card)] border-l-[var(--color-muted-foreground)]/40 hover:bg-[var(--color-muted)]',
          )}
        >
          <SlidersHorizontal size={14} className={cn('shrink-0', setupSelected ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)]')} />
          <span className={cn('text-xs font-bold uppercase tracking-wider', setupSelected ? 'text-[var(--color-accent)]' : 'text-[var(--color-foreground)]')}>
            Setup
          </span>
          {variableCount > 0 && (
            <span className="px-1.5 rounded-full bg-[var(--color-muted)] text-[10px] text-[var(--color-muted-foreground)]">
              {variableCount} {variableCount === 1 ? 'variable' : 'variables'}
            </span>
          )}
          {setupAlert && (
            <span
              role="img"
              aria-label={setupAlert === 'error' ? 'Setup error' : 'Setup warning'}
              className={cn('ml-auto w-2 h-2 rounded-full', setupAlert === 'error' ? 'bg-red-400' : 'bg-amber-400')}
            />
          )}
        </div>
      )}

      <div className="shrink-0 flex items-center gap-2 px-3 py-1.5 bg-[var(--color-background)] border-b border-[var(--color-border)]">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          Steps
        </span>
        {steps.length > 0 && (
          <span className="px-1.5 rounded-full bg-[var(--color-muted)] text-[10px] text-[var(--color-muted-foreground)]">{steps.length}</span>
        )}
      </div>

      <div
        ref={listRef}
        role="listbox"
        aria-label="Scenario steps"
        className="flex-1 overflow-y-auto py-1 outline-none"
        onKeyDown={e => {
          if (!steps.length) return
          if (e.key === 'ArrowUp' && selectedStepIndex === 0) {
            e.preventDefault()
            onSelectSetup()
            return
          }
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
          <button
            type="button"
            onClick={onAddStep}
            className="w-full flex items-center gap-2 px-3 py-2.5 text-left border-l-2 border-transparent opacity-70 hover:opacity-100 hover:bg-[var(--color-muted)] transition-colors"
          >
            <span className="shrink-0 w-5 h-5 rounded-full border border-dashed border-[var(--color-muted-foreground)] flex items-center justify-center text-[11px] text-[var(--color-muted-foreground)]">1</span>
            <span className="flex items-center gap-1 text-xs italic text-[var(--color-muted-foreground)]">
              <Plus size={11} /> Start here
            </span>
          </button>
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

      {steps.length > 0 && (
        <div className="shrink-0 px-3 py-2 border-t border-[var(--color-border)]">
          <button
            type="button"
            onClick={onAddStep}
            className="text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors flex items-center gap-1"
          >
            <Plus size={11} /> Add step
          </button>
        </div>
      )}
    </div>
  )
}
