import type { StepResult } from '@/lib/bridge'

// Spec Conformance Testing summary for one run: how many checked responses match the spec, the
// violations per step, and which steps were not checked (and why). Renders nothing without a spec.
export function SpecConformance({ steps, specFile }: { steps: StepResult[]; specFile?: string }) {
  const withSpec = steps.filter(s => s.specState)
  if (withSpec.length === 0) return null

  const matched = withSpec.filter(s => s.specState === 'matched').length
  const violating = withSpec.filter(s => s.specState === 'violations')
  const unchecked = withSpec.filter(s => s.specState === 'not-checked')
  const checked = matched + violating.length
  const name = specFile ? specFile.replace(/\\/g, '/').split('/').pop() : 'the spec'

  const summary = violating.length === 0
    ? `${matched} of ${checked} responses match ${name}`
    : `${matched} of ${checked} match ${name}, ${violating.length} ${violating.length === 1 ? 'step has' : 'steps have'} violations`

  return (
    <section aria-label="Spec conformance" className="shrink-0 px-4 py-2 text-xs border-b border-[var(--color-border)] bg-[var(--color-card)] flex flex-col gap-1.5">
      <p data-testid="spec-conformance-summary" className={violating.length > 0 ? 'text-red-400 font-semibold' : checked > 0 ? 'text-emerald-400 font-semibold' : 'text-[var(--color-muted-foreground)]'}>
        {checked === 0 ? 'No responses were checked against the spec' : summary}
      </p>
      {violating.map(s => (
        <div key={s.label}>
          <p className="font-medium">{s.label}</p>
          <ul className="list-disc pl-5 font-mono text-[11px] text-red-400">
            {(s.specViolations ?? []).map((v, i) => <li key={i} className="break-all">{v}</li>)}
          </ul>
        </div>
      ))}
      {unchecked.length > 0 && (
        <ul className="text-[11px] text-[var(--color-muted-foreground)]">
          {unchecked.map(s => <li key={s.label}>Not checked: {s.label} ({s.specReason})</li>)}
        </ul>
      )}
    </section>
  )
}
