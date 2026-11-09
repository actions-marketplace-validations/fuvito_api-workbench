import { useMemo, memo } from 'react'
import { cn } from '@/lib/utils'

type DiffLine = { type: 'equal' | 'add' | 'remove'; text: string }

function computeDiff(prev: string, curr: string): DiffLine[] | null {
  const a = prev.split('\n')
  const b = curr.split('\n')
  if (a.length + b.length > 4000) return null
  const m = a.length, n = b.length
  const dp = Array.from({ length: m + 1 }, () => new Int32Array(n + 1))
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      dp[i][j] = a[i - 1] === b[j - 1] ? dp[i - 1][j - 1] + 1 : Math.max(dp[i - 1][j], dp[i][j - 1])
    }
  }
  const result: DiffLine[] = []
  let i = m, j = n
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && a[i - 1] === b[j - 1]) {
      result.push({ type: 'equal', text: a[i - 1] }); i--; j--
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      result.push({ type: 'add', text: b[j - 1] }); j--
    } else {
      result.push({ type: 'remove', text: a[i - 1] }); i--
    }
  }
  return result.reverse()
}

export const DiffView = memo(function DiffView({ prev, curr }: { prev: string; curr: string }) {
  const diff = useMemo(() => computeDiff(prev, curr), [prev, curr])
  if (!diff) {
    return (
      <p className="text-xs text-[var(--color-muted-foreground)] italic">
        Bodies are too large to diff (combined &gt; 4000 lines).
      </p>
    )
  }
  const added = diff.filter(l => l.type === 'add').length
  const removed = diff.filter(l => l.type === 'remove').length
  return (
    <div className="text-xs font-mono select-text">
      <div className="flex gap-4 mb-2 text-[10px]">
        {added > 0 && <span className="text-emerald-400">+{added} added</span>}
        {removed > 0 && <span className="text-red-400">−{removed} removed</span>}
        {added === 0 && removed === 0 && <span className="text-[var(--color-muted-foreground)]">Bodies are identical</span>}
      </div>
      {diff.map((line, idx) => (
        <div
          key={idx}
          className={cn(
            'flex whitespace-pre leading-relaxed rounded-sm',
            line.type === 'add' && 'bg-emerald-500/10',
            line.type === 'remove' && 'bg-red-500/15',
          )}
        >
          <span className={cn(
            'select-none w-4 shrink-0 font-semibold',
            line.type === 'add' && 'text-emerald-400',
            line.type === 'remove' && 'text-red-400',
            line.type === 'equal' && 'opacity-20 text-[var(--color-foreground)]',
          )}>
            {line.type === 'add' ? '+' : line.type === 'remove' ? '−' : ' '}
          </span>
          <span className={cn(
            'flex-1 break-all',
            line.type === 'add' && 'text-emerald-300',
            line.type === 'remove' && 'text-red-400',
            line.type === 'equal' && 'text-[var(--color-foreground)] opacity-30',
          )}>
            {line.text}
          </span>
        </div>
      ))}
    </div>
  )
})
