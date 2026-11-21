import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SpecConformance } from './SpecConformance'
import type { StepResult } from '@/lib/bridge'

const step = (label: string, over: Partial<StepResult> = {}): StepResult => ({
  label, passed: true, skipped: false, statusCode: 200, durationMs: 1, extracted: {}, assertions: [], ...over,
})

describe('SpecConformance', () => {
  it('renders nothing without a spec', () => {
    const { container } = render(<SpecConformance steps={[step('a')]} specFile="" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('says all checked responses match', () => {
    render(<SpecConformance steps={[step('a', { specState: 'matched' }), step('b', { specState: 'matched' })]} specFile="specs/openapi.yaml" />)
    expect(screen.getByTestId('spec-conformance-summary')).toHaveTextContent('2 of 2 responses match openapi.yaml')
  })

  it('lists violations under their step', () => {
    render(<SpecConformance steps={[
      step('ok', { specState: 'matched' }),
      step('bad', { specState: 'violations', passed: false, specViolations: ['price: expected number'] }),
    ]} specFile="openapi.yaml" />)
    expect(screen.getByTestId('spec-conformance-summary')).toHaveTextContent('1 of 2 match openapi.yaml, 1 step has violations')
    expect(screen.getByText('bad')).toBeInTheDocument()
    expect(screen.getByText('price: expected number')).toBeInTheDocument()
  })

  it('explains a violation like the Check Conformance page when the step carries findings', () => {
    render(<SpecConformance steps={[
      step('bad', {
        specState: 'violations', passed: false, specViolations: ['price: expected number'],
        specFindings: [{
          kind: 'wrong-type', where: 'price', spec: 'GET /items/{id} → response 200 → body schema', summary: '"price" is text, the spec says number',
          expected: 'number', got: '"9.99"', why: 'The server sent text where the spec declares a number.', fix: 'Send a number from the server.', detail: 'Error at "/price"',
        }],
      }),
    ]} specFile="openapi.yaml" />)
    expect(screen.getByText('"price" is text, the spec says number')).toBeInTheDocument()
    expect(screen.getByText('The server sent text where the spec declares a number.')).toBeInTheDocument()
    expect(screen.getByText('Send a number from the server.')).toBeInTheDocument()
    expect(screen.getByTestId('finding-spec')).toHaveTextContent('GET /items/{id}')
    // The raw validator line is not repeated as a bullet; it stays under Technical detail.
    expect(screen.queryByText('price: expected number')).not.toBeInTheDocument()
  })

  it('shows unchecked steps with the reason and does not count them as matches', () => {
    render(<SpecConformance steps={[
      step('a', { specState: 'matched' }),
      step('b', { specState: 'not-checked', specReason: 'no matching operation in the spec' }),
    ]} specFile="openapi.yaml" />)
    expect(screen.getByTestId('spec-conformance-summary')).toHaveTextContent('1 of 1 responses match')
    expect(screen.getByText(/Not checked: b \(no matching operation in the spec\)/)).toBeInTheDocument()
  })

  it('says nothing was checked when every step is unchecked', () => {
    render(<SpecConformance steps={[step('a', { specState: 'not-checked', specReason: 'x' })]} specFile="s.yaml" />)
    expect(screen.getByTestId('spec-conformance-summary')).toHaveTextContent('No responses were checked')
  })
})
