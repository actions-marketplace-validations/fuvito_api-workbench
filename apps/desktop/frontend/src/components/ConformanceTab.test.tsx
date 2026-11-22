import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within, act } from '@testing-library/react'
import { ConformanceTab } from './ConformanceTab'
import { isLocalUrl, splitList, parseVars, cliValues, findingWhere } from '@/lib/conformHelpers'

const bridge = vi.hoisted(() => ({
  getConformContext: vi.fn(),
  inspectConformSpec: vi.fn(),
  runConformance: vi.fn(),
  cancelConformance: vi.fn(),
  onConformanceProgress: vi.fn((_cb: unknown): (() => void) => () => {}),
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
      tags: [], paths: [], negative: false, failFast: false, timeoutMs: 0, maxResponseTimeMs: 0, recordCoverage: false, vars: [],
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
      tags: ['users', 'orders'], paths: ['/users'], timeoutMs: 3000, maxResponseTimeMs: 0, negative: true, failFast: true, recordCoverage: false,
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

  it('carries the response-time limit and ignores a bad one', () => {
    expect(cliValues({ ...base, maxResponseText: '500', allowWrites: true })['max-response-time']).toBe('500')
    expect(cliValues({ ...base, maxResponseText: 'fast', allowWrites: true })).not.toHaveProperty('max-response-time')
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

describe('variable overrides', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(ctx)
    bridge.inspectConformSpec.mockResolvedValue(info)
    bridge.runConformance.mockResolvedValue(output)
  })

  it('parseVars keeps name=value lines and reports the bad ones', () => {
    expect(parseVars('id=1\n\n  name = Lamp  \nurl=http://a?b=c')).toEqual({ vars: ['id=1', 'name = Lamp', 'url=http://a?b=c'], bad: [] })
    expect(parseVars('novalue\n=x\nok=1')).toEqual({ vars: ['ok=1'], bad: ['novalue', '=x'] })
    expect(parseVars('')).toEqual({ vars: [], bad: [] })
  })

  it('sends the overrides with the run', async () => {
    await chooseSpec()
    fireEvent.click(screen.getByText('Options'))
    fireEvent.change(screen.getByLabelText('Variable overrides'), { target: { value: 'id=1\nname=Lamp' } })
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await waitFor(() => expect(bridge.runConformance).toHaveBeenCalledWith(expect.objectContaining({ vars: ['id=1', 'name=Lamp'] })))
  })

  it('blocks a run while a line is not name=value', async () => {
    await chooseSpec()
    fireEvent.click(screen.getByText('Options'))
    fireEvent.change(screen.getByLabelText('Variable overrides'), { target: { value: 'oops' } })
    expect(screen.getByRole('button', { name: /Run conformance check/ })).toBeDisabled()
    expect(screen.getByRole('alert')).toHaveTextContent('name=value: oops')
  })
})

describe('run progress', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(ctx)
    bridge.inspectConformSpec.mockResolvedValue(info)
  })

  it('shows how far a running check is, and stops listening when it ends', async () => {
    let listener: ((p: { done: number; total: number; operation: string }) => void) | undefined
    const stop = vi.fn()
    bridge.onConformanceProgress.mockImplementation((cb: unknown) => { listener = cb as typeof listener; return stop })
    let finish: (v: unknown) => void = () => {}
    bridge.runConformance.mockReturnValue(new Promise(resolve => { finish = resolve }))

    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByTestId('conform-progress')).toHaveTextContent('Starting…')
    await waitFor(() => expect(listener).toBeDefined())

    act(() => listener?.({ done: 3, total: 7, operation: 'GET /products' }))
    expect(screen.getByTestId('conform-progress')).toHaveTextContent('3 of 7 · GET /products')
    const bar = screen.getByRole('progressbar', { name: 'Conformance run progress' })
    expect(bar).toHaveAttribute('value', '3')
    expect(bar).toHaveAttribute('max', '7')

    await act(async () => { finish(output) })
    await screen.findByRole('region', { name: 'Conformance results' })
    expect(screen.queryByTestId('conform-progress')).not.toBeInTheDocument()
    expect(stop).toHaveBeenCalled()
  })
})

describe('operations that could not be sent', () => {
  const blockedOp = (id: string, method: string, path: string, missing: string[]) => ({
    operationId: id, method, path, status: 'blocked' as const, statusCode: 0, durationMs: 0, violations: [],
    findings: [{
      kind: 'missing-value' as const, where: missing.join(', '), summary: `No value for ${missing.map(m => `{{${m}}}`).join(', ')}`,
      got: 'not sent', why: 'The request needs a value nothing provides. It was not sent, so nothing was compared with the spec.',
      fix: `Add ${missing.join(', ')} to the environment, or give it as an override.`, detail: 'not sent',
    }],
  })
  const blockedOutput = {
    ...output, failing: 0, passing: 1, covered: 1, notCovered: 2, blockedOperations: 2, missingValues: ['id', 'name'],
    operations: [output.operations[0], blockedOp('a', 'GET', '/a/{{id}}', ['id']), blockedOp('b', 'POST', '/b', ['id', 'name'])],
  }

  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(ctx)
    bridge.inspectConformSpec.mockResolvedValue(info)
  })

  it('shows them as not sent, never as passing, and says what to set once', async () => {
    bridge.runConformance.mockResolvedValue(blockedOutput)
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByTestId('conform-headline')).toHaveTextContent('2 of 7 operations could not be checked')

    const card = screen.getByRole('article', { name: 'POST /b fails' })
    expect(card).toHaveTextContent('NOT SENT')
    expect(card).toHaveTextContent('No value for {{id}}, {{name}}')
    expect(screen.getByRole('region', { name: 'Passing operations' })).not.toHaveTextContent('/a/')

    const panel = screen.getByRole('region', { name: 'Missing values' })
    expect(panel).toHaveTextContent('2 operations were not sent: no value for {{id}}, {{name}}')
    expect(screen.getByText(/missing value × 2/)).toBeInTheDocument()
  })

  it('mixes failures and blocked operations in one headline', async () => {
    bridge.runConformance.mockResolvedValue({
      ...blockedOutput, failing: 1,
      operations: [output.operations[0], output.operations[1], blockedOp('a', 'GET', '/a/{{id}}', ['id'])],
      blockedOperations: 1, missingValues: ['id'],
    })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByTestId('conform-headline')).toHaveTextContent('1 of 7 operations break the spec, 1 could not be checked')
    expect(screen.getByRole('region', { name: 'Missing values' })).toHaveTextContent('1 operation was not sent')
  })

  it('says nothing was reached when every sent operation got no answer, and counts skipped ones apart', async () => {
    const dead = (id: string) => ({ ...output.operations[0], operationId: id, path: `/${id}`, status: 'error', error: 'dial tcp: refused', statusCode: 0, violations: [], findings: [] })
    bridge.runConformance.mockResolvedValue({
      ...output, failed: true, covered: 0, passing: 0, failing: 0, notCovered: 3, total: 3,
      operations: [dead('a'), dead('b'), { ...dead('c'), status: 'skipped', error: '' }],
    })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByTestId('conform-headline')).toHaveTextContent('No operation was reached: is the server running at')
    expect(screen.getByTestId('conform-skipped')).toHaveTextContent('1 skipped')
  })

  it('adds the missing names to the variable overrides, once each, and opens the options', async () => {
    bridge.runConformance.mockResolvedValue(blockedOutput)
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await screen.findByRole('region', { name: 'Missing values' })

    fireEvent.click(screen.getByRole('button', { name: 'Add these to Variable overrides' }))
    const box = screen.getByLabelText('Variable overrides') as HTMLTextAreaElement
    expect(box.value).toBe('id=\nname=')
    expect(screen.getByText('Options').closest('details')).toHaveAttribute('open')

    // Clicking again does not duplicate lines; a value already typed is kept.
    fireEvent.change(box, { target: { value: 'id=1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add these to Variable overrides' }))
    expect((screen.getByLabelText('Variable overrides') as HTMLTextAreaElement).value).toBe('id=1\nname=')
  })

  it('shows no missing-values box when everything was sent', async () => {
    bridge.runConformance.mockResolvedValue(output)
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await screen.findByTestId('conform-headline')
    expect(screen.queryByRole('region', { name: 'Missing values' })).not.toBeInTheDocument()
  })
})

describe('where the problem is', () => {
  it('findingWhere says it in words', () => {
    expect(findingWhere({})).toBe('')
    expect(findingWhere({ part: 'response status', where: '503' })).toBe('response status 503')
    expect(findingWhere({ part: 'response header', where: 'X-Total-Count' })).toBe('response header "X-Total-Count"')
    expect(findingWhere({ part: 'response body', where: '[0].id' })).toBe('response body, field [0].id')
    expect(findingWhere({ part: 'response body' })).toBe('response body')
  })

  const finding = (kind: string, over: object) => ({
    kind, summary: 's', why: 'because', fix: 'do this', detail: 'raw', ...over,
  })
  const failing = (id: string, method: string, path: string, statusCode: number, f: object) => ({
    operationId: id, method, path, status: 'failed' as const, statusCode, durationMs: 3, violations: ['raw'], findings: [f],
  })

  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(ctx)
    bridge.inspectConformSpec.mockResolvedValue(info)
  })

  it('shows what, where in the response, where in the spec, why and how to fix it', async () => {
    bridge.runConformance.mockResolvedValue({
      ...output, passing: 0, failing: 3, covered: 3, operations: [
        failing('h', 'GET', '/inventory', 200, finding('missing-header', {
          summary: 'The required header "X-Total-Count" was not sent', part: 'response header', where: 'X-Total-Count',
          spec: 'GET /inventory → response 200 → headers → X-Total-Count', expected: 'the header X-Total-Count', got: 'not sent',
          why: 'The spec marks it as required.', fix: 'Send it.',
        })),
        failing('s', 'GET', '/status', 503, finding('undeclared-status', {
          summary: 'Status 503 is not declared for GET /status', part: 'response status', where: '503',
          spec: 'GET /status → responses', expected: 'one of: 200', got: '503',
        })),
        failing('c', 'GET', '/widgets', 200, finding('wrong-content-type', {
          summary: 'Content-Type is "text/plain", the spec says application/json', part: 'response header', where: 'Content-Type',
          spec: 'GET /widgets → response 200 → content', expected: 'application/json', got: 'text/plain',
        })),
      ],
    })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await screen.findByTestId('conform-headline')

    const card = screen.getByRole('article', { name: 'GET /inventory fails' })
    expect(within(card).getByTestId('finding-where')).toHaveTextContent('response header "X-Total-Count"')
    expect(within(card).getByTestId('finding-spec')).toHaveTextContent('GET /inventory → response 200 → headers → X-Total-Count')
    expect(card).toHaveTextContent('The required header "X-Total-Count" was not sent')
    expect(card).toHaveTextContent('Expected')
    expect(card).toHaveTextContent('not sent')
    expect(card).toHaveTextContent('The spec marks it as required.')
    expect(card).toHaveTextContent('Send it.')
    expect(within(card).getByText('Technical detail')).toBeInTheDocument()

    expect(within(screen.getByRole('article', { name: 'GET /status fails' })).getByTestId('finding-where')).toHaveTextContent('response status 503')
    expect(within(screen.getByRole('article', { name: 'GET /widgets fails' })).getByTestId('finding-spec')).toHaveTextContent('GET /widgets → response 200 → content')

    for (const label of ['missing header', 'undeclared status', 'wrong content type']) {
      expect(screen.getByText(new RegExp(`${label} × 1`))).toBeInTheDocument()
    }
  })

  it('leaves the location rows out when a finding has none (older data)', async () => {
    bridge.runConformance.mockResolvedValue({
      ...output,
      operations: [{ ...output.operations[1], findings: [finding('other', { summary: 'something differs' })] }],
    })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await screen.findByTestId('conform-headline')
    const card = screen.getByRole('article', { name: 'GET /bad fails' })
    expect(within(card).queryByTestId('finding-where')).not.toBeInTheDocument()
    expect(within(card).queryByTestId('finding-spec')).not.toBeInTheDocument()
  })
  it('applies a choice made in the sidebar and reports the selection and the last result', async () => {
    const onSelectionChange = vi.fn()
    const onResult = vi.fn()
    const { rerender } = render(<ConformanceTab onSelectionChange={onSelectionChange} onResult={onResult} />)
    await screen.findByLabelText(/Spec/)
    rerender(<ConformanceTab request={{ id: 1, specPath: '/ws/api.yaml' }} onSelectionChange={onSelectionChange} onResult={onResult} />)
    await waitFor(() => expect(screen.getByLabelText(/Base URL/)).toHaveValue('http://localhost:8120'))
    expect(screen.getByLabelText(/Spec/)).toHaveValue('/ws/api.yaml')
    expect(onSelectionChange).toHaveBeenLastCalledWith({ specPath: '/ws/api.yaml', envPath: '' })

    rerender(<ConformanceTab request={{ id: 2, envPath: '/ws/environments/local.yaml' }} onSelectionChange={onSelectionChange} onResult={onResult} />)
    expect(screen.getByLabelText(/Environment/)).toHaveValue('/ws/environments/local.yaml')
    expect(screen.getByLabelText(/Spec/)).toHaveValue('/ws/api.yaml')

    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await screen.findByTestId('conform-headline')
    expect(onResult).toHaveBeenCalledWith(expect.objectContaining({ specPath: '/ws/api.yaml', specTitle: 'Shop', total: 7, covered: 5, passing: 4, failing: 1 }))
  })
})
