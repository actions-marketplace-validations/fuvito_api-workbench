import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QualityPanel } from './QualityPanel'
import type { QualityReport } from '@/lib/bridge'

const empty: QualityReport = { neverExercised: [], happyPathOnly: [], securedNeverAuthed: [], errorsNeverTriggered: [] }
const full: QualityReport = {
  neverExercised: ['DELETE /users/{id}'],
  happyPathOnly: ['GET /users', 'POST /users'],
  securedNeverAuthed: ['GET /admin'],
  errorsNeverTriggered: [{ operation: 'GET /users', missing: [404, 500] }],
}

describe('QualityPanel — full view', () => {
  it('celebrates when there is nothing to report', () => {
    render(<QualityPanel quality={empty} />)
    expect(screen.getByText('Quality signals clear')).toBeInTheDocument()
  })

  it('tolerates missing arrays from the backend', () => {
    render(<QualityPanel quality={{} as QualityReport} />)
    expect(screen.getByText('Quality signals clear')).toBeInTheDocument()
  })

  it('counts gaps, excluding never-exercised operations', () => {
    render(<QualityPanel quality={full} />)
    expect(screen.getByText('4 gaps')).toBeInTheDocument()
  })

  it('uses the singular for one gap', () => {
    render(<QualityPanel quality={{ ...empty, happyPathOnly: ['GET /a'] }} />)
    expect(screen.getByText('1 gap')).toBeInTheDocument()
  })

  it('shows no gap counter when only never-exercised operations exist', () => {
    render(<QualityPanel quality={{ ...empty, neverExercised: ['GET /a'] }} />)
    expect(screen.queryByText(/gap/)).not.toBeInTheDocument()
    expect(screen.getByText('Never exercised')).toBeInTheDocument()
  })

  it('opens the three issue sections by default and shows their operations', () => {
    render(<QualityPanel quality={full} />)
    expect(screen.getByText('Happy-path only')).toBeInTheDocument()
    expect(screen.getByText('POST')).toBeInTheDocument()
    expect(screen.getByText('/admin')).toBeInTheDocument()
    expect(screen.getByText('Never triggered:')).toBeInTheDocument()
    expect(screen.getByText('404')).toBeInTheDocument()
    expect(screen.getByText('500')).toBeInTheDocument()
  })

  it('keeps "Never exercised" collapsed until clicked, and toggles', () => {
    render(<QualityPanel quality={full} />)
    expect(screen.queryByText('/users/{id}')).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Never exercised'))
    expect(screen.getByText('/users/{id}')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Never exercised'))
    expect(screen.queryByText('/users/{id}')).not.toBeInTheDocument()
  })

  it('collapses an open section when its header is clicked', () => {
    render(<QualityPanel quality={full} />)
    fireEvent.click(screen.getByText('Secured — never tested with auth'))
    expect(screen.queryByText('/admin')).not.toBeInTheDocument()
  })

  it('omits sections that have no entries', () => {
    render(<QualityPanel quality={{ ...empty, happyPathOnly: ['GET /a'] }} />)
    expect(screen.queryByText('Never exercised')).not.toBeInTheDocument()
    expect(screen.queryByText('Error responses never triggered')).not.toBeInTheDocument()
  })

  it('renders operations without a path or with an unknown method', () => {
    render(<QualityPanel quality={{ ...empty, happyPathOnly: ['WEIRD', 'TRACE /x'] }} />)
    expect(screen.getByText('WEIRD')).toBeInTheDocument()
    expect(screen.getByText('TRACE')).toBeInTheDocument()
  })
})

describe('QualityPanel — compact', () => {
  it('renders nothing when clear', () => {
    const { container } = render(<QualityPanel quality={empty} compact />)
    expect(container).toBeEmptyDOMElement()
  })

  it('summarises every kind of gap on one line', () => {
    render(<QualityPanel quality={full} compact />)
    expect(screen.getByText('Quality gaps: 1 never run · 2 happy-path only · 1 auth gap · 1 error gap')).toBeInTheDocument()
  })

  it('lists only the gaps that exist', () => {
    render(<QualityPanel quality={{ ...empty, securedNeverAuthed: ['GET /a'] }} compact />)
    expect(screen.getByText('Quality gaps: 1 auth gap')).toBeInTheDocument()
  })
})
