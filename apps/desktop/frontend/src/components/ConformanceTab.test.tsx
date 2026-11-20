import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ConformanceTab, isLocalUrl } from './ConformanceTab'

const bridge = vi.hoisted(() => ({
  getConformContext: vi.fn(),
  inspectConformSpec: vi.fn(),
  runConformance: vi.fn(),
  cancelConformance: vi.fn(),
}))
vi.mock('@/lib/bridge', () => bridge)

const ctx = { workspace: '/ws', specs: [{ path: '/ws/api.yaml', label: 'api.yaml' }], environments: [{ path: '/ws/environments/local.yaml', label: 'local' }] }
const info = { title: 'Shop', version: '1', serverUrl: 'http://localhost:8120', reads: 5, writes: 2 }
const output = {
  specTitle: 'Shop', baseUrl: 'http://localhost:8120', writes: false, durationMs: 12, total: 7,
  covered: 5, notCovered: 2, passing: 4, failing: 1,
  operations: [
    { operationId: 'a', method: 'GET', path: '/ok', status: 'passed', statusCode: 200, durationMs: 3, violations: [] },
    { operationId: 'b', method: 'GET', path: '/bad', status: 'failed', statusCode: 200, durationMs: 4, violations: ['price: expected number'] },
    { operationId: 'c', method: 'DELETE', path: '/x', status: 'skipped', statusCode: 0, durationMs: 0, violations: [] },
  ],
}

async function chooseSpec() {
  render(<ConformanceTab />)
  fireEvent.change(await screen.findByLabelText(/Spec/), { target: { value: '/ws/api.yaml' } })
  await waitFor(() => expect(screen.getByLabelText(/Base URL/)).toHaveValue('http://localhost:8120'))
}

describe('ConformanceTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(ctx)
    bridge.inspectConformSpec.mockResolvedValue(info)
    bridge.runConformance.mockResolvedValue(output)
  })

  it('sends read requests only by default and says how many writes are skipped', async () => {
    await chooseSpec()
    expect(screen.getByTestId('conform-send-note')).toHaveTextContent('Will send 5 read requests. 2 write operations are skipped.')
    expect(screen.getByRole('checkbox', { name: /write requests/i })).not.toBeChecked()
  })

  it('shows the write count before the user opts in to writes', async () => {
    await chooseSpec()
    fireEvent.click(screen.getByRole('checkbox', { name: /write requests/i }))
    expect(screen.getByTestId('conform-send-note')).toHaveTextContent('Will send 5 read and 2 write requests')
  })

  it('runs with the chosen options and shows the four totals and the violations', async () => {
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByText('Passing')).toBeInTheDocument()
    expect(bridge.runConformance).toHaveBeenCalledWith({
      specPath: '/ws/api.yaml', baseUrl: 'http://localhost:8120', envPath: '', allowWrites: false, confirmedRemote: false,
    })
    for (const label of ['Covered', 'Not covered', 'Passing', 'Failing']) expect(screen.getByText(label)).toBeInTheDocument()
    expect(screen.getByText('price: expected number')).toBeInTheDocument()
    expect(screen.getByText('skipped')).toBeInTheDocument()
  })

  it('blocks a base URL that is not on this computer until the user confirms it', async () => {
    await chooseSpec()
    fireEvent.change(screen.getByLabelText(/Base URL/), { target: { value: 'https://api.example.com' } })
    const run = screen.getByRole('button', { name: /Run conformance check/ })
    expect(run).toBeDisabled()
    fireEvent.click(screen.getByRole('checkbox', { name: /not on your computer/i }))
    expect(run).toBeEnabled()
    fireEvent.click(run)
    await waitFor(() => expect(bridge.runConformance).toHaveBeenCalledWith(expect.objectContaining({ confirmedRemote: true })))
  })

  it('cannot run without a spec', async () => {
    render(<ConformanceTab />)
    expect(await screen.findByRole('button', { name: /Run conformance check/ })).toBeDisabled()
  })

  it('shows an error returned by the run', async () => {
    bridge.runConformance.mockResolvedValue({ ...output, error: 'confirm the base URL', operations: [] })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByRole('alert')).toHaveTextContent('confirm the base URL')
  })
})

describe('isLocalUrl', () => {
  it('accepts this computer only', () => {
    expect(isLocalUrl('http://localhost:8080')).toBe(true)
    expect(isLocalUrl('http://127.0.0.1:1')).toBe(true)
    expect(isLocalUrl('https://api.example.com')).toBe(false)
    expect(isLocalUrl('http://localhost.evil.com')).toBe(false)
    expect(isLocalUrl('nonsense')).toBe(false)
  })
})
