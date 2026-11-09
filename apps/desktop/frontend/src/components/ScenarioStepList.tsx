import { Loader2, CheckCircle2, XCircle, MinusCircle, ChevronDown, ChevronUp, Eye, EyeOff, Copy, Trash2, Play } from 'lucide-react'
import { cn } from '@/lib/utils'
import { METHOD_BADGE } from '@/lib/httpUtils'
import type { StepDetail, StepResult } from '@/lib/bridge'

export function StepListRow({ step, index, result, selected, onSelect, editMode = false, isFirst = false, isLast = false, onMoveUp, onMoveDown, onRemove, onDuplicate, onToggleDisabled, onRunFrom, running = false }: {
  step: StepDetail
  index: number
  result: StepResult | null
  selected: boolean
  onSelect: (i: number) => void
  editMode?: boolean
  isFirst?: boolean
  isLast?: boolean
  onMoveUp?: () => void
  onMoveDown?: () => void
  onRemove?: () => void
  onDuplicate?: () => void
  onToggleDisabled?: () => void
  onRunFrom?: () => void
  running?: boolean
}) {
  const badge = step.method ? METHOD_BADGE[step.method] ?? '' : ''
  const isDisabled = step.disabled ?? false
  const isPending = running && !result
  return (
    <div
      role="option"
      aria-selected={selected}
      tabIndex={0}
      onClick={() => onSelect(index)}
      onKeyDown={e => {
        if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelect(index) }
      }}
      className={cn(
        'group relative flex items-start gap-2 px-3 py-2.5 cursor-pointer transition-colors border-l-2 select-none',
        selected
          ? 'bg-[var(--color-accent)]/10 border-[var(--color-accent)]'
          : 'border-transparent hover:bg-[var(--color-muted)]',
        isDisabled && 'opacity-50',
      )}
    >
      <span className={cn(
        'shrink-0 w-5 h-5 mt-0.5 rounded-full flex items-center justify-center text-[11px] font-bold',
        isPending
          ? 'bg-[var(--color-accent)]/20 text-[var(--color-accent)] animate-pulse'
          : result
            ? result.skipped
              ? 'bg-[var(--color-muted)] text-[var(--color-muted-foreground)]'
              : result.passed
                ? 'bg-emerald-500/20 text-emerald-400'
                : 'bg-red-500/20 text-red-400'
            : 'bg-[var(--color-border)] text-[var(--color-muted-foreground)]',
      )}>
        {isPending ? <Loader2 size={10} className="animate-spin" /> : index + 1}
      </span>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <span className={cn('text-xs font-medium truncate', isDisabled ? 'line-through text-[var(--color-muted-foreground)]' : selected ? 'text-[var(--color-accent)]' : 'text-[var(--color-foreground)]')}>
            {step.title || 'Untitled step'}
          </span>
          {isDisabled && <EyeOff size={9} className="shrink-0 text-amber-400" />}
          {result && (
            result.skipped
              ? <MinusCircle size={15} className="shrink-0 text-[var(--color-muted-foreground)]" />
              : result.passed
                ? <CheckCircle2 size={15} className="shrink-0 text-emerald-400" />
                : <XCircle size={15} className="shrink-0 text-red-400" />
          )}
        </div>
        {step.requestRef ? (
          <p className="text-xs text-[var(--color-muted-foreground)] truncate font-mono mt-0.5">↗ {step.requestRef}</p>
        ) : step.method ? (
          <div className="flex items-center gap-1 mt-0.5">
            <span className={cn('font-mono font-bold text-xs shrink-0', badge)}>{step.method}</span>
            <span className="text-xs text-[var(--color-muted-foreground)] truncate">{step.url}</span>
          </div>
        ) : null}
      </div>
      {editMode ? (
        <div className="shrink-0 flex items-center gap-0 opacity-0 group-hover:opacity-100 transition-opacity">
          <button type="button" onClick={e => { e.stopPropagation(); onToggleDisabled?.() }}
            title={isDisabled ? 'Enable step' : 'Disable step'}
            className={cn('p-1 transition-colors', isDisabled ? 'text-amber-400 hover:text-amber-300' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}>
            {isDisabled ? <EyeOff size={11} /> : <Eye size={11} />}
          </button>
          <button type="button" onClick={e => { e.stopPropagation(); onDuplicate?.() }} title="Duplicate step"
            className="p-1 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
            <Copy size={11} />
          </button>
          <button type="button" onClick={e => { e.stopPropagation(); onMoveUp?.() }} disabled={isFirst} title="Move up (Alt+↑)"
            className={cn('p-1 transition-colors', isFirst ? 'opacity-20 cursor-not-allowed text-[var(--color-muted-foreground)]' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}>
            <ChevronUp size={11} />
          </button>
          <button type="button" onClick={e => { e.stopPropagation(); onMoveDown?.() }} disabled={isLast} title="Move down (Alt+↓)"
            className={cn('p-1 transition-colors', isLast ? 'opacity-20 cursor-not-allowed text-[var(--color-muted-foreground)]' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}>
            <ChevronDown size={11} />
          </button>
          <button type="button" onClick={e => { e.stopPropagation(); onRemove?.() }} title="Remove step"
            className="p-1 text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors">
            <Trash2 size={11} />
          </button>
        </div>
      ) : onRunFrom ? (
        <button
          type="button"
          onClick={e => { e.stopPropagation(); onRunFrom() }}
          title="Run from this step"
          className="shrink-0 opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] text-[var(--color-muted-foreground)] hover:text-[var(--color-accent)] hover:bg-[var(--color-accent)]/10 transition-colors"
        >
          <Play size={9} /> from here
        </button>
      ) : null}
    </div>
  )
}
