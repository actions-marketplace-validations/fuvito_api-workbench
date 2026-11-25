import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within, act } from '@testing-library/react'
import { ConformanceTab } from './ConformanceTab'
import { isLocalUrl, splitList, parseVars, parsePathLimits, cliValues, findingWhere } from '@/lib/conformHelpers'

const bridge = vi.hoisted(() => ({
  getConformContext: vi.fn(),
  inspectConformSpec: vi.fn(),
  runConformance: vi.fn(),
  cancelConformance: vi.fn(),
  onConformanceProgress: vi.fn((_cb: unknown): (() => void) => () => {}),
  exportConformReport: vi.fn(),
  recordConformBaseline: vi.fn(),
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
      tags: [], paths: [], negative: false, failFast: false, failOnWarning: false, setupScenario: '', baselinePath: '', timeoutMs: 0, maxResponseTimeMs: 0, maxResponseTimeFor: [], recordCoverage: false, vars: [],
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
      tags: ['users', 'orders'], paths: ['/users'], timeoutMs: 3000, maxResponseTimeMs: 0, maxResponseTimeFor: [], negative: true, failFast: true, recordCoverage: false,
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

describe('parsePathLimits', () => {
  it('reads /prefix=ms lines and names the ones that do not fit', () => {
    expect(parsePathLimits(['/reports=2000', '', ' /a/b=0 '].join('\n'))).toEqual({ lines: ['/reports=2000', '/a/b=0'], bad: [] })
    expect(parsePathLimits(['reports=1', '/x=abc'].join('\n')).bad).toEqual(['reports=1', '/x=abc'])
  })
})

describe('warnings', () => {
  const warned = {
    ...output, failing: 0, passing: 1, covered: 1, notCovered: 0, total: 1, failed: false, withWarnings: 1,
    operations: [{
      operationId: 'w', method: 'GET', path: '/warned', status: 'passed', statusCode: 200, durationMs: 5, violations: ['price: expected number'], warnings: 1,
      findings: [{
        kind: 'wrong-type', checkId: 'CS-RESP-006', severity: 'warning', where: 'price', summary: '"price" is text, the spec says number',
        why: 'The server sent text.', fix: 'Send a number.',
      }],
    }],
  }

  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(ctx)
    bridge.inspectConformSpec.mockResolvedValue(info)
    bridge.runConformance.mockResolvedValue(warned)
  })

  it('lists a passing operation with its warning and explains it', async () => {
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByTestId('conform-headline')).toHaveTextContent('All 1 checked operations match the spec, 1 with warnings')
    expect(screen.getByTestId('op-warnings')).toHaveTextContent('1 warning')
    expect(screen.getByTestId('finding-warning')).toBeInTheDocument()
    expect(screen.getByText('Send a number.')).toBeInTheDocument()
  })

  it('sends "fail on warnings" with the run and hands it to the CLI builder', async () => {
    await chooseSpec()
    fireEvent.click(screen.getByText('Options'))
    fireEvent.click(screen.getByRole('checkbox', { name: /Fail on warnings/ }))
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await waitFor(() => expect(bridge.runConformance).toHaveBeenCalledWith(expect.objectContaining({ failOnWarning: true })))
  })

  it('cliValues carries fail-on only when warnings fail the run', () => {
    const base = { specPath: '/ws/api.yaml', baseUrl: 'http://x', envPath: '', activeEnvironment: '', tags: [], paths: [], negative: false, failFast: false, timeoutText: '', allowWrites: false }
    expect(cliValues({ ...base, failOnWarning: true })['fail-on']).toBe('warning')
    expect(cliValues(base)['fail-on']).toBeUndefined()
  })
})

describe('starting values from the workspace project file', () => {
  const defaults = {
    found: true, file: 'clrsig.yaml', note: '', applied: ['spec', 'environment', 'baseUrl', 'conform.negative', 'conform.failOn', 'conform.tags'],
    specPath: '/ws/api.yaml', environmentPath: '/ws/environments/local.yaml', baseUrl: 'http://localhost:9000',
    tags: ['users'], paths: [], negative: true, failOnWarning: true, requestTimeoutMs: 3000, maxResponseTimeMs: 0, maxResponseTimeFor: [],
  }

  beforeEach(() => {
    vi.clearAllMocks()
    bridge.inspectConformSpec.mockResolvedValue(info)
    bridge.runConformance.mockResolvedValue(output)
  })

  it('preselects the spec, environment, base URL and options, and says where they came from', async () => {
    bridge.getConformContext.mockResolvedValue({ ...ctx, projectDefaults: defaults })
    render(<ConformanceTab />)
    expect(await screen.findByTestId('project-defaults-note')).toHaveTextContent('Starting values from clrsig.yaml: spec, environment, baseUrl')
    await waitFor(() => expect(screen.getByLabelText(/Base URL/)).toHaveValue('http://localhost:9000'))
    expect(screen.getByLabelText(/Spec/)).toHaveValue('/ws/api.yaml')
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await waitFor(() => expect(bridge.runConformance).toHaveBeenCalledWith(expect.objectContaining({
      specPath: '/ws/api.yaml', envPath: '/ws/environments/local.yaml', baseUrl: 'http://localhost:9000', tags: ['users'], negative: true, failOnWarning: true, timeoutMs: 3000,
    })))
  })

  it('never turns on writes, whatever the file says', async () => {
    bridge.getConformContext.mockResolvedValue({ ...ctx, projectDefaults: { ...defaults, note: 'conform.readOnly: false is not applied; this page sends only reads until you tick Allow writes.' } })
    render(<ConformanceTab />)
    expect(await screen.findByTestId('project-defaults-note')).toHaveTextContent('readOnly: false is not applied')
    await waitFor(() => expect(screen.getByLabelText(/Base URL/)).toHaveValue('http://localhost:9000'))
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await waitFor(() => expect(bridge.runConformance).toHaveBeenCalledWith(expect.objectContaining({ allowWrites: false })))
  })

  it('ignores a spec that is not among the workspace files, and does nothing without a file', async () => {
    bridge.getConformContext.mockResolvedValue({ ...ctx, projectDefaults: { ...defaults, specPath: '/elsewhere/api.yaml' } })
    const { unmount } = render(<ConformanceTab />)
    await screen.findByTestId('project-defaults-note')
    expect(screen.getByLabelText(/Spec/)).toHaveValue('')
    unmount()

    bridge.getConformContext.mockResolvedValue({ ...ctx, projectDefaults: { found: false, applied: [], tags: [], paths: [], negative: false, failOnWarning: false, requestTimeoutMs: 0, maxResponseTimeMs: 0, maxResponseTimeFor: [] } })
    render(<ConformanceTab />)
    await screen.findByLabelText(/Spec/)
    expect(screen.queryByTestId('project-defaults-note')).not.toBeInTheDocument()
  })
})

describe('run a scenario first', () => {
  const withScenarios = { ...ctx, scenarios: [{ path: '/ws/scenarios/login.yaml', label: 'scenarios/login.yaml' }] }

  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(withScenarios)
    bridge.inspectConformSpec.mockResolvedValue(info)
    bridge.runConformance.mockResolvedValue({ ...output, setupNote: 'Setup scenario login.yaml passed; values available to the check: token' })
  })

  it('sends the chosen scenario with the run and shows what it made available', async () => {
    await chooseSpec()
    fireEvent.click(screen.getByText('Options'))
    fireEvent.change(screen.getByLabelText('Run a scenario first'), { target: { value: '/ws/scenarios/login.yaml' } })
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await waitFor(() => expect(bridge.runConformance).toHaveBeenCalledWith(expect.objectContaining({ setupScenario: '/ws/scenarios/login.yaml' })))
    expect(await screen.findByTestId('conform-setup-note')).toHaveTextContent('values available to the check: token')
  })

  it('offers no scenario by default and hands the choice to the CLI builder', async () => {
    const open = vi.fn()
    render(<ConformanceTab onOpenCliBuilder={open} />)
    fireEvent.change(await screen.findByLabelText(/Spec/), { target: { value: '/ws/api.yaml' } })
    await waitFor(() => expect(screen.getByLabelText(/Base URL/)).toHaveValue('http://localhost:8120'))
    fireEvent.click(screen.getByText('Options'))
    expect(screen.getByLabelText('Run a scenario first')).toHaveValue('')
    fireEvent.change(screen.getByLabelText('Run a scenario first'), { target: { value: '/ws/scenarios/login.yaml' } })
    fireEvent.click(screen.getByRole('button', { name: /Get CLI command/ }))
    expect(open).toHaveBeenCalledWith(expect.objectContaining({ setup: '/ws/scenarios/login.yaml' }))
  })

  it('cliValues carries the scenario to run first', () => {
    const base = { specPath: '/ws/api.yaml', baseUrl: 'http://x', envPath: '', activeEnvironment: '', tags: [], paths: [], negative: false, failFast: false, timeoutText: '', allowWrites: false }
    expect(cliValues({ ...base, setupPath: '/ws/scenarios/login.yaml' }).setup).toBe('/ws/scenarios/login.yaml')
    expect(cliValues(base).setup).toBeUndefined()
  })
})

describe('copy a finding as an issue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(ctx)
    bridge.inspectConformSpec.mockResolvedValue(info)
  })

  it('puts the check, the operation, what was wrong and how to fix it on the clipboard', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })
    bridge.runConformance.mockResolvedValue({
      ...output,
      operations: [{
        operationId: 'b', method: 'GET', path: '/bad', status: 'failed', statusCode: 200, durationMs: 4, violations: [],
        findings: [{
          kind: 'wrong-type', checkId: 'CS-RESP-006', where: 'price', part: 'response body', spec: 'GET /bad → response 200',
          summary: '"price" is text, the spec says number', expected: 'number', got: '"9.99"',
          why: 'The server sent text.', fix: 'Send a number.', detail: 'raw',
        }],
      }],
    })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    fireEvent.click(await screen.findByRole('button', { name: 'Copy as issue' }))
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1))
    const text = writeText.mock.calls[0][0] as string
    for (const want of ['[CS-RESP-006] "price" is text, the spec says number', '- Operation: GET /bad', '- Expected: number', '- Got: "9.99"', '- In the spec: GET /bad → response 200', '- Why: The server sent text.', '- Fix: Send a number.', 'Found by Clear Signal ']) {
      expect(text).toContain(want)
    }
    expect(text).not.toContain('raw')
    expect(await screen.findByRole('button', { name: 'Copied' })).toBeInTheDocument()
  })
})

describe('baselines', () => {
  const withBaselines = { ...ctx, workspace: '/ws', baselines: [{ path: '/ws/conform-baseline.yaml', label: 'conform-baseline.yaml' }] }
  const failing = {
    ...output, failed: true, failing: 1, passing: 0, covered: 1,
    operations: [{ ...output.operations[1], status: 'failed' }],
  }

  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(withBaselines)
    bridge.inspectConformSpec.mockResolvedValue(info)
    bridge.runConformance.mockResolvedValue(failing)
    bridge.recordConformBaseline = vi.fn().mockResolvedValue({ path: '/ws/conform-baseline.yaml', violations: 1, replaced: true })
  })

  it('sends the chosen baseline and shows known, new and fixed against it', async () => {
    bridge.runConformance.mockResolvedValue({
      ...failing, failed: true,
      baseline: {
        path: '/ws/conform-baseline.yaml', legacy: true,
        known: [{ operation: 'GET /a', check: 'CS-RESP-006', where: 'price', summary: 'text, not number' }],
        new: [{ operation: 'GET /b', check: 'CS-RESP-005', where: 'id', summary: 'id is missing' }],
        fixed: [{ operation: 'GET /c', summary: 'old wording' }],
      },
    })
    await chooseSpec()
    fireEvent.click(screen.getByText('Options'))
    fireEvent.change(screen.getByLabelText('Baseline'), { target: { value: '/ws/conform-baseline.yaml' } })
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await waitFor(() => expect(bridge.runConformance).toHaveBeenCalledWith(expect.objectContaining({ baselinePath: '/ws/conform-baseline.yaml' })))
    expect(await screen.findByTestId('baseline-summary')).toHaveTextContent('1 known, 1 new, 1 fixed')
    const region = screen.getByRole('region', { name: 'Baseline' })
    expect(region).toHaveTextContent('GET /b')
    expect(region).toHaveTextContent('CS-RESP-005')
    expect(region).toHaveTextContent('old format')
  })

  it('says when every failure is known to the baseline', async () => {
    bridge.runConformance.mockResolvedValue({ ...failing, failed: false, baseline: { path: '/ws/conform-baseline.yaml', legacy: false, known: [{ operation: 'GET /bad', summary: 'x' }], new: [], fixed: [] } })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    expect(await screen.findByTestId('conform-headline')).toHaveTextContent('all known to the baseline')
  })

  it('records the run as a baseline, asking first before replacing one', async () => {
    await chooseSpec()
    expect(screen.getByRole('button', { name: 'Record as baseline' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await screen.findByTestId('conform-headline')
    fireEvent.click(screen.getByRole('button', { name: 'Record as baseline' }))
    expect(screen.getByRole('alertdialog', { name: 'Replace the baseline' })).toHaveTextContent('conform-baseline.yaml')
    expect(bridge.recordConformBaseline).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Replace' }))
    await waitFor(() => expect(bridge.recordConformBaseline).toHaveBeenCalledWith('/ws/conform-baseline.yaml'))
    expect(await screen.findByTestId('baseline-recorded')).toHaveTextContent('Recorded 1 violation in conform-baseline.yaml (replaced the old file)')
  })

  it('records a new file without asking, and not while tags or paths limit the run', async () => {
    bridge.getConformContext.mockResolvedValue({ ...ctx, workspace: '/ws', baselines: [] })
    bridge.recordConformBaseline = vi.fn().mockResolvedValue({ path: '/ws/conform-baseline.yaml', violations: 1, replaced: false })
    await chooseSpec()
    fireEvent.click(screen.getByRole('button', { name: /Run conformance check/ }))
    await screen.findByTestId('conform-headline')
    fireEvent.click(screen.getByRole('button', { name: 'Record as baseline' }))
    await waitFor(() => expect(bridge.recordConformBaseline).toHaveBeenCalledWith('/ws/conform-baseline.yaml'))
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()

    fireEvent.click(screen.getByText('Options'))
    fireEvent.change(screen.getByLabelText('Tags'), { target: { value: 'users' } })
    expect(screen.getByRole('button', { name: 'Record as baseline' })).toBeDisabled()
  })
})
