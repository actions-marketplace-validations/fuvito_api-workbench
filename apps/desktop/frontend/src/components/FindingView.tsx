import type { ConformFinding } from '@/lib/bridge'
import { findingWhere } from '@/lib/conformHelpers'

// One explained violation: what the spec expected, what came back, where, why it fails and how to fix it.
// Shared by the Check Conformance page and the scenario result.
export function FindingView({ f }: { f: ConformFinding }) {
  return (
    <div className="rounded border border-[var(--color-border)] bg-[var(--color-background)] p-2.5">
      <p className="text-xs font-semibold text-[var(--color-foreground)]">{f.summary}</p>
      <dl className="mt-1.5 grid grid-cols-[5.5rem_1fr] gap-x-2 gap-y-1 text-xs">
        {f.expected && <><dt className="text-[var(--color-muted-foreground)]">Expected</dt><dd className="font-mono">{f.expected}</dd></>}
        {f.got && <><dt className="text-[var(--color-muted-foreground)]">Got</dt><dd className="font-mono break-all">{f.got}</dd></>}
        {findingWhere(f) && <><dt className="text-[var(--color-muted-foreground)]">Where</dt><dd data-testid="finding-where" className="font-mono break-all">{findingWhere(f)}</dd></>}
        {f.spec && <><dt className="text-[var(--color-muted-foreground)]">In the spec</dt><dd data-testid="finding-spec" className="font-mono break-all">{f.spec}</dd></>}
        <dt className="text-[var(--color-muted-foreground)]">Why</dt><dd>{f.why}</dd>
        <dt className="text-[var(--color-muted-foreground)]">Fix</dt><dd>{f.fix}</dd>
      </dl>
      {f.detail && (
        <details className="mt-1.5">
          <summary className="cursor-pointer text-[11px] text-[var(--color-muted-foreground)]">Technical detail</summary>
          <pre className="mt-1 whitespace-pre-wrap break-all font-mono text-[11px] text-[var(--color-muted-foreground)]">{f.detail}</pre>
        </details>
      )}
    </div>
  )
}
