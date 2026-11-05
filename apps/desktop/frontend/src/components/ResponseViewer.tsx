import { useState } from 'react'
import { cn } from '@/lib/utils'
import { statusColor, formatSize, prettyBody } from '@/lib/httpUtils'
import type { ResponseData } from '@/lib/bridge'

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
          <div
            key={s.label}
            title={`${s.label}: ${s.ms}ms`}
            className={cn('h-full', s.color)}
            style={{ width: `${Math.max((s.ms / (total || sum)) * 100 * scale, 1)}%` }}
          />
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

function highlightJSON(json: string): React.ReactNode {
  const nodes: React.ReactNode[] = []
  // Order matters: key (string + lookahead colon) must come before plain string
  const re = /("(?:[^"\\]|\\.)*")(?=\s*:)|("(?:[^"\\]|\\.)*")|([-]?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|(true|false|null)|([{}\[\],:])/g
  let last = 0
  let i = 0
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

function JsonBody({ body }: { body: string }) {
  const trimmed = body.trim()
  const looksLikeJson = trimmed.startsWith('{') || trimmed.startsWith('[')
  let isJson = false
  if (looksLikeJson) try { JSON.parse(body); isJson = true } catch { /* plain text */ }

  const content = isJson && body.length < 500_000
    ? highlightJSON(prettyBody(body))
    : prettyBody(body)

  return (
    <pre className="text-xs font-mono whitespace-pre-wrap break-all leading-relaxed" style={{ color: 'var(--color-foreground)' }}>
      {content}
    </pre>
  )
}

function timeAgo(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime()
  const s = Math.floor(diffMs / 1000)
  if (s < 60) return 'just now'
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

type ResponseViewerProps = {
  response: ResponseData | null
  loading: boolean
  cachedAt?: string | null
}

export function ResponseViewer({ response, loading, cachedAt }: ResponseViewerProps) {
  const [activeTab, setActiveTab] = useState<'Body' | 'Headers'>('Body')

  return (
    <div className="flex flex-col h-full border-l border-[var(--color-border)]">
      {/* Status bar */}
      <div className="flex items-center gap-3 px-3 py-2 border-b border-[var(--color-border)] shrink-0">
        {response && !response.error && response.statusCode > 0 ? (
          <>
            <span className={cn('text-xs font-mono font-semibold', statusColor(response.statusCode))}>
              {response.statusCode}
            </span>
            <span className="text-xs text-[var(--color-muted-foreground)]">{response.status}</span>
            <span className="ml-auto text-xs text-[var(--color-muted-foreground)]">
              {response.durationMs} ms
            </span>
            <span className="text-xs text-[var(--color-muted-foreground)]">
              {formatSize(response.size)}
            </span>
          </>
        ) : (
          <span className="text-xs text-[var(--color-muted-foreground)]">Response</span>
        )}
        {response?.timing && !response.error && <TimingBar timing={response.timing} total={response.durationMs} />}
        {cachedAt && !loading && (
          <span className="ml-auto shrink-0 text-[10px] text-[var(--color-muted-foreground)] italic" title={new Date(cachedAt).toLocaleString()}>
            Last run {timeAgo(cachedAt)}
          </span>
        )}
      </div>

      {/* Tabs */}
      {response && !response.error && (
        <div className="flex border-b border-[var(--color-border)] px-3 shrink-0">
          {(['Body', 'Headers'] as const).map(tab => (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              className={cn(
                'px-3 py-1.5 text-xs border-b-2 transition-colors',
                activeTab === tab
                  ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                  : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
              )}
            >
              {tab}
            </button>
          ))}
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-auto p-3">
        {loading && (
          <p className="text-xs text-[var(--color-muted-foreground)] animate-pulse">Sending request…</p>
        )}
        {!loading && !response && (
          <p className="text-xs text-[var(--color-muted-foreground)]">
            Send a request to see the response.
          </p>
        )}
        {!loading && response?.error && (
          <div className="text-xs text-red-400 font-mono whitespace-pre-wrap break-all">
            {response.error}
          </div>
        )}
        {!loading && response && !response.error && activeTab === 'Body' && (
          <JsonBody body={response.body} />
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
