import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { StepDetailPane } from './ScenarioDetailPane'
import type { ScenarioDetail, ScenarioResult, StepResult } from '@/lib/bridge'

const detail: ScenarioDetail = {
  path: '/s.yaml', title: 'S', description: '',
  steps: [
    { label: 'First', method: 'GET', url: '/a', assertions: [], extractions: [] },
    { label: 'Second', method: 'POST', url: '/b', assertions: [], extractions: [] },
  ],
}

function res(over: Partial<StepResult> = {}): StepResult {
  return { label: 'x', passed: true, skipped: false, statusCode: 200, durationMs: 12, extracted: {}, assertions: [], ...over }
}

function resultOf(...steps: StepResult[]): ScenarioResult {
  return { title: 't', passed: true, durationMs: 1, steps }
}

function openResult() {
  fireEvent.click(screen.getByRole('tab', { name: /Result/ }))
}

function setup(over: Partial<React.ComponentProps<typeof StepDetailPane>> = {}) {
  const props: React.ComponentProps<typeof StepDetailPane> = {
    visualDetail: detail,
    result: null,
    running: false,
    selectedStepIndex: 0,
    editStepCount: 2,
    onUpdateStep: vi.fn(),
    onRemoveStep: vi.fn(),
    onMoveStep: vi.fn(),
    onToggleDisabled: vi.fn(),
    onRunFrom: vi.fn(),
    ...over,
  }
  render(<StepDetailPane {...props} />)
  return props
}

describe('StepDetailPane', () => {
  it('prompts to select a step when none is selected', () => {
    setup({ selectedStepIndex: null })
    expect(screen.getByText('Select a step to edit, or add a new one.')).toBeInTheDocument()
  })

  it('prompts when the detail has not loaded', () => {
    setup({ visualDetail: null })
    expect(screen.getByText('Select a step to edit, or add a new one.')).toBeInTheDocument()
  })

  it('prompts when the selected index is out of range', () => {
    setup({ selectedStepIndex: 9 })
    expect(screen.getByText('Select a step to edit, or add a new one.')).toBeInTheDocument()
  })

  it('renders the editor for the selected step with no result strip', () => {
    setup()
    expect(screen.getByRole('region', { name: 'Step details' })).toBeInTheDocument()
    expect(screen.getByDisplayValue('First')).toBeInTheDocument()
    expect(screen.queryByText('PASSED')).not.toBeInTheDocument()
  })

  it('shows a passed result in the Result tab with status, duration and assertions', () => {
    setup({ result: resultOf(res({ assertions: [{ type: 'status', passed: true, message: 'status eq 200' }] })) })
    expect(screen.getByText('Last run:')).toBeInTheDocument()
    expect(screen.getByText('PASSED')).toBeInTheDocument()
    expect(screen.getByDisplayValue('First')).toBeInTheDocument()
    openResult()
    expect(screen.queryByDisplayValue('First')).not.toBeInTheDocument()
    expect(screen.getAllByText('200').length).toBeGreaterThan(0)
    expect(screen.getAllByText('12 ms').length).toBeGreaterThan(0)
    expect(screen.getByText('status eq 200')).toBeInTheDocument()
    fireEvent.click(screen.getByText('PASSED'))
    expect(screen.queryByText('status eq 200')).not.toBeInTheDocument()
  })

  it('opens a failed result by default with error and failing assertions only', () => {
    setup({
      result: resultOf(res({
        passed: false, statusCode: 500, error: 'boom',
        assertions: [
          { type: 'status', passed: false, message: 'expected 200 got 500' },
          { type: 'header', passed: true, message: 'header ok' },
        ],
      })),
    })
    expect(screen.getByText('1 assertion failed')).toBeInTheDocument()
    openResult()
    expect(screen.getByText('FAILED')).toBeInTheDocument()
    expect(screen.getByText('boom')).toBeInTheDocument()
    expect(screen.getByText('expected 200 got 500')).toBeInTheDocument()
    expect(screen.queryByText('header ok')).not.toBeInTheDocument()
  })

  it('shows skipped steps with an explanation and no status or duration', () => {
    setup({ result: resultOf(res({ passed: false, skipped: true, statusCode: 0, durationMs: 0 })) })
    expect(screen.getByText('SKIPPED')).toBeInTheDocument()
    expect(screen.queryByText('Skipped — previous step failed.')).not.toBeInTheDocument()
    openResult()
    expect(screen.getByText('Skipped — previous step failed.')).toBeInTheDocument()
    expect(screen.queryByText(/ ms$/)).not.toBeInTheDocument()
  })

  it('lists extracted variables when the strip is open', () => {
    setup({ result: resultOf(res({ passed: false, extracted: { token: 'abc' } })) })
    openResult()
    expect(screen.getByText('→ token:')).toBeInTheDocument()
    expect(screen.getByText('abc')).toBeInTheDocument()
  })

  it('tolerates a result with no assertions or extracted map', () => {
    setup({ result: resultOf(res({ passed: false, assertions: undefined as never, extracted: undefined as never })) })
    expect(screen.getByText('FAILED')).toBeInTheDocument()
    openResult()
    expect(screen.getByText('FAILED')).toBeInTheDocument()
  })

  it('runs from the selected step and disables while running', () => {
    const p = setup({ selectedStepIndex: 1, result: resultOf(res(), res()) })
    fireEvent.click(screen.getByTitle('Run scenario from this step'))
    expect(p.onRunFrom).toHaveBeenCalledWith(1)
  })

  it('disables run-from while running', () => {
    setup({ running: true, result: resultOf(res()) })
    expect(screen.getByTitle('Run scenario from this step')).toBeDisabled()
  })

  it('shows no result strip when the selected step has no result', () => {
    setup({ selectedStepIndex: 1, result: resultOf(res()) })
    expect(screen.queryByText('PASSED')).not.toBeInTheDocument()
  })

  it('disables the Result tab until the step has a result', () => {
    setup({ selectedStepIndex: 1, result: resultOf(res()) })
    expect(screen.getByRole('tab', { name: /Result/ })).toBeDisabled()
    expect(screen.getByRole('tab', { name: 'Definition' })).toHaveAttribute('aria-selected', 'true')
  })

  it('switches to Result once when a run ends on a failed step, and back to Definition when a run starts', () => {
    const failed = resultOf(res({ passed: false, statusCode: 404 }))
    const base: React.ComponentProps<typeof StepDetailPane> = {
      visualDetail: detail, result: null, running: true, selectedStepIndex: 0, editStepCount: 2,
      onUpdateStep: vi.fn(), onRemoveStep: vi.fn(), onMoveStep: vi.fn(), onToggleDisabled: vi.fn(), onRunFrom: vi.fn(),
    }
    const { rerender } = render(<StepDetailPane {...base} />)
    rerender(<StepDetailPane {...base} running={false} result={failed} />)
    expect(screen.getByRole('tab', { name: /Result/ })).toHaveAttribute('aria-selected', 'true')
    fireEvent.click(screen.getByRole('tab', { name: 'Definition' }))
    rerender(<StepDetailPane {...base} running={false} result={{ ...failed }} />)
    expect(screen.getByRole('tab', { name: 'Definition' })).toHaveAttribute('aria-selected', 'true')
    rerender(<StepDetailPane {...base} running result={null} />)
    rerender(<StepDetailPane {...base} running={false} result={resultOf(res())} />)
    expect(screen.getByRole('tab', { name: 'Definition' })).toHaveAttribute('aria-selected', 'true')
  })

  it('marks a result stale once its step is edited', () => {
    setup({ result: resultOf(res()) })
    fireEvent.change(screen.getByDisplayValue('First'), { target: { value: 'Renamed' } })
    expect(screen.getByText('stale: edited since this run')).toBeInTheDocument()
    openResult()
    expect(screen.getByText(/edited after the run/)).toBeInTheDocument()
  })

  it('opens a step\'s filled sections when the selection moves to it', () => {
    const withHeader: ScenarioDetail = {
      ...detail,
      steps: [detail.steps[0], { ...detail.steps[1], headers: [{ name: 'X-Two', value: '2' }] }],
    }
    const base: React.ComponentProps<typeof StepDetailPane> = {
      visualDetail: withHeader, result: null, running: false, selectedStepIndex: 0, editStepCount: 2,
      onUpdateStep: vi.fn(), onRemoveStep: vi.fn(), onMoveStep: vi.fn(), onToggleDisabled: vi.fn(), onRunFrom: vi.fn(),
    }
    const { rerender } = render(<StepDetailPane {...base} />)
    expect(screen.queryByDisplayValue('X-Two')).not.toBeInTheDocument()
    rerender(<StepDetailPane {...base} selectedStepIndex={1} />)
    expect(screen.getByDisplayValue('X-Two')).toBeInTheDocument()
  })

  it('warns that run-from-here leaves extracted values unset', () => {
    const steps = [
      { label: 'Login', method: 'POST', url: '/l', assertions: [], extractions: [{ name: 'token', source: 'jsonpath', path: '$.t' }] },
      { label: 'Use', method: 'GET', url: '/u/{{token}}', assertions: [], extractions: [] },
    ]
    setup({ visualDetail: { ...detail, steps }, selectedStepIndex: 1, result: resultOf(res(), res()) })
    expect(screen.getByRole('note')).toHaveTextContent('{{token}} is not set')
    openResult()
    expect(screen.getByRole('note')).toHaveTextContent('{{token}} is not set')
  })

  it('shows no run-from warning when nothing is missing', () => {
    setup({ result: resultOf(res()) })
    expect(screen.queryByRole('note')).not.toBeInTheDocument()
  })

  it('forwards editor changes with the selected index', () => {
    const p = setup({ selectedStepIndex: 1 })
    fireEvent.change(screen.getByDisplayValue('Second'), { target: { value: 'Renamed' } })
    expect(p.onUpdateStep).toHaveBeenCalledWith(1, expect.objectContaining({ label: 'Renamed' }))
  })
})
