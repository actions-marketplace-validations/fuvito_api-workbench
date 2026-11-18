import { describe, it, expect, vi } from 'vitest'
import { createRef } from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { StepListPanel } from './ScenarioStepListPanel'
import { StepListRow } from './ScenarioStepList'
import type { StepDetail, StepResult, ScenarioResult } from '@/lib/bridge'

function step(over: Partial<StepDetail> = {}): StepDetail {
  return { label: 'Step', method: 'GET', url: '/a', assertions: [], extractions: [], ...over }
}

function res(over: Partial<StepResult> = {}): StepResult {
  return { label: 'Step', passed: true, skipped: false, statusCode: 200, durationMs: 1, extracted: {}, assertions: [], ...over }
}

function panel(over: Partial<React.ComponentProps<typeof StepListPanel>> = {}) {
  const props: React.ComponentProps<typeof StepListPanel> = {
    widthPct: 35,
    listRef: createRef<HTMLDivElement>(),
    loadError: null,
    detailLoaded: true,
    steps: [step({ label: 'One' }), step({ label: 'Two' }), step({ label: 'Three' })],
    selectedStepIndex: 0,
    onSelectStep: vi.fn(),
    setupSelected: false,
    onSelectSetup: vi.fn(),
    result: null,
    running: false,
    onMoveStep: vi.fn(),
    onRemoveStep: vi.fn(),
    onDuplicateStep: vi.fn(),
    onToggleDisabled: vi.fn(),
    onAddStep: vi.fn(),
    ...over,
  }
  const utils = render(<StepListPanel {...props} />)
  return { props, ...utils }
}

describe('StepListPanel', () => {
  it('lists steps and applies the width', () => {
    const { container } = panel()
    expect(screen.getAllByRole('option')).toHaveLength(3)
    expect((container.firstChild as HTMLElement).style.width).toBe('35%')
  })

  it('shows the load error instead of steps', () => {
    panel({ loadError: 'bad yaml' })
    expect(screen.getByText('bad yaml')).toBeInTheDocument()
    expect(screen.queryAllByRole('option')).toHaveLength(0)
  })

  it('shows skeleton rows until the detail has loaded', () => {
    const { container } = panel({ detailLoaded: false })
    expect(container.querySelectorAll('.animate-pulse').length).toBe(6)
    expect(screen.queryAllByRole('option')).toHaveLength(0)
  })

  it('shows a Start here row instead of the add button when there are no steps', () => {
    const { props } = panel({ steps: [] })
    expect(screen.queryByRole('button', { name: /Add step/ })).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Start here'))
    expect(props.onAddStep).toHaveBeenCalledTimes(1)
  })

  it('renders the Setup row outside the option set with its badges', () => {
    const { props } = panel({ variableCount: 2, setupAlert: 'error' })
    expect(screen.getAllByRole('option')).toHaveLength(3)
    expect(screen.getByText('2 variables')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Setup error' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Setup' }))
    expect(props.onSelectSetup).toHaveBeenCalledTimes(1)
  })

  it('adds a step', () => {
    const { props } = panel()
    fireEvent.click(screen.getByRole('button', { name: /Add step/ }))
    expect(props.onAddStep).toHaveBeenCalledTimes(1)
  })

  it('navigates with arrow keys within bounds', () => {
    const { props } = panel({ selectedStepIndex: 1 })
    const list = screen.getByRole('listbox')
    fireEvent.keyDown(list, { key: 'ArrowDown' })
    expect(props.onSelectStep).toHaveBeenLastCalledWith(2)
    fireEvent.keyDown(list, { key: 'ArrowUp' })
    expect(props.onSelectStep).toHaveBeenLastCalledWith(0)
  })

  it('clamps arrow navigation at the ends', () => {
    const { props } = panel({ selectedStepIndex: 2 })
    fireEvent.keyDown(screen.getByRole('listbox'), { key: 'ArrowDown' })
    expect(props.onSelectStep).toHaveBeenLastCalledWith(2)
  })

  it('starts from the first step when nothing is selected', () => {
    const { props } = panel({ selectedStepIndex: null })
    fireEvent.keyDown(screen.getByRole('listbox'), { key: 'ArrowDown' })
    expect(props.onSelectStep).toHaveBeenLastCalledWith(0)
    fireEvent.keyDown(screen.getByRole('listbox'), { key: 'ArrowUp' })
    expect(props.onSelectStep).toHaveBeenLastCalledWith(0)
  })

  it('ignores other keys and empty lists', () => {
    const { props, unmount } = panel()
    fireEvent.keyDown(screen.getByRole('listbox'), { key: 'a' })
    expect(props.onSelectStep).not.toHaveBeenCalled()
    unmount()
    const empty = panel({ steps: [] })
    fireEvent.keyDown(screen.getByRole('listbox'), { key: 'ArrowDown' })
    expect(empty.props.onSelectStep).not.toHaveBeenCalled()
  })

  it('wires row actions with the right indices', () => {
    const { props } = panel()
    const rows = screen.getAllByRole('option')
    fireEvent.click(rows[1].querySelector('[title="Move up (Alt+↑)"]')!)
    fireEvent.click(rows[1].querySelector('[title="Move down (Alt+↓)"]')!)
    fireEvent.click(rows[1].querySelector('[title="Remove step"]')!)
    fireEvent.click(rows[1].querySelector('[title="Duplicate step"]')!)
    fireEvent.click(rows[1].querySelector('[title="Disable step"]')!)
    expect(props.onMoveStep).toHaveBeenNthCalledWith(1, 1, -1)
    expect(props.onMoveStep).toHaveBeenNthCalledWith(2, 1, 1)
    expect(props.onRemoveStep).toHaveBeenCalledWith(1)
    expect(props.onDuplicateStep).toHaveBeenCalledWith(1)
    expect(props.onToggleDisabled).toHaveBeenCalledWith(1)
  })

  it('disables moving the first step up and the last step down', () => {
    panel()
    const rows = screen.getAllByRole('option')
    expect(rows[0].querySelector('[title="Move up (Alt+↑)"]')).toBeDisabled()
    expect(rows[2].querySelector('[title="Move down (Alt+↓)"]')).toBeDisabled()
  })

  it('passes per-step results to rows', () => {
    const result: ScenarioResult = { title: 't', passed: false, durationMs: 1, steps: [res(), res({ passed: false })] }
    const { container } = panel({ result })
    expect(container.querySelectorAll('.text-emerald-400').length).toBeGreaterThan(0)
    expect(container.querySelectorAll('.text-red-400').length).toBeGreaterThan(0)
  })
})

describe('StepListRow', () => {
  function row(over: Partial<React.ComponentProps<typeof StepListRow>> = {}) {
    const props: React.ComponentProps<typeof StepListRow> = {
      step: step({ label: 'Login', url: '/login' }),
      index: 0,
      result: null,
      selected: false,
      onSelect: vi.fn(),
      ...over,
    }
    const utils = render(<StepListRow {...props} />)
    return { props, ...utils }
  }

  it('shows number, label, method and url', () => {
    row({ index: 2, step: step({ label: 'Login', method: 'POST', url: '/login' }) })
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('Login')).toBeInTheDocument()
    expect(screen.getByText('POST')).toBeInTheDocument()
    expect(screen.getByText('/login')).toBeInTheDocument()
  })

  it('falls back to a placeholder label', () => {
    row({ step: step({ label: '' }) })
    expect(screen.getByText('Unlabeled step')).toBeInTheDocument()
  })

  it('shows notes and a request reference when present', () => {
    row({ step: step({ notes: 'Needs token', requestRef: 'requests/login.yaml' }) })
    expect(screen.getByText('Needs token')).toBeInTheDocument()
    expect(screen.getByText('↗ requests/login.yaml')).toBeInTheDocument()
    expect(screen.queryByText('/login')).not.toBeInTheDocument()
  })

  it('shows neither method row nor ref for a bare step', () => {
    row({ step: { label: 'Bare', assertions: [], extractions: [] } })
    expect(screen.getByText('Bare')).toBeInTheDocument()
  })

  it('selects on click, Enter and Space only', () => {
    const { props } = row({ index: 4 })
    const opt = screen.getByRole('option')
    fireEvent.click(opt)
    fireEvent.keyDown(opt, { key: 'Enter' })
    fireEvent.keyDown(opt, { key: ' ' })
    fireEvent.keyDown(opt, { key: 'a' })
    expect(props.onSelect).toHaveBeenCalledTimes(3)
    expect(props.onSelect).toHaveBeenCalledWith(4)
  })

  it('reflects selection and disabled styling', () => {
    row({ selected: true, step: step({ disabled: true }) })
    const opt = screen.getByRole('option')
    expect(opt).toHaveAttribute('aria-selected', 'true')
    expect(opt.className).toContain('opacity-50')
    expect(screen.getByText('Step').className).toContain('line-through')
  })

  it('shows a spinner for steps still pending in a running scenario', () => {
    const { container } = row({ running: true, result: null })
    expect(container.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it.each([
    ['passed', res(), 'text-emerald-400'],
    ['failed', res({ passed: false }), 'text-red-400'],
    ['skipped', res({ passed: false, skipped: true }), 'text-[var(--color-muted-foreground)]'],
  ])('renders a %s result icon', (_n, result, cls) => {
    const { container } = row({ result })
    expect(container.querySelector(`svg.${cls.replace(/[[\]()]/g, '\\$&')}`)).toBeInTheDocument()
  })

  it('offers "from here" in read mode and calls onRunFrom without selecting', () => {
    const onRunFrom = vi.fn()
    const { props } = row({ onRunFrom })
    fireEvent.click(screen.getByTitle('Run from this step'))
    expect(onRunFrom).toHaveBeenCalledTimes(1)
    expect(props.onSelect).not.toHaveBeenCalled()
  })

  it('offers neither edit nor run-from controls by default', () => {
    row()
    expect(screen.queryByTitle('Run from this step')).not.toBeInTheDocument()
    expect(screen.queryByTitle('Remove step')).not.toBeInTheDocument()
  })

  it('edit mode: toggles enable state label by disabled flag', () => {
    row({ editMode: true, step: step({ disabled: true }) })
    expect(screen.getByTitle('Enable step')).toBeInTheDocument()
  })

  it('edit mode tolerates missing handlers', () => {
    row({ editMode: true })
    for (const t of ['Disable step', 'Duplicate step', 'Move up (Alt+↑)', 'Move down (Alt+↓)', 'Remove step']) {
      expect(() => fireEvent.click(screen.getByTitle(t))).not.toThrow()
    }
  })
})
