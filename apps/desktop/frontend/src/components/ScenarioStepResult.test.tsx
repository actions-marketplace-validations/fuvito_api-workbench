import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { StepDetailPanel } from './ScenarioStepResult'
import type { StepDetail, StepResult } from '@/lib/bridge'

function step(over: Partial<StepDetail> = {}): StepDetail {
  return { label: 'Login', method: 'POST', url: '/login', assertions: [], extractions: [], ...over }
}

function res(over: Partial<StepResult> = {}): StepResult {
  return { label: 'Login', passed: true, skipped: false, statusCode: 200, durationMs: 15, extracted: {}, assertions: [], ...over }
}

beforeEach(() => {
  Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.resolve()) } })
})

describe('StepDetailPanel — request section', () => {
  it('shows method and url in the header', () => {
    render(<StepDetailPanel step={step()} result={null} />)
    expect(screen.getByText('POST')).toBeInTheDocument()
    expect(screen.getByText('/login')).toBeInTheDocument()
  })

  it('collapses and expands the request section', () => {
    render(<StepDetailPanel step={step({ notes: 'secret note' })} result={null} />)
    expect(screen.getByText('secret note')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Request'))
    expect(screen.queryByText('secret note')).not.toBeInTheDocument()
  })

  it('shows disabled warning, request ref and notes', () => {
    render(<StepDetailPanel step={step({ disabled: true, requestRef: 'requests/login.yaml', notes: 'n' })} result={null} />)
    expect(screen.getByText(/This step is disabled/)).toBeInTheDocument()
    expect(screen.getByText('↗ requests/login.yaml')).toBeInTheDocument()
    expect(screen.getByText('n')).toBeInTheDocument()
  })

  it('renders headers, query params and a pretty-printed JSON body', () => {
    render(<StepDetailPanel step={step({
      headers: [{ name: 'X-Id', value: '7' }],
      queryParams: [{ name: 'page', value: '2' }],
      bodyType: 'json',
      bodyContent: '{"a":1}',
    })} result={null} />)
    expect(screen.getByText('X-Id:')).toBeInTheDocument()
    expect(screen.getByText('?page=2')).toBeInTheDocument()
    expect(screen.getByText('Body · json')).toBeInTheDocument()
    expect(screen.getByText(/"a": 1/)).toBeInTheDocument()
  })

  it('shows non-JSON body verbatim and hides the body for type none', () => {
    const { unmount } = render(<StepDetailPanel step={step({ bodyType: 'raw', bodyContent: 'plain' })} result={null} />)
    expect(screen.getByText('plain')).toBeInTheDocument()
    unmount()
    render(<StepDetailPanel step={step({ bodyType: 'none', bodyContent: 'x' })} result={null} />)
    expect(screen.queryByText(/^Body/)).not.toBeInTheDocument()
  })

  it.each([
    [{ type: 'status', operator: 'eq', expected: 200 }, 'status = 200'],
    [{ type: 'status', operator: 'ne', expected: 500 }, 'status ≠ 500'],
    [{ type: 'jsonpath', path: '$.name', operator: 'contains', expected: 'Al' }, '$.name ∋ Al'],
    [{ type: 'header', path: 'Content-Type', operator: 'matches', expected: 'json' }, 'Content-Type ~ json'],
    [{ type: 'jsonpath', path: '$.n', operator: 'gt', expected: 3 }, '$.n > 3'],
    [{ type: 'jsonpath', path: '$.n', operator: 'lt', expected: 3 }, '$.n < 3'],
    [{ type: 'schema', schema: 'User' }, 'schema: User'],
    [{ type: 'schema' }, 'schema'],
    [{ type: 'status', operator: 'weird', expected: 1 }, 'status weird 1'],
    [{ type: 'status' }, 'status ='],
  ])('summarises assertion %j as "%s"', (a, label) => {
    render(<StepDetailPanel step={step({ assertions: [a] })} result={null} />)
    expect(screen.getByText(label)).toBeInTheDocument()
  })

  it('lists extractions', () => {
    render(<StepDetailPanel step={step({ extractions: [{ name: 'token', source: 'jsonpath', path: '$.t' }] })} result={null} />)
    expect(screen.getByText('→ token')).toBeInTheDocument()
  })

  it('shows execution settings only when non-default', () => {
    const { unmount } = render(<StepDetailPanel step={step({ failPolicy: 'stop' })} result={null} />)
    expect(screen.queryByText(/on fail/)).not.toBeInTheDocument()
    unmount()
    render(<StepDetailPanel step={step({ failPolicy: 'continue', timeoutMs: 500, retryMaxAttempts: 3, retryDelayMs: 250 })} result={null} />)
    expect(screen.getByText('on fail: continue')).toBeInTheDocument()
    expect(screen.getByText('⏱ 500 ms')).toBeInTheDocument()
    expect(screen.getByText('↺ 3× · 250 ms delay')).toBeInTheDocument()
  })

  it('defaults the retry delay to 1000 ms', () => {
    render(<StepDetailPanel step={step({ retryMaxAttempts: 2 })} result={null} />)
    expect(screen.getByText('↺ 2× · 1000 ms delay')).toBeInTheDocument()
  })

  it('runs from here only when a handler is provided, without toggling the section', () => {
    const onRunFrom = vi.fn()
    const { unmount } = render(<StepDetailPanel step={step()} result={null} />)
    expect(screen.queryByTitle('Run scenario from this step')).not.toBeInTheDocument()
    unmount()
    render(<StepDetailPanel step={step({ notes: 'still here' })} result={null} onRunFrom={onRunFrom} />)
    fireEvent.click(screen.getByTitle('Run scenario from this step'))
    expect(onRunFrom).toHaveBeenCalledTimes(1)
    expect(screen.getByText('still here')).toBeInTheDocument()
  })

  it('handles a step with no method or url', () => {
    render(<StepDetailPanel step={{ label: 'bare', assertions: [], extractions: [] }} result={null} />)
    expect(screen.getByText('Request')).toBeInTheDocument()
  })
})

describe('StepDetailPanel — response and result', () => {
  it('shows no response or result sections without a result', () => {
    render(<StepDetailPanel step={step()} result={null} />)
    expect(screen.queryByText('Response')).not.toBeInTheDocument()
    expect(screen.queryByText('Result')).not.toBeInTheDocument()
  })

  it('shows a collapsed response with content type, headers and pretty body', () => {
    render(<StepDetailPanel step={step()} result={res({
      responseBody: '{"ok":true}',
      responseHeaders: [{ name: 'Content-Type', value: 'application/json; charset=utf-8' }],
    })} />)
    expect(screen.getByText('application/json')).toBeInTheDocument()
    expect(screen.queryByText(/"ok": true/)).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Response'))
    expect(screen.getByText(/"ok": true/)).toBeInTheDocument()
    expect(screen.getByText('Content-Type:')).toBeInTheDocument()
  })

  it('omits the content-type badge and header block when absent', () => {
    render(<StepDetailPanel step={step()} result={res({ responseBody: 'text' })} />)
    fireEvent.click(screen.getByText('Response'))
    expect(screen.getByText('text')).toBeInTheDocument()
    expect(screen.queryByText('Headers')).not.toBeInTheDocument()
  })

  it.each([
    [res(), 'PASSED'],
    [res({ passed: false }), 'FAILED'],
    [res({ passed: false, skipped: true }), 'SKIPPED'],
  ])('badges the result as %#', (r, label) => {
    render(<StepDetailPanel step={step()} result={r} />)
    expect(screen.getByText(label)).toBeInTheDocument()
  })

  it.each([
    [200, 'text-emerald-400'],
    [302, 'text-blue-400'],
    [404, 'text-amber-400'],
    [503, 'text-red-400'],
  ])('colours status %i', (code, cls) => {
    render(<StepDetailPanel step={step()} result={res({ statusCode: code })} />)
    expect(screen.getByText(String(code)).className).toContain(cls)
  })

  it('hides status and duration when zero', () => {
    render(<StepDetailPanel step={step()} result={res({ statusCode: 0, durationMs: 0 })} />)
    expect(screen.queryByText(/ ms$/)).not.toBeInTheDocument()
  })

  it('shows the error and skipped explanation', () => {
    const { unmount } = render(<StepDetailPanel step={step()} result={res({ passed: false, error: 'timeout' })} />)
    expect(screen.getByText('timeout')).toBeInTheDocument()
    unmount()
    render(<StepDetailPanel step={step()} result={res({ passed: false, skipped: true })} />)
    expect(screen.getByText('Skipped — previous step failed.')).toBeInTheDocument()
  })

  it('labels assertion results from the step definition and explains failures', () => {
    render(<StepDetailPanel
      step={step({ assertions: [{ type: 'status', operator: 'eq', expected: 200 }, { type: 'status', operator: 'eq', expected: 201 }] })}
      result={res({
        passed: false,
        assertions: [
          { type: 'status', passed: true, message: 'ok' },
          { type: 'status', passed: false, message: 'expected 201, got 200' },
        ],
      })}
    />)
    expect(screen.getAllByText('status = 200').length).toBeGreaterThan(0)
    expect(screen.getByText('expected 201, got 200')).toBeInTheDocument()
    expect(screen.queryByText('ok')).not.toBeInTheDocument()
  })

  it('falls back to the message when there is no matching definition', () => {
    render(<StepDetailPanel step={step()} result={res({ assertions: [{ type: 'status', passed: true, message: 'orphan message' }] })} />)
    expect(screen.getByText('orphan message')).toBeInTheDocument()
  })

  it('does not repeat a failure message identical to its label', () => {
    render(<StepDetailPanel step={step()} result={res({ passed: false, assertions: [{ type: 'status', passed: false, message: 'same' }] })} />)
    expect(screen.getAllByText('same')).toHaveLength(1)
  })

  it('lists extracted values and copies them', () => {
    render(<StepDetailPanel step={step()} result={res({ extracted: { token: 'abc' } })} />)
    expect(screen.getByText('→ token:')).toBeInTheDocument()
    fireEvent.click(screen.getByTitle('Copy value of token'))
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('abc')
  })

  it('tolerates missing assertions/extracted maps', () => {
    render(<StepDetailPanel step={step()} result={res({ assertions: undefined as never, extracted: undefined as never })} />)
    expect(screen.getByText('PASSED')).toBeInTheDocument()
  })
})
