import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react'
import { ScenarioYamlDialog } from './ScenarioYamlDialog'
import * as bridge from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({ validateScenarioYAML: vi.fn() }))
vi.mock('./ScenarioYamlPane', () => ({
  YamlEditorPane: (p: { value: string; error: string | null; onChange: (v: string) => void }) => (
    <div>
      <textarea aria-label="yaml" value={p.value} onChange={e => p.onChange(e.target.value)} />
      {p.error && <p role="alert">{p.error}</p>}
    </div>
  ),
}))

const validate = vi.mocked(bridge.validateScenarioYAML)

function setup(over: Partial<React.ComponentProps<typeof ScenarioYamlDialog>> = {}) {
  const props: React.ComponentProps<typeof ScenarioYamlDialog> = {
    theme: 'dark',
    initialContent: 'title: A',
    onSave: vi.fn(() => Promise.resolve(null)),
    onClose: vi.fn(),
    ...over,
  }
  render(<ScenarioYamlDialog {...props} />)
  return props
}

function edit(value = 'title: B') {
  fireEvent.change(screen.getByLabelText('yaml'), { target: { value } })
}

beforeEach(() => { vi.clearAllMocks(); validate.mockResolvedValue('' as never) })
afterEach(() => vi.useRealTimers())

describe('ScenarioYamlDialog', () => {
  it('shows the initial content with Save disabled until edited', () => {
    setup()
    expect(screen.getByLabelText('yaml')).toHaveValue('title: A')
    expect(screen.getByRole('button', { name: /Save/ })).toBeDisabled()
    edit()
    expect(screen.getByRole('button', { name: /Save/ })).toBeEnabled()
  })

  it('closes immediately from the X button, backdrop when clean', () => {
    const p = setup()
    fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
    expect(p.onClose).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByText('YAML').closest('.fixed')!)
    expect(p.onClose).toHaveBeenCalledTimes(2)
  })

  it('does not close when clicking inside the dialog', () => {
    const p = setup()
    fireEvent.click(screen.getByText('YAML'))
    expect(p.onClose).not.toHaveBeenCalled()
  })

  it('asks before discarding unsaved edits', () => {
    const p = setup()
    edit()
    fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
    expect(p.onClose).not.toHaveBeenCalled()
    expect(screen.getByText(/unsaved YAML edits/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByText(/unsaved YAML edits/)).not.toBeInTheDocument()
    fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
    fireEvent.click(screen.getByRole('button', { name: 'Discard' }))
    expect(p.onClose).toHaveBeenCalledTimes(1)
  })

  describe('validation', () => {
    it('shows Valid briefly after a clean check', async () => {
      vi.useFakeTimers()
      setup()
      await act(async () => { fireEvent.click(screen.getByRole('button', { name: /Check/ })) })
      expect(validate).toHaveBeenCalledWith('title: A')
      expect(screen.getByText('Valid')).toBeInTheDocument()
      act(() => { vi.advanceTimersByTime(2100) })
      expect(screen.getByText('Check')).toBeInTheDocument()
    })

    it('shows the validation error and resets the check state', async () => {
      validate.mockResolvedValue('line 3: bad indent' as never)
      setup()
      fireEvent.click(screen.getByRole('button', { name: /Check/ }))
      expect(await screen.findByRole('alert')).toHaveTextContent('line 3: bad indent')
      expect(screen.getByText('Check')).toBeInTheDocument()
    })

    it('shows a checking state while validation is pending', async () => {
      let release!: (v: string) => void
      validate.mockReturnValue(new Promise(r => { release = r }) as never)
      setup()
      fireEvent.click(screen.getByRole('button', { name: /Check/ }))
      expect(await screen.findByRole('button', { name: /Checking/ })).toBeDisabled()
      await act(async () => { release('') })
    })

    it('clears the valid badge and the error when the content changes', async () => {
      validate.mockResolvedValueOnce('oops' as never)
      setup()
      fireEvent.click(screen.getByRole('button', { name: /Check/ }))
      await screen.findByRole('alert')
      edit('title: C')
      expect(screen.queryByRole('alert')).not.toBeInTheDocument()

      validate.mockResolvedValueOnce('' as never)
      fireEvent.click(screen.getByRole('button', { name: /Check/ }))
      await screen.findByText('Valid')
      edit('title: D')
      expect(screen.queryByText('Valid')).not.toBeInTheDocument()
    })
  })

  describe('saving', () => {
    it('saves the edited content', async () => {
      const p = setup()
      edit('title: Saved')
      fireEvent.click(screen.getByRole('button', { name: /Save/ }))
      await waitFor(() => expect(p.onSave).toHaveBeenCalledWith('title: Saved'))
    })

    it('shows a save error returned by the handler', async () => {
      setup({ onSave: vi.fn(() => Promise.resolve('write failed')) })
      edit()
      fireEvent.click(screen.getByRole('button', { name: /Save/ }))
      expect(await screen.findByRole('alert')).toHaveTextContent('write failed')
    })

    it('shows a saving state while pending', async () => {
      let release!: (v: string | null) => void
      setup({ onSave: vi.fn(() => new Promise<string | null>(r => { release = r })) })
      edit()
      fireEvent.click(screen.getByRole('button', { name: /Save/ }))
      expect(await screen.findByRole('button', { name: /Saving/ })).toBeDisabled()
      await act(async () => { release(null) })
    })
  })
})
