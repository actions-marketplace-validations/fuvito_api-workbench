import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ScenarioHeaderBar } from './ScenarioHeaderBar'
import type { ScenarioResult } from '@/lib/bridge'

const envs = [{ name: 'local', path: '/e/local.yaml' }, { name: 'prod', path: '/e/prod.yaml' }]

function result(over: Partial<ScenarioResult> = {}): ScenarioResult {
  return {
    title: 't', passed: true, durationMs: 120,
    steps: [
      { label: 'a', passed: true, skipped: false, statusCode: 200, durationMs: 1, extracted: {}, assertions: [] },
      { label: 'b', passed: false, skipped: false, statusCode: 500, durationMs: 1, extracted: {}, assertions: [] },
      { label: 'c', passed: false, skipped: true, statusCode: 0, durationMs: 0, extracted: {}, assertions: [] },
    ],
    ...over,
  }
}

function setup(over: Partial<React.ComponentProps<typeof ScenarioHeaderBar>> = {}) {
  const props: React.ComponentProps<typeof ScenarioHeaderBar> = {
    title: 'My scenario',
    saving: false,
    detailLoaded: true,
    environments: [],
    selectedEnvPath: '',
    onEnvPathChange: vi.fn(),
    onTitleChange: vi.fn(),
    result: null,
    running: false,
    onShowStats: vi.fn(),
    onRun: vi.fn(),
    onCancel: vi.fn(),
    saveError: null,
    savedFlash: false,
    isDirty: false,
    onDiscard: vi.fn(),
    onSave: vi.fn(),
    onShowYaml: vi.fn(),
    ...over,
  }
  render(<ScenarioHeaderBar {...props} />)
  return props
}

describe('ScenarioHeaderBar', () => {
  it('edits the title', () => {
    const p = setup()
    const input = screen.getByPlaceholderText('Scenario title')
    expect(input).toHaveValue('My scenario')
    fireEvent.change(input, { target: { value: 'Renamed' } })
    expect(p.onTitleChange).toHaveBeenCalledWith('Renamed')
  })

  it('opens the YAML editor', () => {
    const p = setup()
    fireEvent.click(screen.getByTitle('View / edit YAML'))
    expect(p.onShowYaml).toHaveBeenCalledTimes(1)
  })

  it('offers the CLI command only when the screen can open it', () => {
    const { unmount } = render(<ScenarioHeaderBar {...({
      title: 't', saving: false, detailLoaded: true, environments: [], selectedEnvPath: '',
      onEnvPathChange: vi.fn(), onTitleChange: vi.fn(), result: null, running: false,
      onShowStats: vi.fn(), onRun: vi.fn(), saveError: null, savedFlash: false, isDirty: false,
      onDiscard: vi.fn(), onSave: vi.fn(), onShowYaml: vi.fn(),
    } as React.ComponentProps<typeof ScenarioHeaderBar>)} />)
    expect(screen.queryByRole('button', { name: 'Get CLI command' })).not.toBeInTheDocument()
    unmount()

    const onGetCliCommand = vi.fn()
    setup({ onGetCliCommand })
    fireEvent.click(screen.getByRole('button', { name: 'Get CLI command' }))
    expect(onGetCliCommand).toHaveBeenCalledTimes(1)
  })

  it('hides the environment picker without environments', () => {
    setup()
    expect(screen.queryByTitle('Environment for this run')).not.toBeInTheDocument()
  })

  it('lists environments, reflects the selection and reports changes', () => {
    const p = setup({ environments: envs, selectedEnvPath: '/e/prod.yaml' })
    const select = screen.getByTitle('Environment for this run') as HTMLSelectElement
    expect(select.value).toBe('/e/prod.yaml')
    expect(Array.from(select.options).map(o => o.textContent)).toEqual(['No environment', 'local', 'prod'])
    expect(select.className).toContain('text-[var(--color-accent)]')
    fireEvent.change(select, { target: { value: '/e/local.yaml' } })
    expect(p.onEnvPathChange).toHaveBeenCalledWith('/e/local.yaml')
  })

  it('mutes the picker when no environment is chosen', () => {
    setup({ environments: envs })
    expect(screen.getByTitle('Environment for this run').className).toContain('text-[var(--color-muted-foreground)]')
  })

  describe('result badge', () => {
    it('shows PASSED with counts and duration, and opens stats', () => {
      const p = setup({ result: result() })
      expect(screen.getByText('PASSED')).toBeInTheDocument()
      expect(screen.getByText('1/3 · 120 ms')).toBeInTheDocument()
      fireEvent.click(screen.getByTitle('View run statistics'))
      expect(p.onShowStats).toHaveBeenCalledTimes(1)
    })

    it('shows FAILED for a failing result', () => {
      setup({ result: result({ passed: false }) })
      expect(screen.getByText('FAILED')).toBeInTheDocument()
    })

    it('handles a result without steps', () => {
      setup({ result: { title: 't', passed: true, durationMs: 5, steps: undefined as never } })
      expect(screen.getByText('0/0 · 5 ms')).toBeInTheDocument()
    })

    it('hides the badge while running or without a result', () => {
      const { unmount } = render(<div />)
      unmount()
      setup({ result: result(), running: true })
      expect(screen.queryByText('PASSED')).not.toBeInTheDocument()
    })

    it('shows no badge without a result', () => {
      setup()
      expect(screen.queryByText('PASSED')).not.toBeInTheDocument()
      expect(screen.queryByText('FAILED')).not.toBeInTheDocument()
    })
  })

  describe('dirty state', () => {
    it('hides save controls when clean', () => {
      setup()
      expect(screen.queryByRole('button', { name: /Save/ })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /Discard/ })).not.toBeInTheDocument()
    })

    it('saves and discards when dirty', () => {
      const p = setup({ isDirty: true })
      fireEvent.click(screen.getByRole('button', { name: /Save/ }))
      fireEvent.click(screen.getByRole('button', { name: /Discard/ }))
      expect(p.onSave).toHaveBeenCalledTimes(1)
      expect(p.onDiscard).toHaveBeenCalledTimes(1)
    })

    it('shows a save error with a tooltip', () => {
      setup({ isDirty: true, saveError: 'assertion requires a path' })
      expect(screen.getByText('assertion requires a path')).toHaveAttribute('title', 'assertion requires a path')
    })

    it('disables the buttons and shows progress while saving', () => {
      setup({ isDirty: true, saving: true })
      expect(screen.getByRole('button', { name: /Saving/ })).toBeDisabled()
      expect(screen.getByRole('button', { name: /Discard/ })).toBeDisabled()
    })

    it('shows a Saved flash instead of the controls', () => {
      setup({ isDirty: true, savedFlash: true })
      expect(screen.getByText('Saved')).toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /Discard/ })).not.toBeInTheDocument()
    })
  })

  describe('run controls', () => {
    it('runs the scenario', () => {
      const p = setup()
      fireEvent.click(screen.getByRole('button', { name: /Run/ }))
      expect(p.onRun).toHaveBeenCalledTimes(1)
    })

    it('disables Run until the scenario has loaded', () => {
      setup({ detailLoaded: false })
      expect(screen.getByRole('button', { name: /Run/ })).toBeDisabled()
    })

    it('offers Stop while running and cancels', () => {
      const p = setup({ running: true })
      fireEvent.click(screen.getByRole('button', { name: /Stop/ }))
      expect(p.onCancel).toHaveBeenCalledTimes(1)
      expect(screen.queryByRole('button', { name: /^Run/ })).not.toBeInTheDocument()
    })

    it('shows a disabled Running… button when running without a cancel handler', () => {
      setup({ running: true, onCancel: undefined })
      expect(screen.getByRole('button', { name: /Running/ })).toBeDisabled()
    })
  })
})
