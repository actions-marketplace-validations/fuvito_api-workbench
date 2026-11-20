import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { ConformanceTab, isLocalUrl, splitList, cliValues } from './ConformanceTab'

const bridge = vi.hoisted(() => ({
  getConformContext: vi.fn(),
  inspectConformSpec: vi.fn(),
  runConformance: vi.fn(),
  cancelConformance: vi.fn(),
  exportConformReport: vi.fn(),
  saveContentToFile: vi.fn(),
  openInSystemViewer: vi.fn(),
}))
vi.mock('@/lib/bridge', () => bridge)

const ctx = { workspace: '/ws', specs: [{ path: '/ws/api.yaml', label: 'api.yaml' }], environments: [{ path: '/ws/environments/local.yaml', label: 'local' }] }
const info = { title: 'Shop', version: '1', serverUrl: 'http://localhost:8120', reads: 5, writes: 2 }
const output = {
  specTitle: 'Shop', baseUrl: 'http://localhost:8120', writes: false, durationMs: 12, total: 7,
  covered: 5, notCovered: 2, passing: 4, failing: 1, probes: [], warnings: [],
  operations: [
    { operationId: 'a', method: 'GET', path: '/ok', status: 'passed', statusCode: 200, durationMs: 3, violations: [] },
    {
      operationId: 'b', method: 'GET', path: '/bad', status: 'failed', statusCode: 200, durationMs: 4, violations: ['price: expected number'],
      findings: [{
        kind: 'wrong-type', where: 'price', summary: '"price" is text, the spec says number', expected: 'number', got: '"9.99"',
        why: 'The server sent text ("9.99") where the spec declares a number.', fix: 'Send a number from the server.', detail: 'Error at "/price": value must be a number',
      }],
    },
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
      tags: [], paths: [], negative: false, failFast: false, timeoutMs: 0, recordCoverage: false,
    })
    for (const label of ['Covered', 'Not covered', 'Passing', 'Failing']) expect(screen.getByText(label)).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Not run' })).toHaveTextContent('write request, not sent')
  })

  it('keeps the totals and the export button in the Summary panel, not in the report', async () => {
    await chooseSpec()
    const summary = screen.getByRole('region', { name: 'Summary' })
    expect(within(summary).getByRole('button', { name: /Export report/ })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await screen.findByRole('region', { name: 'Conformance results' })
    for (const label of ['Covered', 'Not covered', 'Passing', 'Failing']) expect(within(summary).getByText(label)).toBeInTheDocument()
    expect(within(summary).getByRole('button', { name: /Export report/ })).toBeEnabled()
    const report = screen.getByRole('region', { name: 'Conformance results' })
    expect(within(report).queryByText('Covered')).not.toBeInTheDocument()
  })

  it('lists the failures first, each with what was expected, what came back, why and how to fix it', async () => {
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByTestId('conform-headline')).toHaveTextContent('1 of 7 operations break the spec')
    const problems = screen.getByRole('region', { name: 'Problems' })
    const card = within(problems).getByRole('article', { name: 'GET /bad fails' })
    expect(card).toHaveTextContent('"price" is text, the spec says number')
    expect(card).toHaveTextContent('Expected')
    expect(card).toHaveTextContent('"9.99"')
    expect(card).toHaveTextContent('The server sent text ("9.99") where the spec declares a number.')
    expect(card).toHaveTextContent('Send a number from the server.')
    expect(within(card).getByText('Technical detail')).toBeInTheDocument()
    expect(screen.getByText(/wrong type × 1/)).toBeInTheDocument()

    // The failures come before the passing operations in the page.
    const passing = screen.getByRole('region', { name: 'Passing operations' })
    expect(problems.compareDocumentPosition(passing) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(passing).toHaveTextContent('/ok')
    expect(passing).not.toHaveTextContent('/bad')
  })

  it('says everything matches when nothing fails and falls back to the raw message without findings', async () => {
    bridge.runConformance.mockResolvedValueOnce({ ...output, failing: 0, operations: [output.operations[0]] })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByTestId('conform-headline')).toHaveTextContent('All 1 checked operations match the spec')
    expect(screen.queryByRole('region', { name: 'Problems' })).not.toBeInTheDocument()

    bridge.runConformance.mockResolvedValueOnce({
      ...output,
      operations: [{ ...output.operations[1], findings: undefined }],
    })
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByText('price: expected number')).toBeInTheDocument()
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

describe('ConformanceTab options and export', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(ctx)
    bridge.inspectConformSpec.mockResolvedValue(info)
    bridge.runConformance.mockResolvedValue({
      ...output,
      probes: [
        { operationId: 'a', method: 'GET', path: '/ok', kind: 'missing-auth', description: 'no credentials', status: 'rejected', statusCode: 401 },
        { operationId: 'a', method: 'GET', path: '/ok', kind: 'wrong-type', description: 'id as text', status: 'accepted', statusCode: 200 },
      ],
      coverageNote: 'Added to coverage as api.yaml.',
    })
  })

  it('passes the options to the run', async () => {
    await chooseSpec()
    fireEvent.click(screen.getByText('Options'))
    fireEvent.change(screen.getByLabelText('Tags'), { target: { value: 'users, orders' } })
    fireEvent.change(screen.getByLabelText('Path prefixes'), { target: { value: '/users' } })
    fireEvent.change(screen.getByLabelText(/Request timeout/), { target: { value: '3000' } })
    fireEvent.click(screen.getByRole('checkbox', { name: /negative probes/ }))
    fireEvent.click(screen.getByRole('checkbox', { name: /Stop after the first failure/ }))
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await waitFor(() => expect(bridge.runConformance).toHaveBeenCalledWith(expect.objectContaining({
      tags: ['users', 'orders'], paths: ['/users'], timeoutMs: 3000, negative: true, failFast: true, recordCoverage: false,
    })))
  })

  it('does not offer coverage recording while a filter is set', async () => {
    await chooseSpec()
    const box = screen.getByRole('checkbox', { name: /Add this run to Contracts coverage/ })
    expect(box).toBeEnabled()
    fireEvent.click(box)
    fireEvent.change(screen.getByLabelText('Tags'), { target: { value: 'users' } })
    expect(box).toBeDisabled()
    expect(box).not.toBeChecked()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await waitFor(() => expect(bridge.runConformance).toHaveBeenCalledWith(expect.objectContaining({ recordCoverage: false })))
  })

  it('refuses a timeout that is not a positive number', async () => {
    await chooseSpec()
    fireEvent.change(screen.getByLabelText(/Request timeout/), { target: { value: 'abc' } })
    expect(screen.getByRole('button', { name: /Run conformance check/ })).toBeDisabled()
    expect(screen.getByText(/Enter a number above 0/)).toBeInTheDocument()
  })

  it('shows the probes and the coverage note', async () => {
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByText('Negative probes')).toBeInTheDocument()
    expect(screen.getByText(/1 rejected as they should be/)).toBeInTheDocument()
    expect(screen.getByText(/1 accepted by the server/)).toBeInTheDocument()
    expect(screen.getByTestId('conform-coverage-note')).toHaveTextContent('Added to coverage as api.yaml.')
  })

  it('shows environment warnings above the results', async () => {
    bridge.runConformance.mockResolvedValue({ ...output, warnings: ['environment variable API_TOKEN is not set (the environment variable "token" refers to it as ${API_TOKEN})'] })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByRole('list', { name: 'Environment warnings' })).toHaveTextContent('API_TOKEN is not set')
  })

  it('saves a non-HTML export through the file dialog and shows export errors', async () => {
    bridge.exportConformReport.mockResolvedValueOnce({ content: '<xml/>' }).mockResolvedValueOnce({ content: '', error: 'no result' })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    fireEvent.click(await screen.findByRole('button', { name: /Export report/ }))
    fireEvent.click(screen.getByText('JUnit XML'))
    await waitFor(() => expect(bridge.saveContentToFile).toHaveBeenCalledWith('<xml/>', 'conformance.xml'))
    fireEvent.click(screen.getByRole('button', { name: /Export report/ }))
    fireEvent.click(screen.getByText('JSON'))
    expect(await screen.findByRole('alert')).toHaveTextContent('no result')
  })
})

describe('splitList', () => {
  it('trims and drops empty items', () => {
    expect(splitList(' a, b ,,c ')).toEqual(['a', 'b', 'c'])
    expect(splitList('')).toEqual([])
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

describe('cliValues', () => {
  const base = {
    specPath: '/ws/api.yaml', baseUrl: ' http://localhost:8120 ', envPath: '', activeEnvironment: '',
    tags: [] as string[], paths: [] as string[], negative: false, failFast: false, timeoutText: '', allowWrites: false,
  }

  it('is read-only unless writes are allowed, and leaves unset choices out', () => {
    expect(cliValues(base)).toEqual({ spec: '/ws/api.yaml', 'base-url': 'http://localhost:8120', 'read-only': 'true' })
    expect(cliValues({ ...base, allowWrites: true })).not.toHaveProperty('read-only')
  })

  it('carries every option', () => {
    expect(cliValues({
      ...base, envPath: '/ws/environments/ci.yaml', activeEnvironment: '/ws/environments/local.yaml',
      tags: ['users', 'orders'], paths: ['/users'], negative: true, failFast: true, timeoutText: '3000', allowWrites: true,
    })).toEqual({
      spec: '/ws/api.yaml', 'base-url': 'http://localhost:8120', env: '/ws/environments/ci.yaml',
      tags: 'users,orders', paths: '/users', negative: 'true', 'fail-fast': 'true', 'request-timeout': '3000',
    })
  })

  it('uses the active environment when none is chosen, and ignores a bad timeout', () => {
    const v = cliValues({ ...base, activeEnvironment: '/ws/environments/local.yaml', timeoutText: 'abc', allowWrites: true })
    expect(v['env']).toBe('/ws/environments/local.yaml')
    expect(v).not.toHaveProperty('request-timeout')
  })
})

describe('ConformanceTab Get CLI command', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue({ ...ctx, activeEnvironment: '/ws/environments/local.yaml' })
    bridge.inspectConformSpec.mockResolvedValue(info)
  })

  it('hands the current choices to the builder', async () => {
    const open = vi.fn()
    render(<ConformanceTab onOpenCliBuilder={open} />)
    fireEvent.change(await screen.findByLabelText(/Spec/), { target: { value: '/ws/api.yaml' } })
    await waitFor(() => expect(screen.getByLabelText(/Base URL/)).toHaveValue('http://localhost:8120'))
    fireEvent.click(screen.getByRole('button', { name: /Get CLI command/ }))
    expect(open).toHaveBeenCalledWith({
      spec: '/ws/api.yaml', 'base-url': 'http://localhost:8120', env: '/ws/environments/local.yaml', 'read-only': 'true',
    })
  })
})
