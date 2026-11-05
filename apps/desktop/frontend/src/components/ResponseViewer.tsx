import { useState, useRef, useEffect, useCallback } from 'react'
import { Search, X, Clock, Copy, Check, AlignLeft, Code2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { statusColor, formatSize, prettyBody } from '@/lib/httpUtils'
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

function TimingDialog({ timing, total, onClose }: {
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

function TimingBar({ timing, total, onClick }: { timing: ResponseData['timing']; total: number; onClick: () => void }) {
  if (!timing || total === 0) return null
  const segments = buildSegments(timing)
  if (segments.length === 0) return null
  const sum = segments.reduce((a, s) => a + s.ms, 0)
  const scale = sum > 0 ? total / sum : 1
  return (
    <button
      type="button"
      onClick={onClick}
      title="Click to view timing breakdown"
      className="w-full mt-1.5 flex flex-col gap-0.5 group cursor-pointer"
    >
      <div className="flex h-2 rounded overflow-hidden gap-px group-hover:opacity-80 transition-opacity">
        {segments.map(s => (
          <div key={s.label} title={`${s.label}: ${s.ms}ms`} className={cn('h-full', s.color)}
            style={{ width: `${Math.max((s.ms / (total || sum)) * 100 * scale, 1)}%` }} />
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

// ---- Syntax highlighting ----

function highlightJSON(json: string): React.ReactNode {
  const nodes: React.ReactNode[] = []
  const re = /("(?:[^"\\]|\\.)*")(?=\s*:)|("(?:[^"\\]|\\.)*")|([-]?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|(true|false|null)|([{}\[\],:])/g
  let last = 0, i = 0
  for (const m of json.matchAll(re)) {
    if (m.index > last) nodes.push(json.slice(last, m.index))
    if      (m[1]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-key)' }}>{m[1]}</span>)
    else if (m[2]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-string)' }}>{m[2]}</span>)
    else if (m[3]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-number)' }}>{m[3]}</span>)
    else if (m[4]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-literal)' }}>{m[4]}</span>)
    else if (m[5]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-punct)' }}>{m[5]}</span>)
    last = m.index + m[0].length
  }
  if (last < json.length) nodes.push(json.slice(last))
  return nodes
}

function highlightXML(xml: string): React.ReactNode {
  const nodes: React.ReactNode[] = []
  // Order: comments, CDATA, tags (with attrs), entities
  const re = /(<!--[\s\S]*?-->)|(<!\[CDATA\[[\s\S]*?\]\]>)|(<\/?)([a-zA-Z][^\s>/]*)([^>]*)(\/?>)|(&#?[a-zA-Z0-9]+;)/g
  let last = 0, i = 0
  for (const m of xml.matchAll(re)) {
    if (m.index > last) nodes.push(xml.slice(last, m.index))
    if (m[1]) {
      nodes.push(<span key={i++} className="text-[var(--color-muted-foreground)] italic">{m[1]}</span>)
    } else if (m[2]) {
      nodes.push(<span key={i++} className="text-amber-400">{m[2]}</span>)
    } else if (m[4]) {
      // tag: highlight open/close bracket, tag name, attributes, close bracket
      const attrs = m[5] ?? ''
      const attrNodes: React.ReactNode[] = []
      const attrRe = /\s+([a-zA-Z][^\s=]*)(?:=(?:"([^"]*)"|'([^']*)'|([^\s>]*)))?/g
      let al = 0, ai = 0
      for (const am of attrs.matchAll(attrRe)) {
        if (am.index > al) attrNodes.push(attrs.slice(al, am.index))
        attrNodes.push(
          <span key={ai++}>{' '}<span className="text-emerald-400">{am[1]}</span>
          {am[0].includes('=') && <>{'='}<span className="text-amber-400">{`"${am[2] ?? am[3] ?? am[4] ?? ''}"`}</span></>}</span>
        )
        al = am.index + am[0].length
      }
      if (al < attrs.length) attrNodes.push(attrs.slice(al))
      nodes.push(
        <span key={i++}>
          <span className="text-[var(--color-muted-foreground)]">{m[3]}</span>
          <span className="text-blue-400">{m[4]}</span>
          {attrNodes}
          <span className="text-[var(--color-muted-foreground)]">{m[6]}</span>
        </span>
      )
    } else if (m[7]) {
      nodes.push(<span key={i++} className="text-purple-400">{m[7]}</span>)
    }
    last = m.index + m[0].length
  }
  if (last < xml.length) nodes.push(xml.slice(last))
  return nodes
}

type BodyFormat = 'json' | 'xml' | 'html' | 'plain'

function detectFormat(body: string, contentType?: string): BodyFormat {
  const ct = contentType?.toLowerCase() ?? ''
  const t = body.trim()
  if (ct.includes('json') || (!ct && (t.startsWith('{') || t.startsWith('[')))) {
    try { JSON.parse(body); return 'json' } catch { /* fall through */ }
  }
  if (ct.includes('xml') || ct.includes('html') || (!ct && t.startsWith('<'))) {
    return ct.includes('html') ? 'html' : 'xml'
  }
  return 'plain'
}

// ---- Search highlight overlay ----

function highlightSearch(text: string, query: string, activeIdx: number): React.ReactNode {
  if (!query) return text
  const nodes: React.ReactNode[] = []
  const lower = text.toLowerCase()
  const q = query.toLowerCase()
  let pos = 0, matchIdx = 0
  while (pos < text.length) {
    const idx = lower.indexOf(q, pos)
    if (idx === -1) { nodes.push(text.slice(pos)); break }
    if (idx > pos) nodes.push(text.slice(pos, idx))
    nodes.push(
      <mark key={matchIdx} className={cn('rounded-sm', matchIdx === activeIdx ? 'bg-orange-400 text-black' : 'bg-yellow-300/60 text-inherit')}>
        {text.slice(idx, idx + q.length)}
      </mark>
    )
    matchIdx++
    pos = idx + q.length
  }
  return nodes
}

function countMatches(text: string, query: string): number {
  if (!query) return 0
  let count = 0, pos = 0
  const lower = text.toLowerCase(), q = query.toLowerCase()
  while ((pos = lower.indexOf(q, pos)) !== -1) { count++; pos += q.length }
  return count
}

// ---- Body component ----

function BodyContent({ body, headers, search, activeMatch, raw }: {
  body: string
  headers: Array<{ name: string; value: string }>
  search: string
  activeMatch: number
  raw?: boolean
}) {
  const ct = headers.find(h => h.name.toLowerCase() === 'content-type')?.value
  const displayBody = raw ? body : prettyBody(body)
  const fmt = detectFormat(body, ct)

  const lines = displayBody.split('\n')
  // Don't count a trailing empty entry from a trailing newline as an extra line
  const lineCount = lines.length > 1 && lines[lines.length - 1] === '' ? lines.length - 1 : lines.length
  const showLineNumbers = lineCount > 1
  const gutterWidth = `${String(lineCount).length + 1}ch`

  let highlighted: React.ReactNode
  if (!search) {
    if (!raw && fmt === 'json' && body.length < 500_000) highlighted = highlightJSON(displayBody)
    else if (!raw && (fmt === 'xml' || fmt === 'html')) highlighted = highlightXML(displayBody)
    else highlighted = displayBody
  } else {
    highlighted = highlightSearch(displayBody, search, activeMatch)
  }

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
}

// ---- Time ago ----

function timeAgo(iso: string): string {
  const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (s < 60) return 'just now'
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

// ---- Main component ----

type ResponseViewerProps = {
  response: ResponseData | null
  loading: boolean
  cachedAt?: string | null
}

export function ResponseViewer({ response, loading, cachedAt }: ResponseViewerProps) {
  const [activeTab, setActiveTab] = useState<'Body' | 'Headers'>('Body')
  const [showSearch, setShowSearch] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [activeMatch, setActiveMatch] = useState(0)
  const [showTimingDialog, setShowTimingDialog] = useState(false)
  const [rawMode, setRawMode] = useState(false)
  const [copied, setCopied] = useState(false)
  const searchRef = useRef<HTMLInputElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)

  const bodyText = response && !response.error ? (rawMode ? response.body : prettyBody(response.body)) : ''

  async function handleCopy() {
    if (!response?.body) return
    try {
      await navigator.clipboard.writeText(response.body)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch { /* ignore */ }
  }
  const matchCount = countMatches(bodyText, searchQuery)

  const openSearch = useCallback(() => {
    setShowSearch(true)
    setActiveMatch(0)
    setTimeout(() => searchRef.current?.focus(), 0)
  }, [])

  const closeSearch = useCallback(() => {
    setShowSearch(false)
    setSearchQuery('')
    setActiveMatch(0)
  }, [])

  // Keyboard shortcut: Ctrl+F inside the response panel
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === 'f' && response && !response.error) {
        e.preventDefault()
        openSearch()
      }
      if (e.key === 'Escape' && showSearch) closeSearch()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [response, showSearch, openSearch, closeSearch])

  // Reset search when response changes
  useEffect(() => { closeSearch() }, [response, closeSearch])

  function stepMatch(dir: 1 | -1) {
    if (!matchCount) return
    setActiveMatch(m => (m + dir + matchCount) % matchCount)
  }

  return (
    <div className="flex flex-col h-full border-l border-[var(--color-border)]">
      {/* Status bar */}
      <div className="flex items-center gap-3 px-3 py-2 border-b border-[var(--color-border)] shrink-0 flex-wrap">
        {response && !response.error && response.statusCode > 0 ? (
          <>
            <span className={cn('text-xs font-mono font-semibold', statusColor(response.statusCode))}>
              {response.statusCode}
            </span>
            <span className="text-xs text-[var(--color-muted-foreground)]">{response.status}</span>
            <span className="ml-auto text-xs text-[var(--color-muted-foreground)]">{response.durationMs} ms</span>
            <span className="text-xs text-[var(--color-muted-foreground)]">{formatSize(response.size)}</span>
            <button
              type="button"
              onClick={openSearch}
              title="Search body (Ctrl+F)"
              className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
            >
              <Search size={12} />
            </button>
            <button
              type="button"
              onClick={() => setRawMode(m => !m)}
              title={rawMode ? 'Switch to pretty view' : 'Switch to raw view'}
              className={cn('transition-colors', rawMode ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}
            >
              {rawMode ? <Code2 size={12} /> : <AlignLeft size={12} />}
            </button>
            <button
              type="button"
              onClick={handleCopy}
              title="Copy response body"
              className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
            >
              {copied ? <Check size={12} className="text-emerald-400" /> : <Copy size={12} />}
            </button>
          </>
        ) : response?.error ? (
          <span className="text-xs font-semibold text-red-400">Network Error</span>
        ) : (
          <span className="text-xs text-[var(--color-muted-foreground)]">Response</span>
        )}
        {response?.timing && !response.error && (
          <TimingBar timing={response.timing} total={response.durationMs} onClick={() => setShowTimingDialog(true)} />
        )}
        {showTimingDialog && response?.timing && (
          <TimingDialog timing={response.timing} total={response.durationMs} onClose={() => setShowTimingDialog(false)} />
        )}
        {cachedAt && !loading && (
          <span className="shrink-0 text-[10px] text-[var(--color-muted-foreground)] italic" title={new Date(cachedAt).toLocaleString()}>
            Last run {timeAgo(cachedAt)}
          </span>
        )}
      </div>

      {/* Find bar */}
      {showSearch && (
        <div className="flex items-center gap-2 px-3 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-muted)] shrink-0">
          <Search size={11} className="text-[var(--color-muted-foreground)] shrink-0" />
          <input
            ref={searchRef}
            type="text"
            value={searchQuery}
            onChange={e => { setSearchQuery(e.target.value); setActiveMatch(0) }}
            onKeyDown={e => {
              if (e.key === 'Enter') stepMatch(e.shiftKey ? -1 : 1)
              if (e.key === 'Escape') closeSearch()
            }}
            placeholder="Find in response…"
            className="flex-1 bg-transparent text-xs font-mono outline-none text-[var(--color-foreground)] placeholder:text-[var(--color-muted-foreground)]"
          />
          {searchQuery && (
            <span className="text-[10px] text-[var(--color-muted-foreground)] shrink-0">
              {matchCount === 0 ? 'No results' : `${activeMatch + 1} / ${matchCount}`}
            </span>
          )}
          <button type="button" onClick={() => stepMatch(-1)} className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] text-xs px-1 disabled:opacity-30" disabled={!matchCount}>↑</button>
          <button type="button" onClick={() => stepMatch(1)}  className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] text-xs px-1 disabled:opacity-30" disabled={!matchCount}>↓</button>
          <button type="button" onClick={closeSearch} className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"><X size={11} /></button>
        </div>
      )}

      {/* Tabs */}
      {response && !response.error && (
        <div className="flex border-b border-[var(--color-border)] px-3 shrink-0">
          {(['Body', 'Headers'] as const).map(tab => (
            <button key={tab} type="button" onClick={() => setActiveTab(tab)}
              className={cn('px-3 py-1.5 text-xs border-b-2 transition-colors',
                activeTab === tab
                  ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                  : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}>
              {tab}
            </button>
          ))}
        </div>
      )}

      {/* Content */}
      <div ref={contentRef} className="flex-1 overflow-auto p-3">
        {loading && <p className="text-xs text-[var(--color-muted-foreground)] animate-pulse">Sending request…</p>}
        {!loading && !response && <p className="text-xs text-[var(--color-muted-foreground)]">Send a request to see the response.</p>}
        {!loading && response?.error && (
          <div className="text-xs text-red-400 font-mono whitespace-pre-wrap break-all">{response.error}</div>
        )}
        {!loading && response && !response.error && activeTab === 'Body' && (
          <BodyContent body={response.body} headers={response.headers} search={searchQuery} activeMatch={activeMatch} raw={rawMode} />
        )}
        {!loading && response && !response.error && activeTab === 'Headers' && (
          <table className="w-full text-xs font-mono">
            <tbody>
              {response.headers.map((h, i) => (
                <tr key={i} className="border-b border-[var(--color-border)]">
                  <td className="py-1 pr-4 text-[var(--color-muted-foreground)] whitespace-nowrap">{h.name}</td>
                  <td className="py-1 text-[var(--color-foreground)] break-all">{h.value}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
