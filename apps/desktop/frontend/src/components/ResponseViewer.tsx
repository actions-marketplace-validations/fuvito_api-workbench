import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { Search, X, Copy, Check, AlignLeft, Code2, ExternalLink, Trash2, GitCompare, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import { statusColor, formatSize, prettyBody } from '@/lib/httpUtils'
import { openInSystemViewer } from '@/lib/bridge'
import type { ResponseData } from '@/lib/bridge'
import { detectFormat, countMatches } from '@/lib/responseHighlight'
import { TimingBar, TimingDialog } from './ResponseTiming'
import { DiffView } from './ResponseDiffView'
import { BodyContent } from './ResponseBody'

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
  previousResponse?: ResponseData | null
  loading: boolean
  cachedAt?: string | null
  onClear?: () => void
  onCollapse?: () => void
}

export function ResponseViewer({ response, previousResponse, loading, cachedAt, onClear, onCollapse }: ResponseViewerProps) {
  const [activeTab, setActiveTab] = useState<'Body' | 'Headers'>('Body')
  const [showSearch, setShowSearch] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [activeMatch, setActiveMatch] = useState(0)
  const [showTimingDialog, setShowTimingDialog] = useState(false)
  const [rawMode, setRawMode] = useState(false)
  const [showDiff, setShowDiff] = useState(false)
  const [copied, setCopied] = useState(false)
  const searchRef = useRef<HTMLInputElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)

  const bodyText = useMemo(
    () => response && !response.error ? (rawMode ? response.body : prettyBody(response.body)) : '',
    [response, rawMode]
  )

  const bodyFormat = useMemo(() =>
    response && !response.error
      ? detectFormat(response.body, response.headers.find(h => h.name.toLowerCase() === 'content-type')?.value)
      : null,
    [response]
  )

  // Debounce search so highlight/count recomputation doesn't block every keystroke
  useEffect(() => {
    const id = setTimeout(() => setDebouncedSearch(searchQuery), 150)
    return () => clearTimeout(id)
  }, [searchQuery])

  const matchCount = useMemo(() => countMatches(bodyText, debouncedSearch), [bodyText, debouncedSearch])

  const canDiff = !!(
    response && !response.error &&
    previousResponse && !previousResponse.error &&
    response.body !== previousResponse.body
  )

  const prevBodyPretty = useMemo(
    () => (canDiff && previousResponse ? prettyBody(previousResponse.body) : ''),
    [canDiff, previousResponse]
  )

  async function handleCopy() {
    if (!bodyText) return
    try {
      await navigator.clipboard.writeText(bodyText)
      setCopied(true)
      setTimeout(() => setCopied(false), 2500)
    } catch { /* ignore */ }
  }

  async function handleOpenExternal() {
    if (!response?.body || !bodyFormat) return
    const displayBody = rawMode ? response.body : prettyBody(response.body)
    await openInSystemViewer(displayBody, bodyFormat)
  }

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

  // Reset search and diff when response changes
  // eslint-disable-next-line react/set-state-in-effect
  useEffect(() => { closeSearch(); setShowDiff(false) }, [response, closeSearch])

  // Scroll the active match into view whenever it changes
  useEffect(() => {
    if (!searchQuery || !contentRef.current) return
    const el = contentRef.current.querySelector<HTMLElement>(`mark[data-match="${activeMatch}"]`)
    el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  }, [activeMatch, searchQuery])

  function stepMatch(dir: 1 | -1) {
    if (!matchCount) return
    setActiveMatch(m => (m + dir + matchCount) % matchCount)
  }

  return (
    <div className="flex flex-col h-full border-l border-[var(--color-border)]">
      {/* Status bar */}
      <div className="flex items-center gap-3 px-3 py-2 border-b border-[var(--color-border)] shrink-0 flex-wrap">
        {onCollapse && (
          <button
            type="button"
            onClick={onCollapse}
            title="Collapse response panel"
            className="p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
          >
            <ChevronRight size={12} />
          </button>
        )}
        {response && !response.error && response.statusCode > 0 ? (
          <>
            <span data-testid="response-status" className={cn('text-sm font-mono font-bold', statusColor(response.statusCode))}>
              {response.statusCode}
            </span>
            <span className="text-xs text-[var(--color-muted-foreground)]">{response.status}</span>
            <span className="ml-auto text-xs font-mono text-[var(--color-muted-foreground)]">{response.durationMs} ms</span>
            <span className="text-xs font-mono text-[var(--color-muted-foreground)]">{formatSize(response.size)}</span>
            <button
              type="button"
              onClick={openSearch}
              title="Search body (Ctrl+F)"
              aria-label="Search body"
              className="p-1 rounded flex items-center gap-1 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
            >
              <Search size={12} />
              <span className="text-[10px] font-mono opacity-60 hidden sm:inline">Ctrl+F</span>
            </button>
            <button
              type="button"
              onClick={() => setRawMode(m => !m)}
              title={rawMode ? 'Switch to pretty view' : 'Switch to raw view'}
              aria-label={rawMode ? 'Switch to pretty view' : 'Switch to raw view'}
              className={cn('p-1 rounded transition-colors', rawMode ? 'text-[var(--color-accent)] bg-[var(--color-muted)]' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]')}
            >
              {rawMode ? <Code2 size={12} /> : <AlignLeft size={12} />}
            </button>
            {canDiff && (
              <button
                type="button"
                onClick={() => setShowDiff(d => !d)}
                title={showDiff ? 'Show response body' : 'Diff against previous response'}
                aria-label={showDiff ? 'Show response body' : 'Diff against previous response'}
                className={cn('p-1 rounded transition-colors', showDiff ? 'text-[var(--color-accent)] bg-[var(--color-muted)]' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]')}
              >
                <GitCompare size={12} />
              </button>
            )}
            <div className="relative">
              <button
                type="button"
                onClick={handleCopy}
                title={copied ? 'Copied!' : 'Copy response body'}
                aria-label="Copy response body"
                className="p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
              >
                {copied ? <Check size={12} className="text-emerald-400" /> : <Copy size={12} />}
              </button>
              {copied && (
                <span className="absolute -top-7 left-1/2 -translate-x-1/2 bg-[var(--color-foreground)] text-[var(--color-background)] text-[10px] font-semibold px-1.5 py-0.5 rounded whitespace-nowrap pointer-events-none">
                  Copied!
                </span>
              )}
            </div>
            <button
              type="button"
              onClick={handleOpenExternal}
              title="Open in system viewer"
              aria-label="Open in system viewer"
              className="p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
            >
              <ExternalLink size={12} />
            </button>
            {onClear && (
              <button
                type="button"
                onClick={onClear}
                title="Clear response"
                aria-label="Clear response"
                className="p-1 rounded text-[var(--color-muted-foreground)] hover:text-red-400 hover:bg-[var(--color-muted)] transition-colors"
              >
                <Trash2 size={12} />
              </button>
            )}
          </>
        ) : response?.error ? (
          <>
            <span className="text-xs font-semibold text-red-400">Network Error</span>
            {onClear && (
              <button
                type="button"
                onClick={onClear}
                title="Clear response"
                aria-label="Clear response"
                className="ml-auto p-1 rounded text-[var(--color-muted-foreground)] hover:text-red-400 hover:bg-[var(--color-muted)] transition-colors"
              >
                <Trash2 size={12} />
              </button>
            )}
          </>
        ) : (
          <span className="text-xs text-[var(--color-muted-foreground)]">Response</span>
        )}
        {response?.timing && (
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
      <div data-testid="response-body" ref={contentRef} className="flex-1 overflow-auto p-3">
        {loading && <p className="text-xs text-[var(--color-muted-foreground)] animate-pulse">Sending request…</p>}
        {!loading && !response && <p className="text-xs text-[var(--color-muted-foreground)]">Send a request to see the response.</p>}
        {!loading && response?.error && (
          <div className="text-xs text-red-400 font-mono whitespace-pre-wrap break-all">{response.error}</div>
        )}
        {!loading && response && !response.error && activeTab === 'Body' && !showDiff && (
          <BodyContent body={response.body} headers={response.headers} search={debouncedSearch} activeMatch={activeMatch} raw={rawMode} />
        )}
        {!loading && response && !response.error && activeTab === 'Body' && showDiff && (
          <DiffView prev={prevBodyPretty} curr={bodyText} />
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
