import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { useState } from 'react'
import { render, screen, fireEvent, waitFor, within, act } from '@testing-library/react'
import { ScenarioViewer } from './ScenarioViewer'
import * as bridge from '@/lib/bridge'
import type { ScenarioDetail, ScenarioResult, StepDetail } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({
  loadScenario: vi.fn(),
  runScenario: vi.fn(),
  saveScenarioRaw: vi.fn(),
  saveScenarioResult: vi.fn(),
  loadScenarioResult: vi.fn(),
  getAccumulatedCoverage: vi.fn(),
  resetAccumulatedCoverage: vi.fn(),
  cancelScenario: vi.fn(),
  validateScenarioYAML: vi.fn(),
}))
vi.mock('./ScenarioYamlPane', () => ({
  YamlEditorPane: (p: { value: string; onChange: (v: string) => void }) => (
    <textarea aria-label="yaml" value={p.value} onChange={e => p.onChange(e.target.value)} />
  ),
}))

const b = vi.mocked(bridge)

function step(over: Partial<StepDetail> = {}): StepDetail {
  return { label: 'One', method: 'GET', url: '/one', assertions: [{ type: 'status', operator: 'eq', expected: 200 }], extractions: [], ...over }
}

function detail(over: Partial<ScenarioDetail> = {}): ScenarioDetail {
  return {
    path: '/ws/scenarios/s.yaml',
    title: 'My scenario',
    description: 'desc',
    steps: [step({ label: 'One' }), step({ label: 'Two', url: '/two' }), step({ label: 'Three', url: '/three' })],
    ...over,
  }
}

function resultOf(steps: Array<{ passed: boolean; skipped?: boolean }>, over: Partial<ScenarioResult> = {}): ScenarioResult {
  return {
    title: 't',
    passed: steps.every(s => s.passed || s.skipped),
    durationMs: 99,
    steps: steps.map((s, i) => ({
      label: `r${i}`, passed: s.passed, skipped: !!s.skipped, statusCode: 200, durationMs: 1, extracted: {}, assertions: [],
    })),
    ...over,
  }
}

type Spy = Mock<(...args: unknown[]) => void>
type Spies = {
  onResultChange: Spy
  onRunningChange: Spy
  onDirtyChange: Spy
  onRunComplete: Spy
}

function Harness({ spies, initialResult = null, ...rest }: { spies: Spies; initialResult?: ScenarioResult | null } & Partial<React.ComponentProps<typeof ScenarioViewer>>) {
  const [result, setResult] = useState<ScenarioResult | null>(initialResult)
  const [running, setRunning] = useState(false)
  return (
    <ScenarioViewer
      scenarioPath="/ws/scenarios/s.yaml"
      scenarioName="My scenario"
      result={result}
      running={running}
      onResultChange={r => { spies.onResultChange(r); setResult(r) }}
      onRunningChange={r => { spies.onRunningChange(r); setRunning(r) }}
      onDirtyChange={spies.onDirtyChange}
      onRunComplete={spies.onRunComplete}
      {...rest}
    />
  )
}

function mount(props: Partial<React.ComponentProps<typeof ScenarioViewer>> & { initialResult?: ScenarioResult | null } = {}) {
  const spies: Spies = { onResultChange: vi.fn(), onRunningChange: vi.fn(), onDirtyChange: vi.fn(), onRunComplete: vi.fn() }
  const utils = render(<Harness spies={spies} {...props} />)
  return { spies, ...utils }
}

async function loaded() {
  await screen.findByRole('button', { name: 'Setup' })
}

function list() {
  return screen.getByRole('listbox', { name: 'Scenario steps' })
}

function labels() {
  return within(list()).getAllByRole('option').map(o => o.querySelector('span.truncate')?.textContent)
}

function editTitle(value = 'Renamed') {
  fireEvent.change(screen.getByPlaceholderText('Scenario title'), { target: { value } })
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  Element.prototype.scrollIntoView = vi.fn()
  b.loadScenario.mockResolvedValue(detail())
  b.loadScenarioResult.mockResolvedValue(null as never)
  b.getAccumulatedCoverage.mockResolvedValue(null as never)
  b.saveScenarioRaw.mockResolvedValue('' as never)
  b.saveScenarioResult.mockResolvedValue(undefined as never)
  b.resetAccumulatedCoverage.mockResolvedValue(undefined as never)
  b.cancelScenario.mockResolvedValue(undefined as never)
  b.validateScenarioYAML.mockResolvedValue('' as never)
})
afterEach(() => vi.useRealTimers())

describe('ScenarioViewer — loading', () => {
  it('loads the scenario, lists steps and selects the first one', async () => {
    mount()
    await loaded()
    expect(b.loadScenario).toHaveBeenCalledWith('/ws/scenarios/s.yaml')
    expect(labels()).toEqual(['One', 'Two', 'Three'])
    expect(within(list()).getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'true')
    expect(screen.queryByDisplayValue('desc')).not.toBeInTheDocument()
  })

  it('shows the load error reported by the backend', async () => {
    b.loadScenario.mockResolvedValue(detail({ error: 'invalid yaml' }))
    mount()
    expect(await screen.findByText('invalid yaml')).toBeInTheDocument()
  })

  it('shows a load failure from a rejected call', async () => {
    b.loadScenario.mockRejectedValue(new Error('ipc down'))
    mount()
    expect(await screen.findByText(/ipc down/)).toBeInTheDocument()
  })

  it('opens Setup and shows the Start here row for a scenario without steps', async () => {
    b.loadScenario.mockResolvedValue(detail({ steps: [] }))
    mount()
    await loaded()
    expect(screen.getByRole('button', { name: 'Setup' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByText('Start here')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Add step/ })).not.toBeInTheDocument()
    expect(screen.getByDisplayValue('desc')).toBeInTheDocument()
  })

  it('creates a real first step from the Start here row', async () => {
    b.loadScenario.mockResolvedValue(detail({ steps: [] }))
    mount()
    await loaded()
    fireEvent.click(screen.getByText('Start here'))
    expect(within(list()).getAllByRole('option')).toHaveLength(1)
    expect(within(list()).getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'true')
    expect(screen.queryByText('Start here')).not.toBeInTheDocument()
  })

  it('disables Run when the scenario has no steps', async () => {
    b.loadScenario.mockResolvedValue(detail({ steps: [] }))
    mount()
    await loaded()
    const run = screen.getByRole('button', { name: /^Run/ })
    expect(run).toBeDisabled()
    expect(run).toHaveAttribute('title', 'Add a step to run')
    fireEvent.click(run)
    expect(b.runScenario).not.toHaveBeenCalled()
  })

  it('switches between Setup and steps with the keyboard', async () => {
    mount()
    await loaded()
    fireEvent.keyDown(list(), { key: 'ArrowUp' })
    expect(screen.getByRole('button', { name: 'Setup' })).toHaveAttribute('aria-pressed', 'true')
    expect(within(list()).getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'false')
    fireEvent.keyDown(screen.getByRole('button', { name: 'Setup' }), { key: 'ArrowDown' })
    expect(within(list()).getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('restores a cached result', async () => {
    const cached = resultOf([{ passed: true }])
    b.loadScenarioResult.mockResolvedValue(cached)
    const { spies } = mount()
    await waitFor(() => expect(spies.onResultChange).toHaveBeenCalledWith(cached))
    expect(await screen.findByTitle('View run statistics')).toBeInTheDocument()
  })

  it('ignores a failing cached-result lookup', async () => {
    b.loadScenarioResult.mockRejectedValue(new Error('none'))
    mount()
    await loaded()
    expect(screen.queryByText('PASSED')).not.toBeInTheDocument()
  })

  it('loads accumulated coverage when the scenario binds a spec', async () => {
    b.loadScenario.mockResolvedValue(detail({ specFile: 'api.yaml' }))
    b.getAccumulatedCoverage.mockResolvedValue({
      specFile: 'api.yaml', covered: 1, total: 2, percent: 50, totalRuns: 2, passedRuns: 2, failedRuns: 0, exercised: ['GET /a'], missed: ['GET /b'],
    })
    mount()
    fireEvent.click(await screen.findByRole('button', { name: /Coverage 1\/2 · 50%/ }))
    expect(await screen.findByText('Accumulated Coverage')).toBeInTheDocument()
    expect(b.getAccumulatedCoverage).toHaveBeenCalledWith('api.yaml')
  })

  it('tolerates a failing coverage lookup', async () => {
    b.loadScenario.mockResolvedValue(detail({ specFile: 'api.yaml' }))
    b.getAccumulatedCoverage.mockRejectedValue(new Error('x'))
    mount()
    await loaded()
    expect(screen.queryByText('Accumulated Coverage')).not.toBeInTheDocument()
  })

  it('resets accumulated coverage from the banner', async () => {
    b.loadScenario.mockResolvedValue(detail({ specFile: 'api.yaml' }))
    b.getAccumulatedCoverage.mockResolvedValue({
      specFile: 'api.yaml', covered: 1, total: 2, percent: 50, totalRuns: 2, passedRuns: 2, failedRuns: 0, exercised: [], missed: [],
    })
    mount()
    fireEvent.click(await screen.findByRole('button', { name: /Coverage 1\/2/ }))
    fireEvent.click(await screen.findByTitle('Reset accumulated coverage'))
    await waitFor(() => expect(b.resetAccumulatedCoverage).toHaveBeenCalledWith('api.yaml'))
    await waitFor(() => expect(screen.queryByText('Accumulated Coverage')).not.toBeInTheDocument())
  })

  it('reloads when the scenario path changes', async () => {
    const spies: Spies = { onResultChange: vi.fn(), onRunningChange: vi.fn(), onDirtyChange: vi.fn(), onRunComplete: vi.fn() }
    const { rerender } = render(<Harness spies={spies} />)
    await loaded()
    b.loadScenario.mockResolvedValue(detail({ title: 'Other', steps: [step({ label: 'Solo' })] }))
    rerender(<Harness spies={spies} scenarioPath="/ws/scenarios/other.yaml" />)
    expect(await screen.findByDisplayValue('Other')).toBeInTheDocument()
    expect(b.loadScenario).toHaveBeenLastCalledWith('/ws/scenarios/other.yaml')
  })
})

describe('ScenarioViewer — editing', () => {
  it('becomes dirty on edit and reports it', async () => {
    const { spies } = mount()
    await loaded()
    expect(screen.queryByRole('button', { name: /Save/ })).not.toBeInTheDocument()
    editTitle()
    expect(screen.getByRole('button', { name: /Save/ })).toBeInTheDocument()
    await waitFor(() => expect(spies.onDirtyChange).toHaveBeenLastCalledWith(true))
  })

  it('edits description and spec file', async () => {
    mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: 'Setup' }))
    fireEvent.change(screen.getByPlaceholderText('What does this scenario test?'), { target: { value: 'new desc' } })
    fireEvent.change(screen.getByPlaceholderText(/OpenAPI spec path/), { target: { value: 'spec.yaml' } })
    expect(screen.getByRole('button', { name: /Save/ })).toBeInTheDocument()
  })

  it('adds a step and selects it', async () => {
    mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /Add step/ }))
    expect(within(list()).getAllByRole('option')).toHaveLength(4)
    expect(within(list()).getAllByRole('option')[3]).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByDisplayValue('Step 4')).toBeInTheDocument()
  })

  it('removes the selected step and keeps a valid selection', async () => {
    mount()
    await loaded()
    fireEvent.click(within(within(list()).getAllByRole('option')[0]).getByTitle('Remove step'))
    expect(labels()).toEqual(['Two', 'Three'])
    expect(within(list()).getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('shifts the selection when a step before it is removed', async () => {
    mount()
    await loaded()
    fireEvent.click(within(list()).getAllByRole('option')[2])
    fireEvent.click(within(within(list()).getAllByRole('option')[0]).getByTitle('Remove step'))
    expect(within(list()).getAllByRole('option')[1]).toHaveAttribute('aria-selected', 'true')
  })

  it('keeps the selection when a later step is removed', async () => {
    mount()
    await loaded()
    fireEvent.click(within(within(list()).getAllByRole('option')[2]).getByTitle('Remove step'))
    expect(within(list()).getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('selects Setup when the last step is removed', async () => {
    b.loadScenario.mockResolvedValue(detail({ steps: [step({ label: 'Only' })] }))
    mount()
    await screen.findByDisplayValue('My scenario')
    fireEvent.click(within(within(list()).getAllByRole('option')[0]).getByTitle('Remove step'))
    expect(screen.getByText('Start here')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Setup' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('duplicates a step right after the original and selects the copy', async () => {
    mount()
    await loaded()
    fireEvent.click(within(within(list()).getAllByRole('option')[0]).getByTitle('Duplicate step'))
    expect(labels()).toEqual(['One', 'One copy', 'Two', 'Three'])
    expect(within(list()).getAllByRole('option')[1]).toHaveAttribute('aria-selected', 'true')
  })

  it('moves steps with the row buttons and keeps the selection on the moved step', async () => {
    mount()
    await loaded()
    fireEvent.click(within(within(list()).getAllByRole('option')[0]).getByTitle('Move down (Alt+↓)'))
    expect(labels()).toEqual(['Two', 'One', 'Three'])
    expect(within(list()).getAllByRole('option')[1]).toHaveAttribute('aria-selected', 'true')
  })

  it('adjusts the selection when another step moves past it', async () => {
    mount()
    await loaded()
    fireEvent.click(within(list()).getAllByRole('option')[1])
    fireEvent.click(within(within(list()).getAllByRole('option')[0]).getByTitle('Move down (Alt+↓)'))
    expect(labels()).toEqual(['Two', 'One', 'Three'])
    expect(within(list()).getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('leaves the selection alone when unrelated steps move', async () => {
    mount()
    await loaded()
    fireEvent.click(within(within(list()).getAllByRole('option')[1]).getByTitle('Move down (Alt+↓)'))
    expect(within(list()).getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('moves the selected step with Alt+Arrow and ignores plain arrows', async () => {
    mount()
    await loaded()
    await waitFor(() => expect(within(list()).getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'true'))
    fireEvent.keyDown(document, { key: 'ArrowDown' })
    expect(labels()).toEqual(['One', 'Two', 'Three'])
    fireEvent.keyDown(document, { key: 'ArrowDown', altKey: true })
    expect(labels()).toEqual(['Two', 'One', 'Three'])
    fireEvent.keyDown(document, { key: 'ArrowUp', altKey: true })
    expect(labels()).toEqual(['One', 'Two', 'Three'])
    fireEvent.keyDown(document, { key: 'ArrowUp', altKey: true })
    expect(labels()).toEqual(['One', 'Two', 'Three'])
  })

  it('ignores Alt+Arrow when the tab is not active', async () => {
    mount({ isActive: false })
    await loaded()
    fireEvent.keyDown(document, { key: 'ArrowDown', altKey: true })
    expect(labels()).toEqual(['One', 'Two', 'Three'])
  })

  it('toggles a step disabled', async () => {
    mount()
    await loaded()
    fireEvent.click(within(within(list()).getAllByRole('option')[0]).getByTitle('Disable step'))
    expect(within(within(list()).getAllByRole('option')[0]).getByTitle('Enable step')).toBeInTheDocument()
  })

  it('edits the selected step through the editor', async () => {
    mount()
    await loaded()
    fireEvent.change(screen.getByDisplayValue('One'), { target: { value: 'Uno' } })
    expect(labels()[0]).toBe('Uno')
  })

  it('keyboard-navigates the step list', async () => {
    mount()
    await loaded()
    fireEvent.keyDown(list(), { key: 'ArrowDown' })
    expect(within(list()).getAllByRole('option')[1]).toHaveAttribute('aria-selected', 'true')
  })
})

describe('ScenarioViewer — discard', () => {
  it('asks for confirmation and keeps the edits when cancelled', async () => {
    mount()
    await loaded()
    editTitle()
    fireEvent.click(screen.getByRole('button', { name: /Discard/ }))
    expect(screen.getByText('Discard unsaved changes to this scenario?')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByText('Discard unsaved changes to this scenario?')).not.toBeInTheDocument()
    expect(screen.getByDisplayValue('Renamed')).toBeInTheDocument()
  })

  it('restores the loaded values after confirming', async () => {
    mount()
    await loaded()
    editTitle()
    fireEvent.click(screen.getByRole('button', { name: /Discard/ }))
    const confirm = screen.getAllByRole('button', { name: 'Discard' }).at(-1)!
    fireEvent.click(confirm)
    expect(await screen.findByDisplayValue('My scenario')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Save/ })).not.toBeInTheDocument()
  })
})

describe('ScenarioViewer — saving', () => {
  it('saves serialized YAML, reloads and clears the result', async () => {
    const { spies } = mount()
    await loaded()
    editTitle('Renamed')
    fireEvent.click(screen.getByRole('button', { name: /Save/ }))
    await waitFor(() => expect(b.saveScenarioRaw).toHaveBeenCalledTimes(1))
    const [path, content] = b.saveScenarioRaw.mock.calls[0]
    expect(path).toBe('/ws/scenarios/s.yaml')
    expect(content).toContain('title: Renamed')
    expect(b.loadScenario).toHaveBeenCalledTimes(2)
    await waitFor(() => expect(spies.onResultChange).toHaveBeenCalledWith(null))
    expect(await screen.findByText('Saved')).toBeInTheDocument()
  })

  it('hides the Saved flash after a moment', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    mount()
    await loaded()
    editTitle()
    fireEvent.click(screen.getByRole('button', { name: /Save/ }))
    await screen.findByText('Saved')
    act(() => { vi.advanceTimersByTime(1600) })
    expect(screen.queryByText('Saved')).not.toBeInTheDocument()
  })

  it('shows a save error from the backend and stays dirty', async () => {
    b.saveScenarioRaw.mockResolvedValue('permission denied' as never)
    mount()
    await loaded()
    editTitle()
    fireEvent.click(screen.getByRole('button', { name: /Save/ }))
    expect(await screen.findByText('permission denied')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Save/ })).toBeInTheDocument()
  })

  it('shows a save failure from a rejected call', async () => {
    b.saveScenarioRaw.mockRejectedValue(new Error('ipc down'))
    mount()
    await loaded()
    editTitle()
    fireEvent.click(screen.getByRole('button', { name: /Save/ }))
    expect(await screen.findByText(/ipc down/)).toBeInTheDocument()
  })

  it('refuses to save a jsonpath assertion without a path', async () => {
    b.loadScenario.mockResolvedValue(detail({ steps: [step({ label: 'Bad', assertions: [{ type: 'jsonpath', operator: 'eq', expected: 1 }] })] }))
    mount()
    await loaded()
    editTitle()
    fireEvent.click(screen.getByRole('button', { name: /Save/ }))
    expect(await screen.findByText('Step "Bad": jsonpath assertion requires a path.')).toBeInTheDocument()
    expect(b.saveScenarioRaw).not.toHaveBeenCalled()
  })

  it('refuses to save an extraction without a variable name', async () => {
    b.loadScenario.mockResolvedValue(detail({ steps: [step({ label: 'Bad', extractions: [{ name: ' ', source: 'jsonpath', path: '$.a' }] })] }))
    mount()
    await loaded()
    editTitle()
    fireEvent.click(screen.getByRole('button', { name: /Save/ }))
    expect(await screen.findByText('Step "Bad": extraction is missing a variable name.')).toBeInTheDocument()
    expect(b.saveScenarioRaw).not.toHaveBeenCalled()
  })

  it('saves when the save signal increments, but not for zero or undefined', async () => {
    const spies: Spies = { onResultChange: vi.fn(), onRunningChange: vi.fn(), onDirtyChange: vi.fn(), onRunComplete: vi.fn() }
    const { rerender } = render(<Harness spies={spies} saveSignal={0} />)
    await loaded()
    editTitle()
    expect(b.saveScenarioRaw).not.toHaveBeenCalled()
    rerender(<Harness spies={spies} saveSignal={1} />)
    await waitFor(() => expect(b.saveScenarioRaw).toHaveBeenCalledTimes(1))
  })
})

describe('ScenarioViewer — running', () => {
  it('runs the scenario with the selected environment and reports lifecycle', async () => {
    b.runScenario.mockResolvedValue(resultOf([{ passed: true }]))
    const { spies } = mount({ environments: [{ name: 'local', path: '/e/local.yaml' }], activeEnvPath: '/e/local.yaml' })
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    await waitFor(() => expect(b.runScenario).toHaveBeenCalledWith('/ws/scenarios/s.yaml', '/e/local.yaml', 0))
    await waitFor(() => expect(spies.onRunComplete).toHaveBeenCalledTimes(1))
    expect(spies.onRunningChange).toHaveBeenNthCalledWith(1, true)
    expect(spies.onRunningChange).toHaveBeenLastCalledWith(false)
    expect(b.saveScenarioResult).toHaveBeenCalled()
    expect(await screen.findByTitle('View run statistics')).toBeInTheDocument()
  })

  it('lets the user change the environment before running', async () => {
    b.runScenario.mockResolvedValue(resultOf([{ passed: true }]))
    mount({ environments: [{ name: 'local', path: '/e/local.yaml' }, { name: 'prod', path: '/e/prod.yaml' }] })
    await loaded()
    fireEvent.change(screen.getByTitle('Environment for this run'), { target: { value: '/e/prod.yaml' } })
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    await waitFor(() => expect(b.runScenario).toHaveBeenCalledWith('/ws/scenarios/s.yaml', '/e/prod.yaml', 0))
  })

  it('selects the first failing step after a run', async () => {
    b.runScenario.mockResolvedValue(resultOf([{ passed: true }, { passed: false }, { passed: true }]))
    mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    await waitFor(() => expect(within(list()).getAllByRole('option')[1]).toHaveAttribute('aria-selected', 'true'))
  })

  it('shows spec warnings from the result', async () => {
    b.runScenario.mockResolvedValue(resultOf([{ passed: true }], { specError: 'cannot fetch spec' }))
    mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    expect(await screen.findByRole('img', { name: 'Setup warning' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Setup' }))
    fireEvent.click(screen.getByRole('tab', { name: 'result' }))
    expect(screen.getByText('cannot fetch spec')).toBeInTheDocument()
    expect(screen.getByText('specFile warning:')).toBeInTheDocument()
  })

  it('shows a setup error under the header and opens its details in Setup', async () => {
    b.runScenario.mockResolvedValue(resultOf([], { passed: false, error: 'variable "x" is bad' }))
    mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    expect(await screen.findByText('Scenario could not start.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Setup' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getAllByText(/variable "x" is bad/).length).toBeGreaterThan(0)
  })

  it('shows accumulated coverage returned by the run', async () => {
    b.runScenario.mockResolvedValue(resultOf([{ passed: true }], {
      accumulatedCoverage: { specFile: 'api.yaml', covered: 2, total: 2, percent: 100, totalRuns: 1, passedRuns: 1, failedRuns: 0, exercised: [], missed: [] },
    }))
    mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    fireEvent.click(await screen.findByRole('button', { name: /Coverage 2\/2 · 100%/ }))
    expect(await screen.findByText('Accumulated Coverage')).toBeInTheDocument()
  })

  it('reports a run failure as a failed result', async () => {
    b.runScenario.mockRejectedValue(new Error('engine crashed'))
    const { spies } = mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    await waitFor(() => expect(spies.onResultChange).toHaveBeenCalledWith(expect.objectContaining({ passed: false, error: expect.stringContaining('engine crashed') })))
    expect(spies.onRunComplete).toHaveBeenCalled()
  })

  it('saves unsaved edits before running, and aborts the run if saving fails', async () => {
    b.runScenario.mockResolvedValue(resultOf([{ passed: true }]))
    mount()
    await loaded()
    editTitle()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    await waitFor(() => expect(b.saveScenarioRaw).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(b.runScenario).toHaveBeenCalledTimes(1))
  })

  it('does not run when the pre-run save fails', async () => {
    b.saveScenarioRaw.mockResolvedValue('disk full' as never)
    mount()
    await loaded()
    editTitle()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    expect(await screen.findByText('disk full')).toBeInTheDocument()
    expect(b.runScenario).not.toHaveBeenCalled()
  })

  it('runs from a chosen step and selects the failure or the start step', async () => {
    b.runScenario.mockResolvedValueOnce(resultOf([{ passed: true }, { passed: true }, { passed: true }]))
    mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    await screen.findByTitle('View run statistics')
    fireEvent.click(within(list()).getAllByRole('option')[2])
    b.runScenario.mockResolvedValueOnce(resultOf([{ passed: true }, { passed: true }, { passed: true }]))
    fireEvent.click(await screen.findByTitle('Run scenario from this step'))
    await waitFor(() => expect(b.runScenario).toHaveBeenLastCalledWith('/ws/scenarios/s.yaml', '', 2))
    await waitFor(() => expect(within(list()).getAllByRole('option')[2]).toHaveAttribute('aria-selected', 'true'))
  })

  it('selects the failing step after running from a step', async () => {
    b.runScenario.mockResolvedValueOnce(resultOf([{ passed: true }, { passed: true }, { passed: true }]))
    mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    await screen.findByTitle('View run statistics')
    b.runScenario.mockResolvedValueOnce(resultOf([{ passed: true }, { passed: false }, { passed: true }]))
    fireEvent.click(await screen.findByTitle('Run scenario from this step'))
    await waitFor(() => expect(within(list()).getAllByRole('option')[1]).toHaveAttribute('aria-selected', 'true'))
  })

  it('reports a run-from failure as a failed result', async () => {
    b.runScenario.mockResolvedValueOnce(resultOf([{ passed: true }, { passed: true }, { passed: true }]))
    const { spies } = mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    await screen.findByTitle('View run statistics')
    b.runScenario.mockRejectedValueOnce(new Error('boom'))
    fireEvent.click(await screen.findByTitle('Run scenario from this step'))
    await waitFor(() => expect(spies.onResultChange).toHaveBeenCalledWith(expect.objectContaining({ error: expect.stringContaining('boom') })))
  })

  it('offers Stop while running and cancels', async () => {
    let release!: (r: ScenarioResult) => void
    b.runScenario.mockReturnValue(new Promise(r => { release = r }))
    mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    fireEvent.click(await screen.findByRole('button', { name: /Stop/ }))
    expect(b.cancelScenario).toHaveBeenCalledTimes(1)
    await act(async () => { release(resultOf([{ passed: true }])) })
  })

  it('swallows a failing cancel call', async () => {
    let release!: (r: ScenarioResult) => void
    b.runScenario.mockReturnValue(new Promise(r => { release = r }))
    b.cancelScenario.mockRejectedValue(new Error('no'))
    mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    fireEvent.click(await screen.findByRole('button', { name: /Stop/ }))
    await act(async () => { release(resultOf([{ passed: true }])) })
  })

  it('shows a progress bar while running and no extra bar afterwards', async () => {
    let release!: (r: ScenarioResult) => void
    b.runScenario.mockReturnValue(new Promise(r => { release = r }))
    const { container } = mount()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /^Run/ }))
    await screen.findByRole('button', { name: /Stop/ })
    expect(container.querySelector('[style*="slide"]')).toBeInTheDocument()
    await act(async () => { release(resultOf([{ passed: true }, { passed: false }, { passed: false, skipped: true }])) })
    expect(container.querySelector('[style*="slide"]')).not.toBeInTheDocument()
    expect(container.querySelector('.bg-emerald-500.transition-all')).not.toBeInTheDocument()
    expect(screen.getAllByText('FAILED').length).toBeGreaterThan(0)
  })

  it('opens and closes the stats dialog', async () => {
    b.loadScenarioResult.mockResolvedValue(resultOf([{ passed: true }, { passed: false }]))
    mount()
    fireEvent.click(await screen.findByTitle('View run statistics'))
    expect(await screen.findByText('1 passed · 1 failed · 0 skipped')).toBeInTheDocument()
  })
})

describe('ScenarioViewer — YAML dialog', () => {
  it('opens with the serialized scenario', async () => {
    mount()
    await loaded()
    fireEvent.click(screen.getByTitle('View / edit YAML'))
    expect((await screen.findByLabelText('yaml') as HTMLTextAreaElement).value).toContain('title: My scenario')
  })

  it('saves edited YAML, reloads and closes the dialog', async () => {
    const { spies } = mount()
    await loaded()
    fireEvent.click(screen.getByTitle('View / edit YAML'))
    fireEvent.change(await screen.findByLabelText('yaml'), { target: { value: 'title: From YAML' } })
    fireEvent.click(screen.getByRole('button', { name: /^Save$/ }))
    await waitFor(() => expect(b.saveScenarioRaw).toHaveBeenCalledWith('/ws/scenarios/s.yaml', 'title: From YAML'))
    await waitFor(() => expect(screen.queryByLabelText('yaml')).not.toBeInTheDocument())
    expect(spies.onResultChange).toHaveBeenCalledWith(null)
  })

  it('keeps the dialog open and shows the error when the YAML save fails', async () => {
    b.saveScenarioRaw.mockResolvedValue('yaml: invalid' as never)
    mount()
    await loaded()
    fireEvent.click(screen.getByTitle('View / edit YAML'))
    fireEvent.change(await screen.findByLabelText('yaml'), { target: { value: 'oops' } })
    fireEvent.click(screen.getByRole('button', { name: /^Save$/ }))
    await waitFor(() => expect(b.saveScenarioRaw).toHaveBeenCalled())
    expect(screen.getByLabelText('yaml')).toBeInTheDocument()
  })

  it('surfaces a reload error after saving YAML', async () => {
    mount()
    await loaded()
    fireEvent.click(screen.getByTitle('View / edit YAML'))
    fireEvent.change(await screen.findByLabelText('yaml'), { target: { value: 'broken' } })
    b.loadScenario.mockResolvedValueOnce(detail({ error: 'parse error after save' }))
    fireEvent.click(screen.getByRole('button', { name: /^Save$/ }))
    expect(await screen.findByText('parse error after save')).toBeInTheDocument()
  })
})

describe('ScenarioViewer — layout', () => {
  it('persists the divider position after dragging', async () => {
    mount()
    await loaded()
    fireEvent.mouseDown(screen.getByTitle('Drag to resize'), { clientX: 100 })
    fireEvent.mouseMove(document, { clientX: 300 })
    fireEvent.mouseUp(document)
    expect(Number(localStorage.getItem('ui.scenarioStepPanelPct'))).toBeGreaterThanOrEqual(20)
    expect(Number(localStorage.getItem('ui.scenarioStepPanelPct'))).toBeLessThanOrEqual(50)
  })

  it('clamps the panel to its limits', async () => {
    mount()
    await loaded()
    fireEvent.mouseDown(screen.getByTitle('Drag to resize'), { clientX: 0 })
    fireEvent.mouseMove(document, { clientX: -100000 })
    fireEvent.mouseUp(document)
    expect(localStorage.getItem('ui.scenarioStepPanelPct')).toBe('20')
    fireEvent.mouseDown(screen.getByTitle('Drag to resize'), { clientX: 0 })
    fireEvent.mouseMove(document, { clientX: 100000 })
    fireEvent.mouseUp(document)
    expect(localStorage.getItem('ui.scenarioStepPanelPct')).toBe('50')
  })

  it('restores a saved, in-range width', async () => {
    localStorage.setItem('ui.scenarioStepPanelPct', '42')
    const { container } = mount()
    await loaded()
    expect(container.querySelector<HTMLElement>('[style*="width: 42%"]')).toBeInTheDocument()
  })

  it('falls back to the default for an invalid or out-of-range saved width', async () => {
    localStorage.setItem('ui.scenarioStepPanelPct', 'abc')
    const { container, unmount } = mount()
    await loaded()
    expect(container.querySelector<HTMLElement>('[style*="width: 35%"]')).toBeInTheDocument()
    unmount()
    localStorage.setItem('ui.scenarioStepPanelPct', '90')
    const second = mount()
    await loaded()
    expect(second.container.querySelector<HTMLElement>('[style*="width: 50%"]')).toBeInTheDocument()
  })
})
