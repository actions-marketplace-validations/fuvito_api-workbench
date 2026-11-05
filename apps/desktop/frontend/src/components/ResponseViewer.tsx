import { useState, useRef, useEffect, useCallback } from 'react'
import { Search, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { statusColor, formatSize, prettyBody } from '@/lib/httpUtils'
import type { ResponseData } from '@/lib/bridge'

// ---- Timing bar ----

type TimingSegment = { label: string; ms: number; color: string }

function TimingBar({ timing, total }: { timing: ResponseData['timing']; total: number }) {
  if (!timing || total === 0) return null
  const segments: TimingSegment[] = [
    { label: 'DNS',      ms: timing.dnsMs,      color: 'bg-purple-500' },
    { label: 'Connect',  ms: timing.connectMs,  color: 'bg-blue-500' },
    { label: 'TLS',      ms: timing.tlsMs,      color: 'bg-yellow-500' },
    { label: 'TTFB',     ms: timing.ttfbMs,     color: 'bg-emerald-500' },
    { label: 'Transfer', ms: timing.transferMs, color: 'bg-cyan-500' },
  ].filter(s => s.ms > 0)
  if (segments.length === 0) return null
  const sum = segments.reduce((a, s) => a + s.ms, 0)
  const scale = sum > 0 ? total / sum : 1
  return (
    <div className="w-full mt-1.5 flex flex-col gap-0.5">
      <div className="flex h-2 rounded overflow-hidden gap-px">
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
    </div>
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

function BodyContent({ body, headers, search, activeMatch }: {
  body: string
  headers: Array<{ name: string; value: string }>
  search: string
  activeMatch: number
}) {
  const ct = headers.find(h => h.name.toLowerCase() === 'content-type')?.value
  const pretty = prettyBody(body)
  const fmt = detectFormat(body, ct)

  let highlighted: React.ReactNode
  if (!search) {
    if (fmt === 'json' && body.length < 500_000) highlighted = highlightJSON(pretty)
    else if (fmt === 'xml' || fmt === 'html') highlighted = highlightXML(pretty)
    else highlighted = pretty
  } else {
    highlighted = highlightSearch(pretty, search, activeMatch)
  }

  return (
    <pre className="text-xs font-mono whitespace-pre-wrap break-all leading-relaxed" style={{ color: 'var(--color-foreground)' }}>
      {highlighted}
    </pre>
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
  const searchRef = useRef<HTMLInputElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)

  const bodyText = response && !response.error ? prettyBody(response.body) : ''
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
          </>
        ) : (
          <span className="text-xs text-[var(--color-muted-foreground)]">Response</span>
        )}
        {response?.timing && !response.error && <TimingBar timing={response.timing} total={response.durationMs} />}
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
          <BodyContent body={response.body} headers={response.headers} search={searchQuery} activeMatch={activeMatch} />
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
