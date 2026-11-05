import { useState, useRef } from 'react'
import { cn } from '@/lib/utils'
import { METHOD_COLOR, HTTP_METHODS, HTTP_REQUEST_HEADERS } from '@/lib/httpUtils'
import type { HttpMethod } from '@/lib/httpUtils'
import type { KVPair, BodyState, AuthState } from '@/types'
import { KeyValueEditor } from '@/components/KeyValueEditor'
import { BodyEditor } from '@/components/BodyEditor'
import { AuthEditor } from '@/components/AuthEditor'

function renderVarTokens(text: string): React.ReactNode[] {
  const nodes: React.ReactNode[] = []
  const re = /(\{\{[^}]*\}\})/g
  let last = 0, i = 0
  for (const m of text.matchAll(re)) {
    if (m.index! > last) nodes.push(<span key={`t${i++}`}>{text.slice(last, m.index)}</span>)
    nodes.push(<span key={`v${i++}`} style={{ color: 'var(--color-accent)' }}>{m[0]}</span>)
    last = m.index! + m[0].length
  }
  if (last < text.length) nodes.push(<span key={`t${i++}`}>{text.slice(last)}</span>)
  return nodes
}

function HighlightedInput({
  value, onChange, onKeyDown, onPaste, placeholder, className,
}: {
  value: string
  onChange: (v: string) => void
  onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void
  onPaste?: (e: React.ClipboardEvent<HTMLInputElement>) => void
  placeholder?: string
  className?: string
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const mirrorRef = useRef<HTMLDivElement>(null)
  const hasVars = /\{\{/.test(value)

  function syncScroll() {
    if (mirrorRef.current && inputRef.current) {
      mirrorRef.current.scrollLeft = inputRef.current.scrollLeft
    }
  }

  return (
    <div className="relative flex-1 min-w-0">
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={e => { onChange(e.target.value); syncScroll() }}
        onKeyDown={onKeyDown}
        onPaste={onPaste}
        onScroll={syncScroll}
        placeholder={placeholder}
        className={cn(className, 'w-full', hasVars && 'text-transparent placeholder:text-transparent')}
        style={hasVars ? { caretColor: 'var(--color-foreground)' } : undefined}
      />
      {hasVars && (
        <div
          ref={mirrorRef}
          aria-hidden
          className="absolute inset-0 flex items-center pointer-events-none overflow-hidden px-3 text-xs font-mono"
        >
          {renderVarTokens(value)}
        </div>
      )}
    </div>
  )
}

type RequestEditorTab = 'Params' | 'Headers' | 'Body' | 'Auth'

export type RequestEditorProps = {
  method: HttpMethod
  url: string
  loading: boolean
  queryParams: KVPair[]
  headers: KVPair[]
  body: BodyState
  auth: AuthState
  onMethodChange: (m: HttpMethod) => void
  onUrlChange: (u: string) => void
  onUrlPaste: (pastedUrl: string) => void
  onQueryParamsChange: (p: KVPair[]) => void
  onHeadersChange: (h: KVPair[]) => void
  onBodyChange: (b: BodyState) => void
  onAuthChange: (a: AuthState) => void
  onSend: () => void
}

export function RequestEditor({
  method, url, loading,
  queryParams, headers, body, auth,
  onMethodChange, onUrlChange, onUrlPaste,
  onQueryParamsChange, onHeadersChange, onBodyChange, onAuthChange,
  onSend,
}: RequestEditorProps) {
  const [activeTab, setActiveTab] = useState<RequestEditorTab>('Params')

  const activeParamCount = queryParams.filter(p => p.enabled && p.name).length
  const activeHeaderCount = headers.filter(h => h.enabled && h.name).length

  const tabBadge: Record<RequestEditorTab, string | undefined> = {
    Params: activeParamCount > 0 ? String(activeParamCount) : undefined,
    Headers: activeHeaderCount > 0 ? String(activeHeaderCount) : undefined,
    Body: body.type !== 'none' ? '•' : undefined,
    Auth: auth.type !== 'none' ? '•' : undefined,
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' && !loading) onSend()
  }

  return (
    <div className="flex flex-col h-full">
      {/* URL bar */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-[var(--color-border)] bg-[var(--color-muted)]">
        <select
          value={method}
          onChange={e => onMethodChange(e.target.value as HttpMethod)}
          className={cn(
            'bg-[var(--color-muted)] text-xs font-mono font-semibold rounded px-2 py-1',
            'border border-[var(--color-border)] outline-none cursor-pointer',
            METHOD_COLOR[method],
          )}
        >
          {HTTP_METHODS.map(m => (
            <option key={m} value={m}>{m}</option>
          ))}
        </select>

        <HighlightedInput
          value={url}
          onChange={onUrlChange}
          onKeyDown={handleKeyDown}
          onPaste={e => {
            const text = e.clipboardData.getData('text').trim()
            if (/^curl\b/i.test(text) || text.includes('?')) {
              e.preventDefault()
              onUrlPaste(text)
            }
          }}
          placeholder="https://api.example.com/endpoint"
          className={cn(
            'bg-[var(--color-muted)] text-xs font-mono rounded px-3 py-1',
            'border border-[var(--color-border)] outline-none',
            'focus:border-[var(--color-accent)]',
            'placeholder:text-[var(--color-muted-foreground)]',
            'text-[var(--color-foreground)]',
          )}
        />

        <button
          type="button"
          onClick={onSend}
          disabled={loading || !url.trim()}
          className={cn(
            'bg-[var(--color-accent)] text-white text-xs font-semibold rounded px-4 py-1',
            'transition-opacity disabled:opacity-40 hover:opacity-90 shrink-0',
          )}
        >
          {loading ? 'Sending…' : 'Send'}
        </button>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-[var(--color-border)] px-3 shrink-0">
        {(['Params', 'Headers', 'Body', 'Auth'] as const).map(tab => (
          <button
            key={tab}
            type="button"
            onClick={() => setActiveTab(tab)}
            className={cn(
              'flex items-center gap-1 px-3 py-1.5 text-xs border-b-2 transition-colors',
              activeTab === tab
                ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
            )}
          >
            {tab}
            {tabBadge[tab] && (
              <span className="text-[10px] text-[var(--color-accent)] font-semibold">
                {tabBadge[tab]}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-hidden flex flex-col">
        {activeTab === 'Params' && (
          <div className="flex-1 overflow-auto p-3">
            <KeyValueEditor
              pairs={queryParams}
              onChange={onQueryParamsChange}
              keyPlaceholder="Parameter"
              valuePlaceholder="Value"
            />
          </div>
        )}
        {activeTab === 'Headers' && (
          <div className="flex-1 overflow-auto p-3">
            <KeyValueEditor
              pairs={headers}
              onChange={onHeadersChange}
              keyPlaceholder="Header"
              valuePlaceholder="Value"
              keySuggestions={HTTP_REQUEST_HEADERS}
            />
          </div>
        )}
        {activeTab === 'Body' && (
          <div className="flex-1 overflow-hidden flex flex-col p-3">
            <BodyEditor body={body} onChange={onBodyChange} />
          </div>
        )}
        {activeTab === 'Auth' && (
          <div className="flex-1 overflow-auto p-3">
            <AuthEditor auth={auth} onChange={onAuthChange} />
          </div>
        )}
      </div>
    </div>
  )
}
