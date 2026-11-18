import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { StepExchange } from './StepExchange'
import type { StepResult } from '@/lib/bridge'

const base: StepResult = {
  label: 'go', passed: true, skipped: false, statusCode: 201, durationMs: 42, extracted: {}, assertions: [],
}

describe('StepExchange', () => {
  it('renders nothing when the step sent nothing and got nothing', () => {
    const { container } = render(<StepExchange result={{ ...base, statusCode: 0, durationMs: 0 }} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('shows the request as sent and the response in separately coloured sections', () => {
    render(<StepExchange result={{
      ...base,
      requestMethod: 'POST', requestUrl: 'http://x/users', requestHeaders: [{ name: 'Authorization', value: 'Bearer ***' }],
      requestBody: '{"name":"Ada"}',
      responseHeaders: [{ name: 'Content-Type', value: 'application/json; charset=utf-8' }], responseBody: '{"id":7}',
    }} />)
    const req = screen.getByRole('region', { name: 'Request' })
    const res = screen.getByRole('region', { name: 'Response' })
    expect(req.className).toContain('sky')
    expect(res.className).toContain('emerald')
    expect(req).toHaveTextContent('POST')
    expect(req).toHaveTextContent('Authorization:')
    expect(req).toHaveTextContent('Bearer ***')
    expect(req).toHaveTextContent('"name": "Ada"')
    expect(res).toHaveTextContent('201')
    expect(res).toHaveTextContent('42 ms')
    expect(res).toHaveTextContent('application/json')
    expect(res).toHaveTextContent('"id": 7')
  })

  it('keeps response headers closed until asked, request headers open', () => {
    render(<StepExchange result={{
      ...base, requestMethod: 'GET', requestUrl: 'http://x', requestHeaders: [{ name: 'X-A', value: '1' }],
      responseHeaders: [{ name: 'X-B', value: '2' }], responseBody: 'hi',
    }} />)
    expect(screen.getByText('X-A:')).toBeInTheDocument()
    expect(screen.queryByText('X-B:')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /Headers \(1\)/, expanded: false }))
    expect(screen.getByText('X-B:')).toBeInTheDocument()
  })

  it('collapses a section from its header and says so when a body is missing', () => {
    render(<StepExchange result={{ ...base, requestMethod: 'GET', requestUrl: 'http://x' }} />)
    expect(screen.getByText('No body sent.')).toBeInTheDocument()
    expect(screen.getByText('Empty response body.')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /^Request/ }))
    expect(screen.queryByText('No body sent.')).not.toBeInTheDocument()
  })

  it('still shows a response for results saved before requests were recorded', () => {
    render(<StepExchange result={{ ...base, responseBody: 'old' }} />)
    expect(screen.queryByRole('region', { name: 'Request' })).not.toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Response' })).toHaveTextContent('old')
  })
})
