import { useState, useEffect, useRef } from 'react'
import { Code2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TabState } from '@/lib/tabState'
import { buildCurl, buildFetch, buildPython, SNIPPET_FORMATS } from '@/lib/snippets'
import type { SnippetFormat } from '@/lib/snippets'

export function SnippetButton({ tab }: { tab: TabState }) {
  const [open, setOpen] = useState(false)
  const [copied, setCopied] = useState<SnippetFormat | null>(null)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  async function copy(fmt: SnippetFormat) {
    const snippet = fmt === 'curl' ? buildCurl(tab) : fmt === 'fetch' ? buildFetch(tab) : buildPython(tab)
    await navigator.clipboard.writeText(snippet)
    setCopied(fmt)
    setTimeout(() => { setCopied(null); setOpen(false) }, 1400)
  }

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        title="Copy as code snippet"
        className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
      >
        <Code2 size={13} />
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-50 bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg shadow-xl py-1 min-w-[200px]">
          {SNIPPET_FORMATS.map(f => (
            <button
              key={f.key}
              type="button"
              onClick={() => copy(f.key)}
              className={cn(
                'w-full text-left px-3 py-1.5 text-xs flex items-center justify-between gap-4 transition-colors',
                'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]',
              )}
            >
              <span>{f.label}</span>
              {copied === f.key && <span className="text-emerald-400 text-[10px] shrink-0 font-medium">Copied!</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
