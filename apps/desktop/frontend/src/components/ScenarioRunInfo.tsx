import type { ScenarioResult } from '@/lib/bridge'

// What the run generated: the setup error (the scenario never started), the seed that replays the
// random values, and the scenario variables as they were generated.
export function ScenarioRunInfo({ result }: { result: ScenarioResult }) {
  const hasSeed = result.seed !== undefined
  const vars = result.variables ?? []
  if (!result.error && !hasSeed && vars.length === 0) return null

  return (
    <div className="shrink-0 px-4 py-2 text-xs border-b border-[var(--color-border)] bg-[var(--color-card)] flex flex-col gap-1">
      {result.error && (
        <p role="alert" className="text-red-400 font-mono break-all">
          <span className="font-semibold font-sans">Scenario could not start: </span>{result.error}
        </p>
      )}
      {vars.length > 0 && (
        <div className="flex flex-wrap gap-x-4 gap-y-0.5 font-mono text-[11px]">
          {vars.map(v => (
            <span key={v.name} className="text-[var(--color-muted-foreground)]">
              {v.name} = <span className="text-[var(--color-foreground)]">{v.value}</span>
            </span>
          ))}
        </div>
      )}
      {hasSeed && (
        <p className="text-[11px] text-[var(--color-muted-foreground)]">
          Random values used seed <span className="font-mono text-[var(--color-foreground)]">{result.seed}</span>
          {' '}— replay with <span className="font-mono">clrsig run --seed {result.seed}</span>
        </p>
      )}
    </div>
  )
}
