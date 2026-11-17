import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react'
import type { BundlePreview } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({
  pickBundleFile: vi.fn(), previewBundle: vi.fn(), importBundle: vi.fn(),
}))

import * as bridge from '@/lib/bridge'
import { BundleImportDialog } from './BundleImportDialog'

const b = vi.mocked(bridge)

const okPreview: BundlePreview = {
  title: 'Suite', specFile: 'https://api.example.com/openapi.json', hasErrors: false, issues: [],
  entries: [
    { kind: 'environment', name: 'staging', path: 'environments/staging.yaml', requestedPath: 'environments/staging.yaml', action: 'create' },
    { kind: 'scenario', name: 'Smoke', path: 'scenarios/smoke.yaml', requestedPath: 'scenarios/smoke.yaml', action: 'create' },
  ],
}

function setup() {
  const h = { onClose: vi.fn(), onComplete: vi.fn() }
  render(<BundleImportDialog {...h} />)
  const textarea = () => screen.getByLabelText('Bundle') as HTMLTextAreaElement
  const type = async (value: string) => {
    fireEvent.change(textarea(), { target: { value } })
    await act(async () => { await vi.advanceTimersByTimeAsync(300) })
  }
  return { ...h, textarea, type }
}

beforeEach(() => {
  vi.useFakeTimers({ shouldAdvanceTime: true })
  vi.resetAllMocks()
  b.previewBundle.mockResolvedValue(okPreview)
})
afterEach(() => vi.useRealTimers())

describe('before anything is pasted', () => {
  it('explains what to do and offers no import', () => {
    setup()
    expect(screen.getByText(/Nothing is written until you click Import/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Import' })).toBeDisabled()
    expect(b.previewBundle).not.toHaveBeenCalled()
  })
})

describe('previewing', () => {
  it('checks the pasted text after a short pause and lists the files', async () => {
    const h = setup()
    await h.type('schemaVersion: "1"')
    expect(b.previewBundle).toHaveBeenCalledWith('schemaVersion: "1"', '', 'rename')
    expect(screen.getByText('environments/staging.yaml')).toBeInTheDocument()
    expect(screen.getByText('scenarios/smoke.yaml')).toBeInTheDocument()
    expect(screen.getByText('— Suite')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Import 2 files' })).toBeEnabled()
  })

  it('debounces typing into one check', async () => {
    const h = setup()
    fireEvent.change(h.textarea(), { target: { value: 'a' } })
    fireEvent.change(h.textarea(), { target: { value: 'ab' } })
    fireEvent.change(h.textarea(), { target: { value: 'abc' } })
    await act(async () => { await vi.advanceTimersByTimeAsync(300) })
    expect(b.previewBundle).toHaveBeenCalledTimes(1)
    expect(b.previewBundle).toHaveBeenCalledWith('abc', '', 'rename')
  })

  it('re-checks when the spec file override changes and trims it', async () => {
    const h = setup()
    await h.type('x')
    fireEvent.change(screen.getByLabelText(/Spec file/), { target: { value: '  C:/specs/api.yaml  ' } })
    await act(async () => { await vi.advanceTimersByTimeAsync(300) })
    expect(b.previewBundle).toHaveBeenLastCalledWith('x', 'C:/specs/api.yaml', 'rename')
  })

  it("shows the bundle's own spec as the placeholder", async () => {
    const h = setup()
    await h.type('x')
    expect(screen.getByPlaceholderText('Bundle uses: https://api.example.com/openapi.json')).toBeInTheDocument()
  })

  it('shows a parse error and blocks the import', async () => {
    b.previewBundle.mockResolvedValue({ ...okPreview, entries: [], hasErrors: true, parseError: 'parsing yaml: line 3: unknown field "timeoutMs"' })
    const h = setup()
    await h.type('bad')
    expect(screen.getByText(/unknown field "timeoutMs"/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Import' })).toBeDisabled()
  })

  it('lists problems with their locations and blocks the import', async () => {
    b.previewBundle.mockResolvedValue({
      ...okPreview, entries: [], hasErrors: true,
      issues: [
        { severity: 'error', where: 'scenarios[0] "Flow" › step 2 "Login"', message: 'requestRef "x" does not match any request' },
        { severity: 'error', where: '', message: 'schemaVersion is required' },
      ],
    })
    const h = setup()
    await h.type('x')
    expect(screen.getByText('2 problems to fix')).toBeInTheDocument()
    expect(screen.getByText(/step 2 "Login"/)).toBeInTheDocument()
    expect(screen.getByText('schemaVersion is required')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Import' })).toBeDisabled()
  })

  it('shows warnings without blocking the import', async () => {
    b.previewBundle.mockResolvedValue({
      ...okPreview, issues: [{ severity: 'warning', where: 'scenarios[0] › step 1', message: '{{ghost}} is not defined' }],
    })
    const h = setup()
    await h.type('x')
    expect(screen.getByText('1 warning')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Import 2 files' })).toBeEnabled()
  })

  it('treats a failing preview call as an error', async () => {
    b.previewBundle.mockRejectedValue(new Error('ipc down'))
    const h = setup()
    await h.type('x')
    expect(screen.getByText(/ipc down/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Import' })).toBeDisabled()
  })

  it('ignores a slow answer for text that has since changed', async () => {
    let resolveFirst: (p: BundlePreview) => void = () => {}
    b.previewBundle
      .mockImplementationOnce(() => new Promise<BundlePreview>(r => { resolveFirst = r }))
      .mockResolvedValueOnce({ ...okPreview, entries: [{ ...okPreview.entries[0], path: 'environments/new.yaml' }] })
    const h = setup()
    await h.type('first')
    await h.type('second')
    expect(screen.getByText('environments/new.yaml')).toBeInTheDocument()
    await act(async () => { resolveFirst({ ...okPreview, entries: [{ ...okPreview.entries[0], path: 'environments/stale.yaml' }] }) })
    expect(screen.queryByText('environments/stale.yaml')).not.toBeInTheDocument()
  })

  it('clears the preview when the text is emptied', async () => {
    const h = setup()
    await h.type('x')
    expect(screen.getByText('scenarios/smoke.yaml')).toBeInTheDocument()
    fireEvent.change(h.textarea(), { target: { value: '' } })
    expect(screen.queryByText('scenarios/smoke.yaml')).not.toBeInTheDocument()
  })
})

describe('conflicts', () => {
  const conflicted: BundlePreview = {
    ...okPreview,
    entries: [
      { ...okPreview.entries[0], path: 'environments/staging-2.yaml', action: 'rename' },
      { ...okPreview.entries[1], action: 'create' },
    ],
  }

  it('offers a strategy only when something already exists, defaulting to rename', async () => {
    const h = setup()
    await h.type('x')
    expect(screen.queryByText('Files that already exist')).not.toBeInTheDocument()

    b.previewBundle.mockResolvedValue(conflicted)
    await h.type('xy')
    expect(screen.getByText('Files that already exist')).toBeInTheDocument()
    expect((screen.getByLabelText(/Rename/) as HTMLInputElement).checked).toBe(true)
  })

  it('re-previews with the chosen strategy', async () => {
    b.previewBundle.mockResolvedValue(conflicted)
    const h = setup()
    await h.type('x')
    fireEvent.click(screen.getByLabelText(/Skip/))
    await act(async () => { await vi.advanceTimersByTimeAsync(300) })
    expect(b.previewBundle).toHaveBeenLastCalledWith('x', '', 'skip')
    fireEvent.click(screen.getByLabelText(/Overwrite/))
    await act(async () => { await vi.advanceTimersByTimeAsync(300) })
    expect(b.previewBundle).toHaveBeenLastCalledWith('x', '', 'overwrite')
  })

  it('only counts files that will actually be written', async () => {
    b.previewBundle.mockResolvedValue({ ...okPreview, entries: [{ ...okPreview.entries[0], action: 'skip' }, okPreview.entries[1]] })
    const h = setup()
    await h.type('x')
    expect(screen.getByRole('button', { name: 'Import 1 file' })).toBeEnabled()
  })

  it('has nothing to import when every file is skipped', async () => {
    b.previewBundle.mockResolvedValue({ ...okPreview, entries: okPreview.entries.map(e => ({ ...e, action: 'skip' as const })) })
    const h = setup()
    await h.type('x')
    expect(screen.getByRole('button', { name: 'Import' })).toBeDisabled()
  })
})

describe('opening a file', () => {
  it('loads the file text into the editor', async () => {
    b.pickBundleFile.mockResolvedValue({ path: '/b.yaml', text: 'from file' })
    const h = setup()
    fireEvent.click(screen.getByText('Open file…'))
    await waitFor(() => expect(h.textarea().value).toBe('from file'))
  })

  it('keeps the existing text when the dialog is cancelled', async () => {
    b.pickBundleFile.mockResolvedValue({ path: '', text: '' })
    const h = setup()
    fireEvent.change(h.textarea(), { target: { value: 'mine' } })
    fireEvent.click(screen.getByText('Open file…'))
    await act(async () => {})
    expect(h.textarea().value).toBe('mine')
  })

  it('shows read errors', async () => {
    b.pickBundleFile.mockResolvedValue({ path: '/big', text: '', error: 'file is too large for a bundle' })
    setup()
    fireEvent.click(screen.getByText('Open file…'))
    expect(await screen.findByText('file is too large for a bundle')).toBeInTheDocument()
  })
})

describe('importing', () => {
  it('imports with the current settings, then shows what was written', async () => {
    b.importBundle.mockResolvedValue({ created: 2, overwritten: 0, renamed: 1, skipped: 0, written: ['scenarios/smoke.yaml', 'environments/staging-2.yaml'] })
    const h = setup()
    await h.type('doc')
    fireEvent.change(screen.getByLabelText(/Spec file/), { target: { value: '/spec.yaml' } })
    await act(async () => { await vi.advanceTimersByTimeAsync(300) })
    fireEvent.click(screen.getByRole('button', { name: 'Import 2 files' }))

    expect(await screen.findByText('Import complete')).toBeInTheDocument()
    expect(b.importBundle).toHaveBeenCalledWith('doc', '/spec.yaml', 'rename')
    expect(screen.getByText('scenarios/smoke.yaml')).toBeInTheDocument()
    expect(screen.getByText('created', { exact: false })).toBeInTheDocument()
    expect(screen.getByText('renamed', { exact: false })).toBeInTheDocument()
    expect(h.onComplete).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByText('Done'))
    expect(h.onClose).toHaveBeenCalled()
  })

  it('shows an import error and does not refresh the workspace', async () => {
    b.importBundle.mockResolvedValue({ created: 0, overwritten: 0, renamed: 0, skipped: 0, written: [], error: 'disk full' })
    const h = setup()
    await h.type('doc')
    fireEvent.click(screen.getByRole('button', { name: 'Import 2 files' }))
    expect(await screen.findByText('disk full')).toBeInTheDocument()
    expect(h.onComplete).not.toHaveBeenCalled()
  })

  it('reports a failing import call', async () => {
    b.importBundle.mockRejectedValue(new Error('ipc'))
    const h = setup()
    await h.type('doc')
    fireEvent.click(screen.getByRole('button', { name: 'Import 2 files' }))
    expect(await screen.findByText(/ipc/)).toBeInTheDocument()
  })

  it('cancel closes without importing', async () => {
    const h = setup()
    await h.type('doc')
    fireEvent.click(screen.getByText('Cancel'))
    expect(h.onClose).toHaveBeenCalled()
    expect(b.importBundle).not.toHaveBeenCalled()
  })
})
