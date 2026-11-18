import { useState } from 'react'
import type { ReactNode } from 'react'
import { ChevronDown, ChevronRight, Copy } from 'lucide-react'
import { cn } from '@/lib/utils'
import { METHOD_BADGE, statusColor } from '@/lib/httpUtils'
import type { StepResult } from '@/lib/bridge'

type Header = { name: string; value: string }

function tryPrettyJSON(s: string): string {
  try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s }
}

// Request and response get their own colour so they are easy to tell apart at a glance:
// blue for what was sent, green for what came back.
const TONES = {
  request: {
    frame: 'border-sky-500/40 bg-sky-500/5',
    head: 'bg-sky-500/15 hover:bg-sky-500/25',
    label: 'text-sky-400',
    bar: 'bg-sky-500',
  },
  response: {
    frame: 'border-emerald-500/40 bg-emerald-500/5',
    head: 'bg-emerald-500/15 hover:bg-emerald-500/25',
    label: 'text-emerald-400',
    bar: 'bg-emerald-500',
  },
} as const

function Section({ tone, title, summary, defaultOpen = true, children }: {
  tone: keyof typeof TONES
  title: string
  summary?: ReactNode
  defaultOpen?: boolean
  children: ReactNode
}) {
  const [open, setOpen] = useState(defaultOpen)
  const t = TONES[tone]
  return (
    <section aria-label={title} className={cn('rounded border overflow-hidden', t.frame)}>
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        aria-expanded={open}
        className={cn('w-full flex items-center gap-2 px-3 py-1.5 text-left transition-colors', t.head)}
      >
        <span className={cn('w-1 self-stretch rounded-full shrink-0', t.bar)} />
        {open ? <ChevronDown size={12} className={t.label} /> : <ChevronRight size={12} className={t.label} />}
        <span className={cn('text-[11px] font-semibold uppercase tracking-wider shrink-0', t.label)}>{title}</span>
        <span className="flex-1 min-w-0 flex items-center gap-2 text-xs overflow-hidden">{summary}</span>
      </button>
      {open && <div className="px-3 py-2 flex flex-col gap-2">{children}</div>}
    </section>
  )
}

function HeaderList({ headers, tone, defaultOpen }: { headers: Header[]; tone: keyof typeof TONES; defaultOpen: boolean }) {
  const [open, setOpen] = useState(defaultOpen)
  if (headers.length === 0) return null
  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        aria-expanded={open}
        className={cn('flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider', TONES[tone].label)}
      >
        {open ? <ChevronDown size={10} /> : <ChevronRight size={10} />}
        Headers ({headers.length})
      </button>
      {open && (
        <div className="mt-1 max-h-40 overflow-y-auto flex flex-col gap-0.5">
          {headers.map((h, i) => (
            <div key={i} className="flex items-baseline gap-1 text-xs font-mono">
              <span className="text-[var(--color-muted-foreground)] shrink-0">{h.name}:</span>
              <span className="text-[var(--color-foreground)] break-all">{h.value}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function BodyBlock({ text, tone }: { text: string; tone: keyof typeof TONES }) {
  const pretty = tryPrettyJSON(text)
  return (
    <div>
      <div className="flex items-center justify-between mb-0.5">
        <span className={cn('text-[10px] font-semibold uppercase tracking-wider', TONES[tone].label)}>Body</span>
        <button
          type="button"
          onClick={() => navigator.clipboard?.writeText(pretty)}
          title="Copy body"
          aria-label={`Copy ${tone} body`}
          className="p-0.5 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
        >
          <Copy size={10} />
        </button>
      </div>
      <pre className="text-xs font-mono text-[var(--color-foreground)] bg-[var(--color-background)] border border-[var(--color-border)] rounded px-2 py-1.5 overflow-auto max-h-72 whitespace-pre-wrap break-all">
        {pretty}
      </pre>
    </div>
  )
}

// The exchange of one step: the request as it was sent and the response that came back.
export function StepExchange({ result }: { result: StepResult }) {
  const reqHeaders = result.requestHeaders ?? []
  const resHeaders = result.responseHeaders ?? []
  const hasRequest = !!result.requestMethod || !!result.requestUrl
  const hasResponse = result.statusCode > 0 || !!result.responseBody || resHeaders.length > 0
  if (!hasRequest && !hasResponse) return null

  const contentType = resHeaders.find(h => h.name.toLowerCase() === 'content-type')?.value?.split(';')[0]

  return (
    <div className="flex flex-col gap-2" aria-label="Step request and response">
      {hasRequest && (
        <Section
          tone="request"
          title="Request"
          summary={
            <>
              {result.requestMethod && (
                <span className={cn('font-mono font-bold shrink-0', METHOD_BADGE[result.requestMethod] ?? '')}>{result.requestMethod}</span>
              )}
              <span className="font-mono text-[var(--color-muted-foreground)] truncate">{result.requestUrl}</span>
            </>
          }
        >
          <HeaderList headers={reqHeaders} tone="request" defaultOpen />
          {result.requestBody ? <BodyBlock text={result.requestBody} tone="request" /> : (
            <p className="text-xs text-[var(--color-muted-foreground)] italic">No body sent.</p>
          )}
        </Section>
      )}

      {hasResponse && (
        <Section
          tone="response"
          title="Response"
          summary={
            <>
              {result.statusCode > 0 && <span className={cn('font-mono font-bold shrink-0', statusColor(result.statusCode))}>{result.statusCode}</span>}
              {result.durationMs > 0 && <span className="text-[var(--color-muted-foreground)] shrink-0">{result.durationMs} ms</span>}
              {contentType && <span className="font-mono text-[var(--color-muted-foreground)] truncate">{contentType}</span>}
            </>
          }
        >
          <HeaderList headers={resHeaders} tone="response" defaultOpen={false} />
          {result.responseBody ? <BodyBlock text={result.responseBody} tone="response" /> : (
            <p className="text-xs text-[var(--color-muted-foreground)] italic">Empty response body.</p>
          )}
        </Section>
      )}
    </div>
  )
}
