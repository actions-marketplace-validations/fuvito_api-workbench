import { useState, useEffect } from 'react'
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

const sectionLabelCls = 'text-[10px] font-semibold uppercase tracking-wider text-[var(--color-muted-foreground)] mb-1.5'
const addBtnCls = 'flex items-center gap-1 text-[11px] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors mt-1'

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
          <span className="shrink-0 w-5 h-5 rounded-full bg-[var(--color-border)] text-[var(--color-muted-foreground)] flex items-center justify-center text-[9px] font-bold">
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
          className="w-full mt-1 bg-transparent border-b border-transparent focus:border-[var(--color-accent)] outline-none text-[11px] text-[var(--color-muted-foreground)] py-0.5 italic placeholder:not-italic"
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
            className={cn(selectCls, 'text-[10px]')}
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
              className={cn('shrink-0 text-[10px] px-1.5 py-0.5 rounded border border-[var(--color-border)] transition-colors',
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
      <div className="px-3 py-2 flex flex-wrap gap-x-4 gap-y-1.5">
        <div className="flex items-center gap-1.5">
          <span className="text-[var(--color-muted-foreground)] text-[10px] shrink-0">On failure:</span>
          <select
            value={step.failPolicy ?? 'stop'}
            onChange={e => onChange({ failPolicy: e.target.value })}
            className={cn(selectCls, 'text-[10px]')}
          >
            <option value="stop">stop remaining steps</option>
            <option value="continue">continue next steps</option>
            <option value="skip">skip this step if prev failed</option>
          </select>
        </div>
        <div className="flex items-center gap-1">
          <span className="text-[var(--color-muted-foreground)] text-[10px] shrink-0">Timeout:</span>
          <input
            type="number"
            min={0}
            value={step.timeoutMs ?? 0}
            onChange={e => onChange({ timeoutMs: Math.max(0, parseInt(e.target.value) || 0) })}
            title="Step timeout in milliseconds (0 = use global default)"
            className={cn(inputCls, 'w-16 text-[10px]')}
          />
          <span className="text-[var(--color-muted-foreground)] text-[10px]">ms</span>
        </div>
        <div className="flex items-center gap-1">
          <span className="text-[var(--color-muted-foreground)] text-[10px] shrink-0">Retry:</span>
          <input
            type="number"
            min={0}
            max={10}
            value={step.retryMaxAttempts ?? 0}
            onChange={e => onChange({ retryMaxAttempts: Math.max(0, parseInt(e.target.value) || 0) })}
            title="Maximum retry attempts (0 = no retry)"
            className={cn(inputCls, 'w-10 text-[10px]')}
          />
          <span className="text-[var(--color-muted-foreground)] text-[10px]">× delay</span>
          <input
            type="number"
            min={0}
            value={step.retryDelayMs ?? 1000}
            onChange={e => onChange({ retryDelayMs: Math.max(0, parseInt(e.target.value) || 0) })}
            title="Delay between retry attempts in milliseconds"
            disabled={(step.retryMaxAttempts ?? 0) === 0}
            className={cn(inputCls, 'w-16 text-[10px]', (step.retryMaxAttempts ?? 0) === 0 && 'opacity-40')}
          />
          <span className="text-[var(--color-muted-foreground)] text-[10px]">ms</span>
        </div>
      </div>
    </div>
  )
}

function VisualEditor({ value, onChange }: { value: ScenarioDetail; onChange: (d: ScenarioDetail) => void }) {
  function update<K extends keyof ScenarioDetail>(key: K, val: ScenarioDetail[K]) {
    onChange({ ...value, [key]: val })
  }

  function updateStep(i: number, updates: Partial<StepDetail>) {
    update('steps', value.steps.map((s, idx) => idx === i ? { ...s, ...updates } : s))
  }

  function addStep() {
    update('steps', [...value.steps, {
      name: `Step ${value.steps.length + 1}`,
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
    }])
  }

  function removeStep(i: number) {
    update('steps', value.steps.filter((_, idx) => idx !== i))
  }

  function moveStep(from: number, dir: -1 | 1) {
    const to = from + dir
    if (to < 0 || to >= value.steps.length) return
    const steps = [...value.steps]
    ;[steps[from], steps[to]] = [steps[to], steps[from]]
    update('steps', steps)
  }

  return (
    <div className="flex-1 overflow-y-auto py-3 px-3 flex flex-col gap-2">
      {/* Scenario meta */}
      <div className="flex flex-col gap-1.5 pb-3 mb-1 border-b border-[var(--color-border)]">
        <input
          value={value.name}
          onChange={e => update('name', e.target.value)}
          placeholder="Scenario name"
          className="bg-transparent border-b border-transparent focus:border-[var(--color-accent)] outline-none text-sm font-semibold text-[var(--color-foreground)] py-0.5"
        />
        <input
          value={value.description}
          onChange={e => update('description', e.target.value)}
          placeholder="Description (optional)"
          className="bg-transparent border-b border-transparent focus:border-[var(--color-accent)] outline-none text-xs text-[var(--color-muted-foreground)] py-0.5"
        />
      </div>

      {/* Steps */}
      {value.steps.map((step, i) => (
        <StepEditorCard
          key={i}
          step={step}
          index={i}
          isFirst={i === 0}
          isLast={i === value.steps.length - 1}
          onChange={updates => updateStep(i, updates)}
          onRemove={() => removeStep(i)}
          onMoveUp={() => moveStep(i, -1)}
          onMoveDown={() => moveStep(i, 1)}
        />
      ))}

      {/* Add step */}
      <button
        type="button"
        onClick={addStep}
        className={cn(
          'flex items-center justify-center gap-1.5 py-2 rounded border border-dashed border-[var(--color-border)]',
          'text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:border-[var(--color-accent)] transition-colors',
        )}
      >
        <Plus size={12} /> Add step
      </button>
    </div>
  )
}

// ---- Main view sub-components ----

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

function StepCard({ step, index, result }: {
  step: ScenarioDetail['steps'][number]
  index: number
  result: StepResult | null
}) {
  const badge = step.method ? METHOD_BADGE[step.method] ?? 'text-[var(--color-muted-foreground)]' : ''
  return (
    <div className={cn(
      'mx-2 mb-1.5 rounded border text-xs',
      'bg-[var(--color-muted)] border-[var(--color-border)]',
    )}>
      <div className="flex items-start gap-2 px-2.5 py-2">
        <span className="shrink-0 w-4 h-4 rounded-full bg-[var(--color-border)] text-[var(--color-muted-foreground)] flex items-center justify-center text-[9px] font-bold mt-0.5">
          {index + 1}
        </span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5 mb-0.5">
            <span className="font-semibold text-[var(--color-foreground)]">{step.name}</span>
            {result && (
              result.skipped
                ? <MinusCircle size={11} className="text-[var(--color-muted-foreground)] shrink-0" />
                : result.passed
                  ? <CheckCircle2 size={11} className="text-emerald-400 shrink-0" />
                  : <XCircle size={11} className="text-red-400 shrink-0" />
            )}
          </div>
          {step.description && (
            <p className="text-[11px] text-[var(--color-muted-foreground)] italic leading-snug mb-0.5">{step.description}</p>
          )}
          {step.requestRef ? (
            <p className="text-[var(--color-muted-foreground)] truncate">ref: {step.requestRef}</p>
          ) : step.method ? (
            <p className="flex items-center gap-1 truncate">
              <span className={cn('font-mono font-bold text-[10px] shrink-0', badge)}>{step.method}</span>
              <span className="text-[var(--color-muted-foreground)] truncate">{step.url}</span>
              {step.bodyType && step.bodyType !== 'none' && (
                <span className="shrink-0 rounded bg-[var(--color-background)] border border-[var(--color-border)] px-1 py-0.5 text-[10px] text-[var(--color-muted-foreground)]">
                  {step.bodyType}
                </span>
              )}
            </p>
          ) : null}
          {step.headers && step.headers.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-1">
              {step.headers.map((h, i) => (
                <span key={i} className="rounded bg-[var(--color-background)] border border-[var(--color-border)] px-1 py-0.5 text-[10px] text-[var(--color-muted-foreground)] font-mono">
                  {h.name}: {h.value}
                </span>
              ))}
            </div>
          )}
          {step.queryParams && step.queryParams.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-1">
              {step.queryParams.map((q, i) => (
                <span key={i} className="rounded bg-[var(--color-background)] border border-[var(--color-border)] px-1 py-0.5 text-[10px] text-[var(--color-muted-foreground)] font-mono">
                  ?{q.name}={q.value}
                </span>
              ))}
            </div>
          )}
          {(step.assertions?.length ?? 0) > 0 && (
            <div className="flex flex-wrap gap-1 mt-1">
              {step.assertions.map((a, i) => (
                <span key={i} className="rounded bg-[var(--color-background)] border border-[var(--color-border)] px-1 py-0.5 text-[10px] text-[var(--color-muted-foreground)]">
                  {assertionSummary(a)}
                </span>
              ))}
            </div>
          )}
          {(step.extractions?.length ?? 0) > 0 && (
            <div className="flex flex-wrap gap-1 mt-1">
              {step.extractions.map((e, i) => (
                <span key={i} className="rounded bg-violet-500/10 border border-violet-500/20 px-1 py-0.5 text-[10px] text-violet-400">
                  → {e.name}
                </span>
              ))}
            </div>
          )}
          {(step.failPolicy && step.failPolicy !== 'stop') || (step.timeoutMs && step.timeoutMs > 0) || (step.retryMaxAttempts && step.retryMaxAttempts > 0) ? (
            <div className="flex flex-wrap gap-1.5 mt-0.5">
              {step.failPolicy && step.failPolicy !== 'stop' && (
                <span className="text-[10px] text-[var(--color-muted-foreground)]">on fail: {step.failPolicy}</span>
              )}
              {step.timeoutMs && step.timeoutMs > 0 ? (
                <span className="text-[10px] text-[var(--color-muted-foreground)]">⏱ {step.timeoutMs} ms</span>
              ) : null}
              {step.retryMaxAttempts && step.retryMaxAttempts > 0 ? (
                <span className="text-[10px] text-[var(--color-muted-foreground)]">↺ {step.retryMaxAttempts}×{step.retryDelayMs ? ` ${step.retryDelayMs} ms` : ''}</span>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}

function StepResultCard({ sr, index }: { sr: StepResult; index: number }) {
  const [expanded, setExpanded] = useState(true)

  return (
    <div className={cn(
      'mx-2 mb-1.5 rounded border text-xs',
      sr.skipped
        ? 'bg-[var(--color-muted)] border-[var(--color-border)] opacity-60'
        : sr.passed
          ? 'bg-emerald-500/5 border-emerald-500/20'
          : 'bg-red-500/5 border-red-500/20',
    )}>
      <button
        type="button"
        className="w-full flex items-center gap-2 px-2.5 py-2 text-left"
        onClick={() => setExpanded(e => !e)}
      >
        <span className="shrink-0 w-4 h-4 rounded-full bg-[var(--color-border)] text-[var(--color-muted-foreground)] flex items-center justify-center text-[9px] font-bold">
          {index + 1}
        </span>
        {sr.skipped ? (
          <MinusCircle size={12} className="shrink-0 text-[var(--color-muted-foreground)]" />
        ) : sr.passed ? (
          <CheckCircle2 size={12} className="shrink-0 text-emerald-400" />
        ) : (
          <XCircle size={12} className="shrink-0 text-red-400" />
        )}
        <span className="flex-1 font-semibold text-[var(--color-foreground)]">{sr.name}</span>
        {sr.statusCode > 0 && (
          <span className={cn(
            'shrink-0 font-mono text-[10px] font-bold',
            sr.statusCode < 300 ? 'text-emerald-400' : sr.statusCode < 400 ? 'text-blue-400' : sr.statusCode < 500 ? 'text-amber-400' : 'text-red-400',
          )}>{sr.statusCode}</span>
        )}
        {sr.durationMs > 0 && (
          <span className="shrink-0 text-[var(--color-muted-foreground)] text-[10px]">{sr.durationMs} ms</span>
        )}
        {expanded ? <ChevronDown size={10} className="shrink-0 text-[var(--color-muted-foreground)]" /> : <ChevronRight size={10} className="shrink-0 text-[var(--color-muted-foreground)]" />}
      </button>

      {expanded && (
        <div className="px-2.5 pb-2 pt-0 flex flex-col gap-1">
          {sr.error && (
            <p className="text-red-400 text-[11px] font-mono break-all">{sr.error}</p>
          )}
          {(sr.assertions ?? []).map((ar, i) => (
            <div key={i} className="flex items-start gap-1.5">
              {ar.passed
                ? <CheckCircle2 size={10} className="shrink-0 text-emerald-400 mt-0.5" />
                : <XCircle size={10} className="shrink-0 text-red-400 mt-0.5" />}
              <span className={cn('text-[11px] break-all', ar.passed ? 'text-[var(--color-muted-foreground)]' : 'text-red-300')}>
                {ar.message}
              </span>
            </div>
          ))}
          {Object.entries(sr.extracted ?? {}).map(([k, v]) => (
            <div key={k} className="flex items-baseline gap-1 text-[11px]">
              <span className="text-violet-400 shrink-0">→ {k}:</span>
              <span className="text-[var(--color-foreground)] font-mono break-all">{v}</span>
            </div>
          ))}
          {sr.skipped && (
            <p className="text-[var(--color-muted-foreground)] text-[11px]">Skipped — previous step failed.</p>
          )}
        </div>
      )}
    </div>
  )
}

// ---- Main component ----

export function ScenarioViewer({ scenarioPath, scenarioName, result, running, theme = 'dark', environments = [], activeEnvPath = '', onResultChange, onRunningChange }: Props) {
  const [detail, setDetail] = useState<ScenarioDetail | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [selectedEnvPath, setSelectedEnvPath] = useState(activeEnvPath)

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
    loadScenario(scenarioPath)
      .then(d => { if (d.error) setLoadError(d.error); else setDetail(d) })
      .catch(e => setLoadError(String(e)))
    loadScenarioResult(scenarioPath)
      .then(cached => { if (cached) onResultChange(cached) })
      .catch(() => {})
  }, [scenarioPath]) // eslint-disable-line react-hooks/exhaustive-deps

  async function handleRun() {
    onRunningChange(true)
    onResultChange(null)
    try {
      const r = await runScenario(scenarioPath, selectedEnvPath)
      onResultChange(r)
      saveScenarioResult(scenarioPath, r).catch(() => {})
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
    setVisualDetail({
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
    })
    setVisualDirty(false)
    setMode('visual')
  }

  function handleVisualChange(d: ScenarioDetail) {
    setVisualDetail(d)
    setVisualDirty(true)
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
          <button
            type="button"
            onClick={() => setMode('view')}
            disabled={mode === 'view' || saving}
            className={cn('px-2.5 py-1 transition-colors', mode === 'view' ? toggleActiveCls : toggleInactiveCls)}
          >
            View
          </button>
          <button
            type="button"
            onClick={handleEnterVisual}
            disabled={mode === 'visual' || saving || !detail}
            className={cn('px-2.5 py-1 transition-colors', mode === 'visual' ? toggleActiveCls : toggleInactiveCls)}
          >
            Edit
          </button>
          <button
            type="button"
            onClick={handleEnterYaml}
            disabled={mode === 'yaml' || loadingRaw || saving}
            className={cn('px-2.5 py-1 transition-colors flex items-center gap-1', mode === 'yaml' ? toggleActiveCls : toggleInactiveCls)}
          >
            {loadingRaw ? <Loader2 size={10} className="animate-spin" /> : null}
            YAML
          </button>
        </div>

        {/* Right side: view controls or save/discard */}
        {!isEditing ? (
          <>
            {environments.length > 0 && (
              <select
                value={selectedEnvPath}
                onChange={e => setSelectedEnvPath(e.target.value)}
                title="Environment for this run"
                className={cn(
                  'shrink-0 bg-[var(--color-muted)] text-xs rounded px-2 py-0.5',
                  'border border-[var(--color-border)] outline-none cursor-pointer',
                  selectedEnvPath ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)]',
                )}
              >
                <option value="">No environment</option>
                {environments.map(env => (
                  <option key={env.path} value={env.path}>{env.name}</option>
                ))}
              </select>
            )}
            {result && !running && (
              <div className="shrink-0 flex items-center gap-2">
                <span className={cn(
                  'text-xs font-semibold px-2 py-0.5 rounded',
                  result.passed ? 'bg-emerald-500/15 text-emerald-400' : 'bg-red-500/15 text-red-400',
                )}>
                  {result.passed ? 'PASSED' : 'FAILED'}
                </span>
                <span className="text-xs text-[var(--color-muted-foreground)]">
                  {passed}/{passed + failed + skipped} steps · {result.durationMs} ms
                </span>
              </div>
            )}
            <button
              type="button"
              onClick={handleRun}
              disabled={running || !detail}
              className={cn(
                'shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold transition-opacity',
                'bg-[var(--color-accent)] text-white',
                (running || !detail) ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90',
              )}
            >
              {running
                ? <><Loader2 size={12} className="animate-spin" /> Running…</>
                : <><Play size={12} /> Run</>}
            </button>
          </>
        ) : (
          <>
            {saveError && (
              <span className="text-xs text-red-400 shrink-0 max-w-[200px] truncate" title={saveError}>{saveError}</span>
            )}
            <button
              type="button"
              onClick={handleDiscard}
              disabled={saving}
              className="shrink-0 flex items-center gap-1 px-2.5 py-1.5 rounded text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
            >
              <X size={11} /> Discard
            </button>
            <button
              type="button"
              onClick={handleSave}
              disabled={saving || (mode === 'yaml' && !isDirty)}
              className={cn(
                'shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold transition-opacity',
                'bg-[var(--color-accent)] text-white',
                (saving || (mode === 'yaml' && !isDirty)) ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90',
              )}
            >
              {saving
                ? <><Loader2 size={12} className="animate-spin" /> Saving…</>
                : <><Save size={12} /> Save</>}
            </button>
          </>
        )}
      </div>

      {/* Body */}
      {mode === 'yaml' ? (
        <div className="flex-1 overflow-hidden">
          {rawError ? (
            <p className="px-3 py-2 text-xs text-red-400">{rawError}</p>
          ) : (
            <CodeMirror
              value={editorContent}
              extensions={[yaml()]}
              theme={theme === 'light' ? 'light' : 'dark'}
              onChange={v => setEditorContent(v)}
              height="100%"
              style={{ height: '100%', fontSize: '12px' }}
            />
          )}
        </div>
      ) : mode === 'visual' && visualDetail ? (
        <VisualEditor value={visualDetail} onChange={handleVisualChange} />
      ) : (
        <div className="flex-1 flex overflow-hidden">
          {/* Left: step definitions */}
          <div className="w-[42%] shrink-0 border-r border-[var(--color-border)] overflow-y-auto py-2">
            {loadError ? (
              <p className="px-3 py-2 text-xs text-red-400">{loadError}</p>
            ) : !detail ? (
              <div className="px-3 py-2 flex items-center gap-2 text-xs text-[var(--color-muted-foreground)]">
                <Loader2 size={12} className="animate-spin" /> Loading…
              </div>
            ) : (
              <>
                {detail.description && (
                  <div className="mx-2 mb-2 px-2.5 py-2 rounded bg-[var(--color-muted)] border border-[var(--color-border)]">
                    <p className="text-xs text-[var(--color-muted-foreground)] leading-relaxed whitespace-pre-wrap">{detail.description}</p>
                  </div>
                )}
                {detail.steps.length === 0 ? (
                  <p className="px-3 py-2 text-xs text-[var(--color-muted-foreground)]">No steps defined.</p>
                ) : (
                  detail.steps.map((step, i) => (
                    <StepCard
                      key={i}
                      step={step}
                      index={i}
                      result={result?.steps[i] ?? null}
                    />
                  ))
                )}
              </>
            )}
          </div>

          {/* Right: results */}
          <div className="flex-1 overflow-y-auto py-2 min-w-0">
            {running ? (
              <div className="px-4 py-6 flex flex-col items-center gap-3 text-[var(--color-muted-foreground)]">
                <Loader2 size={20} className="animate-spin" />
                <p className="text-sm">Running scenario…</p>
              </div>
            ) : !result ? (
              <div className="px-4 py-6 text-center text-xs text-[var(--color-muted-foreground)]">
                <Play size={24} className="mx-auto mb-2 opacity-30" />
                Click <strong>Run</strong> to execute this scenario.
              </div>
            ) : result.error ? (
              <div className="mx-2 p-3 rounded border border-red-500/30 bg-red-500/5 text-xs text-red-400">
                {result.error}
              </div>
            ) : (
              result.steps.map((sr, i) => (
                <StepResultCard key={i} sr={sr} index={i} />
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}
