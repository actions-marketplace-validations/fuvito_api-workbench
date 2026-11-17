import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ScenarioTemplateGallery } from './ScenarioTemplateGallery'
import type { TemplateInfo } from '@/lib/bridge'

const templates: TemplateInfo[] = [
  { id: 'pagination', title: 'Pagination walk', description: 'Walks every page', category: 'Conformance', tags: ['pages', 'list'], stepCount: 3 },
  { id: 'github', title: 'GitHub smoke', description: 'Hits the public API', category: 'Public APIs', tags: ['rest'], stepCount: 1, note: 'Rate limited' },
  { id: 'misc', title: 'Odd one', description: 'Something else', category: 'Unknown', tags: undefined as never, stepCount: 2 },
]

function setup(over: Partial<React.ComponentProps<typeof ScenarioTemplateGallery>> = {}) {
  const props: React.ComponentProps<typeof ScenarioTemplateGallery> = {
    templates, loading: false, onUse: vi.fn(), onClose: vi.fn(), ...over,
  }
  render(<ScenarioTemplateGallery {...props} />)
  return props
}

describe('ScenarioTemplateGallery', () => {
  it('shows a spinner while loading', () => {
    const { container } = render(<ScenarioTemplateGallery templates={[]} loading onUse={vi.fn()} onClose={vi.fn()} />)
    expect(container.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('renders cards with title, description, tags, step count and notes', () => {
    setup()
    expect(screen.getByText('Pagination walk')).toBeInTheDocument()
    expect(screen.getByText('Walks every page')).toBeInTheDocument()
    expect(screen.getByText('pages')).toBeInTheDocument()
    expect(screen.getByText('3 steps')).toBeInTheDocument()
    expect(screen.getByText('1 step')).toBeInTheDocument()
    expect(screen.getByText('Rate limited')).toBeInTheDocument()
  })

  it('builds category tabs from the templates', () => {
    setup()
    for (const c of ['All', 'Conformance', 'Public APIs', 'Unknown']) {
      expect(screen.getByRole('button', { name: c })).toBeInTheDocument()
    }
  })

  it('filters by category', () => {
    setup()
    fireEvent.click(screen.getByRole('button', { name: 'Public APIs' }))
    expect(screen.getByText('GitHub smoke')).toBeInTheDocument()
    expect(screen.queryByText('Pagination walk')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'All' }))
    expect(screen.getByText('Pagination walk')).toBeInTheDocument()
  })

  it.each([
    ['PAGINATION', 'Pagination walk'],
    ['public api', 'GitHub smoke'],
    ['rest', 'GitHub smoke'],
  ])('searches title, description and tags: %s', (q, expected) => {
    setup()
    fireEvent.change(screen.getByPlaceholderText('Search templates…'), { target: { value: q } })
    expect(screen.getByText(expected)).toBeInTheDocument()
    expect(screen.queryByText('Odd one')).not.toBeInTheDocument()
  })

  it('shows an empty message and clears the search', () => {
    setup()
    const input = screen.getByPlaceholderText('Search templates…')
    fireEvent.change(input, { target: { value: 'zzz' } })
    expect(screen.getByText('No templates match your search.')).toBeInTheDocument()
    fireEvent.click(input.parentElement!.querySelector('button')!)
    expect(screen.getByText('Pagination walk')).toBeInTheDocument()
  })

  it('uses a template with its default title', () => {
    const p = setup()
    fireEvent.click(screen.getAllByRole('button', { name: /Use this template/ })[0])
    fireEvent.click(screen.getByRole('button', { name: 'Create' }))
    expect(p.onUse).toHaveBeenCalledWith('pagination', 'Pagination walk')
  })

  it('uses a template with a custom name and Enter', () => {
    const p = setup()
    fireEvent.click(screen.getAllByRole('button', { name: /Use this template/ })[1])
    const input = screen.getByPlaceholderText('Scenario name…')
    fireEvent.change(input, { target: { value: '  My copy  ' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(p.onUse).toHaveBeenCalledWith('github', 'My copy')
  })

  it('falls back to the title when the name is blank', () => {
    const p = setup()
    fireEvent.click(screen.getAllByRole('button', { name: /Use this template/ })[0])
    fireEvent.change(screen.getByPlaceholderText('Scenario name…'), { target: { value: '   ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create' }))
    expect(p.onUse).toHaveBeenCalledWith('pagination', 'Pagination walk')
  })

  it('cancels naming with Escape and the cancel button, restoring the title', () => {
    const p = setup()
    fireEvent.click(screen.getAllByRole('button', { name: /Use this template/ })[0])
    const input = screen.getByPlaceholderText('Scenario name…')
    fireEvent.change(input, { target: { value: 'edited' } })
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(screen.queryByPlaceholderText('Scenario name…')).not.toBeInTheDocument()

    fireEvent.click(screen.getAllByRole('button', { name: /Use this template/ })[0])
    expect(screen.getByPlaceholderText('Scenario name…')).toHaveValue('Pagination walk')
    fireEvent.click(screen.getByPlaceholderText('Scenario name…').parentElement!.querySelector('button:last-child')!)
    expect(screen.queryByPlaceholderText('Scenario name…')).not.toBeInTheDocument()
    expect(p.onUse).not.toHaveBeenCalled()
  })

  it('closes from the X button and backdrop but not from inside the dialog', () => {
    const p = setup()
    fireEvent.mouseDown(screen.getByText('Scenario templates'))
    expect(p.onClose).not.toHaveBeenCalled()
    fireEvent.mouseDown(screen.getByText('Scenario templates').closest('.fixed')!)
    expect(p.onClose).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
    expect(p.onClose).toHaveBeenCalledTimes(2)
  })
})
