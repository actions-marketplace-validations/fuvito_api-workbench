import { useState, useEffect, useMemo, useCallback } from 'react'
import { Code2, Copy, Check, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TabState } from '@/lib/tabState'
import { buildCurl, buildFetch, buildPython, SNIPPET_FORMATS } from '@/lib/snippets'
import type { SnippetFormat } from '@/lib/snippets'

// ---- Syntax highlighting ----

type Span = { text: string; cls?: string }

function tokenize(code: string, lang: SnippetFormat): Span[] {
  const spans: Span[] = []

  if (lang === 'curl') {
    const re = /("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')|(--?[a-zA-Z][a-zA-Z-]*)|(curl\b)|(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)|(\\$)/gm
    let last = 0, m: RegExpExecArray | null
    while ((m = re.exec(code)) !== null) {
      if (m.index > last) spans.push({ text: code.slice(last, m.index) })
      if (m[1]) {
        const inner = m[1].slice(1, -1)
        spans.push({ text: m[1], cls: inner.startsWith('http') ? 'text-emerald-400' : 'text-amber-400' })
      } else if (m[2]) spans.push({ text: m[2], cls: 'text-purple-400' })
      else if (m[3]) spans.push({ text: m[3], cls: 'text-[var(--color-accent)]' })
      else if (m[4]) spans.push({ text: m[4], cls: 'text-cyan-400' })
      else if (m[5]) spans.push({ text: m[5], cls: 'text-[var(--color-muted-foreground)]' })
      last = m.index + m[0].length
    }
    if (last < code.length) spans.push({ text: code.slice(last) })
    return spans
  }

  if (lang === 'fetch') {
    const re = /('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")|\b(const|await|let|var|true|false|null|undefined)\b|\b(fetch|method|headers|body|JSON|stringify|response|json|data)\b/g
    let last = 0, m: RegExpExecArray | null
    while ((m = re.exec(code)) !== null) {
      if (m.index > last) spans.push({ text: code.slice(last, m.index) })
      if (m[1]) spans.push({ text: m[1], cls: 'text-amber-400' })
      else if (m[2]) spans.push({ text: m[2], cls: 'text-purple-400' })
      else if (m[3]) spans.push({ text: m[3], cls: 'text-blue-400' })
      last = m.index + m[0].length
    }
    if (last < code.length) spans.push({ text: code.slice(last) })
    return spans
  }

  // python
  const re = /('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")|\b(import|True|False|None|raise_for_status)\b|\b(requests)\b|(\b[a-z_]+(?=\s*=\s*(?!>))(?=.*\())/g
  let last = 0, m: RegExpExecArray | null
  while ((m = re.exec(code)) !== null) {
    if (m.index > last) spans.push({ text: code.slice(last, m.index) })
    if (m[1]) spans.push({ text: m[1], cls: 'text-amber-400' })
    else if (m[2]) spans.push({ text: m[2], cls: 'text-purple-400' })
    else if (m[3]) spans.push({ text: m[3], cls: 'text-blue-400' })
    else if (m[4]) spans.push({ text: m[4], cls: 'text-emerald-400' })
    last = m.index + m[0].length
  }
  if (last < code.length) spans.push({ text: code.slice(last) })
  return spans
}

function HighlightedCode({ code, lang }: { code: string; lang: SnippetFormat }) {
  const spans = useMemo(() => tokenize(code, lang), [code, lang])
  return (
    <code>
      {spans.map((s, i) =>
        s.cls ? <span key={i} className={s.cls}>{s.text}</span> : s.text,
      )}
    </code>
  )
}

// ---- Dialog ----

export function SnippetButton({ tab }: { tab: TabState }) {
  const [open, setOpen] = useState(false)
  const [format, setFormat] = useState<SnippetFormat>('curl')
  const [copied, setCopied] = useState(false)

  const snippet = useMemo(
    () => format === 'curl' ? buildCurl(tab) : format === 'fetch' ? buildFetch(tab) : buildPython(tab),
    [format, tab],
  )

  const copyToClipboard = useCallback((text: string) => {
    navigator.clipboard.writeText(text).catch(() => {})
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open])

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        title="Copy as code snippet"
        className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
      >
        <Code2 size={13} />
      </button>

      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
          onClick={e => { if (e.target === e.currentTarget) setOpen(false) }}
        >
          <div className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-xl shadow-2xl w-[640px] max-h-[80vh] flex flex-col">

            {/* Header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border)]">
              <span className="text-sm font-semibold text-[var(--color-foreground)]">Code snippet</span>
              <button
                type="button"
                onClick={() => setOpen(false)}
                className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                <X size={15} />
              </button>
            </div>

            {/* Format tabs */}
            <div className="flex border-b border-[var(--color-border)] px-4 gap-0">
              {SNIPPET_FORMATS.map(f => (
                <button
                  key={f.key}
                  type="button"
                  onClick={() => setFormat(f.key)}
                  className={cn(
                    'px-3 py-2 text-xs font-medium border-b-2 transition-colors',
                    format === f.key
                      ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                      : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
                  )}
                >
                  {f.label}
                </button>
              ))}
            </div>

            {/* Code */}
            <div className="flex-1 overflow-auto p-4 min-h-0">
              <pre className="text-xs font-mono leading-relaxed text-[var(--color-foreground)] whitespace-pre-wrap break-words">
                <HighlightedCode code={snippet} lang={format} />
              </pre>
            </div>

            {/* Footer */}
            <div className="flex items-center justify-end gap-2 px-4 py-3 border-t border-[var(--color-border)]">
              <span className={cn('text-xs transition-opacity duration-300', copied ? 'text-emerald-400 opacity-100' : 'opacity-0')}>
                Copied to clipboard
              </span>
              <button
                type="button"
                onClick={() => copyToClipboard(snippet)}
                className={cn(
                  'flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors',
                  copied
                    ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                    : 'bg-[var(--color-accent)] text-white hover:opacity-90',
                )}
              >
                {copied ? <Check size={12} /> : <Copy size={12} />}
                {copied ? 'Copied!' : 'Copy'}
              </button>
            </div>

          </div>
        </div>
      )}
    </>
  )
}
