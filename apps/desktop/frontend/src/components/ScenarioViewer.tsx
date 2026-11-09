import { useState, useEffect, useRef } from 'react'
import { Play, Loader2, CheckCircle2, XCircle, Save, X, Plus, Clock } from 'lucide-react'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { cn } from '@/lib/utils'
import { loadScenario, runScenario, loadScenarioRaw, saveScenarioRaw, saveScenarioResult, loadScenarioResult } from '@/lib/bridge'
import { DiscardChangesDialog } from '@/components/ConfirmDialogs'
import type { ScenarioDetail, ScenarioResult, StepDetail, EnvInfo } from '@/lib/bridge'
import { serializeScenarioToYAML } from './scenario-yaml'
import { StepEditorCard } from './ScenarioStepEditor'
import { StepDetailPanel, statusColor } from './ScenarioStepResult'
import { StepListRow } from './ScenarioStepList'

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
  onDirtyChange?: (dirty: boolean) => void
}

// ---- Main component ----

export function ScenarioViewer({ scenarioPath, scenarioName, result, running, theme = 'dark', environments = [], activeEnvPath = '', onResultChange, onRunningChange, onDirtyChange }: Props) {
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
  const [savedFlash, setSavedFlash] = useState(false)
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false)

  const [leftPanelWidth, setLeftPanelWidth] = useState(320)
  const [showStats, setShowStats] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const stepListRef = useRef<HTMLDivElement>(null)
  const selectedStepIndexRef = useRef<number | null>(null)
  useEffect(() => { selectedStepIndexRef.current = selectedStepIndex }, [selectedStepIndex])

  function handleDividerMouseDown(e: React.MouseEvent) {
    e.preventDefault()
    const startX = e.clientX
    const startWidth = leftPanelWidth
    function onMouseMove(ev: MouseEvent) {
      setLeftPanelWidth(Math.max(200, Math.min(600, startWidth + (ev.clientX - startX))))
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

  // Scroll selected step into view and focus the row for keyboard users
  useEffect(() => {
    if (selectedStepIndex === null || !stepListRef.current) return
    const row = stepListRef.current.querySelectorAll<HTMLElement>('[role="option"]')[selectedStepIndex]
    row?.scrollIntoView({ block: 'nearest' })
  }, [selectedStepIndex])

  // Keyboard shortcuts: Alt+Up/Down to reorder steps in visual edit mode
  useEffect(() => {
    if (mode !== 'visual') return
    function handleKey(e: KeyboardEvent) {
      if (!e.altKey || (e.key !== 'ArrowUp' && e.key !== 'ArrowDown')) return
      e.preventDefault()
      const idx = selectedStepIndexRef.current
      if (idx === null) return
      moveVisualStep(idx, e.key === 'ArrowUp' ? -1 : 1)
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [mode]) // eslint-disable-line react-hooks/exhaustive-deps

  async function handleRun() {
    onRunningChange(true)
    onResultChange(null)
    try {
      const r = await runScenario(scenarioPath, selectedEnvPath, 0)
      onResultChange(r)
      saveScenarioResult(scenarioPath, r).catch(() => {})
      const firstFailed = (r.steps ?? []).findIndex(s => !s.passed && !s.skipped)
      if (firstFailed >= 0) setSelectedStepIndex(firstFailed)
    } catch (e) {
      onResultChange({ title: scenarioName, passed: false, durationMs: 0, steps: [], error: String(e) })
    } finally {
      onRunningChange(false)
    }
  }

  async function handleRunFrom(fromStep: number) {
    onRunningChange(true)
    onResultChange(null)
    try {
      const r = await runScenario(scenarioPath, selectedEnvPath, fromStep)
      onResultChange(r)
      saveScenarioResult(scenarioPath, r).catch(() => {})
      const firstFailed = (r.steps ?? []).findIndex(s => !s.passed && !s.skipped)
      if (firstFailed >= 0) setSelectedStepIndex(firstFailed)
      else setSelectedStepIndex(fromStep)
    } catch (e) {
      onResultChange({ title: scenarioName, passed: false, durationMs: 0, steps: [], error: String(e) })
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
      title: `Step ${visualDetail.steps.length + 1}`,
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
    setVisualDetail(prev => {
      if (!prev) return prev
      const to = from + dir
      if (to < 0 || to >= prev.steps.length) return prev
      const steps = [...prev.steps]
      ;[steps[from], steps[to]] = [steps[to], steps[from]]
      return { ...prev, steps }
    })
    setVisualDirty(true)
    setSelectedStepIndex(prev => {
      if (prev === from) return from + dir
      if (prev === from + dir) return from
      return prev
    })
  }

  function duplicateVisualStep(i: number) {
    if (!visualDetail) return
    const src = visualDetail.steps[i]
    const copy: StepDetail = { ...src, title: `${src.title} copy`, assertions: [...src.assertions], extractions: [...src.extractions], headers: [...(src.headers ?? [])], queryParams: [...(src.queryParams ?? [])] }
    const newSteps = [...visualDetail.steps.slice(0, i + 1), copy, ...visualDetail.steps.slice(i + 1)]
    setVisualDetail({ ...visualDetail, steps: newSteps })
    setVisualDirty(true)
    setSelectedStepIndex(i + 1)
  }

  function toggleVisualStepDisabled(i: number) {
    setVisualDetail(prev => {
      if (!prev) return prev
      return { ...prev, steps: prev.steps.map((s, idx) => idx === i ? { ...s, disabled: !s.disabled } : s) }
    })
    setVisualDirty(true)
  }

  async function handleSave() {
    setSaving(true)
    setSaveError(null)
    // Validate visual editor state before serializing
    if (mode === 'visual' && visualDetail) {
      for (const step of visualDetail.steps) {
        for (const a of step.assertions) {
          if ((a.type === 'jsonpath' || a.type === 'header') && !a.path?.trim()) {
            setSaveError(`Step "${step.title}": ${a.type} assertion requires a path.`)
            setSaving(false)
            return
          }
        }
        for (const e of step.extractions) {
          if (!e.name?.trim()) {
            setSaveError(`Step "${step.title}": extraction is missing a variable name.`)
            setSaving(false)
            return
          }
        }
      }
    }
    try {
      const content = mode === 'yaml' ? editorContent : serializeScenarioToYAML(visualDetail!)
      const err = await saveScenarioRaw(scenarioPath, content)
      if (err) { setSaveError(err); return }
      if (mode === 'yaml') setRawContent(editorContent)
      const d = await loadScenario(scenarioPath)
      if (d.error) setLoadError(d.error); else setDetail(d)
      onResultChange(null)
      setVisualDirty(false)
      setSavedFlash(true)
      setTimeout(() => { setSavedFlash(false); setMode('view') }, 1500)
    } catch (e) {
      setSaveError(String(e))
    } finally {
      setSaving(false)
    }
  }

  function doDiscard() {
    if (mode === 'yaml') setEditorContent(rawContent)
    setVisualDirty(false)
    setSaveError(null)
    setMode('view')
  }

  function handleDiscard() {
    if (isDirty) { setShowDiscardConfirm(true); return }
    doDiscard()
  }

  const isEditing = mode === 'visual' || mode === 'yaml'
  const isDirty = (mode === 'yaml' && editorContent !== rawContent) || (mode === 'visual' && visualDirty)

  useEffect(() => { onDirtyChange?.(isDirty) }, [isDirty]) // eslint-disable-line react-hooks/exhaustive-deps

  const passed = result ? (result.steps ?? []).filter(s => s.passed && !s.skipped).length : 0
  const failed = result ? (result.steps ?? []).filter(s => !s.passed && !s.skipped).length : 0
  const skipped = result ? (result.steps ?? []).filter(s => s.skipped).length : 0

  const toggleActiveCls = 'border-b-2 border-[var(--color-accent)] text-[var(--color-foreground)]'
  const toggleInactiveCls = 'border-b-2 border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]'

  // Current steps list (view: from loaded detail, edit: from visualDetail)
  const viewSteps = detail?.steps ?? []
  const editSteps = visualDetail?.steps ?? []

  return (
    <div className="flex flex-col h-full w-full overflow-hidden">

      {/* Header */}
      <div className="shrink-0 flex items-center gap-2 px-4 py-2.5 border-b border-[var(--color-border)] bg-[var(--color-card)]">
        <div className="flex-1 min-w-0">
          <h2 className="text-sm font-semibold text-[var(--color-foreground)] truncate">
            {detail?.title ?? scenarioName}
          </h2>
          {!isEditing && detail?.description && (
            <p className="text-xs text-[var(--color-muted-foreground)] truncate">{detail.description}</p>
          )}
        </div>

        {/* View / Edit / YAML toggle — underline-style tabs */}
        <div className="shrink-0 flex text-xs">
          <button type="button" onClick={() => setMode('view')} disabled={mode === 'view' || saving}
            className={cn('px-3 py-1 transition-colors', mode === 'view' ? toggleActiveCls : toggleInactiveCls)}>
            View
          </button>
          <button type="button" onClick={handleEnterVisual} disabled={mode === 'visual' || saving || !detail}
            className={cn('px-3 py-1 transition-colors', mode === 'visual' ? toggleActiveCls : toggleInactiveCls)}>
            Edit
          </button>
          <button type="button" onClick={handleEnterYaml} disabled={mode === 'yaml' || loadingRaw || saving}
            className={cn('px-3 py-1 transition-colors flex items-center gap-1', mode === 'yaml' ? toggleActiveCls : toggleInactiveCls)}>
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
                <button
                  type="button"
                  onClick={() => setShowStats(true)}
                  title="View run statistics"
                  className="p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">
                  <Clock size={13} />
                </button>
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
            {savedFlash ? (
              <span className="shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold text-emerald-400 bg-emerald-500/10">
                <CheckCircle2 size={12} /> Saved
              </span>
            ) : (
              <>
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
          </>
        )}
      </div>

      {/* Progress bar */}
      {running && (
        <div className="shrink-0 h-1 bg-[var(--color-border)] overflow-hidden">
          <div className="h-full w-1/3 bg-[var(--color-accent)] opacity-80 animate-[slide_1.4s_ease-in-out_infinite]"
            style={{animation: 'slide 1.4s ease-in-out infinite'}} />
        </div>
      )}
      {result && !running && (passed + failed + skipped) > 0 && (
        <div className="shrink-0 h-1 flex">
          {passed   > 0 && <div className="bg-emerald-500 transition-all" style={{flex: passed}} />}
          {failed   > 0 && <div className="bg-red-500 transition-all"     style={{flex: failed}} />}
          {skipped  > 0 && <div className="bg-[var(--color-muted-foreground)]/30 transition-all" style={{flex: skipped}} />}
        </div>
      )}

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
                  value={visualDetail.title}
                  onChange={e => { setVisualDetail({ ...visualDetail, title: e.target.value }); setVisualDirty(true) }}
                  placeholder="Scenario title"
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

            {/* Panel column header */}
            <div className="shrink-0 px-3 py-1 border-b border-[var(--color-border)] bg-[var(--color-background)]">
              <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
                {isEditing ? 'Edit Steps' : 'Steps'}
              </span>
            </div>

            {/* Step list */}
            <div
              ref={stepListRef}
              role="listbox"
              aria-label="Scenario steps"
              className="flex-1 overflow-y-auto py-1 outline-none"
              onKeyDown={e => {
                const steps = mode === 'view' ? viewSteps : editSteps
                if (!steps.length) return
                if (e.key === 'ArrowDown') {
                  e.preventDefault()
                  setSelectedStepIndex(prev => Math.min((prev ?? -1) + 1, steps.length - 1))
                } else if (e.key === 'ArrowUp') {
                  e.preventDefault()
                  setSelectedStepIndex(prev => Math.max((prev ?? 1) - 1, 0))
                }
              }}
            >
              {loadError ? (
                <p className="px-3 py-2 text-xs text-red-400">{loadError}</p>
              ) : mode === 'view' && !detail ? (
                <div className="flex flex-col py-1">
                  {[0.7, 0.9, 0.6].map((w, i) => (
                    <div key={i} className="flex items-center gap-2 px-3 py-2">
                      <div className="w-4 h-4 rounded-full bg-[var(--color-muted)] animate-pulse shrink-0" />
                      <div className="h-2 rounded bg-[var(--color-muted)] animate-pulse" style={{ width: `${w * 100}%` }} />
                    </div>
                  ))}
                </div>
              ) : mode === 'view' ? (
                viewSteps.length === 0 ? (
                  <p className="px-3 py-4 text-xs text-center text-[var(--color-muted-foreground)]">
                    No steps yet. Switch to <strong>Edit</strong> to add your first step.
                  </p>
                ) : (
                  viewSteps.map((step, i) => (
                    <StepListRow
                      key={i} step={step} index={i}
                      result={result?.steps?.[i] ?? null}
                      selected={selectedStepIndex === i}
                      onSelect={setSelectedStepIndex}
                      onRunFrom={() => handleRunFrom(i)}
                      running={running}
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
                      result={result?.steps?.[i] ?? null}
                      selected={selectedStepIndex === i}
                      onSelect={setSelectedStepIndex}
                      editMode
                      isFirst={i === 0}
                      isLast={i === editSteps.length - 1}
                      onMoveUp={() => moveVisualStep(i, -1)}
                      onMoveDown={() => moveVisualStep(i, 1)}
                      onRemove={() => removeVisualStep(i)}
                      onDuplicate={() => duplicateVisualStep(i)}
                      onToggleDisabled={() => toggleVisualStepDisabled(i)}
                    />
                  ))
                )
              )}
            </div>

            {/* Edit mode: add step */}
            {mode === 'visual' && (
              <div className="shrink-0 px-3 py-2 border-t border-[var(--color-border)]">
                <button type="button" onClick={addVisualStep}
                  className="text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors flex items-center gap-1">
                  <Plus size={11} /> Add Step
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
          <div className="flex-1 overflow-hidden min-w-0 flex flex-col">
            {/* Panel column header */}
            <div className="shrink-0 px-3 py-1 border-b border-[var(--color-border)] bg-[var(--color-background)]">
              <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
                {isEditing ? 'Step Editor' : 'Results'}
              </span>
            </div>
            <div className="flex-1 overflow-hidden" role="region" aria-label="Step details">
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
                  onRunFrom={!running ? () => handleRunFrom(selectedStepIndex) : undefined}
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
                    onToggleDisabled={() => toggleVisualStepDisabled(selectedStepIndex)}
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

        </div>
      )}

      {showDiscardConfirm && (
        <DiscardChangesDialog
          message="Discard unsaved changes to this scenario?"
          onCancel={() => setShowDiscardConfirm(false)}
          onDiscard={() => { setShowDiscardConfirm(false); doDiscard() }}
        />
      )}

      {/* Stats dialog */}
      {showStats && result && (() => {
        const steps = result.steps ?? []
        const maxMs = Math.max(...steps.map(s => s.durationMs ?? 0), 1)
        return (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
            onClick={() => setShowStats(false)}>
            <div className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg shadow-xl w-[540px] max-h-[70vh] flex flex-col overflow-hidden"
              onClick={e => e.stopPropagation()}>
              <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border)]">
                <h3 className="text-sm font-semibold text-[var(--color-foreground)]">Run Statistics</h3>
                <button type="button" onClick={() => setShowStats(false)}
                  className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
                  <X size={14} />
                </button>
              </div>

              {/* Summary */}
              <div className="shrink-0 flex items-center gap-3 px-4 py-3 border-b border-[var(--color-border)] bg-[var(--color-muted)]/40">
                <span className={cn('text-xs font-semibold px-2 py-0.5 rounded',
                  result.passed ? 'bg-emerald-500/15 text-emerald-400' : 'bg-red-500/15 text-red-400')}>
                  {result.passed ? 'PASSED' : 'FAILED'}
                </span>
                <span className="text-xs text-[var(--color-muted-foreground)]">
                  {passed} passed · {failed} failed · {skipped} skipped
                </span>
                <span className="ml-auto text-xs font-mono text-[var(--color-foreground)]">
                  {result.durationMs} ms total
                </span>
              </div>

              {/* Per-step table */}
              <div className="overflow-y-auto flex-1">
                <table className="w-full text-xs">
                  <thead className="sticky top-0 bg-[var(--color-card)] border-b border-[var(--color-border)]">
                    <tr className="text-[var(--color-muted-foreground)]">
                      <th className="text-left px-4 py-2 font-medium w-6">#</th>
                      <th className="text-left px-2 py-2 font-medium">Step</th>
                      <th className="text-right px-2 py-2 font-medium w-14">Status</th>
                      <th className="text-right px-4 py-2 font-medium w-24">Duration</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[var(--color-border)]">
                    {steps.map((sr, i) => (
                      <tr key={i} className="hover:bg-[var(--color-muted)]/30 transition-colors">
                        <td className="px-4 py-2.5">
                          <span className={cn('w-5 h-5 rounded-full flex items-center justify-center font-bold text-[11px]',
                            sr.skipped ? 'bg-[var(--color-muted)] text-[var(--color-muted-foreground)]'
                            : sr.passed ? 'bg-emerald-500/20 text-emerald-400'
                            : 'bg-red-500/20 text-red-400')}>
                            {i + 1}
                          </span>
                        </td>
                        <td className="px-2 py-2.5">
                          <div className="font-medium text-[var(--color-foreground)] truncate max-w-[220px]">
                            {sr.title || `Step ${i + 1}`}
                          </div>
                          <div className="flex items-center gap-1.5 mt-1">
                            {sr.skipped
                              ? <span className="text-[var(--color-muted-foreground)]">skipped</span>
                              : sr.passed
                                ? <CheckCircle2 size={11} className="text-emerald-400" />
                                : <XCircle size={11} className="text-red-400" />}
                            <span className="text-[var(--color-muted-foreground)]">
                              {(sr.assertions ?? []).filter(a => a.passed).length}/{(sr.assertions ?? []).length} assertions
                            </span>
                          </div>
                        </td>
                        <td className="px-2 py-2.5 text-right">
                          {sr.statusCode > 0 && (
                            <span className={cn('font-mono font-bold', statusColor(sr.statusCode))}>
                              {sr.statusCode}
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-2.5">
                          <div className="flex flex-col items-end gap-1">
                            <span className="font-mono text-[var(--color-foreground)]">
                              {sr.durationMs > 0 ? `${sr.durationMs} ms` : '—'}
                            </span>
                            {sr.durationMs > 0 && (
                              <div className="w-full h-1 bg-[var(--color-border)] rounded-full overflow-hidden">
                                <div
                                  className={cn('h-full rounded-full', sr.passed ? 'bg-emerald-500' : 'bg-red-500')}
                                  style={{width: `${Math.round((sr.durationMs / maxMs) * 100)}%`}}
                                />
                              </div>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )
      })()}

    </div>
  )
}
