import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ScenarioSetupPane } from './ScenarioSetupPane'
import type { ScenarioDetail, ScenarioResult, AccumulatedCoverage } from '@/lib/bridge'

const detail: ScenarioDetail = {
  path: '/s.yaml', title: 'S', description: 'about it', specFile: 'api.yaml',
  variables: [{ name: 'email', value: 'a@b.c' }],
  steps: [],
}

const result: ScenarioResult = {
  title: 't', passed: true, durationMs: 1, steps: [],
  seed: 42, variables: [{ name: 'email', value: 'gen@x.io' }],
  coverage: { covered: 1, total: 2, percent: 50, exercised: ['GET /a'], missed: ['GET /b'] },
}

const accumulated: AccumulatedCoverage = {
  specFile: 'api.yaml', covered: 2, total: 2, percent: 100, totalRuns: 3, passedRuns: 3, failedRuns: 0, exercised: [], missed: [],
}

function setup(over: Partial<React.ComponentProps<typeof ScenarioSetupPane>> = {}) {
  const props: React.ComponentProps<typeof ScenarioSetupPane> = {
    tab: 'definition',
    onTabChange: vi.fn(),
    detail,
    result: null,
    running: false,
    accumulatedCoverage: null,
    onChange: vi.fn(),
    onResetCoverage: vi.fn(),
    ...over,
  }
  const utils = render(<ScenarioSetupPane {...props} />)
  return { props, ...utils }
}

describe('ScenarioSetupPane', () => {
  it('shows description, spec file and variables in the Definition tab', () => {
    setup()
    expect(screen.getByDisplayValue('about it')).toBeInTheDocument()
    expect(screen.getByDisplayValue('api.yaml')).toBeInTheDocument()
    expect(screen.getByLabelText('Variable 1 name')).toHaveValue('email')
    expect(screen.getByRole('tab', { name: 'definition' })).toHaveAttribute('aria-selected', 'true')
  })

  it('opens the variables section even when there are none', () => {
    setup({ detail: { ...detail, variables: [] } })
    expect(screen.getByRole('button', { name: 'Add variable' })).toBeInTheDocument()
  })

  it('forwards edits', () => {
    const { props } = setup()
    fireEvent.change(screen.getByPlaceholderText('What does this scenario test?'), { target: { value: 'new' } })
    expect(props.onChange).toHaveBeenCalledWith({ description: 'new' })
    fireEvent.click(screen.getByRole('button', { name: 'Add variable' }))
    expect(props.onChange).toHaveBeenLastCalledWith({ variables: [...detail.variables!, { name: '', value: '' }] })
  })

  describe('Result tab gating', () => {
    it('is disabled with no result and no coverage, and a requested Result falls back to Definition', () => {
      setup({ tab: 'result' })
      expect(screen.getByRole('tab', { name: 'result' })).toBeDisabled()
      expect(screen.getByRole('tab', { name: 'definition' })).toHaveAttribute('aria-selected', 'true')
      expect(screen.getByDisplayValue('about it')).toBeInTheDocument()
    })

    it('is disabled while a run is in progress', () => {
      setup({ result: { title: 't', passed: true, durationMs: 1, steps: [] }, running: true })
      expect(screen.getByRole('tab', { name: 'result' })).toBeDisabled()
    })

    it('is enabled by a finished result', () => {
      const { props } = setup({ result })
      const tab = screen.getByRole('tab', { name: 'result' })
      expect(tab).toBeEnabled()
      fireEvent.click(tab)
      expect(props.onTabChange).toHaveBeenCalledWith('result')
    })

    it('is enabled by accumulated coverage alone', () => {
      setup({ accumulatedCoverage: accumulated, tab: 'result' })
      expect(screen.getByRole('tab', { name: 'result' })).toBeEnabled()
      expect(screen.getByText('Accumulated Coverage')).toBeInTheDocument()
    })
  })

  describe('Result tab content', () => {
    it('shows the seed, generated variables and coverage of the run', () => {
      setup({ result, tab: 'result' })
      expect(screen.getByText('clrsig run --seed 42')).toBeInTheDocument()
      expect(screen.getByText('gen@x.io')).toBeInTheDocument()
      expect(screen.getByText('Spec Coverage')).toBeInTheDocument()
      expect(screen.queryByDisplayValue('about it')).not.toBeInTheDocument()
    })

    it('shows the spec warning and the setup error', () => {
      setup({ result: { ...result, specError: 'cannot fetch spec', error: 'unknown function' }, tab: 'result' })
      expect(screen.getByText('cannot fetch spec')).toBeInTheDocument()
      expect(screen.getByRole('alert')).toHaveTextContent('unknown function')
    })

    it('resets accumulated coverage', () => {
      const { props } = setup({ result, accumulatedCoverage: accumulated, tab: 'result' })
      fireEvent.click(screen.getByTitle('Reset accumulated coverage'))
      expect(props.onResetCoverage).toHaveBeenCalledTimes(1)
    })

    it('says so when a run produced nothing to report', () => {
      setup({ result: { title: 't', passed: true, durationMs: 1, steps: [] }, tab: 'result' })
      expect(screen.getByText('Nothing was generated or reported for this run.')).toBeInTheDocument()
    })
  })
})
