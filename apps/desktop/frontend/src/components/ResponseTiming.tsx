import { useEffect, useMemo } from 'react'
import { Clock, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ResponseData } from '@/lib/bridge'

// ---- Timing bar ----

type TimingSegment = { label: string; ms: number; color: string; barColor: string }

const TIMING_SEGMENTS: Array<{ key: keyof NonNullable<ResponseData['timing']>; label: string; color: string; barColor: string }> = [
  { key: 'dnsMs',      label: 'DNS',      color: 'bg-purple-500',  barColor: 'bg-purple-500' },
  { key: 'connectMs',  label: 'Connect',  color: 'bg-blue-500',    barColor: 'bg-blue-500'   },
  { key: 'tlsMs',      label: 'TLS',      color: 'bg-yellow-500',  barColor: 'bg-yellow-500' },
  { key: 'ttfbMs',     label: 'TTFB',     color: 'bg-emerald-500', barColor: 'bg-emerald-500'},
  { key: 'transferMs', label: 'Transfer', color: 'bg-cyan-500',    barColor: 'bg-cyan-500'   },
]

function buildSegments(timing: NonNullable<ResponseData['timing']>): TimingSegment[] {
  return TIMING_SEGMENTS
    .map(({ key, label, color, barColor }) => ({ label, ms: timing[key], color, barColor }))
    .filter(s => s.ms > 0)
}

// ---- Timing detail dialog ----

export function TimingDialog({ timing, total, onClose }: {
  timing: NonNullable<ResponseData['timing']>
  total: number
  onClose: () => void
}) {
  const segments = buildSegments(timing)
  const sum = segments.reduce((a, s) => a + s.ms, 0)
  const denom = Math.max(sum, 1)

  useEffect(() => {
    function onKey(e: KeyboardEvent) { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-xl shadow-2xl w-[480px] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3 border-b border-[var(--color-border)]">
          <div className="flex items-center gap-2">
            <Clock size={14} className="text-[var(--color-muted-foreground)]" />
            <span className="text-sm font-semibold text-[var(--color-foreground)]">Request Timing</span>
          </div>
          <button type="button" onClick={onClose} className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
            <X size={15} />
          </button>
        </div>

        {/* Rows */}
        <div className="flex flex-col gap-3 p-5">
          {segments.map(s => {
            const pct = (s.ms / denom) * 100
            return (
              <div key={s.label} className="flex flex-col gap-1">
                <div className="flex items-center justify-between text-xs">
                  <div className="flex items-center gap-2">
                    <span className={cn('inline-block w-2.5 h-2.5 rounded-sm shrink-0', s.color)} />
                    <span className="font-medium text-[var(--color-foreground)] w-14">{s.label}</span>
                  </div>
                  <div className="flex items-center gap-4 font-mono text-[var(--color-muted-foreground)]">
                    <span className="w-16 text-right">{s.ms} ms</span>
                    <span className="w-12 text-right">{pct.toFixed(1)}%</span>
                  </div>
                </div>
                <div className="h-2 w-full rounded bg-[var(--color-muted)] overflow-hidden">
                  <div
                    className={cn('h-full rounded transition-all', s.barColor)}
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            )
          })}

          {/* Divider + total */}
          <div className="border-t border-[var(--color-border)] pt-3 flex items-center justify-between text-xs font-mono">
            <span className="text-[var(--color-muted-foreground)] font-sans font-semibold uppercase tracking-wider text-[10px]">Total</span>
            <div className="flex items-center gap-4 text-[var(--color-foreground)] font-semibold">
              <span className="w-16 text-right">{total} ms</span>
              <span className="w-12 text-right text-[var(--color-muted-foreground)]">100%</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// ---- Timing bar (mini, clickable) ----

export function TimingBar({ timing, total, onClick }: { timing: ResponseData['timing']; total: number; onClick: () => void }) {
  const segments = useMemo(() => (timing ? buildSegments(timing) : []), [timing])
  if (!timing || total === 0) return null
  if (segments.length === 0) {
    return <p className="text-xs text-[var(--color-muted-foreground)] mt-1 italic">No timing breakdown available.</p>
  }
  const sum = segments.reduce((a, s) => a + s.ms, 0)
  // Compute proportional widths, enforce 1% minimum, re-normalise, then round so total stays at 100%.
  const raw = segments.map(s => (s.ms / (sum || 1)) * 100)
  const withMin = raw.map(w => Math.max(w, 1))
  const inflated = withMin.reduce((a, b) => a + b, 0)
  const widths = withMin.map((w, i, arr) => {
    const rounded = Math.round((w / inflated) * 100 * 10) / 10
    // Assign last segment the remainder so widths sum to exactly 100
    if (i === arr.length - 1) {
      const sumSoFar = arr.slice(0, -1).reduce((a, b) => a + Math.round((b / inflated) * 100 * 10) / 10, 0)
      return Math.max(0, 100 - sumSoFar)
    }
    return rounded
  })
  return (
    <button
      type="button"
      onClick={onClick}
      title="Click to view timing breakdown"
      className="w-full mt-1.5 flex flex-col gap-0.5 group cursor-pointer"
    >
      <div className="flex h-2 rounded overflow-hidden gap-px group-hover:opacity-80 transition-opacity">
        {segments.map((s, i) => (
          <div key={s.label} title={`${s.label}: ${s.ms}ms`} className={cn('h-full', s.color)}
            style={{ width: `${widths[i]}%` }} />
        ))}
      </div>
      <div className="flex gap-3 flex-wrap">
        {segments.map(s => (
          <span key={s.label} className="flex items-center gap-1 text-label text-[var(--color-muted-foreground)]">
            <span className={cn('inline-block rounded-sm shrink-0', s.color)} style={{ width: 'calc(var(--font-size-code) * 0.8)', height: 'calc(var(--font-size-code) * 0.8)' }} />
            {s.label} {s.ms}ms
          </span>
        ))}
      </div>
    </button>
  )
}
