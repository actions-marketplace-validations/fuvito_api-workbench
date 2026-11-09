import { useState, useEffect, useRef } from 'react'
import { loadScenario, runScenario, loadScenarioRaw, saveScenarioRaw, saveScenarioResult, loadScenarioResult } from '@/lib/bridge'
import { DiscardChangesDialog } from '@/components/ConfirmDialogs'
import type { ScenarioDetail, ScenarioResult, StepDetail, EnvInfo } from '@/lib/bridge'
import { serializeScenarioToYAML } from './scenario-yaml'
import { ScenarioHeaderBar } from './ScenarioHeaderBar'
import { StepListPanel } from './ScenarioStepListPanel'
import { StepDetailPane } from './ScenarioDetailPane'
import { YamlEditorPane } from './ScenarioYamlPane'
import { CoverageBanner } from './ScenarioCoverageBanner'
import { ScenarioStatsDialog } from './ScenarioStatsDialog'

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

  // Current steps list (view: from loaded detail, edit: from visualDetail)
  const viewSteps = detail?.steps ?? []
  const editSteps = visualDetail?.steps ?? []

  return (
    <div className="flex flex-col h-full w-full overflow-hidden">

      <ScenarioHeaderBar
        title={detail?.title ?? scenarioName}
        description={detail?.description}
        mode={mode}
        isEditing={isEditing}
        saving={saving}
        loadingRaw={loadingRaw}
        detailLoaded={!!detail}
        environments={environments}
        selectedEnvPath={selectedEnvPath}
        onEnvPathChange={setSelectedEnvPath}
        result={result}
        running={running}
        onShowStats={() => setShowStats(true)}
        onRun={handleRun}
        saveError={saveError}
        savedFlash={savedFlash}
        isDirty={isDirty}
        onModeView={() => setMode('view')}
        onEnterVisual={handleEnterVisual}
        onEnterYaml={handleEnterYaml}
        onDiscard={handleDiscard}
        onSave={handleSave}
      />

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

      {/* Coverage banner — shown when scenario has a specFile and run produced coverage data */}
      {result?.coverage && !running && (
        <CoverageBanner coverage={result.coverage} />
      )}

      {/* Body */}
      {mode === 'yaml' ? (
        <YamlEditorPane theme={theme} value={editorContent} error={rawError} onChange={setEditorContent} />
      ) : (
        <div ref={containerRef} className="flex-1 flex overflow-hidden">

          {/* Left panel: step list */}
          <StepListPanel
            width={leftPanelWidth}
            mode={mode}
            listRef={stepListRef}
            loadError={loadError}
            detailLoaded={!!detail}
            viewSteps={viewSteps}
            editSteps={editSteps}
            visualDetail={visualDetail}
            onMetaChange={patch => { if (visualDetail) { setVisualDetail({ ...visualDetail, ...patch }); setVisualDirty(true) } }}
            selectedStepIndex={selectedStepIndex}
            onSelectStep={setSelectedStepIndex}
            result={result}
            running={running}
            onRunFrom={handleRunFrom}
            onMoveStep={moveVisualStep}
            onRemoveStep={removeVisualStep}
            onDuplicateStep={duplicateVisualStep}
            onToggleDisabled={toggleVisualStepDisabled}
            onAddStep={addVisualStep}
          />

          {/* Draggable divider */}
          <div
            onMouseDown={handleDividerMouseDown}
            className="group w-[5px] shrink-0 cursor-col-resize flex items-center justify-center hover:bg-[var(--color-accent)]/15 transition-colors active:bg-[var(--color-accent)]/25"
            title="Drag to resize"
          >
            <div className="w-px h-full bg-[var(--color-border)] group-hover:bg-[var(--color-accent)] transition-colors" />
          </div>

          {/* Right panel */}
          <StepDetailPane
            mode={mode}
            running={running}
            detail={detail}
            visualDetail={visualDetail}
            result={result}
            selectedStepIndex={selectedStepIndex}
            editStepCount={editSteps.length}
            onRunFrom={handleRunFrom}
            onUpdateStep={updateVisualStep}
            onRemoveStep={removeVisualStep}
            onMoveStep={moveVisualStep}
            onToggleDisabled={toggleVisualStepDisabled}
          />

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
      {showStats && result && (
        <ScenarioStatsDialog result={result} onClose={() => setShowStats(false)} />
      )}

    </div>
  )
}
