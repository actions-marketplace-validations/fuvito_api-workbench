import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SnippetButton } from './SnippetButton'
import { createBlankTab } from '@/lib/tabState'
import { buildCurl, buildFetch, buildPython } from '@/lib/snippets'

const tab = { ...createBlankTab(), method: 'POST' as const, url: 'https://api.example.com/users', name: 'Create user' }

beforeEach(() => {
  Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.resolve()) } })
})

function open() {
  fireEvent.click(screen.getByTitle('Copy as code snippet'))
}

describe('SnippetButton', () => {
  it('is closed until clicked', () => {
    render(<SnippetButton tab={tab} />)
    expect(screen.queryByText('Code snippet')).not.toBeInTheDocument()
    open()
    expect(screen.getByText('Code snippet')).toBeInTheDocument()
  })

  it('shows curl by default', () => {
    render(<SnippetButton tab={tab} />)
    open()
    expect(document.querySelector('pre')!.textContent).toBe(buildCurl(tab))
  })

  it('switches between formats', () => {
    render(<SnippetButton tab={tab} />)
    open()
    fireEvent.click(screen.getByRole('button', { name: 'JavaScript · fetch' }))
    expect(document.querySelector('pre')!.textContent).toBe(buildFetch(tab))
    fireEvent.click(screen.getByRole('button', { name: 'Python · requests' }))
    expect(document.querySelector('pre')!.textContent).toBe(buildPython(tab))
    fireEvent.click(screen.getByRole('button', { name: 'curl' }))
    expect(document.querySelector('pre')!.textContent).toBe(buildCurl(tab))
  })

  it('highlights tokens for each language', () => {
    render(<SnippetButton tab={tab} />)
    open()
    expect(document.querySelector('pre .text-emerald-400')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'JavaScript · fetch' }))
    expect(document.querySelector('pre .text-blue-400')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Python · requests' }))
    expect(document.querySelector('pre .text-purple-400')).toBeInTheDocument()
  })

  it('copies the snippet and shows a confirmation', () => {
    render(<SnippetButton tab={tab} />)
    open()
    fireEvent.click(screen.getByRole('button', { name: 'Copy' }))
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(buildCurl(tab))
    expect(screen.getByRole('button', { name: 'Copied!' })).toBeInTheDocument()
  })

  it('closes with the X button, Escape and the backdrop but not from inside', () => {
    render(<SnippetButton tab={tab} />)
    open()
    fireEvent.click(screen.getByText('Code snippet'))
    expect(screen.getByText('Code snippet')).toBeInTheDocument()
    fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
    expect(screen.queryByText('Code snippet')).not.toBeInTheDocument()

    open()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByText('Code snippet')).not.toBeInTheDocument()

    open()
    fireEvent.click(screen.getByText('Code snippet').closest('.fixed')!)
    expect(screen.queryByText('Code snippet')).not.toBeInTheDocument()
  })

  it('ignores other keys', () => {
    render(<SnippetButton tab={tab} />)
    open()
    fireEvent.keyDown(document, { key: 'a' })
    expect(screen.getByText('Code snippet')).toBeInTheDocument()
  })
})
