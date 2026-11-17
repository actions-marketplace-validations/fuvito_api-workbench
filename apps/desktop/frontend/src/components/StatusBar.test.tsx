import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { StatusBar, shortenPath } from './StatusBar'

const envs = [{ name: 'local', path: '/e/local.yaml' }, { name: 'prod', path: '/e/prod.yaml' }]

function renderBar(over: Partial<React.ComponentProps<typeof StatusBar>> = {}) {
  const handlers = { onEnvChange: vi.fn(), onEditEnv: vi.fn(), onStopDemo: vi.fn() }
  render(<StatusBar environments={envs} activeEnv={envs[0]} workspacePath="/ws/my-project" demoRunning={false} demoPort={0} {...handlers} {...over} />)
  return handlers
}

describe('StatusBar', () => {
  beforeEach(() => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.resolve()) } })
  })

  it('shows only the footer shell without a workspace', () => {
    renderBar({ workspacePath: '' })
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
    expect(screen.queryByTitle(/environment/i)).not.toBeInTheDocument()
  })

  it('shows the workspace folder name and full path as a tooltip', () => {
    renderBar({ workspacePath: 'C:\\Users\\me\\proj\\' })
    const name = screen.getByText('proj')
    expect(name).toHaveAttribute('title', 'C:\\Users\\me\\proj\\')
  })

  it('shows the full path of the file in focus, with the same path as a tooltip', () => {
    const path = String.raw`C:\Users\me\proj\scenarios\login.yaml`
    renderBar({ activeFilePath: path })
    const el = screen.getByTestId('status-file-path')
    expect(el).toHaveTextContent(path)
    expect(el).toHaveAttribute('title', path)
  })

  it('shows no file path when nothing is open or the file is unsaved', () => {
    renderBar({ activeFilePath: null })
    expect(screen.queryByTestId('status-file-path')).not.toBeInTheDocument()
    renderBar({ activeFilePath: '' })
    expect(screen.queryByTestId('status-file-path')).not.toBeInTheDocument()
  })

  it('shortens a very long path from the left and keeps the file name', () => {
    const long = '/very/' + 'deep/'.repeat(40) + 'requests/get-user.yaml'
    renderBar({ activeFilePath: long })
    const el = screen.getByTestId('status-file-path')
    expect(el.textContent!.startsWith('…')).toBe(true)
    expect(el.textContent!.endsWith('requests/get-user.yaml')).toBe(true)
    expect(el.textContent!.length).toBe(100)
    expect(el).toHaveAttribute('title', long)
  })

  it('shortenPath leaves short paths alone', () => {
    expect(shortenPath('/a/b.yaml')).toBe('/a/b.yaml')
    expect(shortenPath('x'.repeat(100))).toBe('x'.repeat(100))
    expect(shortenPath('x'.repeat(101))).toBe('…' + 'x'.repeat(99))
  })

  it('lists environments with the active one selected, and reports changes', () => {
    const h = renderBar()
    const select = screen.getByRole('combobox') as HTMLSelectElement
    expect(select.value).toBe('/e/local.yaml')
    expect(Array.from(select.options).map(o => o.textContent)).toEqual(['No environment', 'local', 'prod'])
    fireEvent.change(select, { target: { value: '/e/prod.yaml' } })
    expect(h.onEnvChange).toHaveBeenCalledWith('/e/prod.yaml')
    fireEvent.change(select, { target: { value: '' } })
    expect(h.onEnvChange).toHaveBeenLastCalledWith('')
  })

  it('selects "No environment" when none is active', () => {
    renderBar({ activeEnv: null })
    expect((screen.getByRole('combobox') as HTMLSelectElement).value).toBe('')
  })

  it('hides the selector but still allows creating an environment when there are none', () => {
    const h = renderBar({ environments: [], activeEnv: null })
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
    fireEvent.click(screen.getByTitle('Create environment'))
    expect(h.onEditEnv).toHaveBeenCalled()
  })

  it('names the active environment on the edit button', () => {
    const h = renderBar()
    fireEvent.click(screen.getByTitle('Edit environment: local'))
    expect(h.onEditEnv).toHaveBeenCalledTimes(1)
  })

  it('hides demo controls while the demo is stopped', () => {
    renderBar()
    expect(screen.queryByText(/Demo :/)).not.toBeInTheDocument()
  })

  it('shows the demo port, copies the base URL and can stop the demo', () => {
    const h = renderBar({ demoRunning: true, demoPort: 8090 })
    fireEvent.click(screen.getByText('Demo :8090'))
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('http://localhost:8090')
    fireEvent.click(screen.getByTitle('Stop demo server'))
    expect(h.onStopDemo).toHaveBeenCalled()
  })

  it('shows demo controls even without a workspace', () => {
    renderBar({ workspacePath: '', demoRunning: true, demoPort: 8091 })
    expect(screen.getByText('Demo :8091')).toBeInTheDocument()
  })

  it('ignores clipboard failures', () => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.reject(new Error('denied'))) } })
    renderBar({ demoRunning: true, demoPort: 8090 })
    expect(() => fireEvent.click(screen.getByText('Demo :8090'))).not.toThrow()
  })
})
