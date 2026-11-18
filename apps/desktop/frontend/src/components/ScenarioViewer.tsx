import { useState, useEffect, useRef } from 'react'
import {
  loadScenario, runScenario, saveScenarioRaw, saveScenarioResult,
  loadScenarioResult, getAccumulatedCoverage, resetAccumulatedCoverage, cancelScenario,
} from '@/lib/bridge'
import { DiscardChangesDialog } from '@/components/ConfirmDialogs'
import type { ScenarioDetail, ScenarioResult, StepDetail, EnvInfo, AccumulatedCoverage, ScenarioVariable } from '@/lib/bridge'
import { serializeScenarioToYAML } from './scenario-yaml'
import { ScenarioHeaderBar } from './ScenarioHeaderBar'
import { ScenarioMetaSection } from './ScenarioMetaSection'
import { ScenarioVariablesSection } from './ScenarioVariablesSection'
import { ScenarioRunInfo } from './ScenarioRunInfo'
import { StepListPanel } from './ScenarioStepListPanel'
import { StepDetailPane } from './ScenarioDetailPane'
import { ScenarioYamlDialog } from './ScenarioYamlDialog'
import { CoverageBanner } from './ScenarioCoverageBanner'
import { ScenarioStatsDialog } from './ScenarioStatsDialog'

type Props = {
  scenarioPath: string
  scenarioName: string
  result: ScenarioResult | null
  running: boolean
  isActive?: boolean
  saveSignal?: number
  theme?: 'dark' | 'light'
  environments?: EnvInfo[]
  activeEnvPath?: string
  onResultChange: (result: ScenarioResult | null) => void
  onRunningChange: (running: boolean) => void
  onDirtyChange?: (dirty: boolean) => void
  onRunComplete?: () => void
  onGetCliCommand?: () => void
}

export function ScenarioViewer({
  scenarioPath, scenarioName, result, running, isActive = true, saveSignal, theme = 'dark',
  environments = [], activeEnvPath = '',
  onResultChange, onRunningChange, onDirtyChange, onRunComplete, onGetCliCommand,
}: Props) {
  const [detail, setDetail] = useState<ScenarioDetail | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [selectedEnvPath, setSelectedEnvPath] = useState(activeEnvPath)
  const [selectedStepIndex, setSelectedStepIndex] = useState<number | null>(null)

  const [showYamlDialog, setShowYamlDialog] = useState(false)

  const [visualDetail, setVisualDetail] = useState<ScenarioDetail | null>(null)
  const [visualDirty, setVisualDirty] = useState(false)

  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [savedFlash, setSavedFlash] = useState(false)
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false)

  const [accumulatedCoverage, setAccumulatedCoverage] = useState<AccumulatedCoverage | null>(null)
  const [leftPanelPct, setLeftPanelPct] = useState<number>(() => {
    const saved = Number(localStorage.getItem('ui.scenarioStepPanelPct'))
    return isNaN(saved) || saved === 0 ? 35 : Math.min(50, Math.max(20, saved))
  })
  const leftPanelPctRef = useRef(leftPanelPct)
  const [showStats, setShowStats] = useState(false)

  const containerRef = useRef<HTMLDivElement>(null)
  const stepListRef = useRef<HTMLDivElement>(null)
  const selectedStepIndexRef = useRef<number | null>(null)
  useEffect(() => { selectedStepIndexRef.current = selectedStepIndex }, [selectedStepIndex])
  const moveVisualStepRef = useRef<((from: number, dir: -1 | 1) => void) | null>(null)

  const [prevScenarioPath, setPrevScenarioPath] = useState(scenarioPath)
  if (prevScenarioPath !== scenarioPath) {
    setPrevScenarioPath(scenarioPath)
    setDetail(null)
    setLoadError(null)
    setShowYamlDialog(false)
    setSaveError(null)
    setVisualDetail(null)
    setVisualDirty(false)
    setSelectedEnvPath(activeEnvPath)
    setSelectedStepIndex(null)
    setAccumulatedCoverage(null)
  }

  function initDraft(d: ScenarioDetail) {
    setVisualDetail({
      ...d,
      steps: d.steps.map(s => ({
        ...s,
        notes: s.notes ?? '',
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
  }

  function handleDividerMouseDown(e: React.MouseEvent) {
    e.preventDefault()
    const startX = e.clientX
    const startPct = leftPanelPct
    const containerWidth = containerRef.current?.clientWidth ?? 800
    function onMouseMove(ev: MouseEvent) {
      const deltaPct = ((ev.clientX - startX) / containerWidth) * 100
      const clamped = Math.min(50, Math.max(20, startPct + deltaPct))
      leftPanelPctRef.current = clamped
      setLeftPanelPct(clamped)
    }
    function onMouseUp() {
      localStorage.setItem('ui.scenarioStepPanelPct', String(Math.round(leftPanelPctRef.current)))
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
  }

  useEffect(() => {
    loadScenario(scenarioPath)
      .then(d => {
        if (d.error) setLoadError(d.error)
        else {
          setDetail(d)
          initDraft(d)
          if (d.steps.length > 0) setSelectedStepIndex(0)
          if (d.specFile) {
            getAccumulatedCoverage(d.specFile)
              .then(acc => { if (acc) setAccumulatedCoverage(acc) })
              .catch(() => {})
          }
        }
      })
      .catch(e => setLoadError(String(e)))
    loadScenarioResult(scenarioPath)
      .then(cached => { if (cached) onResultChange(cached) })
      .catch(() => {})
  }, [scenarioPath]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (selectedStepIndex === null || !stepListRef.current) return
    const row = stepListRef.current.querySelectorAll<HTMLElement>('[role="option"]')[selectedStepIndex]
    row?.scrollIntoView({ block: 'nearest' })
  }, [selectedStepIndex])

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (!isActive) return
      if (!e.altKey || (e.key !== 'ArrowUp' && e.key !== 'ArrowDown')) return
      e.preventDefault()
      const idx = selectedStepIndexRef.current
      if (idx === null) return
      moveVisualStepRef.current?.(idx, e.key === 'ArrowUp' ? -1 : 1)
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [isActive])

  const handleSaveRef = useRef<() => Promise<boolean>>(() => Promise.resolve(false))
  // eslint-disable-next-line react/immutability
  useEffect(() => { handleSaveRef.current = handleSave })

  useEffect(() => {
    if (saveSignal === undefined || saveSignal === 0) return
    handleSaveRef.current()
  }, [saveSignal]) // eslint-disable-line react-hooks/exhaustive-deps

  function applyRunResult(r: ScenarioResult) {
    onResultChange(r)
    if (r.accumulatedCoverage) setAccumulatedCoverage(r.accumulatedCoverage)
    saveScenarioResult(scenarioPath, r).catch(() => {})
  }

  async function handleRun() {
    if (visualDirty && !(await handleSave())) return
    onRunningChange(true)
    onResultChange(null)
    try {
      const r = await runScenario(scenarioPath, selectedEnvPath, 0)
      applyRunResult(r)
      const firstFailed = (r.steps ?? []).findIndex(s => !s.passed && !s.skipped)
      if (firstFailed >= 0) setSelectedStepIndex(firstFailed)
    } catch (e) {
      onResultChange({ title: scenarioName, passed: false, durationMs: 0, steps: [], error: String(e) })
    } finally {
      onRunningChange(false)
      onRunComplete?.()
    }
  }

  function handleCancel() {
    cancelScenario().catch(() => {})
    // onRunningChange(false) will be called by handleRun/handleRunFrom's finally block
  }

  async function handleRunFrom(fromStep: number) {
    if (visualDirty && !(await handleSave())) return
    onRunningChange(true)
    onResultChange(null)
    try {
      const r = await runScenario(scenarioPath, selectedEnvPath, fromStep)
      applyRunResult(r)
      const firstFailed = (r.steps ?? []).findIndex(s => !s.passed && !s.skipped)
      if (firstFailed >= 0) setSelectedStepIndex(firstFailed)
      else setSelectedStepIndex(fromStep)
    } catch (e) {
      onResultChange({ title: scenarioName, passed: false, durationMs: 0, steps: [], error: String(e) })
    } finally {
      onRunningChange(false)
      onRunComplete?.()
    }
  }

  async function handleResetCoverage() {
    const specFile = detail?.specFile
    if (!specFile) return
    await resetAccumulatedCoverage(specFile).catch(() => {})
    setAccumulatedCoverage(null)
  }

  function updateMeta(patch: { title?: string; description?: string; specFile?: string; variables?: ScenarioVariable[] }) {
    if (!visualDetail) return
    setVisualDetail({ ...visualDetail, ...patch })
    setVisualDirty(true)
  }

  function updateVisualStep(i: number, updates: Partial<StepDetail>) {
    if (!visualDetail) return
    setVisualDetail({ ...visualDetail, steps: visualDetail.steps.map((s, idx) => idx === i ? { ...s, ...updates } : s) })
    setVisualDirty(true)
  }

  function addVisualStep() {
    if (!visualDetail) return
    const newStep: StepDetail = {
      label: `Step ${visualDetail.steps.length + 1}`,
      notes: '',
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
  useEffect(() => { moveVisualStepRef.current = moveVisualStep })

  function duplicateVisualStep(i: number) {
    if (!visualDetail) return
    const src = visualDetail.steps[i]
    const copy: StepDetail = {
      ...src,
      label: `${src.label} copy`,
      assertions: [...src.assertions],
      extractions: [...src.extractions],
      headers: [...(src.headers ?? [])],
      queryParams: [...(src.queryParams ?? [])],
    }
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

  async function handleSave(): Promise<boolean> {
    if (!visualDetail) return false
    setSaving(true)
    setSaveError(null)
    for (const step of visualDetail.steps) {
      for (const a of step.assertions) {
        if ((a.type === 'jsonpath' || a.type === 'header') && !a.path?.trim()) {
          setSaveError(`Step "${step.label}": ${a.type} assertion requires a path.`)
          setSaving(false)
          return false
        }
      }
      for (const e of step.extractions) {
        if (!e.name?.trim()) {
          setSaveError(`Step "${step.label}": extraction is missing a variable name.`)
          setSaving(false)
          return false
        }
      }
    }
    try {
      const content = serializeScenarioToYAML(visualDetail)
      const err = await saveScenarioRaw(scenarioPath, content)
      if (err) { setSaveError(err); return false }
      const d = await loadScenario(scenarioPath)
      if (d.error) setLoadError(d.error); else { setDetail(d); initDraft(d) }
      onResultChange(null)
      setSavedFlash(true)
      setTimeout(() => setSavedFlash(false), 1500)
      return true
    } catch (e) {
      setSaveError(String(e))
      return false
    } finally {
      setSaving(false)
    }
  }

  async function handleSaveYaml(content: string): Promise<string | null> {
    setSaving(true)
    try {
      const err = await saveScenarioRaw(scenarioPath, content)
      if (err) return err
      const d = await loadScenario(scenarioPath)
      if (d.error) { setLoadError(d.error); return d.error }
      setDetail(d)
      initDraft(d)
      onResultChange(null)
      setSavedFlash(true)
      setTimeout(() => setSavedFlash(false), 1500)
      setShowYamlDialog(false)
      return null
    } catch (e) {
      return String(e)
    } finally {
      setSaving(false)
    }
  }

  function doDiscard() {
    if (detail) initDraft(detail)
    setSaveError(null)
  }

  function handleDiscard() {
    if (visualDirty) { setShowDiscardConfirm(true); return }
    doDiscard()
  }

  useEffect(() => { onDirtyChange?.(visualDirty) }, [visualDirty]) // eslint-disable-line react-hooks/exhaustive-deps

  const passed = result ? (result.steps ?? []).filter(s => s.passed && !s.skipped).length : 0
  const failed = result ? (result.steps ?? []).filter(s => !s.passed && !s.skipped).length : 0
  const skipped = result ? (result.steps ?? []).filter(s => s.skipped).length : 0

  return (
    <div className="flex flex-col h-full w-full overflow-hidden">

      <ScenarioHeaderBar
        title={visualDetail?.title ?? scenarioName}
        saving={saving}
        detailLoaded={!!detail}
        environments={environments}
        selectedEnvPath={selectedEnvPath}
        onEnvPathChange={setSelectedEnvPath}
        onTitleChange={t => updateMeta({ title: t })}
        result={result}
        running={running}
        onShowStats={() => setShowStats(true)}
        onRun={handleRun}
        onCancel={handleCancel}
        saveError={saveError}
        savedFlash={savedFlash}
        isDirty={visualDirty}
        onDiscard={handleDiscard}
        onSave={handleSave}
        onShowYaml={() => setShowYamlDialog(true)}
        onGetCliCommand={onGetCliCommand}
      />

      {running && (
        <div className="shrink-0 h-1 bg-[var(--color-border)] overflow-hidden">
          <div className="h-full w-1/3 bg-[var(--color-accent)] opacity-80"
            style={{ animation: 'slide 1.4s ease-in-out infinite' }} />
        </div>
      )}
      {result && !running && (passed + failed + skipped) > 0 && (
        <div className="shrink-0 h-1 flex">
          {passed  > 0 && <div className="bg-emerald-500 transition-all" style={{ flex: passed }} />}
          {failed  > 0 && <div className="bg-red-500 transition-all" style={{ flex: failed }} />}
          {skipped > 0 && <div className="bg-[var(--color-muted-foreground)]/30 transition-all" style={{ flex: skipped }} />}
        </div>
      )}

      {result?.specError && !running && (
        <div className="shrink-0 px-4 py-2 text-xs text-amber-400 bg-amber-500/10 border-b border-amber-500/20">
          <span className="font-semibold">specFile warning: </span>{result.specError}
        </div>
      )}

      {(accumulatedCoverage || result?.coverage) && !running && (
        <CoverageBanner
          coverage={result?.coverage}
          accumulated={accumulatedCoverage ?? undefined}
          onReset={handleResetCoverage}
        />
      )}

      {visualDetail && (
        <ScenarioMetaSection
          description={visualDetail.description ?? ''}
          specFile={visualDetail.specFile ?? ''}
          onChange={updateMeta}
        />
      )}

      {visualDetail && (
        <ScenarioVariablesSection
          variables={visualDetail.variables ?? []}
          onChange={variables => updateMeta({ variables })}
        />
      )}

      {result && !running && <ScenarioRunInfo result={result} />}

      <div ref={containerRef} className="flex-1 flex overflow-hidden">
        <StepListPanel
          widthPct={leftPanelPct}
          listRef={stepListRef}
          loadError={loadError}
          detailLoaded={!!detail}
          steps={visualDetail?.steps ?? []}
          selectedStepIndex={selectedStepIndex}
          onSelectStep={setSelectedStepIndex}
          result={result}
          running={running}
          onMoveStep={moveVisualStep}
          onRemoveStep={removeVisualStep}
          onDuplicateStep={duplicateVisualStep}
          onToggleDisabled={toggleVisualStepDisabled}
          onAddStep={addVisualStep}
        />

        <div
          onMouseDown={handleDividerMouseDown}
          className="group w-[5px] shrink-0 cursor-col-resize flex items-center justify-center hover:bg-[var(--color-accent)]/15 transition-colors active:bg-[var(--color-accent)]/25"
          title="Drag to resize"
        >
          <div className="w-px h-full bg-[var(--color-border)] group-hover:bg-[var(--color-accent)] transition-colors" />
        </div>

        <StepDetailPane
          visualDetail={visualDetail}
          result={result}
          running={running}
          selectedStepIndex={selectedStepIndex}
          editStepCount={visualDetail?.steps.length ?? 0}
          onUpdateStep={updateVisualStep}
          onRemoveStep={removeVisualStep}
          onMoveStep={moveVisualStep}
          onToggleDisabled={toggleVisualStepDisabled}
          onRunFrom={handleRunFrom}
        />
      </div>

      {showYamlDialog && visualDetail && (
        <ScenarioYamlDialog
          theme={theme}
          initialContent={serializeScenarioToYAML(visualDetail)}
          onSave={handleSaveYaml}
          onClose={() => setShowYamlDialog(false)}
        />
      )}

      {showDiscardConfirm && (
        <DiscardChangesDialog
          message="Discard unsaved changes to this scenario?"
          onCancel={() => setShowDiscardConfirm(false)}
          onDiscard={() => { setShowDiscardConfirm(false); doDiscard() }}
        />
      )}

      {showStats && result && (
        <ScenarioStatsDialog result={result} onClose={() => setShowStats(false)} />
      )}

    </div>
  )
}
