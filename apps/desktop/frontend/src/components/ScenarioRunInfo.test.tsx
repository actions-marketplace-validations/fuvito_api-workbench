import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ScenarioRunInfo } from './ScenarioRunInfo'
import type { ScenarioResult } from '@/lib/bridge'

const base: ScenarioResult = { title: 'S', passed: true, durationMs: 1, steps: [] }

describe('ScenarioRunInfo', () => {
  it('renders nothing for a static run', () => {
    const { container } = render(<ScenarioRunInfo result={base} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('shows the seed with the replay command and the generated variables', () => {
    render(<ScenarioRunInfo result={{ ...base, seed: 4242, variables: [{ name: 'email', value: 'u@example.com' }, { name: 'tok', value: '***', secret: true }] }} />)
    expect(screen.getByText('4242')).toBeInTheDocument()
    expect(screen.getByText(/clrsig run --seed 4242/)).toBeInTheDocument()
    expect(screen.getByText('u@example.com')).toBeInTheDocument()
    expect(screen.getByText('***')).toBeInTheDocument()
  })

  it('shows a setup error', () => {
    render(<ScenarioRunInfo result={{ ...base, passed: false, error: 'variable x: unknown function $nope' }} />)
    expect(screen.getByRole('alert')).toHaveTextContent('Scenario could not start')
    expect(screen.getByRole('alert')).toHaveTextContent('unknown function $nope')
  })
})
