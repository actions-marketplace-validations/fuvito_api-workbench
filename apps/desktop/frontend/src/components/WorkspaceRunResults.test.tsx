import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { FolderRunResults, RunAllResults } from './WorkspaceRunResults'
import type { FolderRunState, RunAllState } from '@/types'

function folder(over: Partial<FolderRunState> = {}): FolderRunState {
  return {
    folderPath: '/ws/requests/users',
    folderName: 'users',
    running: false,
    results: [
      { name: 'list', path: '/a', method: 'GET', statusCode: 200, durationMs: 12, passed: true },
      { name: 'create', path: '/b', method: 'POST', statusCode: 500, durationMs: 30, passed: false },
      { name: 'broken', path: '/c', method: 'GET', statusCode: 0, durationMs: 0, passed: false, error: 'refused' },
    ],
    ...over,
  }
}

function runAll(over: Partial<RunAllState> = {}): RunAllState {
  return {
    running: false, done: 2, total: 3, passed: 1,
    results: [
      { path: '/s1', name: 'smoke', passed: true, durationMs: 50 },
      { path: '/s2', name: 'login', passed: false, durationMs: 80 },
      { path: '/s3', name: 'errored', passed: false, durationMs: 0, error: 'spec missing' },
    ],
    ...over,
  }
}

describe('FolderRunResults', () => {
  it('offers Stop only while the folder is running, and calls it', () => {
    const onCancel = vi.fn()
    const { rerender } = render(<FolderRunResults state={folder({ running: true })} onSelectFile={vi.fn()} onCancel={onCancel} />)
    fireEvent.click(screen.getByRole('button', { name: 'Stop' }))
    expect(onCancel).toHaveBeenCalledTimes(1)

    rerender(<FolderRunResults state={folder({ running: false })} onSelectFile={vi.fn()} onCancel={onCancel} />)
    expect(screen.queryByRole('button', { name: 'Stop' })).not.toBeInTheDocument()
  })

  it('says when a run was stopped and counts only the requests that finished', () => {
    render(<FolderRunResults state={folder({ stopped: true, results: [{ name: 'list', path: '/a', method: 'GET', statusCode: 200, durationMs: 12, passed: true }] })} onSelectFile={vi.fn()} />)
    expect(screen.getByText(/1\/1 ok, stopped/)).toBeInTheDocument()
  })

  it('summarises a finished run', () => {
    render(<FolderRunResults state={folder()} onSelectFile={vi.fn()} />)
    expect(screen.getByText('users')).toBeInTheDocument()
    expect(screen.getByText('— 1/3 ok')).toBeInTheDocument()
  })

  it('shows progress while running and hides dismiss', () => {
    render(<FolderRunResults state={folder({ running: true })} onSelectFile={vi.fn()} onDismiss={vi.fn()} />)
    expect(screen.getByText('Running users…')).toBeInTheDocument()
    expect(screen.queryAllByRole('button').filter(b => b.querySelector('svg.lucide-x'))).toHaveLength(0)
  })

  it('lists each request with method, status and duration, or "err"', () => {
    render(<FolderRunResults state={folder()} onSelectFile={vi.fn()} />)
    expect(screen.getByText('200 · 12ms')).toBeInTheDocument()
    expect(screen.getByText('500 · 30ms')).toBeInTheDocument()
    expect(screen.getByText('err')).toBeInTheDocument()
    expect(screen.getAllByTitle('refused')).toHaveLength(2)
    expect(screen.getAllByText('GET')).toHaveLength(2)
    expect(screen.getByText('POST')).toBeInTheDocument()
  })

  it('opens a request on click', () => {
    const onSelectFile = vi.fn()
    render(<FolderRunResults state={folder()} onSelectFile={onSelectFile} />)
    fireEvent.click(screen.getByText('create'))
    expect(onSelectFile).toHaveBeenCalledWith('/b')
  })

  it('dismisses when finished', () => {
    const onDismiss = vi.fn()
    render(<FolderRunResults state={folder()} onSelectFile={vi.fn()} onDismiss={onDismiss} />)
    fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('omits the dismiss button without a handler', () => {
    render(<FolderRunResults state={folder()} onSelectFile={vi.fn()} />)
    expect(screen.getAllByRole('button').filter(b => b.querySelector('svg.lucide-x'))).toHaveLength(0)
  })

  it('omits the method badge when a result has no method', () => {
    render(<FolderRunResults state={folder({ results: [{ name: 'n', path: '/p', method: '', statusCode: 200, durationMs: 1, passed: true }] })} onSelectFile={vi.fn()} />)
    expect(screen.queryByText('GET')).not.toBeInTheDocument()
  })
})

describe('RunAllResults', () => {
  it('summarises a finished run', () => {
    render(<RunAllResults state={runAll()} />)
    expect(screen.getByText('1/3 passed')).toBeInTheDocument()
  })

  it('shows 1-based progress while running and offers stop', () => {
    const onCancel = vi.fn()
    render(<RunAllResults state={runAll({ running: true, done: 1 })} onCancel={onCancel} onDismiss={vi.fn()} />)
    expect(screen.getByText('Running 2/3…')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Stop' }))
    expect(onCancel).toHaveBeenCalledTimes(1)
    expect(screen.queryAllByRole('button').filter(b => b.querySelector('svg.lucide-x'))).toHaveLength(0)
  })

  it('does not offer stop without a cancel handler', () => {
    render(<RunAllResults state={runAll({ running: true })} />)
    expect(screen.queryByRole('button', { name: 'Stop' })).not.toBeInTheDocument()
  })

  it('shows durations and "err" with tooltips', () => {
    render(<RunAllResults state={runAll()} />)
    expect(screen.getByText('50ms')).toBeInTheDocument()
    expect(screen.getByText('80ms')).toBeInTheDocument()
    expect(screen.getByText('err')).toBeInTheDocument()
    expect(screen.getByTitle('spec missing')).toBeInTheDocument()
    expect(screen.getAllByTitle('Click to open scenario with results')).toHaveLength(2)
  })

  it('opens a scenario on click and tolerates a missing handler', () => {
    const onSelectScenario = vi.fn()
    const { unmount } = render(<RunAllResults state={runAll()} onSelectScenario={onSelectScenario} />)
    fireEvent.click(screen.getByText('login'))
    expect(onSelectScenario).toHaveBeenCalledWith('/s2', 'login')
    unmount()
    render(<RunAllResults state={runAll()} />)
    expect(() => fireEvent.click(screen.getByText('login'))).not.toThrow()
  })

  it('dismisses when finished', () => {
    const onDismiss = vi.fn()
    render(<RunAllResults state={runAll()} onDismiss={onDismiss} />)
    fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })
})
