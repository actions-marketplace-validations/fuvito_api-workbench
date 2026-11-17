import { useState } from 'react'
import { ChevronDown, ChevronRight, ShieldOff, AlertTriangle, Lock, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { QualityReport, QualityGap } from '@/lib/bridge'

const METHOD_COLOR: Record<string, string> = {
  GET: 'text-sky-400', POST: 'text-emerald-400', PUT: 'text-amber-400',
  PATCH: 'text-orange-400', DELETE: 'text-red-400', HEAD: 'text-purple-400',
  OPTIONS: 'text-pink-400',
}
const METHOD_BG: Record<string, string> = {
  GET: 'bg-sky-400/10', POST: 'bg-emerald-400/10', PUT: 'bg-amber-400/10',
  PATCH: 'bg-orange-400/10', DELETE: 'bg-red-400/10', HEAD: 'bg-purple-400/10',
  OPTIONS: 'bg-pink-400/10',
}

function parseOp(op: string): { method: string; path: string } {
  const idx = op.indexOf(' ')
  if (idx === -1) return { method: op, path: '' }
  return { method: op.slice(0, idx), path: op.slice(idx + 1) }
}

function OpPill({ op }: { op: string }) {
  const { method, path } = parseOp(op)
  return (
    <span className="inline-flex items-center gap-1.5 font-mono text-[11px]">
      <span className={cn(
        'px-1.5 py-0.5 rounded font-bold text-[10px]',
        METHOD_COLOR[method] ?? 'text-[var(--color-foreground)]',
        METHOD_BG[method] ?? '',
      )}>
        {method}
      </span>
      <span className="text-[var(--color-foreground)]">{path}</span>
    </span>
  )
}

function ErrorGapRow({ gap }: { gap: QualityGap }) {
  const { method, path } = parseOp(gap.operation)
  return (
    <div className="flex items-start gap-3 px-4 py-2 border-b border-[var(--color-border)] last:border-0 text-xs">
      <span className={cn(
        'font-mono font-bold w-16 shrink-0 px-1.5 py-0.5 rounded text-center text-[10px] mt-0.5',
        METHOD_COLOR[method] ?? 'text-[var(--color-foreground)]',
        METHOD_BG[method] ?? '',
      )}>
        {method}
      </span>
      <span className="font-mono text-[var(--color-foreground)] flex-1 min-w-0 truncate mt-0.5" title={path}>{path}</span>
      <span className="shrink-0 text-[var(--color-muted-foreground)] text-[10px] mt-0.5">
        Never triggered: {gap.missing.map(c => (
          <span key={c} className="inline-block mx-0.5 px-1 py-0.5 rounded bg-orange-500/10 text-orange-400 font-mono font-bold">{c}</span>
        ))}
      </span>
    </div>
  )
}

type SectionProps = {
  title: string
  hint: string
  count: number
  accentText: string
  accentBg: string
  accentBorder: string
  icon: React.ReactNode
  defaultOpen?: boolean
  children: React.ReactNode
}

function QualitySection({
  title, hint, count, accentText, accentBg, accentBorder,
  icon, defaultOpen = false, children,
}: SectionProps) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <section>
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        className={cn(
          'w-full px-4 py-2 border-b border-[var(--color-border)] flex items-center gap-2 transition-colors',
          accentBg,
          `hover:${accentBg.replace('/5', '/10')}`,
        )}
      >
        {open
          ? <ChevronDown size={11} className={cn('shrink-0', accentText)} />
          : <ChevronRight size={11} className={cn('shrink-0', accentText)} />}
        <span className={cn('shrink-0', accentText)}>{icon}</span>
        <span className={cn('text-[10px] font-semibold uppercase tracking-widest', accentText)}>{title}</span>
        <span className={cn('text-[10px] tabular-nums', accentText, 'opacity-60')}>{count}</span>
        {open && <span className="ml-auto text-[10px] text-[var(--color-muted-foreground)] italic">{hint}</span>}
      </button>
      {open && children}
    </section>
  )
}

type Props = {
  quality: QualityReport
  compact?: boolean
}

export function QualityPanel({ quality, compact = false }: Props) {
  const neverExercised   = quality.neverExercised   ?? []
  const happyPathOnly    = quality.happyPathOnly    ?? []
  const securedNeverAuthed = quality.securedNeverAuthed ?? []
  const errorsNeverTriggered = quality.errorsNeverTriggered ?? []

  const totalIssues = happyPathOnly.length + securedNeverAuthed.length + errorsNeverTriggered.length

  if (totalIssues === 0 && neverExercised.length === 0) {
    if (compact) return null
    return (
      <div className="px-6 py-4 border-t border-[var(--color-border)] flex items-center gap-2 text-xs text-emerald-400">
        <span className="font-semibold">Quality signals clear</span>
        <span className="text-[var(--color-muted-foreground)]">— all exercised paths have been tested across error and auth cases</span>
      </div>
    )
  }

  if (compact) {
    const parts: string[] = []
    if (neverExercised.length)      parts.push(`${neverExercised.length} never run`)
    if (happyPathOnly.length)       parts.push(`${happyPathOnly.length} happy-path only`)
    if (securedNeverAuthed.length)  parts.push(`${securedNeverAuthed.length} auth gap`)
    if (errorsNeverTriggered.length) parts.push(`${errorsNeverTriggered.length} error gap`)
    return (
      <p className="text-[10px] text-[var(--color-muted-foreground)]">
        Quality gaps: {parts.join(' · ')}
      </p>
    )
  }

  return (
    <div className="border-t border-[var(--color-border)]">
      <div className="px-4 py-2 bg-[var(--color-background)] border-b border-[var(--color-border)] flex items-center gap-2">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">Quality Signals</span>
        <span className="text-[10px] text-[var(--color-muted-foreground)] opacity-60">what have you NOT proven?</span>
        {totalIssues > 0 && (
          <span className="ml-auto text-[10px] font-semibold text-amber-400 tabular-nums">
            {totalIssues} gap{totalIssues !== 1 ? 's' : ''}
          </span>
        )}
      </div>

      <div className="divide-y divide-[var(--color-border)]">
        {happyPathOnly.length > 0 && (
          <QualitySection
            title="Happy-path only"
            hint="Only 2xx responses observed — error paths untested"
            count={happyPathOnly.length}
            accentText="text-amber-400"
            accentBg="bg-amber-500/5"
            accentBorder="border-amber-500/20"
            icon={<AlertTriangle size={10} />}
            defaultOpen
          >
            <ul className="py-1">
              {happyPathOnly.map(op => (
                <li key={op} className="flex items-center px-4 py-1.5 border-b border-[var(--color-border)] last:border-0">
                  <OpPill op={op} />
                </li>
              ))}
            </ul>
          </QualitySection>
        )}

        {errorsNeverTriggered.length > 0 && (
          <QualitySection
            title="Error responses never triggered"
            hint="Declared non-2xx codes never observed in any run"
            count={errorsNeverTriggered.length}
            accentText="text-orange-400"
            accentBg="bg-orange-500/5"
            accentBorder="border-orange-500/20"
            icon={<XCircle size={10} />}
            defaultOpen
          >
            {errorsNeverTriggered.map(gap => <ErrorGapRow key={gap.operation} gap={gap} />)}
          </QualitySection>
        )}

        {securedNeverAuthed.length > 0 && (
          <QualitySection
            title="Secured — never tested with auth"
            hint="Operations require auth but were always called without credentials"
            count={securedNeverAuthed.length}
            accentText="text-blue-400"
            accentBg="bg-blue-500/5"
            accentBorder="border-blue-500/20"
            icon={<Lock size={10} />}
            defaultOpen
          >
            <ul className="py-1">
              {securedNeverAuthed.map(op => (
                <li key={op} className="flex items-center px-4 py-1.5 border-b border-[var(--color-border)] last:border-0">
                  <OpPill op={op} />
                </li>
              ))}
            </ul>
          </QualitySection>
        )}

        {neverExercised.length > 0 && (
          <QualitySection
            title="Never exercised"
            hint="These operations have not been reached by any recorded run"
            count={neverExercised.length}
            accentText="text-red-400"
            accentBg="bg-red-500/5"
            accentBorder="border-red-500/20"
            icon={<ShieldOff size={10} />}
          >
            <ul className="py-1">
              {neverExercised.map(op => (
                <li key={op} className="flex items-center px-4 py-1.5 border-b border-[var(--color-border)] last:border-0">
                  <OpPill op={op} />
                </li>
              ))}
            </ul>
          </QualitySection>
        )}
      </div>
    </div>
  )
}
