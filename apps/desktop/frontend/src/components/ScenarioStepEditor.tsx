import { useId, useState } from 'react'
import { Trash2, Plus, ChevronDown, ChevronRight, ChevronUp, Eye, EyeOff } from 'lucide-react'
import { cn } from '@/lib/utils'
import { METHOD_COLOR, HTTP_METHODS, HTTP_REQUEST_HEADERS, BODY_TYPES } from '@/lib/httpUtils'
import type { HttpMethod } from '@/lib/httpUtils'
import { varsIn } from '@/lib/stepVars'
import { formatJsonWithPlaceholders } from '@/lib/jsonFormat'
import type { StepVarInfo } from '@/lib/stepVars'
import { HighlightedInput } from '@/components/HighlightedInput'
import { VarValueInput } from '@/components/KeyValueEditor'
import { AuthEditor } from '@/components/AuthEditor'
import { authPayloadToState, authStateToPayload } from '@/lib/httpUtils'
import { inputCls as baseInputCls, selectCls as baseSelectCls } from '@/lib/styles'
import type { StepDetail, AssertionDetail, ExtractionDetail } from '@/lib/bridge'

const ASSERTION_TYPES = ['status', 'jsonpath', 'header', 'schema']
const STATUS_OPERATORS = ['eq', 'ne', 'gt', 'lt']
const ALL_OPERATORS = ['eq', 'ne', 'gt', 'lt', 'contains', 'matches']
const EXTRACTION_SOURCES = ['jsonpath', 'header', 'status']

const inputCls = cn(baseInputCls, 'px-2 py-1')
const selectCls = cn(baseSelectCls, 'px-2 py-1')
const sectionLabelCls = 'text-xs font-semibold uppercase tracking-wider text-[var(--color-muted-foreground)] mb-1.5'
const addBtnCls = 'flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors mt-1'

export function AssertionRow({ assertion, onChange, onRemove }: {
  assertion: AssertionDetail
  onChange: (u: Partial<AssertionDetail>) => void
  onRemove: () => void
}) {
  const needsPath = assertion.type === 'jsonpath' || assertion.type === 'header'
  const isSchema = assertion.type === 'schema'
  const availableOperators = assertion.type === 'status' ? STATUS_OPERATORS : ALL_OPERATORS

  return (
    <div className="flex items-center gap-1 mb-1 flex-wrap">
      <select
        aria-label="Assertion type"
        value={assertion.type}
        onChange={e => onChange({ type: e.target.value, path: '', expected: '', schema: '', operator: 'eq' })}
        className={selectCls}
      >
        {ASSERTION_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
      </select>
      {needsPath && (
        <input
          value={assertion.path ?? ''}
          onChange={e => onChange({ path: e.target.value })}
          placeholder={assertion.type === 'jsonpath' ? '$.field' : 'Header-Name'}
          className={cn(inputCls, 'w-28 flex-1 min-w-[5rem]', !assertion.path?.trim() && 'border-red-500/70 focus:border-red-500')}
        />
      )}
      {isSchema ? (
        <input
          value={assertion.schema ?? ''}
          onChange={e => onChange({ schema: e.target.value })}
          placeholder="OpenAPI operation ID or spec path"
          className={cn(inputCls, 'flex-1 min-w-[8rem]')}
          aria-label="Schema value"
        />
      ) : (
        <>
          <select
            aria-label="Assertion operator"
            value={assertion.operator ?? 'eq'}
            onChange={e => onChange({ operator: e.target.value })}
            className={selectCls}
          >
            {availableOperators.map(o => <option key={o} value={o}>{o}</option>)}
          </select>
          <input
            value={assertion.expected !== undefined && assertion.expected !== null ? String(assertion.expected) : ''}
            onChange={e => {
              const raw = e.target.value
              const num = Number(raw)
              onChange({ expected: raw === '' ? '' : (!isNaN(num) && raw.trim() !== '' ? num : raw) })
            }}
            placeholder="expected"
            className={cn(inputCls, 'w-24 flex-1 min-w-[4rem]')}
          />
        </>
      )}
      <button
        type="button"
        onClick={onRemove}
        title="Remove assertion"
        className="p-1 shrink-0 text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors"
      >
        <Trash2 size={11} />
      </button>
    </div>
  )
}

export function ExtractionRow({ extraction, onChange, onRemove }: {
  extraction: ExtractionDetail
  onChange: (u: Partial<ExtractionDetail>) => void
  onRemove: () => void
}) {
  const needsPath = extraction.source === 'jsonpath' || extraction.source === 'header'
  return (
    <div className="flex items-center gap-1 mb-1 flex-wrap">
      <input
        value={extraction.name}
        onChange={e => onChange({ name: e.target.value })}
        placeholder="variable name"
        className={cn(inputCls, 'w-28 flex-1 min-w-[5rem]')}
      />
      <select
        value={extraction.source}
        onChange={e => onChange({ source: e.target.value, path: '' })}
        className={selectCls}
      >
        {EXTRACTION_SOURCES.map(s => <option key={s} value={s}>{s}</option>)}
      </select>
      {needsPath && (
        <input
          value={extraction.path ?? ''}
          onChange={e => onChange({ path: e.target.value })}
          placeholder={extraction.source === 'jsonpath' ? '$.field' : 'Header-Name'}
          className={cn(inputCls, 'w-28 flex-1 min-w-[5rem]')}
        />
      )}
      <button
        type="button"
        onClick={onRemove}
        title="Remove extraction"
        className="p-1 shrink-0 text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors"
      >
        <Trash2 size={11} />
      </button>
    </div>
  )
}

export function KVRow({ pair, onChange, onRemove, keyPlaceholder = 'Key', valuePlaceholder = 'Value', keySuggestions, varNames }: {
  pair: { name: string; value: string }
  onChange: (u: { name?: string; value?: string }) => void
  onRemove: () => void
  keyPlaceholder?: string
  valuePlaceholder?: string
  keySuggestions?: string[]
  varNames?: string[]
}) {
  const listId = useId()
  return (
    <div className="flex items-center gap-1 mb-1">
      {keySuggestions && (
        <datalist id={listId}>
          {keySuggestions.map(s => <option key={s} value={s} />)}
        </datalist>
      )}
      <input
        value={pair.name}
        onChange={e => onChange({ name: e.target.value })}
        placeholder={keyPlaceholder}
        list={keySuggestions ? listId : undefined}
        className={cn(inputCls, 'flex-1 min-w-0 font-mono')}
      />
      <VarValueInput
        value={pair.value}
        onChange={v => onChange({ value: v })}
        placeholder={valuePlaceholder}
        envVarNames={varNames}
      />
      <button
        type="button"
        onClick={onRemove}
        title="Remove"
        className="p-1 shrink-0 text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors"
      >
        <Trash2 size={11} />
      </button>
    </div>
  )
}

function EditorSection({ title, count, addLabel, onAdd, badge, children }: {
  title: string
  count: number
  addLabel?: string
  onAdd?: () => void
  badge?: string
  children: React.ReactNode
}) {
  const [open, setOpen] = useState(count > 0)
  return (
    <div className="px-3 py-2 border-b border-[var(--color-border)]">
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={() => setOpen(o => !o)}
          aria-expanded={open}
          className="flex items-center gap-1 text-xs font-semibold uppercase tracking-wider text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
        >
          {open ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
          {title}
          {count > 0 && (
            <span className="px-1.5 rounded-full bg-[var(--color-border)] text-[10px] normal-case tracking-normal">{badge ?? count}</span>
          )}
        </button>
        {!open && addLabel && onAdd && (
          <button type="button" onClick={() => { setOpen(true); onAdd() }} className={addBtnCls}>
            <Plus size={10} /> {addLabel}
          </button>
        )}
      </div>
      {open && <div className="mt-1.5">{children}</div>}
    </div>
  )
}

export function StepEditorCard({ step, index, isFirst, isLast, onChange, onRemove, onMoveUp, onMoveDown, onToggleDisabled, vars }: {
  step: StepDetail
  vars?: StepVarInfo
  index: number
  isFirst: boolean
  isLast: boolean
  onChange: (u: Partial<StepDetail>) => void
  onRemove: () => void
  onMoveUp: () => void
  onMoveDown: () => void
  onToggleDisabled: () => void
}) {
  const hasNonDefaultOptions = !!((step.failPolicy && step.failPolicy !== 'stop') ||
    (step.timeoutMs && step.timeoutMs > 0) ||
    (step.retryMaxAttempts && step.retryMaxAttempts > 0))
  const [showAdvanced, setShowAdvanced] = useState(!!hasNonDefaultOptions)
  const [bodyFormatError, setBodyFormatError] = useState(false)

  function updateHeader(i: number, u: { name?: string; value?: string }) {
    onChange({ headers: (step.headers ?? []).map((h, hi) => hi === i ? { ...h, ...u } : h) })
  }
  function removeHeader(i: number) {
    onChange({ headers: (step.headers ?? []).filter((_, hi) => hi !== i) })
  }
  function addHeader() {
    onChange({ headers: [...(step.headers ?? []), { name: '', value: '' }] })
  }

  function updateQueryParam(i: number, u: { name?: string; value?: string }) {
    onChange({ queryParams: (step.queryParams ?? []).map((q, qi) => qi === i ? { ...q, ...u } : q) })
  }
  function removeQueryParam(i: number) {
    onChange({ queryParams: (step.queryParams ?? []).filter((_, qi) => qi !== i) })
  }
  function addQueryParam() {
    onChange({ queryParams: [...(step.queryParams ?? []), { name: '', value: '' }] })
  }

  function updateAssertion(i: number, u: Partial<AssertionDetail>) {
    onChange({ assertions: step.assertions.map((a, ai) => ai === i ? { ...a, ...u } : a) })
  }
  function removeAssertion(i: number) {
    onChange({ assertions: step.assertions.filter((_, ai) => ai !== i) })
  }
  function addAssertion() {
    onChange({ assertions: [...step.assertions, { type: 'status', operator: 'eq', expected: 200 }] })
  }

  function updateExtraction(i: number, u: Partial<ExtractionDetail>) {
    onChange({ extractions: step.extractions.map((e, ei) => ei === i ? { ...e, ...u } : e) })
  }
  function removeExtraction(i: number) {
    onChange({ extractions: step.extractions.filter((_, ei) => ei !== i) })
  }
  function addExtraction() {
    onChange({ extractions: [...step.extractions, { name: '', source: 'jsonpath', path: '' }] })
  }

  return (
    <div className={cn('rounded border border-[var(--color-border)] bg-[var(--color-muted)] text-xs overflow-hidden', step.disabled && 'opacity-60')}>
      {/* Step header */}
      <div className="px-3 py-2 border-b border-[var(--color-border)] bg-[var(--color-card)]">
        <div className="flex items-center gap-2">
          <span className="shrink-0 w-5 h-5 rounded-full bg-[var(--color-border)] text-[var(--color-muted-foreground)] flex items-center justify-center text-[11px] font-bold">
            {index + 1}
          </span>
          <input
            value={step.label}
            onChange={e => onChange({ label: e.target.value })}
            placeholder="Step label"
            className="flex-1 min-w-0 bg-transparent border-b border-[var(--color-border)] hover:border-[var(--color-muted-foreground)] focus:border-[var(--color-accent)] outline-none text-xs text-[var(--color-foreground)] font-semibold py-0.5 transition-colors"
          />
          <div className="flex items-center shrink-0">
            <button
              type="button"
              onClick={onToggleDisabled}
              title={step.disabled ? 'Enable step' : 'Disable step (skip during run)'}
              className={cn('p-1 transition-colors', step.disabled ? 'text-amber-400 hover:text-amber-300' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}
            >
              {step.disabled ? <EyeOff size={12} /> : <Eye size={12} />}
            </button>
            <button
              type="button"
              onClick={onMoveUp}
              disabled={isFirst}
              title="Move step up (Alt+↑)"
              className={cn('p-1 transition-colors', isFirst ? 'opacity-20 cursor-not-allowed text-[var(--color-muted-foreground)]' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}
            >
              <ChevronUp size={12} />
            </button>
            <button
              type="button"
              onClick={onMoveDown}
              disabled={isLast}
              title="Move step down (Alt+↓)"
              className={cn('p-1 transition-colors', isLast ? 'opacity-20 cursor-not-allowed text-[var(--color-muted-foreground)]' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}
            >
              <ChevronDown size={12} />
            </button>
            <button
              type="button"
              onClick={onRemove}
              title="Remove step"
              className="p-1 text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors"
            >
              <Trash2 size={12} />
            </button>
          </div>
        </div>
        <input
          value={step.notes ?? ''}
          onChange={e => onChange({ notes: e.target.value })}
          placeholder="Notes (optional)"
          className="w-full mt-1 bg-transparent border-b border-transparent focus:border-[var(--color-accent)] outline-none text-xs text-[var(--color-muted-foreground)] py-0.5 italic placeholder:not-italic"
        />
      </div>

      {/* Request */}
      <div className="px-3 py-2 border-b border-[var(--color-border)]">
        <div className={sectionLabelCls}>Request</div>
        <div className="flex gap-1.5 items-center mb-1.5">
          <select
            value={step.method ?? 'GET'}
            onChange={e => onChange({ method: e.target.value })}
            className={cn(selectCls, 'font-mono font-semibold', METHOD_COLOR[(step.method ?? 'GET') as HttpMethod])}
          >
            {HTTP_METHODS.map(m => <option key={m} value={m}>{m}</option>)}
          </select>
          <HighlightedInput
            value={step.url ?? ''}
            onChange={url => onChange({ url })}
            placeholder="https://api.example.com/path"
            className={cn(inputCls, 'font-mono')}
            envVarNames={vars?.names}
            padding="px-2"
          />
        </div>
        {vars && varsIn(step.url ?? '').length > 0 && (
          <div className="flex items-center gap-3 flex-wrap mb-1.5" aria-label="Variables in the URL">
            {varsIn(step.url ?? '').map(name => {
              const h = vars.hint(name)
              const warn = h.kind === 'missing' || h.kind === 'later'
              return (
                <span key={name} className={cn('text-[10px] font-mono', warn ? 'text-amber-400' : 'text-[var(--color-muted-foreground)]')}>
                  {warn ? `⚠ ${name} ` : name}
                  {!warn && <span className="opacity-50">{h.kind === 'extracted' ? ' ← ' : ' = '}</span>}
                  {h.text}
                </span>
              )
            })}
          </div>
        )}
        <div className="flex items-center gap-1.5">
          <select
            value={step.bodyType ?? 'none'}
            onChange={e => onChange({ bodyType: e.target.value })}
            className={selectCls}
            title="Request body type"
          >
            {BODY_TYPES
              .filter(t => t.value !== 'multipart' || step.bodyType === 'multipart')
              .map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
          </select>
          {(step.bodyType === 'json' || step.bodyType === 'raw' || step.bodyType === 'form-urlencoded') && (
            <button
              type="button"
              title="Format JSON"
              disabled={step.bodyType !== 'json'}
              onClick={() => {
                const formatted = formatJsonWithPlaceholders(step.bodyContent ?? '')
                setBodyFormatError(formatted === null)
                if (formatted !== null) onChange({ bodyContent: formatted })
              }}
              className={cn('shrink-0 text-xs px-1.5 py-0.5 rounded border border-[var(--color-border)] transition-colors',
                step.bodyType === 'json'
                  ? 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]'
                  : 'opacity-30 cursor-not-allowed text-[var(--color-muted-foreground)]'
              )}
            >
              Format
            </button>
          )}
          {bodyFormatError && step.bodyType === 'json' && (
            <span role="alert" className="text-xs text-red-400">Not valid JSON, so it was left as it is.</span>
          )}
        </div>
        {(step.bodyType === 'json' || step.bodyType === 'raw' || step.bodyType === 'form-urlencoded') && (
          <textarea
            value={step.bodyContent ?? ''}
            onChange={e => onChange({ bodyContent: e.target.value })}
            placeholder={
              step.bodyType === 'json' ? '{"key": "value"}'
              : step.bodyType === 'form-urlencoded' ? 'key=value&other=value'
              : 'Raw body content'
            }
            rows={4}
            className={cn(inputCls, 'w-full mt-1.5 font-mono resize-y')}
          />
        )}
      </div>

      <EditorSection title="Headers" count={(step.headers ?? []).length} addLabel="Add header" onAdd={addHeader}>
        {(step.headers ?? []).map((h, i) => (
          <KVRow
            key={i}
            pair={h}
            onChange={u => updateHeader(i, u)}
            onRemove={() => removeHeader(i)}
            keyPlaceholder="Header"
            valuePlaceholder="Value"
            keySuggestions={HTTP_REQUEST_HEADERS}
            varNames={vars?.names}
          />
        ))}
        <button type="button" onClick={addHeader} className={addBtnCls}>
          <Plus size={10} /> Add header
        </button>
      </EditorSection>

      <EditorSection title="Query Params" count={(step.queryParams ?? []).length} addLabel="Add param" onAdd={addQueryParam}>
        {(step.queryParams ?? []).map((q, i) => (
          <KVRow
            key={i}
            pair={q}
            onChange={u => updateQueryParam(i, u)}
            onRemove={() => removeQueryParam(i)}
            keyPlaceholder="Parameter"
            valuePlaceholder="Value"
            varNames={vars?.names}
          />
        ))}
        <button type="button" onClick={addQueryParam} className={addBtnCls}>
          <Plus size={10} /> Add param
        </button>
      </EditorSection>

      {!step.requestRef && (
        <EditorSection title="Auth" count={step.auth ? 1 : 0} badge={step.auth?.type}>
          <div className="flex flex-col gap-2">
            <AuthEditor
              auth={authPayloadToState(step.auth)}
              onChange={a => onChange({ auth: authStateToPayload(a) })}
            />
            <p className="text-[11px] text-[var(--color-muted-foreground)]">
              Use a variable such as <code className="font-mono">{'{{token}}'}</code> so the secret is not written into the scenario file.
            </p>
          </div>
        </EditorSection>
      )}

      {/* Assertions */}
      <div className="px-3 py-2 border-b border-[var(--color-border)]">
        <div className={sectionLabelCls}>Assertions</div>
        {step.assertions.map((a, i) => (
          <AssertionRow
            key={i}
            assertion={a}
            onChange={u => updateAssertion(i, u)}
            onRemove={() => removeAssertion(i)}
          />
        ))}
        <button type="button" onClick={addAssertion} className={addBtnCls}>
          <Plus size={10} /> Add assertion
        </button>
      </div>

      <EditorSection title="Extractions" count={step.extractions.length} addLabel="Add extraction" onAdd={addExtraction}>
        {step.extractions.map((e, i) => (
          <ExtractionRow
            key={i}
            extraction={e}
            onChange={u => updateExtraction(i, u)}
            onRemove={() => removeExtraction(i)}
          />
        ))}
        <button type="button" onClick={addExtraction} className={addBtnCls}>
          <Plus size={10} /> Add extraction
        </button>
      </EditorSection>

      {/* Advanced: fail policy, timeout, retry */}
      <div className="border-t border-[var(--color-border)]">
        <button
          type="button"
          onClick={() => setShowAdvanced(v => !v)}
          className="w-full flex items-center gap-1.5 px-3 py-2 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
        >
          {showAdvanced ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
          <span>Advanced</span>
          {hasNonDefaultOptions && !showAdvanced && (
            <span className="ml-1 w-1.5 h-1.5 rounded-full bg-[var(--color-accent)] shrink-0" />
          )}
        </button>
        {showAdvanced && (
          <div className="px-3 pb-2 flex flex-wrap gap-x-4 gap-y-2">
            <div className="flex items-center gap-1.5">
              <span className="text-[var(--color-muted-foreground)] text-xs shrink-0">On failure:</span>
              <select
                value={step.failPolicy ?? 'stop'}
                onChange={e => onChange({ failPolicy: e.target.value })}
                className={selectCls}
              >
                <option value="stop">stop remaining steps</option>
                <option value="continue">continue next steps</option>
                <option value="skip">skip this step if prev failed</option>
              </select>
            </div>
            <div className="flex items-center gap-1">
              <span className="text-[var(--color-muted-foreground)] text-xs shrink-0">Timeout:</span>
              <input
                type="number"
                min={0}
                value={step.timeoutMs ?? 0}
                onChange={e => onChange({ timeoutMs: Math.max(0, parseInt(e.target.value) || 0) })}
                title="Step timeout in milliseconds (0 = use global default)"
                className={cn(inputCls, 'w-16')}
              />
              <span className="text-[var(--color-muted-foreground)] text-xs">ms</span>
            </div>
            <label
              className="flex items-center gap-1 text-[var(--color-muted-foreground)] text-xs cursor-pointer"
              title="Return a 3xx response as it is instead of following the redirect"
            >
              <input
                type="checkbox"
                checked={step.noFollowRedirects ?? false}
                onChange={e => onChange({ noFollowRedirects: e.target.checked })}
                className="accent-[var(--color-accent)]"
              />
              Don&apos;t follow redirects
            </label>
            <div className="flex items-center gap-1">
              <span className="text-[var(--color-muted-foreground)] text-xs shrink-0">Retry:</span>
              <input
                type="number"
                min={0}
                max={10}
                value={step.retryMaxAttempts ?? 0}
                onChange={e => onChange({ retryMaxAttempts: Math.max(0, parseInt(e.target.value) || 0) })}
                title="Maximum retry attempts (0 = no retry)"
                className={cn(inputCls, 'w-12')}
              />
              <span className="text-[var(--color-muted-foreground)] text-xs">× delay</span>
              <input
                type="number"
                min={0}
                value={step.retryDelayMs ?? 1000}
                onChange={e => onChange({ retryDelayMs: Math.max(0, parseInt(e.target.value) || 0) })}
                title={(step.retryMaxAttempts ?? 0) === 0 ? 'Set Retry attempts > 0 to enable delay' : 'Delay between retry attempts in milliseconds'}
                disabled={(step.retryMaxAttempts ?? 0) === 0}
                className={cn(inputCls, 'w-16', (step.retryMaxAttempts ?? 0) === 0 && 'opacity-40')}
              />
              <span className="text-[var(--color-muted-foreground)] text-xs">ms</span>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
