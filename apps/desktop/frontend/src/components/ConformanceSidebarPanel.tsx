import { useEffect, useState } from 'react'
import { CheckCircle2, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { getConformContext } from '@/lib/bridge'
import type { ConformContext } from '@/lib/bridge'

// What the Check Conformance page last ran, kept by the app for this session.
export type ConformSummary = {
  specPath: string
  specTitle: string
  total: number
  covered: number
  passing: number
  failing: number
  failed: boolean
}

type Props = {
  workspacePath: string
  selectedSpec: string
  selectedEnv: string
  lastResult: ConformSummary | null
  // Each picks the input on the page and shows the page.
  onSelectSpec: (path: string) => void
  onSelectEnv: (path: string) => void
}

const heading = 'px-3 pt-3 pb-1 text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]'

function row(selected: boolean) {
  return cn(
    'w-full text-left px-3 py-1.5 truncate hover:bg-[var(--color-muted)]',
    selected && 'bg-[var(--color-muted)] border-l-2 border-[var(--color-accent)] pl-[10px]',
  )
}

// A faster way to pick the two inputs of the Check Conformance page; the page keeps every option.
export function ConformanceSidebarPanel({ workspacePath, selectedSpec, selectedEnv, lastResult, onSelectSpec, onSelectEnv }: Props) {
  const [ctx, setCtx] = useState<ConformContext | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let current = true
    getConformContext()
      .then(c => { if (current) { setCtx(c); setError('') } })
      .catch(e => { if (current) setError(String(e)) })
    return () => { current = false }
  }, [workspacePath])

  return (
    <div data-testid="sidebar-conformance" className="flex flex-col flex-1 overflow-auto text-xs">
      {error && <p role="alert" className="px-3 py-2 text-red-400">{error}</p>}
      {!ctx && !error && <p className="px-3 py-3 text-[var(--color-muted-foreground)]">Reading the workspace…</p>}

      {ctx && (
        <>
          <h3 className={heading}>Specs</h3>
          {ctx.specs.length === 0 ? (
            <p className="px-3 pb-2 text-[var(--color-muted-foreground)]">No OpenAPI spec was found in this workspace. Import one from the Requests panel.</p>
          ) : (
            <ul aria-label="Workspace specs">
              {ctx.specs.map(f => (
                <li key={f.path}>
                  <button type="button" title={f.path} aria-pressed={f.path === selectedSpec} onClick={() => onSelectSpec(f.path)} className={row(f.path === selectedSpec)}>
                    {f.label}
                  </button>
                </li>
              ))}
            </ul>
          )}

          <h3 className={heading}>Environments</h3>
          <ul aria-label="Workspace environments">
            <li>
              <button type="button" aria-pressed={selectedEnv === ''} onClick={() => onSelectEnv('')} className={row(selectedEnv === '')}>
                Active environment
              </button>
            </li>
            {ctx.environments.map(f => (
              <li key={f.path}>
                <button type="button" title={f.path} aria-pressed={f.path === selectedEnv} onClick={() => onSelectEnv(f.path)} className={row(f.path === selectedEnv)}>
                  {f.label}
                </button>
              </li>
            ))}
          </ul>

          {lastResult && (
            <>
              <h3 className={heading}>Last result</h3>
              <button
                type="button"
                data-testid="sidebar-last-result"
                onClick={() => onSelectSpec(lastResult.specPath)}
                className="mx-3 mb-3 flex items-start gap-1.5 rounded border border-[var(--color-border)] px-2 py-1.5 text-left hover:bg-[var(--color-muted)]"
              >
                {lastResult.failed
                  ? <XCircle size={13} className="shrink-0 mt-0.5 text-red-400" aria-label="failed" />
                  : <CheckCircle2 size={13} className="shrink-0 mt-0.5 text-emerald-400" aria-label="passed" />}
                <span>
                  <span className="block font-medium truncate">{lastResult.specTitle || 'Last run'}</span>
                  <span className="block text-[var(--color-muted-foreground)]">
                    Covered {lastResult.covered}, Passing {lastResult.passing}, Failing {lastResult.failing} of {lastResult.total}
                  </span>
                </span>
              </button>
            </>
          )}
        </>
      )}
    </div>
  )
}
