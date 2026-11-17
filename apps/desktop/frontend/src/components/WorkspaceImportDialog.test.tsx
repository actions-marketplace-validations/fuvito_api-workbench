import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { WorkspaceImportDialog } from './WorkspaceImportDialog'
import * as bridge from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({ applyWorkspaceImport: vi.fn() }))

const b = vi.mocked(bridge)

function setup(scan: Partial<React.ComponentProps<typeof WorkspaceImportDialog>['scan']> = {}) {
  const props = {
    scan: { zipPath: '/tmp/ws.zip', totalFiles: 5, conflicts: [] as string[], ...scan },
    onClose: vi.fn(),
    onComplete: vi.fn(),
  }
  render(<WorkspaceImportDialog {...props} />)
  return props
}

beforeEach(() => { vi.clearAllMocks(); b.applyWorkspaceImport.mockResolvedValue({ imported: 5, skipped: 0, renamed: 0 } as never) })

describe('WorkspaceImportDialog', () => {
  it('reports no conflicts and hides the strategy picker', () => {
    setup()
    expect(screen.getByText(/No conflicts/)).toBeInTheDocument()
    expect(screen.queryByRole('radio')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Import 5 files' })).toBeInTheDocument()
  })

  it('uses the singular for one file', () => {
    setup({ totalFiles: 1 })
    expect(screen.getByRole('button', { name: 'Import 1 file' })).toBeInTheDocument()
  })

  it('lists conflicts, truncating after twenty', () => {
    const conflicts = Array.from({ length: 23 }, (_, i) => `requests/f${i}.yaml`)
    setup({ conflicts })
    expect(screen.getByText('23 conflicts with existing files.')).toBeInTheDocument()
    expect(screen.getByText('requests/f19.yaml')).toBeInTheDocument()
    expect(screen.queryByText('requests/f20.yaml')).not.toBeInTheDocument()
    expect(screen.getByText('…and 3 more')).toBeInTheDocument()
  })

  it('uses the singular for one conflict', () => {
    setup({ conflicts: ['a.yaml'] })
    expect(screen.getByText('1 conflict with existing files.')).toBeInTheDocument()
    expect(screen.queryByText(/more$/)).not.toBeInTheDocument()
  })

  it('defaults to skip and lets the user choose a strategy', async () => {
    const p = setup({ conflicts: ['a.yaml'] })
    const radios = screen.getAllByRole('radio') as HTMLInputElement[]
    expect(radios.map(r => r.checked)).toEqual([true, false, false])
    fireEvent.click(radios[1])
    fireEvent.click(screen.getByRole('button', { name: /^Import/ }))
    await waitFor(() => expect(b.applyWorkspaceImport).toHaveBeenCalledWith('/tmp/ws.zip', 'overwrite'))
    await waitFor(() => expect(p.onComplete).toHaveBeenCalledTimes(1))
  })

  it('supports the rename strategy', async () => {
    setup({ conflicts: ['a.yaml'] })
    fireEvent.click(screen.getAllByRole('radio')[2])
    fireEvent.click(screen.getByRole('button', { name: /^Import/ }))
    await waitFor(() => expect(b.applyWorkspaceImport).toHaveBeenCalledWith('/tmp/ws.zip', 'rename'))
  })

  it('shows the result counts and closes with Done', async () => {
    b.applyWorkspaceImport.mockResolvedValue({ imported: 3, skipped: 1, renamed: 2 } as never)
    const p = setup()
    fireEvent.click(screen.getByRole('button', { name: /^Import/ }))
    expect(await screen.findByText('Import complete')).toBeInTheDocument()
    expect(screen.getByText('Imported').nextSibling).toHaveTextContent('3')
    expect(screen.getByText('Renamed').nextSibling).toHaveTextContent('2')
    expect(screen.getByText('Skipped').nextSibling).toHaveTextContent('1')
    fireEvent.click(screen.getByRole('button', { name: 'Done' }))
    expect(p.onClose).toHaveBeenCalledTimes(1)
  })

  it('omits zero renamed/skipped rows', async () => {
    setup()
    fireEvent.click(screen.getByRole('button', { name: /^Import/ }))
    await screen.findByText('Import complete')
    expect(screen.queryByText('Renamed')).not.toBeInTheDocument()
    expect(screen.queryByText('Skipped')).not.toBeInTheDocument()
  })

  it('shows an error result without refreshing the workspace', async () => {
    b.applyWorkspaceImport.mockResolvedValue({ imported: 0, skipped: 0, renamed: 0, error: 'zip corrupt' } as never)
    const p = setup()
    fireEvent.click(screen.getByRole('button', { name: /^Import/ }))
    expect(await screen.findByText('zip corrupt')).toBeInTheDocument()
    expect(p.onComplete).not.toHaveBeenCalled()
  })

  it('disables the buttons while applying', async () => {
    let release!: (v: unknown) => void
    b.applyWorkspaceImport.mockReturnValue(new Promise(r => { release = r }) as never)
    setup()
    fireEvent.click(screen.getByRole('button', { name: /^Import/ }))
    expect(await screen.findByRole('button', { name: 'Importing…' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled()
    release({ imported: 1, skipped: 0, renamed: 0 })
    await screen.findByText('Import complete')
  })

  it('cancels', () => {
    const p = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(p.onClose).toHaveBeenCalledTimes(1)
  })
})
