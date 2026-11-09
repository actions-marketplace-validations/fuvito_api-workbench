import { useMemo, memo } from 'react'
import { prettyBody } from '@/lib/httpUtils'
import { detectFormat, highlightJSON, highlightXML, highlightSearch } from '@/lib/responseHighlight'

export const BodyContent = memo(function BodyContent({ body, headers, search, activeMatch, raw }: {
  body: string
  headers: Array<{ name: string; value: string }>
  search: string
  activeMatch: number
  raw?: boolean
}) {
  const ct = useMemo(
    () => headers.find(h => h.name.toLowerCase() === 'content-type')?.value,
    [headers]
  )

  const displayBody = useMemo(() => raw ? body : prettyBody(body), [body, raw])

  const fmt = useMemo(() => detectFormat(body, ct), [body, ct])

  const { lineCount, showLineNumbers, gutterWidth } = useMemo(() => {
    const ls = displayBody.split('\n')
    // Don't count a trailing empty entry from a trailing newline as an extra line
    const lc = ls.length > 1 && ls[ls.length - 1] === '' ? ls.length - 1 : ls.length
    return { lineCount: lc, showLineNumbers: lc > 1, gutterWidth: `${String(lc).length + 1}ch` }
  }, [displayBody])

  const highlighted = useMemo<React.ReactNode>(() => {
    if (!search) {
      if (!raw && fmt === 'json' && body.length < 500_000) return highlightJSON(displayBody)
      if (!raw && (fmt === 'xml' || fmt === 'html')) return highlightXML(displayBody)
      return displayBody
    }
    return highlightSearch(displayBody, search, activeMatch)
  }, [displayBody, search, activeMatch, raw, fmt, body.length]) // eslint-disable-line react-hooks/exhaustive-deps

  if (!showLineNumbers) {
    return (
      <pre className="text-xs font-mono whitespace-pre-wrap break-all leading-relaxed" style={{ color: 'var(--color-foreground)' }}>
        {highlighted}
      </pre>
    )
  }

  return (
    <div className="flex min-w-0">
      <pre
        aria-hidden
        className="text-xs font-mono leading-relaxed select-none shrink-0 text-right pr-3 border-r border-[var(--color-border)] mr-3 whitespace-pre"
        style={{ color: 'var(--color-muted-foreground)', minWidth: gutterWidth }}
      >
        {Array.from({ length: lineCount }, (_, i) => i + 1).join('\n')}
      </pre>
      <pre
        className="text-xs font-mono whitespace-pre leading-relaxed flex-1 min-w-0 overflow-x-auto"
        style={{ color: 'var(--color-foreground)' }}
      >
        {highlighted}
      </pre>
    </div>
  )
})
