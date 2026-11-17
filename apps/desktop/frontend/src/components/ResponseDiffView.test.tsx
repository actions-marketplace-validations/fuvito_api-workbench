import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { DiffView } from './ResponseDiffView'

describe('DiffView', () => {
  it('reports identical bodies', () => {
    render(<DiffView prev={'a\nb'} curr={'a\nb'} />)
    expect(screen.getByText('Bodies are identical')).toBeInTheDocument()
    expect(screen.queryByText(/added/)).not.toBeInTheDocument()
  })

  it('counts added and removed lines', () => {
    render(<DiffView prev={'a\nb\nc'} curr={'a\nx\nc\nd'} />)
    expect(screen.getByText('+2 added')).toBeInTheDocument()
    expect(screen.getByText('−1 removed')).toBeInTheDocument()
  })

  it('marks lines with +, − or a blank gutter', () => {
    const { container } = render(<DiffView prev={'keep\ngone'} curr={'keep\nnew'} />)
    const rows = Array.from(container.querySelectorAll('.whitespace-pre')).map(r => r.textContent)
    expect(rows).toEqual([' keep', '−gone', '+new'])
  })

  it('handles an empty previous body (everything added)', () => {
    render(<DiffView prev="" curr={'a\nb'} />)
    expect(screen.getByText('+2 added')).toBeInTheDocument()
    expect(screen.getByText('−1 removed')).toBeInTheDocument()
  })

  it('handles pure deletion', () => {
    render(<DiffView prev={'a\nb'} curr={'a'} />)
    expect(screen.getByText('−1 removed')).toBeInTheDocument()
    expect(screen.queryByText(/added/)).not.toBeInTheDocument()
  })

  it('refuses to diff very large bodies', () => {
    const big = Array.from({ length: 2001 }, (_, i) => `line ${i}`).join('\n')
    render(<DiffView prev={big} curr={big} />)
    expect(screen.getByText(/too large to diff/)).toBeInTheDocument()
  })
})
