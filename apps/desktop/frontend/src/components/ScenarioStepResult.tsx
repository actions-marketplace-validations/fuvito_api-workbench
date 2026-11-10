import { useState } from 'react'
import type { ReactNode } from 'react'
import { Play, CheckCircle2, XCircle, ChevronDown, ChevronRight, Copy, EyeOff } from 'lucide-react'
import { cn } from '@/lib/utils'
import { METHOD_BADGE } from '@/lib/httpUtils'
import type { StepDetail, StepResult } from '@/lib/bridge'

const OPERATOR_LABEL: Record<string, string> = {
  eq: '=', ne: '≠', contains: '∋', matches: '~', gt: '>', lt: '<',
}

const sectionLabelCls = 'text-xs font-semibold uppercase tracking-wider text-[var(--color-muted-foreground)] mb-1.5'

function assertionSummary(a: { type: string; path?: string; operator?: string; expected?: unknown; schema?: string }): string {
  const op = OPERATOR_LABEL[a.operator ?? 'eq'] ?? a.operator ?? '='
  const exp = a.expected !== undefined && a.expected !== null ? String(a.expected) : ''
  if (a.type === 'status') return `status ${op} ${exp}`
  if (a.type === 'schema') return a.schema ? `schema: ${a.schema}` : 'schema'
  const path = a.path ? a.path.replace(/^\$\./, '') : ''
  return `${a.type === 'jsonpath' ? '$.'+path : path} ${op} ${exp}`.trim()
}

function tryPrettyJSON(s: string): string {
  try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s }
}

export function statusColor(code: number): string {
  if (code < 300) return 'text-emerald-400'
  if (code < 400) return 'text-blue-400'
  if (code < 500) return 'text-amber-400'
  return 'text-red-400'
}

function CollapsibleSection({ title, badge, defaultOpen = true, children }: {
  title: ReactNode
  badge?: ReactNode
  defaultOpen?: boolean
  children: ReactNode
}) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div className="border-b border-[var(--color-border)] last:border-0">
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center gap-2 px-3 py-2.5 text-left bg-[var(--color-muted)] hover:bg-[var(--color-border)] transition-colors"
      >
        {open
          ? <ChevronDown size={13} className="shrink-0 text-[var(--color-muted-foreground)]" />
          : <ChevronRight size={13} className="shrink-0 text-[var(--color-muted-foreground)]" />}
        <span className="flex-1 text-xs font-semibold uppercase tracking-wide text-[var(--color-foreground)]">{title}</span>
        {badge && <span className="shrink-0">{badge}</span>}
      </button>
      {open && <div className="px-4 pb-4 pt-2">{children}</div>}
    </div>
  )
}

export function StepDetailPanel({ step, result, onRunFrom }: { step: StepDetail; result: StepResult | null; onRunFrom?: () => void }) {
  const badge = step.method ? METHOD_BADGE[step.method] ?? '' : ''
  const bodyPreview = step.bodyContent ? tryPrettyJSON(step.bodyContent) : null
  const responseBodyFormatted = result?.responseBody ? tryPrettyJSON(result.responseBody) : null
  const contentType = result?.responseHeaders?.find(h => h.name.toLowerCase() === 'content-type')?.value?.split(';')[0]

  return (
    <div className="flex flex-col overflow-y-auto h-full">

      {/* Request */}
      <CollapsibleSection
        title={
          <span className="flex items-center gap-2 min-w-0">
            <span className="shrink-0">Request</span>
            {step.method && <span className={cn('font-mono font-bold text-xs shrink-0', badge)}>{step.method}</span>}
            {step.url && <span className="text-[var(--color-muted-foreground)] font-mono text-xs truncate">{step.url}</span>}
          </span>
        }
        badge={onRunFrom ? (
          <button
            type="button"
            onClick={e => { e.stopPropagation(); onRunFrom() }}
            title="Run scenario from this step"
            className="flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-semibold text-[var(--color-muted-foreground)] hover:text-[var(--color-accent)] hover:bg-[var(--color-accent)]/10 transition-colors"
          >
            <Play size={9} /> Run from here
          </button>
        ) : undefined}
        defaultOpen
      >
        {step.disabled && (
          <div className="flex items-center gap-1.5 mb-2 px-2 py-1 rounded bg-amber-500/10 border border-amber-500/20 text-xs text-amber-400">
            <EyeOff size={11} /> This step is disabled and will be skipped during runs.
          </div>
        )}
        {step.requestRef && (
          <p className="text-xs text-[var(--color-muted-foreground)] font-mono mb-2 break-all">↗ {step.requestRef}</p>
        )}
        {step.description && (
          <p className="text-xs text-[var(--color-muted-foreground)] italic mb-2">{step.description}</p>
        )}

        {(step.headers?.length ?? 0) > 0 && (
          <div className="mb-2">
            <p className={sectionLabelCls}>Headers</p>
            {step.headers!.map((h, i) => (
              <div key={i} className="flex items-baseline gap-1 text-xs font-mono mb-0.5">
                <span className="text-[var(--color-muted-foreground)] shrink-0">{h.name}:</span>
                <span className="text-[var(--color-foreground)] break-all">{h.value}</span>
              </div>
            ))}
          </div>
        )}

        {(step.queryParams?.length ?? 0) > 0 && (
          <div className="mb-2">
            <p className={sectionLabelCls}>Query Params</p>
            <div className="flex flex-wrap gap-1">
              {step.queryParams!.map((q, i) => (
                <span key={i} className="font-mono text-xs bg-[var(--color-muted)] border border-[var(--color-border)] rounded px-1.5 py-0.5 text-[var(--color-muted-foreground)]">
                  ?{q.name}={q.value}
                </span>
              ))}
            </div>
          </div>
        )}

        {step.bodyType && step.bodyType !== 'none' && bodyPreview && (
          <div className="mb-2">
            <p className={sectionLabelCls}>Body · {step.bodyType}</p>
            <pre className="text-xs font-mono text-[var(--color-foreground)] bg-[var(--color-background)] border border-[var(--color-border)] rounded px-2 py-1.5 overflow-auto max-h-32 whitespace-pre-wrap break-all">
              {bodyPreview}
            </pre>
          </div>
        )}

        {(step.assertions?.length ?? 0) > 0 && (
          <div className="mb-2">
            <p className={sectionLabelCls}>Assertions</p>
            <div className="flex flex-wrap gap-1">
              {step.assertions.map((a, i) => (
                <span key={i} className="rounded bg-[var(--color-muted)] border border-[var(--color-border)] px-1.5 py-0.5 text-xs text-[var(--color-muted-foreground)]">
                  {assertionSummary(a)}
                </span>
              ))}
            </div>
          </div>
        )}

        {(step.extractions?.length ?? 0) > 0 && (
          <div className="mb-2">
            <p className={sectionLabelCls}>Extractions</p>
            <div className="flex flex-wrap gap-1">
              {step.extractions.map((e, i) => (
                <span key={i} className="rounded bg-violet-500/10 border border-violet-500/20 px-1.5 py-0.5 text-xs text-violet-400">→ {e.name}</span>
              ))}
            </div>
          </div>
        )}

        {((step.failPolicy && step.failPolicy !== 'stop') || (step.timeoutMs && step.timeoutMs > 0) || (step.retryMaxAttempts && step.retryMaxAttempts > 0)) && (
          <div className="flex flex-wrap gap-3 text-xs text-[var(--color-muted-foreground)]">
            {step.failPolicy && step.failPolicy !== 'stop' && <span>on fail: {step.failPolicy}</span>}
            {step.timeoutMs && step.timeoutMs > 0 ? <span>⏱ {step.timeoutMs} ms</span> : null}
            {step.retryMaxAttempts && step.retryMaxAttempts > 0 ? <span>↺ {step.retryMaxAttempts}× · {step.retryDelayMs ?? 1000} ms delay</span> : null}
          </div>
        )}
      </CollapsibleSection>

      {/* Response */}
      {result?.responseBody && (
        <CollapsibleSection
          title="Response"
          badge={contentType ? <span className="text-xs text-[var(--color-muted-foreground)] font-mono">{contentType}</span> : undefined}
          defaultOpen={false}
        >
          {(result.responseHeaders?.length ?? 0) > 0 && (
            <div className="mb-3">
              <p className={sectionLabelCls}>Headers</p>
              <div className="max-h-28 overflow-y-auto flex flex-col gap-0.5">
                {result.responseHeaders!.map((h, i) => (
                  <div key={i} className="flex items-baseline gap-1 text-xs font-mono">
                    <span className="text-[var(--color-muted-foreground)] shrink-0">{h.name}:</span>
                    <span className="text-[var(--color-foreground)] break-all">{h.value}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
          <pre className="text-xs font-mono text-[var(--color-foreground)] bg-[var(--color-background)] border border-[var(--color-border)] rounded px-2 py-1.5 overflow-auto max-h-72 whitespace-pre-wrap break-all">
            {responseBodyFormatted}
          </pre>
        </CollapsibleSection>
      )}

      {/* Result */}
      {result && (
        <CollapsibleSection
          title="Result"
          badge={
            <span className="flex items-center gap-2 text-xs">
              {result.skipped
                ? <span className="text-[var(--color-muted-foreground)]">SKIPPED</span>
                : result.passed
                  ? <span className="font-semibold text-emerald-400">PASSED</span>
                  : <span className="font-semibold text-red-400">FAILED</span>}
              {result.statusCode > 0 && <span className={cn('font-mono font-bold', statusColor(result.statusCode))}>{result.statusCode}</span>}
              {result.durationMs > 0 && <span className="text-[var(--color-muted-foreground)]">{result.durationMs} ms</span>}
            </span>
          }
          defaultOpen
        >
          {result.error && <p className="text-red-400 text-xs font-mono break-all mb-2">{result.error}</p>}
          {result.skipped && <p className="text-xs text-[var(--color-muted-foreground)]">Skipped — previous step failed.</p>}
          <div className="flex flex-col gap-1.5">
            {(result.assertions ?? []).map((ar, i) => {
              const def = step.assertions?.[i]
              const label = def ? assertionSummary(def) : ar.message
              return (
                <div key={i} className="flex items-start gap-2">
                  {ar.passed
                    ? <CheckCircle2 size={14} className="shrink-0 text-emerald-400 mt-0.5" />
                    : <XCircle size={14} className="shrink-0 text-red-400 mt-0.5" />}
                  <div className="flex-1 min-w-0">
                    <span className={cn('text-xs font-mono', ar.passed ? 'text-[var(--color-foreground)]' : 'text-red-300')}>
                      {label}
                    </span>
                    {!ar.passed && ar.message && ar.message !== label && (
                      <p className="text-xs text-red-400 mt-0.5 break-all">{ar.message}</p>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
          {Object.entries(result.extracted ?? {}).length > 0 && (
            <div className="mt-3 flex flex-col gap-1">
              <p className={sectionLabelCls}>Extracted</p>
              {Object.entries(result.extracted ?? {}).map(([k, v]) => (
                <div key={k} className="flex items-center gap-1 text-xs group/extract">
                  <span className="text-violet-400 shrink-0">→ {k}:</span>
                  <span className="text-[var(--color-foreground)] font-mono break-all flex-1">{v}</span>
                  <button
                    type="button"
                    onClick={() => navigator.clipboard.writeText(v)}
                    title={`Copy value of ${k}`}
                    className="shrink-0 opacity-0 group-hover/extract:opacity-100 transition-opacity p-0.5 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
                  >
                    <Copy size={10} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </CollapsibleSection>
      )}
    </div>
  )
}
