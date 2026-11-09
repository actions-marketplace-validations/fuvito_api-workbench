import { cn } from '@/lib/utils'
import type { ScenarioCoverage } from '@/lib/bridge'

export function CoverageBanner({ coverage }: { coverage: ScenarioCoverage }) {
  return (
    <div className="shrink-0 px-3 py-2 border-b border-[var(--color-border)] bg-[var(--color-card)] flex flex-col gap-1.5">
      <div className="flex items-center gap-2">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">Spec Coverage</span>
        <span className={cn('text-xs font-semibold',
          coverage.percent >= 100 ? 'text-emerald-400'
          : coverage.percent >= 50 ? 'text-amber-400'
          : 'text-red-400')}>
          {coverage.covered} / {coverage.total} operations ({Math.round(coverage.percent)}%)
        </span>
      </div>
      <div className="h-1.5 rounded-full bg-[var(--color-muted)] overflow-hidden">
        <div
          className={cn('h-full rounded-full transition-all',
            coverage.percent >= 100 ? 'bg-emerald-500'
            : coverage.percent >= 50 ? 'bg-amber-500'
            : 'bg-red-500')}
          style={{ width: `${coverage.percent}%` }}
        />
      </div>
      {coverage.missed.length > 0 && (
        <p className="text-[10px] text-[var(--color-muted-foreground)]">
          Not exercised: {coverage.missed.join(', ')}
        </p>
      )}
    </div>
  )
}
