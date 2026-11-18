import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ScenarioRunLog } from './ScenarioRunLog'
import * as bridge from '@/lib/bridge'
import type { ScenarioRunEntry, ScenarioResult, StepResult } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({
  getScenarioRunHistory: vi.fn(),
  loadScenarioResult: vi.fn(),
}))

const b = vi.mocked(bridge)

function run(over: Partial<ScenarioRunEntry> = {}): ScenarioRunEntry {
  return {
    id: 1,
    executedAt: '2026-11-15T10:00:00Z',
    scenarioPath: '/ws/scenarios/login-flow.yaml',
    title: 'Login flow',
    passed: true,
    durationMs: 250,
    stepTotal: 2,
    stepPassed: 2,
    stepFailed: 0,
    stepSkipped: 0,
    steps: [
      { label: 'Login', passed: true, skipped: false, statusCode: 200, durationMs: 120 },
      { label: 'Profile', passed: true, skipped: false, statusCode: 200, durationMs: 130 },
    ],
    ...over,
  }
}

function step(over: Partial<StepResult> = {}): StepResult {
  return {
    label: 'Login', passed: true, skipped: false, statusCode: 200, durationMs: 120,
    extracted: {}, assertions: [], ...over,
  }
}

function result(steps: StepResult[]): ScenarioResult {
  return { title: 'Login flow', passed: true, durationMs: 1, steps }
}

beforeEach(() => {
  vi.clearAllMocks()
  b.getScenarioRunHistory.mockResolvedValue([])
  Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.resolve()) } })
})

describe('ScenarioRunLog', () => {
  it('shows a placeholder and does not fetch without a workspace', () => {
    render(<ScenarioRunLog workspacePath="" />)
    expect(screen.getByText('No workspace open')).toBeInTheDocument()
    expect(b.getScenarioRunHistory).not.toHaveBeenCalled()
  })

  it('shows an empty state when there are no runs', async () => {
    render(<ScenarioRunLog workspacePath="/ws" />)
    expect(await screen.findByText('No runs yet')).toBeInTheDocument()
    expect(b.getScenarioRunHistory).toHaveBeenCalledWith(200)
  })

  it('lists runs with name, duration and step counts', async () => {
    b.getScenarioRunHistory.mockResolvedValue([run({ stepFailed: 1, stepSkipped: 2, durationMs: 2500 })])
    render(<ScenarioRunLog workspacePath="/ws" />)
    expect(await screen.findByText('Login flow')).toBeInTheDocument()
    expect(screen.getByText('2.5s')).toBeInTheDocument()
    expect(screen.getByText('2 passed')).toBeInTheDocument()
    expect(screen.getByText('1 failed')).toBeInTheDocument()
    expect(screen.getByText('2 skipped')).toBeInTheDocument()
  })

  it('omits failed/skipped badges when zero and formats sub-second durations in ms', async () => {
    b.getScenarioRunHistory.mockResolvedValue([run({ durationMs: 250 })])
    render(<ScenarioRunLog workspacePath="/ws" />)
    expect(await screen.findByText('250ms')).toBeInTheDocument()
    expect(screen.queryByText(/failed/)).not.toBeInTheDocument()
    expect(screen.queryByText(/skipped/)).not.toBeInTheDocument()
  })

  it('derives a name from the file path when the title is empty', async () => {
    b.getScenarioRunHistory.mockResolvedValue([run({ title: '', scenarioPath: 'C:\\ws\\scenarios\\smoke.yml' })])
    render(<ScenarioRunLog workspacePath="/ws" />)
    expect(await screen.findByText('smoke')).toBeInTheDocument()
  })

  it('falls back to "scenario" when no name can be derived', async () => {
    b.getScenarioRunHistory.mockResolvedValue([run({ title: '', scenarioPath: '' })])
    render(<ScenarioRunLog workspacePath="/ws" />)
    expect(await screen.findByText('scenario')).toBeInTheDocument()
  })

  it('opens the scenario when a row is clicked', async () => {
    b.getScenarioRunHistory.mockResolvedValue([run()])
    const onSelectScenario = vi.fn()
    render(<ScenarioRunLog workspacePath="/ws" onSelectScenario={onSelectScenario} />)
    fireEvent.click(await screen.findByText('Login flow'))
    expect(onSelectScenario).toHaveBeenCalledWith('/ws/scenarios/login-flow.yaml', 'Login flow')
  })

  it('does not throw on row click without a handler', async () => {
    b.getScenarioRunHistory.mockResolvedValue([run()])
    render(<ScenarioRunLog workspacePath="/ws" />)
    const name = await screen.findByText('Login flow')
    expect(() => fireEvent.click(name)).not.toThrow()
  })

  it('refreshes on the refresh button and when refreshTrigger changes', async () => {
    const { rerender } = render(<ScenarioRunLog workspacePath="/ws" refreshTrigger={0} />)
    await screen.findByText('No runs yet')
    expect(b.getScenarioRunHistory).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByTitle('Refresh'))
    await waitFor(() => expect(b.getScenarioRunHistory).toHaveBeenCalledTimes(2))
    rerender(<ScenarioRunLog workspacePath="/ws" refreshTrigger={1} />)
    await waitFor(() => expect(b.getScenarioRunHistory).toHaveBeenCalledTimes(3))
  })

  describe('expanded run detail', () => {
    async function expand(runs = [run()]) {
      b.getScenarioRunHistory.mockResolvedValue(runs)
      render(<ScenarioRunLog workspacePath="/ws" />)
      await screen.findByText(runs[0].title || 'scenario')
      fireEvent.click(screen.getByTitle('Expand detail'))
    }

    it('shows a loading state, then full step detail from the cached result', async () => {
      let release!: (r: ScenarioResult) => void
      b.loadScenarioResult.mockReturnValue(new Promise(r => { release = r }))
      await expand()
      expect(screen.getByText('Loading…')).toBeInTheDocument()
      release(result([step({ label: 'Full step' })]))
      expect(await screen.findByText('Full step')).toBeInTheDocument()
      expect(b.loadScenarioResult).toHaveBeenCalledWith('/ws/scenarios/login-flow.yaml')
    })

    it('falls back to the summary rows when no cached result exists', async () => {
      b.loadScenarioResult.mockRejectedValue(new Error('missing'))
      await expand()
      expect(await screen.findByText('Profile')).toBeInTheDocument()
      expect(screen.getByTitle('Collapse')).toBeInTheDocument()
    })

    it('shows a notice when the cached result has no steps', async () => {
      b.loadScenarioResult.mockResolvedValue(result([]))
      await expand()
      expect(await screen.findByText('No step detail available.')).toBeInTheDocument()
    })

    it('collapses again and does not refetch on re-expand', async () => {
      b.loadScenarioResult.mockResolvedValue(result([step({ label: 'Full step' })]))
      await expand()
      await screen.findByText('Full step')
      fireEvent.click(screen.getByTitle('Collapse'))
      expect(screen.queryByText('Full step')).not.toBeInTheDocument()
      fireEvent.click(screen.getByTitle('Expand detail'))
      expect(await screen.findByText('Full step')).toBeInTheDocument()
      expect(b.loadScenarioResult).toHaveBeenCalledTimes(1)
    })

    it('renders summary rows with status, duration and error variants', async () => {
      b.loadScenarioResult.mockRejectedValue(new Error('x'))
      await expand([run({
        steps: [
          { label: 'Fail', passed: false, skipped: false, statusCode: 500, durationMs: 1500, error: 'boom' },
          { label: 'Skip', passed: false, skipped: true },
          { label: 'Redirect', passed: true, skipped: false, statusCode: 302, durationMs: 5 },
          { label: 'ClientErr', passed: false, skipped: false, statusCode: 404, durationMs: 5 },
        ],
      })])
      expect(await screen.findByText('Fail')).toBeInTheDocument()
      expect(screen.getByText('500').className).toContain('text-red-400')
      expect(screen.getByText('1.5s')).toBeInTheDocument()
      expect(screen.getByText('boom')).toBeInTheDocument()
      expect(screen.getByText('Skip').className).toContain('italic')
      expect(screen.getByText('302').className).toContain('text-blue-400')
      expect(screen.getByText('404').className).toContain('text-amber-400')
    })

    it('renders full steps: error, skipped note, assertions and extracted variables', async () => {
      b.loadScenarioResult.mockResolvedValue(result([
        step({
          label: 'Broken', passed: false, error: 'connection refused',
          assertions: [
            { type: 'status', passed: true, message: 'status eq 200' },
            { type: 'jsonpath', passed: false, message: '$.id missing' },
          ],
          extracted: { token: 'abc123' },
        }),
        step({ label: 'Later', skipped: true, passed: false, statusCode: 0, durationMs: 0 }),
      ]))
      await expand()
      expect(await screen.findByText('connection refused')).toBeInTheDocument()
      expect(screen.getByText('status eq 200')).toBeInTheDocument()
      expect(screen.getByText('$.id missing').className).toContain('text-red-300')
      expect(screen.getByText('→ token:')).toBeInTheDocument()
      expect(screen.getByText('Skipped — previous step failed.')).toBeInTheDocument()
    })

    it('copies an extracted variable to the clipboard', async () => {
      b.loadScenarioResult.mockResolvedValue(result([step({ extracted: { token: 'abc123' } })]))
      await expand()
      await screen.findByText('abc123')
      const copyBtn = screen.getByText('abc123').parentElement!.querySelector('button')!
      fireEvent.click(copyBtn)
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith('abc123')
    })

    it('toggles response body (pretty-printed JSON) and headers', async () => {
      b.loadScenarioResult.mockResolvedValue(result([
        step({
          responseBody: '{"a":1}',
          responseHeaders: [{ name: 'Content-Type', value: 'application/json' }],
        }),
      ]))
      await expand()
      // the response section is open, so the body is visible straight away
      expect(await screen.findByText(/"a": 1/)).toBeInTheDocument()
      expect(screen.queryByText('Content-Type:')).not.toBeInTheDocument()

      fireEvent.click(screen.getByRole('button', { name: /Headers \(1\)/ }))
      expect(screen.getByText('Content-Type:')).toBeInTheDocument()
      fireEvent.click(screen.getByRole('button', { name: /Headers \(1\)/ }))
      expect(screen.queryByText('Content-Type:')).not.toBeInTheDocument()

      fireEvent.click(screen.getByRole('button', { name: /^Response/ }))
      expect(screen.queryByText(/"a": 1/)).not.toBeInTheDocument()
    })

    it('shows the request as sent above the response', async () => {
      b.loadScenarioResult.mockResolvedValue(result([
        step({
          requestMethod: 'POST', requestUrl: 'http://x/users?v=1',
          requestHeaders: [{ name: 'X-Req', value: 'abc' }], requestBody: '{"id":"u1"}',
          responseBody: '{"ok":true}',
        }),
      ]))
      await expand()
      expect(await screen.findByText('http://x/users?v=1')).toBeInTheDocument()
      fireEvent.click(screen.getAllByRole('button', { name: /Headers \(1\)/ })[0])
      expect(screen.getByText('X-Req:')).toBeInTheDocument()
      expect(screen.getByText(/"id": "u1"/)).toBeInTheDocument()
      expect(screen.getByRole('region', { name: 'Request' })).toBeInTheDocument()
      expect(screen.getByRole('region', { name: 'Response' })).toBeInTheDocument()
    })

    it('shows non-JSON bodies verbatim', async () => {
      b.loadScenarioResult.mockResolvedValue(result([step({ responseBody: 'plain text body' })]))
      await expand()
      expect(await screen.findByText('plain text body')).toBeInTheDocument()
    })

    it('shows no body/headers toggles when the step has neither', async () => {
      b.loadScenarioResult.mockResolvedValue(result([step()]))
      await expand()
      await screen.findByText('Login')
      expect(screen.queryByLabelText('Copy response body')).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /Headers/ })).not.toBeInTheDocument()
    })
  })
})
