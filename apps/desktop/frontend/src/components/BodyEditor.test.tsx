import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { BodyEditor } from './BodyEditor'
import type { BodyState } from '@/types'

const none: BodyState = { type: 'none', content: '' }
const json: BodyState = { type: 'json', content: '{"x":1}' }
const formUrlEncoded: BodyState = { type: 'form-urlencoded', content: 'name=alice&role=admin' }
const multipart: BodyState = { type: 'multipart', content: 'file=doc.pdf' }

describe('BodyEditor', () => {
  it('shows "no body" message for none type', () => {
    render(<BodyEditor body={none} onChange={() => {}} />)
    expect(screen.getByText(/no body/i)).toBeInTheDocument()
  })

  it('shows textarea for json type', () => {
    render(<BodyEditor body={json} onChange={() => {}} />)
    expect(screen.getByRole('textbox')).toBeInTheDocument()
  })

  it('shows Format button for non-empty json', () => {
    render(<BodyEditor body={json} onChange={() => {}} />)
    expect(screen.getByText('Format')).toBeInTheDocument()
  })

  it('does not show Format button for empty json', () => {
    render(<BodyEditor body={{ type: 'json', content: '' }} onChange={() => {}} />)
    expect(screen.queryByText('Format')).toBeNull()
  })

  it('Format button pretty-prints JSON', () => {
    const onChange = vi.fn()
    render(<BodyEditor body={{ type: 'json', content: '{"a":1,"b":2}' }} onChange={onChange} />)
    fireEvent.click(screen.getByText('Format'))
    const [updated] = onChange.mock.calls[0]
    expect(updated.content).toBe('{\n  "a": 1,\n  "b": 2\n}')
  })

  it('shows KeyValueEditor for form-urlencoded type', () => {
    render(<BodyEditor body={formUrlEncoded} onChange={() => {}} />)
    // KeyValueEditor renders "+ Add row" button
    expect(screen.getByText('+ Add row')).toBeInTheDocument()
    // The two pairs from content should be pre-populated
    expect(screen.getByDisplayValue('alice')).toBeInTheDocument()
    expect(screen.getByDisplayValue('admin')).toBeInTheDocument()
  })

  it('shows KeyValueEditor for multipart type', () => {
    render(<BodyEditor body={multipart} onChange={() => {}} />)
    expect(screen.getByText('+ Add row')).toBeInTheDocument()
  })

  it('shows textarea for raw type', () => {
    render(<BodyEditor body={{ type: 'raw', content: 'hello' }} onChange={() => {}} />)
    expect(screen.getByRole('textbox')).toBeInTheDocument()
  })

  it('switching type calls onChange preserving content', () => {
    const onChange = vi.fn()
    render(<BodyEditor body={json} onChange={onChange} />)
    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'raw' } })
    const [updated] = onChange.mock.calls[0]
    expect(updated.type).toBe('raw')
    expect(updated.content).toBe(json.content)
  })

  it('switching to none clears content', () => {
    const onChange = vi.fn()
    render(<BodyEditor body={json} onChange={onChange} />)
    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'none' } })
    const [updated] = onChange.mock.calls[0]
    expect(updated.type).toBe('none')
    expect(updated.content).toBe('')
  })
})
