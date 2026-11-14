import { useState, useRef, useMemo } from 'react'
import { cn } from '@/lib/utils'
import { inputCls, selectCls } from '@/lib/styles'
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
  value, onChange, onKeyDown, onPaste, placeholder, className, envVarNames,
}: {
  value: string
  onChange: (v: string) => void
  onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void
  onPaste?: (e: React.ClipboardEvent<HTMLInputElement>) => void
  placeholder?: string
  className?: string
  envVarNames?: string[]
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const mirrorRef = useRef<HTMLDivElement>(null)
  const hasVars = /\{\{/.test(value)
  const [acPartial, setAcPartial] = useState<string | null>(null)
  const [acIndex, setAcIndex] = useState(0)

  const acMatches = useMemo(() => {
    if (acPartial === null || !envVarNames?.length) return []
    const q = acPartial.toLowerCase()
    return envVarNames.filter(n => n.toLowerCase().includes(q))
  }, [acPartial, envVarNames])

  function checkAc(el: HTMLInputElement, v: string) {
    const pos = el.selectionStart ?? v.length
    const before = v.slice(0, pos)
    const oi = before.lastIndexOf('{{')
    if (oi !== -1 && !before.slice(oi + 2).includes('}}')) {
      setAcPartial(before.slice(oi + 2))
      setAcIndex(0)
    } else {
      setAcPartial(null)
    }
  }

  function insertVar(name: string) {
    const el = inputRef.current; if (!el) return
    const pos = el.selectionStart ?? value.length
    const before = value.slice(0, pos)
    const oi = before.lastIndexOf('{{')
    const newVal = value.slice(0, oi) + '{{' + name + '}}' + value.slice(pos)
    onChange(newVal); setAcPartial(null)
    requestAnimationFrame(() => {
      el.setSelectionRange(oi + 4 + name.length, oi + 4 + name.length)
      el.focus()
    })
  }

  function syncScroll() {
    if (mirrorRef.current && inputRef.current) {
      mirrorRef.current.scrollLeft = inputRef.current.scrollLeft
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (acMatches.length > 0 && acPartial !== null) {
      if (e.key === 'ArrowDown') { e.preventDefault(); setAcIndex(i => Math.min(i + 1, acMatches.length - 1)); return }
      if (e.key === 'ArrowUp') { e.preventDefault(); setAcIndex(i => Math.max(i - 1, 0)); return }
      if (e.key === 'Tab' || e.key === 'Enter') {
        if (acMatches[acIndex]) { e.preventDefault(); insertVar(acMatches[acIndex]); return }
      }
      if (e.key === 'Escape') { e.preventDefault(); setAcPartial(null); return }
    }
    onKeyDown?.(e)
  }

  return (
    <div className="relative flex-1 min-w-0">
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={e => { onChange(e.target.value); syncScroll(); checkAc(e.target, e.target.value) }}
        onKeyDown={handleKeyDown}
        onPaste={onPaste}
        onScroll={syncScroll}
        onClick={e => checkAc(e.currentTarget, value)}
        onBlur={() => setTimeout(() => setAcPartial(null), 150)}
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
      {acPartial !== null && acMatches.length > 0 && (
        <ul className="absolute top-full left-0 right-0 z-50 bg-[var(--color-card)] border border-[var(--color-border)] rounded shadow-lg mt-0.5 max-h-40 overflow-auto py-0.5">
          {acMatches.map((name, i) => (
            <li key={name}>
              <button
                type="button"
                onMouseDown={e => { e.preventDefault(); insertVar(name) }}
                className={cn(
                  'w-full text-left px-3 py-1 text-xs font-mono transition-colors',
                  i === acIndex ? 'bg-[var(--color-accent)]/15 text-[var(--color-foreground)]' : 'text-[var(--color-muted-foreground)] hover:bg-[var(--color-muted)]'
                )}
              >
                {name}
              </button>
            </li>
          ))}
        </ul>
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
  timeoutMs: number
  envVarMap?: Record<string, string>
  onMethodChange: (m: HttpMethod) => void
  onUrlChange: (u: string) => void
  onUrlPaste: (pastedUrl: string) => void
  onQueryParamsChange: (p: KVPair[]) => void
  onHeadersChange: (h: KVPair[]) => void
  onBodyChange: (b: BodyState) => void
  onAuthChange: (a: AuthState) => void
  onTimeoutChange: (ms: number) => void
  onPickFile?: () => Promise<string>
  onSend: () => void
  onCancel?: () => void
}

export function RequestEditor({
  method, url, loading,
  queryParams, headers, body, auth, timeoutMs, envVarMap,
  onMethodChange, onUrlChange, onUrlPaste,
  onQueryParamsChange, onHeadersChange, onBodyChange, onAuthChange,
  onTimeoutChange, onPickFile, onSend, onCancel,
}: RequestEditorProps) {
  const [activeTab, setActiveTab] = useState<RequestEditorTab>('Params')

  const urlVarNames = [...new Set(
    [...url.matchAll(/\{\{([^}]+)\}\}/g)].map(m => m[1])
  )]

  const envVarNames = useMemo(() => Object.keys(envVarMap ?? {}), [envVarMap])

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
          className={cn(selectCls, 'font-mono font-semibold px-2 py-1', METHOD_COLOR[method])}
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
          className={cn(inputCls, 'font-mono px-3 py-1')}
          envVarNames={envVarNames}
        />

        <div className="flex items-center gap-1 shrink-0" title="Per-request timeout — 0 uses the global default from Settings">
          <input
            type="number"
            min={0}
            step={1000}
            value={timeoutMs || ''}
            onChange={e => onTimeoutChange(e.target.value === '' ? 0 : Number(e.target.value))}
            placeholder="default"
            className={cn(
              'w-16 text-right',
              inputCls, 'px-2 py-1',
            )}
          />
          <span className="text-[10px] text-[var(--color-muted-foreground)]">ms</span>
        </div>

        {loading ? (
          <button
            type="button"
            onClick={onCancel}
            className="bg-[var(--color-muted)] border border-[var(--color-border)] text-red-400 hover:text-red-300 text-xs font-semibold rounded px-4 py-1.5 transition-colors shrink-0"
          >
            Cancel
          </button>
        ) : (
          <button
            type="button"
            onClick={onSend}
            disabled={!url.trim()}
            title="Send (Ctrl+Enter)"
            className={cn(
              'bg-[var(--color-accent)] text-white text-xs font-semibold rounded px-4 py-1.5',
              'transition-opacity disabled:opacity-40 hover:opacity-90 shrink-0',
            )}
          >
            Send
          </button>
        )}
      </div>

      {/* Variable resolution bar */}
      {envVarMap && urlVarNames.length > 0 && (
        <div className="flex items-center gap-3 px-3 py-1 border-b border-[var(--color-border)] bg-[var(--color-muted)] flex-wrap">
          {urlVarNames.map(name => {
            const resolved = envVarMap[name]
            const missing = resolved === undefined
            return (
              <span
                key={name}
                className={cn(
                  'text-[10px] font-mono',
                  missing ? 'text-amber-400' : 'text-[var(--color-muted-foreground)]',
                )}
                title={missing ? 'Variable not found in active environment' : resolved || '(empty string)'}
              >
                {missing ? `⚠ ${name}` : <>{name}<span className="opacity-50"> = </span>{resolved || <em>empty</em>}</>}
              </span>
            )
          })}
        </div>
      )}

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
              <span className="text-[10px] font-semibold px-1 rounded bg-[var(--color-accent)]/15 text-[var(--color-accent)]">
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
              envVarNames={envVarNames}
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
              envVarNames={envVarNames}
            />
          </div>
        )}
        {activeTab === 'Body' && (
          <div className="flex-1 overflow-hidden flex flex-col p-3">
            <BodyEditor body={body} onChange={onBodyChange} onPickFile={onPickFile} />
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
