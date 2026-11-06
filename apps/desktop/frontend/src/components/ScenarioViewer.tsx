import { useState, useEffect, useRef } from 'react'
import { Play, Loader2, CheckCircle2, XCircle, MinusCircle, ChevronDown, ChevronUp, ChevronRight, Save, X, Plus, Trash2 } from 'lucide-react'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { cn } from '@/lib/utils'
import { inputCls as baseInputCls, selectCls as baseSelectCls } from '@/lib/styles'
import { loadScenario, runScenario, loadScenarioRaw, saveScenarioRaw, saveScenarioResult, loadScenarioResult } from '@/lib/bridge'
import type { ScenarioDetail, ScenarioResult, StepDetail, AssertionDetail, ExtractionDetail, EnvInfo, StepResult } from '@/lib/bridge'
import { METHOD_BADGE } from '@/lib/httpUtils'

// ---- YAML serialization ----

function yamlScalar(s: string): string {
  if (s === '') return '""'
  if (
    /[:#\[\]{},&*!|>'"%@`\n\r\t]/.test(s) ||
    s.startsWith(' ') || s.endsWith(' ') ||
    s === 'true' || s === 'false' || s === 'null'
  ) return JSON.stringify(s)
  return s
}

function yamlValue(v: unknown): string {
  if (v === null || v === undefined) return 'null'
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  return yamlScalar(String(v))
}

function serializeScenarioToYAML(d: ScenarioDetail): string {
  const lines: string[] = ['schemaVersion: "1"', `name: ${yamlScalar(d.name)}`]
  if (d.description) lines.push(`description: ${yamlScalar(d.description)}`)
  if (!d.steps.length) {
    lines.push('steps: []')
  } else {
    lines.push('steps:')
    for (const step of d.steps) {
      lines.push(`  - name: ${yamlScalar(step.name)}`)
      if (step.description) lines.push(`    description: ${yamlScalar(step.description)}`)
      if (step.requestRef) {
        lines.push(`    requestRef: ${yamlScalar(step.requestRef)}`)
      } else if (step.method || step.url) {
        lines.push('    request:')
        if (step.method) lines.push(`      method: ${step.method}`)
        if (step.url) lines.push(`      url: ${yamlScalar(step.url)}`)
        if (step.headers?.length) {
          lines.push('      headers:')
          for (const h of step.headers) {
            lines.push(`        - name: ${yamlScalar(h.name)}`)
            lines.push(`          value: ${yamlScalar(h.value)}`)
          }
        }
        if (step.queryParams?.length) {
          lines.push('      queryParams:')
          for (const q of step.queryParams) {
            lines.push(`        - name: ${yamlScalar(q.name)}`)
            lines.push(`          value: ${yamlScalar(q.value)}`)
          }
        }
        if (step.bodyType && step.bodyType !== 'none') {
          lines.push('      body:')
          lines.push(`        type: ${step.bodyType}`)
          if (step.bodyContent) lines.push(`        content: ${yamlScalar(step.bodyContent)}`)
        }
      }
      if (step.failPolicy && step.failPolicy !== 'stop') {
        lines.push(`    failPolicy: ${step.failPolicy}`)
      }
      if (step.timeoutMs && step.timeoutMs > 0) {
        lines.push(`    timeoutMs: ${step.timeoutMs}`)
      }
      if (step.retryMaxAttempts && step.retryMaxAttempts > 0) {
        lines.push('    retry:')
        lines.push(`      maxAttempts: ${step.retryMaxAttempts}`)
        if (step.retryDelayMs && step.retryDelayMs > 0) {
          lines.push(`      delayMs: ${step.retryDelayMs}`)
        }
      }
      if (step.assertions?.length) {
        lines.push('    assertions:')
        for (const a of step.assertions) {
          lines.push(`      - type: ${a.type}`)
          if (a.path) lines.push(`        path: ${yamlScalar(a.path)}`)
          lines.push(`        operator: ${a.operator ?? 'eq'}`)
          if (a.expected !== undefined && a.expected !== null) {
            lines.push(`        expected: ${yamlValue(a.expected)}`)
          }
          if (a.schema) lines.push(`        schema: ${yamlScalar(a.schema)}`)
        }
      }
      if (step.extractions?.length) {
        lines.push('    extractions:')
        for (const e of step.extractions) {
          lines.push(`      - name: ${yamlScalar(e.name)}`)
          lines.push(`        source: ${e.source}`)
          if (e.path) lines.push(`        path: ${yamlScalar(e.path)}`)
        }
      }
    }
  }
  return lines.join('\n') + '\n'
}

// ---- Visual editor sub-components ----

const BODY_TYPES = ['none', 'json', 'raw', 'form-urlencoded']

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']
const ASSERTION_TYPES = ['status', 'jsonpath', 'header', 'schema']
const STATUS_OPERATORS = ['eq', 'ne', 'gt', 'lt']
const ALL_OPERATORS = ['eq', 'ne', 'gt', 'lt', 'contains', 'matches']
const EXTRACTION_SOURCES = ['jsonpath', 'header', 'status']

const inputCls = cn(baseInputCls, 'px-2 py-1')
const selectCls = cn(baseSelectCls, 'px-2 py-1')

function AssertionRow({ assertion, onChange, onRemove }: {
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
          className={cn(inputCls, 'w-28 flex-1 min-w-[5rem]')}
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

function ExtractionRow({ extraction, onChange, onRemove }: {
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

function KVRow({ pair, onChange, onRemove, keyPlaceholder = 'name', valuePlaceholder = 'value' }: {
  pair: { name: string; value: string }
  onChange: (u: { name?: string; value?: string }) => void
  onRemove: () => void
  keyPlaceholder?: string
  valuePlaceholder?: string
}) {
  return (
    <div className="flex items-center gap-1 mb-1">
      <input
        value={pair.name}
        onChange={e => onChange({ name: e.target.value })}
        placeholder={keyPlaceholder}
        className={cn(inputCls, 'flex-1 min-w-0')}
      />
      <input
        value={pair.value}
        onChange={e => onChange({ value: e.target.value })}
        placeholder={valuePlaceholder}
        className={cn(inputCls, 'flex-1 min-w-0')}
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

const sectionLabelCls = 'text-xs font-semibold uppercase tracking-wider text-[var(--color-muted-foreground)] mb-1.5'
const addBtnCls = 'flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors mt-1'

function StepEditorCard({ step, index, isFirst, isLast, onChange, onRemove, onMoveUp, onMoveDown }: {
  step: StepDetail
  index: number
  isFirst: boolean
  isLast: boolean
  onChange: (u: Partial<StepDetail>) => void
  onRemove: () => void
  onMoveUp: () => void
  onMoveDown: () => void
}) {
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
    <div className="rounded border border-[var(--color-border)] bg-[var(--color-muted)] text-xs overflow-hidden">
      {/* Step header */}
      <div className="px-3 py-2 border-b border-[var(--color-border)] bg-[var(--color-card)]">
        <div className="flex items-center gap-2">
          <span className="shrink-0 w-5 h-5 rounded-full bg-[var(--color-border)] text-[var(--color-muted-foreground)] flex items-center justify-center text-[11px] font-bold">
            {index + 1}
          </span>
          <input
            value={step.name}
            onChange={e => onChange({ name: e.target.value })}
            placeholder="Step name"
            className="flex-1 min-w-0 bg-transparent border-b border-transparent focus:border-[var(--color-accent)] outline-none text-xs text-[var(--color-foreground)] font-semibold py-0.5"
          />
          <div className="flex items-center shrink-0">
            <button
              type="button"
              onClick={onMoveUp}
              disabled={isFirst}
              title="Move step up"
              className={cn('p-1 transition-colors', isFirst ? 'opacity-20 cursor-not-allowed text-[var(--color-muted-foreground)]' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}
            >
              <ChevronUp size={12} />
            </button>
            <button
              type="button"
              onClick={onMoveDown}
              disabled={isLast}
              title="Move step down"
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
          value={step.description ?? ''}
          onChange={e => onChange({ description: e.target.value })}
          placeholder="Step description (optional)"
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
            className={selectCls}
          >
            {HTTP_METHODS.map(m => <option key={m} value={m}>{m}</option>)}
          </select>
          <input
            value={step.url ?? ''}
            onChange={e => onChange({ url: e.target.value })}
            placeholder="https://api.example.com/path"
            className={cn(inputCls, 'flex-1 min-w-0 font-mono')}
          />
        </div>
        <div className="flex items-center gap-1.5">
          <select
            value={step.bodyType ?? 'none'}
            onChange={e => onChange({ bodyType: e.target.value })}
            className={selectCls}
            title="Request body type"
          >
            {BODY_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
          </select>
          {(step.bodyType === 'json' || step.bodyType === 'raw' || step.bodyType === 'form-urlencoded') && (
            <button
              type="button"
              title="Format JSON"
              disabled={step.bodyType !== 'json'}
              onClick={() => {
                try {
                  onChange({ bodyContent: JSON.stringify(JSON.parse(step.bodyContent ?? ''), null, 2) })
                } catch { /* invalid JSON — leave as-is */ }
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

      {/* Headers */}
      <div className="px-3 py-2 border-b border-[var(--color-border)]">
        <div className={sectionLabelCls}>Headers</div>
        {(step.headers ?? []).map((h, i) => (
          <KVRow
            key={i}
            pair={h}
            onChange={u => updateHeader(i, u)}
            onRemove={() => removeHeader(i)}
            keyPlaceholder="Header-Name"
            valuePlaceholder="value"
          />
        ))}
        <button type="button" onClick={addHeader} className={addBtnCls}>
          <Plus size={10} /> Add header
        </button>
      </div>

      {/* Query Params */}
      <div className="px-3 py-2 border-b border-[var(--color-border)]">
        <div className={sectionLabelCls}>Query Params</div>
        {(step.queryParams ?? []).map((q, i) => (
          <KVRow
            key={i}
            pair={q}
            onChange={u => updateQueryParam(i, u)}
            onRemove={() => removeQueryParam(i)}
            keyPlaceholder="param"
            valuePlaceholder="value"
          />
        ))}
        <button type="button" onClick={addQueryParam} className={addBtnCls}>
          <Plus size={10} /> Add param
        </button>
      </div>

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

      {/* Extractions */}
      <div className="px-3 py-2 border-b border-[var(--color-border)]">
        <div className={sectionLabelCls}>Extractions</div>
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
      </div>

      {/* Options: fail policy, timeout, retry */}
      <div className="px-3 py-2 flex flex-wrap gap-x-4 gap-y-2">
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
            title="Delay between retry attempts in milliseconds"
            disabled={(step.retryMaxAttempts ?? 0) === 0}
            className={cn(inputCls, 'w-16', (step.retryMaxAttempts ?? 0) === 0 && 'opacity-40')}
          />
          <span className="text-[var(--color-muted-foreground)] text-xs">ms</span>
        </div>
      </div>
    </div>
  )
}

// ---- Shared helpers ----

const OPERATOR_LABEL: Record<string, string> = {
  eq: '=', ne: '≠', contains: '∋', matches: '~', gt: '>', lt: '<',
}

function assertionSummary(a: { type: string; path?: string; operator?: string; expected?: unknown; schema?: string }): string {
  const op = OPERATOR_LABEL[a.operator ?? 'eq'] ?? a.operator ?? '='
  const exp = a.expected !== undefined && a.expected !== null ? String(a.expected) : ''
  if (a.type === 'status') return `status ${op} ${exp}`
  if (a.type === 'schema') return a.schema ? `schema: ${a.schema}` : 'schema'
  const path = a.path ? a.path.replace(/^\$\./, '') : ''
  return `${a.type === 'jsonpath' ? '$.'+path : path} ${op} ${exp}`.trim()
}

function tryPrettyJSON(s: string): string {
  try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s }
}

function statusColor(code: number): string {
  if (code < 300) return 'text-emerald-400'
  if (code < 400) return 'text-blue-400'
  if (code < 500) return 'text-amber-400'
  return 'text-red-400'
}

// ---- CollapsibleSection ----

function CollapsibleSection({ title, badge, defaultOpen = true, children }: {
  title: React.ReactNode
  badge?: React.ReactNode
  defaultOpen?: boolean
  children: React.ReactNode
}) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div className="border-b border-[var(--color-border)] last:border-0">
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center gap-2 px-3 py-2.5 text-left bg-[var(--color-muted)] hover:bg-[var(--color-border)] transition-colors"
      >
        {open
          ? <ChevronDown size={13} className="shrink-0 text-[var(--color-muted-foreground)]" />
          : <ChevronRight size={13} className="shrink-0 text-[var(--color-muted-foreground)]" />}
        <span className="flex-1 text-xs font-semibold uppercase tracking-wide text-[var(--color-foreground)]">{title}</span>
        {badge && <span className="shrink-0">{badge}</span>}
      </button>
      {open && <div className="px-4 pb-4 pt-2">{children}</div>}
    </div>
  )
}

// ---- StepListRow (compact left-panel row, shared between view and edit modes) ----

function StepListRow({ step, index, result, selected, onSelect, editMode = false, isFirst = false, isLast = false, onMoveUp, onMoveDown, onRemove }: {
  step: StepDetail
  index: number
  result: StepResult | null
  selected: boolean
  onSelect: (i: number) => void
  editMode?: boolean
  isFirst?: boolean
  isLast?: boolean
  onMoveUp?: () => void
  onMoveDown?: () => void
  onRemove?: () => void
}) {
  const badge = step.method ? METHOD_BADGE[step.method] ?? '' : ''
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => onSelect(index)}
      onKeyDown={e => e.key === 'Enter' && onSelect(index)}
      className={cn(
        'group relative flex items-start gap-2 px-3 py-2.5 cursor-pointer transition-colors border-l-2 select-none',
        selected
          ? 'bg-[var(--color-accent)]/10 border-[var(--color-accent)]'
          : 'border-transparent hover:bg-[var(--color-muted)]',
      )}
    >
      <span className="shrink-0 w-5 h-5 mt-0.5 rounded-full bg-[var(--color-border)] text-[var(--color-muted-foreground)] flex items-center justify-center text-[11px] font-bold">
        {index + 1}
      </span>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <span className={cn('text-xs font-medium truncate', selected ? 'text-[var(--color-accent)]' : 'text-[var(--color-foreground)]')}>
            {step.name || 'Untitled step'}
          </span>
          {result && (
            result.skipped
              ? <MinusCircle size={10} className="shrink-0 text-[var(--color-muted-foreground)]" />
              : result.passed
                ? <CheckCircle2 size={10} className="shrink-0 text-emerald-400" />
                : <XCircle size={10} className="shrink-0 text-red-400" />
          )}
        </div>
        {step.requestRef ? (
          <p className="text-xs text-[var(--color-muted-foreground)] truncate font-mono mt-0.5">↗ {step.requestRef}</p>
        ) : step.method ? (
          <div className="flex items-center gap-1 mt-0.5">
            <span className={cn('font-mono font-bold text-xs shrink-0', badge)}>{step.method}</span>
            <span className="text-xs text-[var(--color-muted-foreground)] truncate">{step.url}</span>
          </div>
        ) : null}
      </div>
      {editMode && (
        <div className="shrink-0 flex items-center gap-0 opacity-0 group-hover:opacity-100 transition-opacity">
          <button type="button" onClick={e => { e.stopPropagation(); onMoveUp?.() }} disabled={isFirst} title="Move up"
            className={cn('p-1 transition-colors', isFirst ? 'opacity-20 cursor-not-allowed text-[var(--color-muted-foreground)]' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}>
            <ChevronUp size={11} />
          </button>
          <button type="button" onClick={e => { e.stopPropagation(); onMoveDown?.() }} disabled={isLast} title="Move down"
            className={cn('p-1 transition-colors', isLast ? 'opacity-20 cursor-not-allowed text-[var(--color-muted-foreground)]' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]')}>
            <ChevronDown size={11} />
          </button>
          <button type="button" onClick={e => { e.stopPropagation(); onRemove?.() }} title="Remove step"
            className="p-1 text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors">
            <Trash2 size={11} />
          </button>
        </div>
      )}
    </div>
  )
}

// ---- StepDetailPanel (right panel for view mode) ----

function StepDetailPanel({ step, result }: { step: StepDetail; result: StepResult | null }) {
  const badge = step.method ? METHOD_BADGE[step.method] ?? '' : ''
  const bodyPreview = step.bodyContent ? tryPrettyJSON(step.bodyContent) : null
  const responseBodyFormatted = result?.responseBody ? tryPrettyJSON(result.responseBody) : null
  const contentType = result?.responseHeaders?.find(h => h.name.toLowerCase() === 'content-type')?.value?.split(';')[0]

  return (
    <div className="flex flex-col overflow-y-auto h-full">

      {/* Request */}
      <CollapsibleSection
        title={
          <span className="flex items-center gap-2 min-w-0">
            <span className="shrink-0">Request</span>
            {step.method && <span className={cn('font-mono font-bold text-xs shrink-0', badge)}>{step.method}</span>}
            {step.url && <span className="text-[var(--color-muted-foreground)] font-mono text-xs truncate">{step.url}</span>}
          </span>
        }
        defaultOpen
      >
        {step.requestRef && (
          <p className="text-xs text-[var(--color-muted-foreground)] font-mono mb-2 break-all">↗ {step.requestRef}</p>
        )}
        {step.description && (
          <p className="text-xs text-[var(--color-muted-foreground)] italic mb-2">{step.description}</p>
        )}

        {(step.headers?.length ?? 0) > 0 && (
          <div className="mb-2">
            <p className={sectionLabelCls}>Headers</p>
            {step.headers!.map((h, i) => (
              <div key={i} className="flex items-baseline gap-1 text-xs font-mono mb-0.5">
                <span className="text-[var(--color-muted-foreground)] shrink-0">{h.name}:</span>
                <span className="text-[var(--color-foreground)] break-all">{h.value}</span>
              </div>
            ))}
          </div>
        )}

        {(step.queryParams?.length ?? 0) > 0 && (
          <div className="mb-2">
            <p className={sectionLabelCls}>Query Params</p>
            <div className="flex flex-wrap gap-1">
              {step.queryParams!.map((q, i) => (
                <span key={i} className="font-mono text-xs bg-[var(--color-muted)] border border-[var(--color-border)] rounded px-1.5 py-0.5 text-[var(--color-muted-foreground)]">
                  ?{q.name}={q.value}
                </span>
              ))}
            </div>
          </div>
        )}

        {step.bodyType && step.bodyType !== 'none' && bodyPreview && (
          <div className="mb-2">
            <p className={sectionLabelCls}>Body · {step.bodyType}</p>
            <pre className="text-xs font-mono text-[var(--color-foreground)] bg-[var(--color-background)] border border-[var(--color-border)] rounded px-2 py-1.5 overflow-auto max-h-32 whitespace-pre-wrap break-all">
              {bodyPreview}
            </pre>
          </div>
        )}

        {(step.assertions?.length ?? 0) > 0 && (
          <div className="mb-2">
            <p className={sectionLabelCls}>Assertions</p>
            <div className="flex flex-wrap gap-1">
              {step.assertions.map((a, i) => (
                <span key={i} className="rounded bg-[var(--color-muted)] border border-[var(--color-border)] px-1.5 py-0.5 text-xs text-[var(--color-muted-foreground)]">
                  {assertionSummary(a)}
                </span>
              ))}
            </div>
          </div>
        )}

        {(step.extractions?.length ?? 0) > 0 && (
          <div className="mb-2">
            <p className={sectionLabelCls}>Extractions</p>
            <div className="flex flex-wrap gap-1">
              {step.extractions.map((e, i) => (
                <span key={i} className="rounded bg-violet-500/10 border border-violet-500/20 px-1.5 py-0.5 text-xs text-violet-400">→ {e.name}</span>
              ))}
            </div>
          </div>
        )}

        {((step.failPolicy && step.failPolicy !== 'stop') || (step.timeoutMs && step.timeoutMs > 0) || (step.retryMaxAttempts && step.retryMaxAttempts > 0)) && (
          <div className="flex flex-wrap gap-3 text-xs text-[var(--color-muted-foreground)]">
            {step.failPolicy && step.failPolicy !== 'stop' && <span>on fail: {step.failPolicy}</span>}
            {step.timeoutMs && step.timeoutMs > 0 ? <span>⏱ {step.timeoutMs} ms</span> : null}
            {step.retryMaxAttempts && step.retryMaxAttempts > 0 ? <span>↺ {step.retryMaxAttempts}× · {step.retryDelayMs ?? 1000} ms delay</span> : null}
          </div>
        )}
      </CollapsibleSection>

      {/* Response */}
      {result?.responseBody && (
        <CollapsibleSection
          title="Response"
          badge={contentType ? <span className="text-xs text-[var(--color-muted-foreground)] font-mono">{contentType}</span> : undefined}
          defaultOpen={false}
        >
          {(result.responseHeaders?.length ?? 0) > 0 && (
            <div className="mb-3">
              <p className={sectionLabelCls}>Headers</p>
              <div className="max-h-28 overflow-y-auto flex flex-col gap-0.5">
                {result.responseHeaders!.map((h, i) => (
                  <div key={i} className="flex items-baseline gap-1 text-xs font-mono">
                    <span className="text-[var(--color-muted-foreground)] shrink-0">{h.name}:</span>
                    <span className="text-[var(--color-foreground)] break-all">{h.value}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
          <pre className="text-xs font-mono text-[var(--color-foreground)] bg-[var(--color-background)] border border-[var(--color-border)] rounded px-2 py-1.5 overflow-auto max-h-72 whitespace-pre-wrap break-all">
            {responseBodyFormatted}
          </pre>
        </CollapsibleSection>
      )}

      {/* Result */}
      {result && (
        <CollapsibleSection
          title="Result"
          badge={
            <span className="flex items-center gap-2 text-xs">
              {result.skipped
                ? <span className="text-[var(--color-muted-foreground)]">SKIPPED</span>
                : result.passed
                  ? <span className="font-semibold text-emerald-400">PASSED</span>
                  : <span className="font-semibold text-red-400">FAILED</span>}
              {result.statusCode > 0 && <span className={cn('font-mono font-bold', statusColor(result.statusCode))}>{result.statusCode}</span>}
              {result.durationMs > 0 && <span className="text-[var(--color-muted-foreground)]">{result.durationMs} ms</span>}
            </span>
          }
          defaultOpen
        >
          {result.error && <p className="text-red-400 text-xs font-mono break-all mb-2">{result.error}</p>}
          {result.skipped && <p className="text-xs text-[var(--color-muted-foreground)]">Skipped — previous step failed.</p>}
          <div className="flex flex-col gap-1.5">
            {(result.assertions ?? []).map((ar, i) => {
              const def = step.assertions?.[i]
              const label = def ? assertionSummary(def) : ar.message
              return (
                <div key={i} className="flex items-start gap-2">
                  {ar.passed
                    ? <CheckCircle2 size={14} className="shrink-0 text-emerald-400 mt-0.5" />
                    : <XCircle size={14} className="shrink-0 text-red-400 mt-0.5" />}
                  <div className="flex-1 min-w-0">
                    <span className={cn('text-xs font-mono', ar.passed ? 'text-[var(--color-foreground)]' : 'text-red-300')}>
                      {label}
                    </span>
                    {!ar.passed && ar.message && ar.message !== label && (
                      <p className="text-xs text-red-400 mt-0.5 break-all">{ar.message}</p>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
          {Object.entries(result.extracted ?? {}).length > 0 && (
            <div className="mt-3 flex flex-col gap-1">
              {Object.entries(result.extracted ?? {}).map(([k, v]) => (
                <div key={k} className="flex items-baseline gap-1 text-xs">
                  <span className="text-violet-400 shrink-0">→ {k}:</span>
                  <span className="text-[var(--color-foreground)] font-mono break-all">{v}</span>
                </div>
              ))}
            </div>
          )}
        </CollapsibleSection>
      )}
    </div>
  )
}

// ---- Props ----

type Props = {
  scenarioPath: string
  scenarioName: string
  result: ScenarioResult | null
  running: boolean
  theme?: 'dark' | 'light'
  environments?: EnvInfo[]
  activeEnvPath?: string
  onResultChange: (result: ScenarioResult | null) => void
  onRunningChange: (running: boolean) => void
}

// ---- Main component ----

export function ScenarioViewer({ scenarioPath, scenarioName, result, running, theme = 'dark', environments = [], activeEnvPath = '', onResultChange, onRunningChange }: Props) {
  const [detail, setDetail] = useState<ScenarioDetail | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [selectedEnvPath, setSelectedEnvPath] = useState(activeEnvPath)
  const [selectedStepIndex, setSelectedStepIndex] = useState<number | null>(null)

  const [mode, setMode] = useState<'view' | 'visual' | 'yaml'>('view')

  // YAML editor state
  const [rawContent, setRawContent] = useState('')
  const [editorContent, setEditorContent] = useState('')
  const [loadingRaw, setLoadingRaw] = useState(false)
  const [rawError, setRawError] = useState<string | null>(null)

  // Visual editor state
  const [visualDetail, setVisualDetail] = useState<ScenarioDetail | null>(null)
  const [visualDirty, setVisualDirty] = useState(false)

  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  const [leftPanelWidth, setLeftPanelWidth] = useState(240)
  const containerRef = useRef<HTMLDivElement>(null)

  function handleDividerMouseDown(e: React.MouseEvent) {
    e.preventDefault()
    const startX = e.clientX
    const startWidth = leftPanelWidth
    function onMouseMove(ev: MouseEvent) {
      setLeftPanelWidth(Math.max(160, Math.min(480, startWidth + (ev.clientX - startX))))
    }
    function onMouseUp() {
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
  }

  // Reset when scenario changes
  useEffect(() => {
    setDetail(null)
    setLoadError(null)
    setMode('view')
    setRawContent('')
    setEditorContent('')
    setRawError(null)
    setSaveError(null)
    setVisualDetail(null)
    setVisualDirty(false)
    setSelectedEnvPath(activeEnvPath)
    setSelectedStepIndex(null)
    loadScenario(scenarioPath)
      .then(d => { if (d.error) setLoadError(d.error); else setDetail(d) })
      .catch(e => setLoadError(String(e)))
    loadScenarioResult(scenarioPath)
      .then(cached => { if (cached) onResultChange(cached) })
      .catch(() => {})
  }, [scenarioPath]) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-select first step once detail loads
  useEffect(() => {
    if (detail && selectedStepIndex === null && detail.steps.length > 0) {
      setSelectedStepIndex(0)
    }
  }, [detail]) // eslint-disable-line react-hooks/exhaustive-deps

  async function handleRun() {
    onRunningChange(true)
    onResultChange(null)
    try {
      const r = await runScenario(scenarioPath, selectedEnvPath)
      onResultChange(r)
      saveScenarioResult(scenarioPath, r).catch(() => {})
      // Auto-focus first failed step, else keep current selection
      const firstFailed = r.steps.findIndex(s => !s.passed && !s.skipped)
      if (firstFailed >= 0) setSelectedStepIndex(firstFailed)
    } catch (e) {
      onResultChange({ name: scenarioName, passed: false, durationMs: 0, steps: [], error: String(e) })
    } finally {
      onRunningChange(false)
    }
  }

  async function handleEnterYaml() {
    setLoadingRaw(true)
    setRawError(null)
    try {
      const raw = await loadScenarioRaw(scenarioPath)
      setRawContent(raw)
      setEditorContent(raw)
      setMode('yaml')
    } catch (e) {
      setRawError(String(e))
    } finally {
      setLoadingRaw(false)
    }
  }

  function handleEnterVisual() {
    if (!detail) return
    const vd: ScenarioDetail = {
      ...detail,
      steps: detail.steps.map(s => ({
        ...s,
        description: s.description ?? '',
        headers: s.headers ? [...s.headers] : [],
        queryParams: s.queryParams ? [...s.queryParams] : [],
        bodyType: s.bodyType ?? 'none',
        bodyContent: s.bodyContent ?? '',
        assertions: [...(s.assertions ?? [])],
        extractions: [...(s.extractions ?? [])],
        timeoutMs: s.timeoutMs ?? 0,
        retryMaxAttempts: s.retryMaxAttempts ?? 0,
        retryDelayMs: s.retryDelayMs ?? 1000,
      })),
    }
    setVisualDetail(vd)
    setVisualDirty(false)
    if (selectedStepIndex === null && vd.steps.length > 0) setSelectedStepIndex(0)
    setMode('visual')
  }

  // Visual editor step management
  function updateVisualStep(i: number, updates: Partial<StepDetail>) {
    if (!visualDetail) return
    setVisualDetail({ ...visualDetail, steps: visualDetail.steps.map((s, idx) => idx === i ? { ...s, ...updates } : s) })
    setVisualDirty(true)
  }

  function addVisualStep() {
    if (!visualDetail) return
    const newStep: StepDetail = {
      name: `Step ${visualDetail.steps.length + 1}`,
      description: '',
      method: 'GET',
      url: '',
      headers: [],
      queryParams: [],
      bodyType: 'none',
      bodyContent: '',
      assertions: [{ type: 'status', operator: 'eq', expected: 200 }],
      extractions: [],
      timeoutMs: 0,
      retryMaxAttempts: 0,
      retryDelayMs: 1000,
    }
    const newSteps = [...visualDetail.steps, newStep]
    setVisualDetail({ ...visualDetail, steps: newSteps })
    setVisualDirty(true)
    setSelectedStepIndex(newSteps.length - 1)
  }

  function removeVisualStep(i: number) {
    if (!visualDetail) return
    const newSteps = visualDetail.steps.filter((_, idx) => idx !== i)
    setVisualDetail({ ...visualDetail, steps: newSteps })
    setVisualDirty(true)
    if (selectedStepIndex === i) {
      setSelectedStepIndex(newSteps.length > 0 ? Math.min(i, newSteps.length - 1) : null)
    } else if (selectedStepIndex !== null && selectedStepIndex > i) {
      setSelectedStepIndex(selectedStepIndex - 1)
    }
  }

  function moveVisualStep(from: number, dir: -1 | 1) {
    if (!visualDetail) return
    const to = from + dir
    if (to < 0 || to >= visualDetail.steps.length) return
    const steps = [...visualDetail.steps]
    ;[steps[from], steps[to]] = [steps[to], steps[from]]
    setVisualDetail({ ...visualDetail, steps })
    setVisualDirty(true)
    if (selectedStepIndex === from) setSelectedStepIndex(to)
    else if (selectedStepIndex === to) setSelectedStepIndex(from)
  }

  async function handleSave() {
    setSaving(true)
    setSaveError(null)
    try {
      const content = mode === 'yaml' ? editorContent : serializeScenarioToYAML(visualDetail!)
      const err = await saveScenarioRaw(scenarioPath, content)
      if (err) { setSaveError(err); return }
      if (mode === 'yaml') setRawContent(editorContent)
      const d = await loadScenario(scenarioPath)
      if (d.error) setLoadError(d.error); else setDetail(d)
      onResultChange(null)
      setMode('view')
    } catch (e) {
      setSaveError(String(e))
    } finally {
      setSaving(false)
    }
  }

  function handleDiscard() {
    if (mode === 'yaml') setEditorContent(rawContent)
    setSaveError(null)
    setMode('view')
  }

  const isEditing = mode === 'visual' || mode === 'yaml'
  const isDirty = (mode === 'yaml' && editorContent !== rawContent) || (mode === 'visual' && visualDirty)

  const passed = result ? result.steps.filter(s => s.passed && !s.skipped).length : 0
  const failed = result ? result.steps.filter(s => !s.passed && !s.skipped).length : 0
  const skipped = result ? result.steps.filter(s => s.skipped).length : 0

  const toggleActiveCls = 'bg-[var(--color-accent)] text-white'
  const toggleInactiveCls = 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]'

  // Current steps list (view: from loaded detail, edit: from visualDetail)
  const viewSteps = detail?.steps ?? []
  const editSteps = visualDetail?.steps ?? []

  return (
    <div className="flex flex-col h-full w-full overflow-hidden">

      {/* Header */}
      <div className="shrink-0 flex items-center gap-2 px-4 py-2.5 border-b border-[var(--color-border)] bg-[var(--color-card)]">
        <div className="flex-1 min-w-0">
          <h2 className="text-sm font-semibold text-[var(--color-foreground)] truncate">
            {detail?.name ?? scenarioName}
          </h2>
        </div>

        {/* View / Edit / YAML toggle */}
        <div className="shrink-0 flex rounded overflow-hidden border border-[var(--color-border)] text-[11px]">
          <button type="button" onClick={() => setMode('view')} disabled={mode === 'view' || saving}
            className={cn('px-2.5 py-1 transition-colors', mode === 'view' ? toggleActiveCls : toggleInactiveCls)}>
            View
          </button>
          <button type="button" onClick={handleEnterVisual} disabled={mode === 'visual' || saving || !detail}
            className={cn('px-2.5 py-1 transition-colors', mode === 'visual' ? toggleActiveCls : toggleInactiveCls)}>
            Edit
          </button>
          <button type="button" onClick={handleEnterYaml} disabled={mode === 'yaml' || loadingRaw || saving}
            className={cn('px-2.5 py-1 transition-colors flex items-center gap-1', mode === 'yaml' ? toggleActiveCls : toggleInactiveCls)}>
            {loadingRaw ? <Loader2 size={10} className="animate-spin" /> : null}
            YAML
          </button>
        </div>

        {/* Controls */}
        {!isEditing ? (
          <>
            {environments.length > 0 && (
              <select value={selectedEnvPath} onChange={e => setSelectedEnvPath(e.target.value)} title="Environment for this run"
                className={cn('shrink-0 text-xs rounded px-2 py-0.5 border outline-none cursor-pointer',
                  'bg-[var(--color-input)] border-[var(--color-input-border)]',
                  selectedEnvPath ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)]')}>
                <option value="">No environment</option>
                {environments.map(env => <option key={env.path} value={env.path}>{env.name}</option>)}
              </select>
            )}
            {result && !running && (
              <div className="shrink-0 flex items-center gap-2">
                <span className={cn('text-xs font-semibold px-2 py-0.5 rounded',
                  result.passed ? 'bg-emerald-500/15 text-emerald-400' : 'bg-red-500/15 text-red-400')}>
                  {result.passed ? 'PASSED' : 'FAILED'}
                </span>
                <span className="text-xs text-[var(--color-muted-foreground)]">
                  {passed}/{passed + failed + skipped} · {result.durationMs} ms
                </span>
              </div>
            )}
            <button type="button" onClick={handleRun} disabled={running || !detail}
              className={cn('shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold transition-opacity',
                'bg-[var(--color-accent)] text-white',
                (running || !detail) ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90')}>
              {running ? <><Loader2 size={12} className="animate-spin" /> Running…</> : <><Play size={12} /> Run</>}
            </button>
          </>
        ) : (
          <>
            {saveError && <span className="text-xs text-red-400 shrink-0 max-w-[200px] truncate" title={saveError}>{saveError}</span>}
            <button type="button" onClick={handleDiscard} disabled={saving}
              className="shrink-0 flex items-center gap-1 px-2.5 py-1.5 rounded text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">
              <X size={11} /> Discard
            </button>
            <button type="button" onClick={handleSave} disabled={saving || (mode === 'yaml' && !isDirty)}
              className={cn('shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold transition-opacity',
                'bg-[var(--color-accent)] text-white',
                (saving || (mode === 'yaml' && !isDirty)) ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90')}>
              {saving ? <><Loader2 size={12} className="animate-spin" /> Saving…</> : <><Save size={12} /> Save</>}
            </button>
          </>
        )}
      </div>

      {/* Body */}
      {mode === 'yaml' ? (
        <div className="flex-1 overflow-hidden">
          {rawError
            ? <p className="px-3 py-2 text-xs text-red-400">{rawError}</p>
            : <CodeMirror value={editorContent} extensions={[yaml()]} theme={theme === 'light' ? 'light' : 'dark'}
                onChange={v => setEditorContent(v)} height="100%" style={{ height: '100%', fontSize: '12px' }} />}
        </div>
      ) : (
        <div ref={containerRef} className="flex-1 flex overflow-hidden">

          {/* Left panel: step list */}
          <div className="shrink-0 flex flex-col overflow-hidden" style={{ width: leftPanelWidth }}>

            {/* Edit mode: scenario meta */}
            {mode === 'visual' && visualDetail && (
              <div className="shrink-0 px-3 py-2 border-b border-[var(--color-border)] bg-[var(--color-card)]">
                <input
                  value={visualDetail.name}
                  onChange={e => { setVisualDetail({ ...visualDetail, name: e.target.value }); setVisualDirty(true) }}
                  placeholder="Scenario name"
                  className="w-full bg-transparent border-b border-transparent focus:border-[var(--color-accent)] outline-none text-xs font-semibold text-[var(--color-foreground)] py-0.5 mb-0.5"
                />
                <input
                  value={visualDetail.description}
                  onChange={e => { setVisualDetail({ ...visualDetail, description: e.target.value }); setVisualDirty(true) }}
                  placeholder="Description (optional)"
                  className="w-full bg-transparent border-b border-transparent focus:border-[var(--color-accent)] outline-none text-xs text-[var(--color-muted-foreground)] py-0.5"
                />
              </div>
            )}

            {/* Step list */}
            <div className="flex-1 overflow-y-auto py-1">
              {loadError ? (
                <p className="px-3 py-2 text-xs text-red-400">{loadError}</p>
              ) : mode === 'view' && !detail ? (
                <div className="px-3 py-2 flex items-center gap-2 text-xs text-[var(--color-muted-foreground)]">
                  <Loader2 size={12} className="animate-spin" /> Loading…
                </div>
              ) : mode === 'view' ? (
                viewSteps.length === 0 ? (
                  <p className="px-3 py-4 text-xs text-center text-[var(--color-muted-foreground)]">
                    No steps. Click <strong>Edit</strong> to add one.
                  </p>
                ) : (
                  viewSteps.map((step, i) => (
                    <StepListRow
                      key={i} step={step} index={i}
                      result={result?.steps[i] ?? null}
                      selected={selectedStepIndex === i}
                      onSelect={setSelectedStepIndex}
                    />
                  ))
                )
              ) : (
                // Visual edit mode
                editSteps.length === 0 ? (
                  <p className="px-3 py-4 text-xs text-center text-[var(--color-muted-foreground)]">No steps yet.</p>
                ) : (
                  editSteps.map((step, i) => (
                    <StepListRow
                      key={i} step={step} index={i}
                      result={result?.steps[i] ?? null}
                      selected={selectedStepIndex === i}
                      onSelect={setSelectedStepIndex}
                      editMode
                      isFirst={i === 0}
                      isLast={i === editSteps.length - 1}
                      onMoveUp={() => moveVisualStep(i, -1)}
                      onMoveDown={() => moveVisualStep(i, 1)}
                      onRemove={() => removeVisualStep(i)}
                    />
                  ))
                )
              )}
            </div>

            {/* Edit mode: add step */}
            {mode === 'visual' && (
              <div className="shrink-0 px-2 py-2 border-t border-[var(--color-border)]">
                <button type="button" onClick={addVisualStep}
                  className="w-full flex items-center justify-center gap-1.5 py-2 rounded bg-[var(--color-accent)] text-white text-xs font-semibold hover:opacity-90 transition-opacity">
                  <Plus size={13} /> Add Step
                </button>
              </div>
            )}
          </div>

          {/* Draggable divider */}
          <div
            onMouseDown={handleDividerMouseDown}
            className="group w-[5px] shrink-0 cursor-col-resize flex items-center justify-center hover:bg-[var(--color-accent)]/15 transition-colors active:bg-[var(--color-accent)]/25"
            title="Drag to resize"
          >
            <div className="w-px h-full bg-[var(--color-border)] group-hover:bg-[var(--color-accent)] transition-colors" />
          </div>

          {/* Right panel */}
          <div className="flex-1 overflow-hidden min-w-0">
            {mode === 'view' ? (
              running ? (
                <div className="h-full flex flex-col items-center justify-center gap-3 text-[var(--color-muted-foreground)]">
                  <Loader2 size={20} className="animate-spin" />
                  <p className="text-sm">Running scenario…</p>
                </div>
              ) : selectedStepIndex !== null && detail?.steps[selectedStepIndex] ? (
                <StepDetailPanel
                  step={detail.steps[selectedStepIndex]}
                  result={result?.steps[selectedStepIndex] ?? null}
                />
              ) : (
                <div className="h-full flex flex-col items-center justify-center text-xs text-[var(--color-muted-foreground)]">
                  <Play size={24} className="mb-2 opacity-30" />
                  {detail && detail.steps.length > 0
                    ? 'Select a step to see its details.'
                    : <span>Click <strong>Edit</strong> to add steps, then <strong>Run</strong>.</span>}
                </div>
              )
            ) : (
              // Visual edit mode
              selectedStepIndex !== null && visualDetail?.steps[selectedStepIndex] ? (
                <div className="h-full overflow-y-auto p-3">
                  <StepEditorCard
                    step={visualDetail.steps[selectedStepIndex]}
                    index={selectedStepIndex}
                    isFirst={selectedStepIndex === 0}
                    isLast={selectedStepIndex === editSteps.length - 1}
                    onChange={updates => updateVisualStep(selectedStepIndex, updates)}
                    onRemove={() => removeVisualStep(selectedStepIndex)}
                    onMoveUp={() => moveVisualStep(selectedStepIndex, -1)}
                    onMoveDown={() => moveVisualStep(selectedStepIndex, 1)}
                  />
                </div>
              ) : (
                <div className="h-full flex flex-col items-center justify-center text-xs text-[var(--color-muted-foreground)]">
                  <Plus size={24} className="mb-2 opacity-30" />
                  Select a step to edit, or add a new one.
                </div>
              )
            )}
          </div>

        </div>
      )}
    </div>
  )
}
