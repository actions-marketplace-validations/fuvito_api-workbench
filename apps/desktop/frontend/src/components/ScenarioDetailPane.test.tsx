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

  it('shows a passed result with status, duration and passing assertions, collapsed by default', () => {
    setup({ result: resultOf(res({ assertions: [{ type: 'status', passed: true, message: 'status eq 200' }] })) })
    expect(screen.getByText('PASSED')).toBeInTheDocument()
    expect(screen.getAllByText('200').length).toBeGreaterThan(0)
    expect(screen.getAllByText('12 ms').length).toBeGreaterThan(0)
    expect(screen.queryByText('status eq 200')).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('PASSED'))
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
    expect(screen.getByText('FAILED')).toBeInTheDocument()
    expect(screen.getByText('boom')).toBeInTheDocument()
    expect(screen.getByText('expected 200 got 500')).toBeInTheDocument()
    expect(screen.queryByText('header ok')).not.toBeInTheDocument()
  })

  it('shows skipped steps with an explanation and no status or duration', () => {
    setup({ result: resultOf(res({ passed: false, skipped: true, statusCode: 0, durationMs: 0 })) })
    expect(screen.getByText('SKIPPED')).toBeInTheDocument()
    expect(screen.queryByText('Skipped — previous step failed.')).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('SKIPPED'))
    expect(screen.getByText('Skipped — previous step failed.')).toBeInTheDocument()
    expect(screen.queryByText(/ ms$/)).not.toBeInTheDocument()
  })

  it('lists extracted variables when the strip is open', () => {
    setup({ result: resultOf(res({ passed: false, extracted: { token: 'abc' } })) })
    expect(screen.getByText('→ token:')).toBeInTheDocument()
    expect(screen.getByText('abc')).toBeInTheDocument()
  })

  it('tolerates a result with no assertions or extracted map', () => {
    setup({ result: resultOf(res({ passed: false, assertions: undefined as never, extracted: undefined as never })) })
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

  it('forwards editor changes with the selected index', () => {
    const p = setup({ selectedStepIndex: 1 })
    fireEvent.change(screen.getByDisplayValue('Second'), { target: { value: 'Renamed' } })
    expect(p.onUpdateStep).toHaveBeenCalledWith(1, expect.objectContaining({ label: 'Renamed' }))
  })
})
