import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { CoverageSidebarPanel } from './CoverageSidebarPanel'
import * as bridge from '@/lib/bridge'
import type { AccumulatedCoverage } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({ listAccumulatedCoverage: vi.fn() }))

const b = vi.mocked(bridge)

function entry(over: Partial<AccumulatedCoverage> = {}): AccumulatedCoverage {
  return {
    specFile: 'specs/users.yaml', covered: 9, total: 10, percent: 90, totalRuns: 3, passedRuns: 3, failedRuns: 0,
    exercised: [], missed: [], ...over,
  }
}

beforeEach(() => { vi.clearAllMocks(); b.listAccumulatedCoverage.mockResolvedValue([]) })

describe('CoverageSidebarPanel', () => {
  it('shows a placeholder without a workspace and does not fetch', () => {
    render(<CoverageSidebarPanel workspacePath="" selectedSpecFile={null} onSelectSpec={vi.fn()} />)
    expect(screen.getByText('No workspace open')).toBeInTheDocument()
    expect(b.listAccumulatedCoverage).not.toHaveBeenCalled()
  })

  it('shows an empty state with no data', async () => {
    render(<CoverageSidebarPanel workspacePath="/ws" selectedSpecFile={null} onSelectSpec={vi.fn()} />)
    expect(await screen.findByText('No coverage data yet')).toBeInTheDocument()
  })

  it('lists specs with percentage, ops and run summary', async () => {
    b.listAccumulatedCoverage.mockResolvedValue([entry({ failedRuns: 1, totalRuns: 1, passedRuns: 0 })])
    render(<CoverageSidebarPanel workspacePath="/ws" selectedSpecFile="specs/users.yaml" onSelectSpec={vi.fn()} />)
    expect(await screen.findByText('users.yaml')).toBeInTheDocument()
    expect(screen.getByText('90%')).toBeInTheDocument()
    expect(screen.getByText('9/10 ops')).toBeInTheDocument()
    expect(screen.getByText('1 run')).toBeInTheDocument()
    expect(screen.getByText('1 failed')).toBeInTheDocument()
  })

  it.each([[90, 'text-emerald-400'], [50, 'text-amber-400'], [5, 'text-red-400']])('colours %i%%', async (percent, cls) => {
    b.listAccumulatedCoverage.mockResolvedValue([entry({ percent })])
    render(<CoverageSidebarPanel workspacePath="/ws" selectedSpecFile="specs/users.yaml" onSelectSpec={vi.fn()} />)
    expect((await screen.findByText(`${percent}%`)).className).toContain(cls)
  })

  it('auto-selects the first spec when nothing is selected', async () => {
    b.listAccumulatedCoverage.mockResolvedValue([entry({ specFile: 'a.yaml' }), entry({ specFile: 'b.yaml' })])
    const onSelectSpec = vi.fn()
    render(<CoverageSidebarPanel workspacePath="/ws" selectedSpecFile={null} onSelectSpec={onSelectSpec} />)
    await waitFor(() => expect(onSelectSpec).toHaveBeenCalledWith('a.yaml'))
  })

  it('does not override an existing selection', async () => {
    b.listAccumulatedCoverage.mockResolvedValue([entry({ specFile: 'a.yaml' })])
    const onSelectSpec = vi.fn()
    render(<CoverageSidebarPanel workspacePath="/ws" selectedSpecFile="b.yaml" onSelectSpec={onSelectSpec} />)
    await screen.findByText('a.yaml')
    expect(onSelectSpec).not.toHaveBeenCalled()
  })

  it('selects a spec on click and refreshes on demand', async () => {
    b.listAccumulatedCoverage.mockResolvedValue([entry({ specFile: 'a.yaml' }), entry({ specFile: 'b.yaml' })])
    const onSelectSpec = vi.fn()
    render(<CoverageSidebarPanel workspacePath="/ws" selectedSpecFile="a.yaml" onSelectSpec={onSelectSpec} />)
    fireEvent.click(await screen.findByText('b.yaml'))
    expect(onSelectSpec).toHaveBeenCalledWith('b.yaml')
    fireEvent.click(screen.getByTitle('Refresh'))
    await waitFor(() => expect(b.listAccumulatedCoverage).toHaveBeenCalledTimes(2))
  })
})
