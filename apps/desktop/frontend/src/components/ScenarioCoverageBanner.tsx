import { RotateCcw } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScenarioCoverage, AccumulatedCoverage } from '@/lib/bridge'
import { QualityPanel } from './QualityPanel'

type Props = {
  coverage?: ScenarioCoverage
  accumulated?: AccumulatedCoverage
  onReset?: () => void
}

export function CoverageBanner({ coverage, accumulated, onReset }: Props) {
  const primary = accumulated ?? coverage
  if (!primary) return null

  const exercised = coverage?.exercised ?? []
  const missed = primary.missed ?? []
  const pct = Math.round(primary.percent)
  const color = pct >= 100 ? 'text-emerald-400' : pct >= 50 ? 'text-amber-400' : 'text-red-400'
  const barColor = pct >= 100 ? 'bg-emerald-500' : pct >= 50 ? 'bg-amber-500' : 'bg-red-500'

  return (
    <div className="shrink-0 px-3 py-2 border-b border-[var(--color-border)] bg-[var(--color-card)] flex flex-col gap-1.5">
      <div className="flex items-center gap-2">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          {accumulated ? 'Accumulated Coverage' : 'Spec Coverage'}
        </span>
        <span className={cn('text-xs font-semibold', color)}>
          {primary.covered} / {primary.total} operations ({pct}%)
        </span>
        {accumulated && (
          <span className="text-[10px] text-[var(--color-muted-foreground)]">
            {accumulated.totalRuns} run{accumulated.totalRuns === 1 ? '' : 's'} · {accumulated.passedRuns} passed · {accumulated.failedRuns} failed
          </span>
        )}
        {onReset && accumulated && (
          <button
            type="button"
            onClick={onReset}
            title="Reset accumulated coverage"
            className="ml-auto p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
          >
            <RotateCcw size={11} />
          </button>
        )}
      </div>
      <div className="h-1.5 rounded-full bg-[var(--color-muted)] overflow-hidden">
        <div
          className={cn('h-full rounded-full transition-all', barColor)}
          style={{ width: `${primary.percent}%` }}
        />
      </div>
      {/* This run row — only shown when we have both, to highlight what this run contributed */}
      {accumulated && exercised.length > 0 && (
        <p className="text-[10px] text-[var(--color-muted-foreground)] truncate">
          This run: {exercised.join(' · ')}
        </p>
      )}
      {missed.length > 0 && (
        <p className="text-[10px] text-[var(--color-muted-foreground)]">
          Not yet covered: {missed.join(' · ')}
        </p>
      )}
      {accumulated?.quality && (
        <QualityPanel quality={accumulated.quality} compact />
      )}
    </div>
  )
}
