import { cn } from '@/lib/utils'
import { ScenarioMetaSection } from './ScenarioMetaSection'
import { ScenarioVariablesSection } from './ScenarioVariablesSection'
import { ScenarioRunInfo } from './ScenarioRunInfo'
import { CoverageBanner } from './ScenarioCoverageBanner'
import type { ScenarioDetail, ScenarioResult, ScenarioVariable, AccumulatedCoverage } from '@/lib/bridge'

export type SetupTab = 'definition' | 'result'

type Props = {
  tab: SetupTab
  onTabChange: (tab: SetupTab) => void
  detail: ScenarioDetail
  result: ScenarioResult | null
  running: boolean
  accumulatedCoverage: AccumulatedCoverage | null
  onChange: (patch: { description?: string; specFile?: string; variables?: ScenarioVariable[] }) => void
  onResetCoverage: () => void
}

export function ScenarioSetupPane({
  tab, onTabChange, detail, result, running, accumulatedCoverage, onChange, onResetCoverage,
}: Props) {
  const hasResult = !!result && !running
  const hasCoverage = !!(accumulatedCoverage || result?.coverage)
  const resultAvailable = hasResult || hasCoverage
  const active: SetupTab = tab === 'result' && resultAvailable ? 'result' : 'definition'

  return (
    <div className="flex-1 overflow-hidden min-w-0 flex flex-col">
      <div role="tablist" aria-label="Setup views" className="shrink-0 flex items-end gap-1 px-3 pt-1 border-b border-[var(--color-border)] bg-[var(--color-card)]">
        {(['definition', 'result'] as const).map(t => {
          const disabled = t === 'result' && !resultAvailable
          return (
            <button
              key={t}
              type="button"
              role="tab"
              aria-selected={active === t}
              disabled={disabled}
              onClick={() => onTabChange(t)}
              className={cn(
                'px-4 py-2 -mb-px text-xs font-medium capitalize border-b-2 transition-colors',
                active === t
                  ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                  : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:border-[var(--color-border)]',
                disabled && 'opacity-40 cursor-not-allowed hover:text-[var(--color-muted-foreground)] hover:border-transparent',
              )}
            >
              {t}
            </button>
          )
        })}
      </div>

      <div className="flex-1 overflow-y-auto" role="region" aria-label="Setup details">
        {active === 'definition' ? (
          <>
            <ScenarioMetaSection
              description={detail.description ?? ''}
              specFile={detail.specFile ?? ''}
              onChange={onChange}
            />
            <ScenarioVariablesSection
              variables={detail.variables ?? []}
              onChange={variables => onChange({ variables })}
              defaultOpen
            />
          </>
        ) : (
          <>
            {result?.specError && (
              <div className="px-4 py-2 text-xs text-amber-400 bg-amber-500/10 border-b border-amber-500/20">
                <span className="font-semibold">specFile warning: </span>{result.specError}
              </div>
            )}
            {result && <ScenarioRunInfo result={result} />}
            <CoverageBanner
              coverage={result?.coverage}
              accumulated={accumulatedCoverage ?? undefined}
              onReset={onResetCoverage}
            />
            {!result?.specError && !result?.error && !result?.seed && !(result?.variables ?? []).length && !hasCoverage && (
              <p className="px-4 py-3 text-xs text-[var(--color-muted-foreground)]">Nothing was generated or reported for this run.</p>
            )}
          </>
        )}
      </div>
    </div>
  )
}
