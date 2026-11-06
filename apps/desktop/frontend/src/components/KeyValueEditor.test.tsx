import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { KeyValueEditor } from './KeyValueEditor'
import type { KVPair } from '@/types'

function makePairs(...names: string[]): KVPair[] {
  return names.map((name, i) => ({ id: String(i), name, value: `v${i}`, enabled: true }))
}

describe('KeyValueEditor', () => {
  it('renders + Add row button', () => {
    render(<KeyValueEditor pairs={[]} onChange={() => {}} />)
    expect(screen.getByText('+ Add row')).toBeInTheDocument()
  })

  it('calls onChange when Add row is clicked', () => {
    const onChange = vi.fn()
    render(<KeyValueEditor pairs={[]} onChange={onChange} />)
    fireEvent.click(screen.getByText('+ Add row'))
    expect(onChange).toHaveBeenCalledOnce()
    const [newPairs] = onChange.mock.calls[0]
    expect(newPairs).toHaveLength(1)
    expect(newPairs[0].name).toBe('')
    expect(newPairs[0].enabled).toBe(true)
  })

  it('shows toggle-all checkbox when pairs exist', () => {
    render(<KeyValueEditor pairs={makePairs('a', 'b')} onChange={() => {}} />)
    expect(screen.getByTitle('Toggle all')).toBeInTheDocument()
  })

  it('toggle-all checked when all enabled', () => {
    render(<KeyValueEditor pairs={makePairs('a', 'b')} onChange={() => {}} />)
    const toggleAll = screen.getByTitle('Toggle all') as HTMLInputElement
    expect(toggleAll.checked).toBe(true)
    expect(toggleAll.indeterminate).toBe(false)
  })

  it('toggle-all unchecked when all disabled', () => {
    const pairs = makePairs('a', 'b').map(p => ({ ...p, enabled: false }))
    render(<KeyValueEditor pairs={pairs} onChange={() => {}} />)
    const toggleAll = screen.getByTitle('Toggle all') as HTMLInputElement
    expect(toggleAll.checked).toBe(false)
    expect(toggleAll.indeterminate).toBe(false)
  })

  it('toggle-all indeterminate when mixed', () => {
    const pairs = [
      { id: '0', name: 'a', value: 'v', enabled: true },
      { id: '1', name: 'b', value: 'v', enabled: false },
    ]
    render(<KeyValueEditor pairs={pairs} onChange={() => {}} />)
    const toggleAll = screen.getByTitle('Toggle all') as HTMLInputElement
    expect(toggleAll.indeterminate).toBe(true)
  })

  it('toggle-all disables all when checked', () => {
    const onChange = vi.fn()
    render(<KeyValueEditor pairs={makePairs('a', 'b')} onChange={onChange} />)
    const toggleAll = screen.getByTitle('Toggle all')
    fireEvent.click(toggleAll)
    const [updated] = onChange.mock.calls[0]
    expect(updated.every((p: KVPair) => !p.enabled)).toBe(true)
  })

  it('toggle-all enables all when unchecked', () => {
    const onChange = vi.fn()
    const pairs = makePairs('a', 'b').map(p => ({ ...p, enabled: false }))
    render(<KeyValueEditor pairs={pairs} onChange={onChange} />)
    const toggleAll = screen.getByTitle('Toggle all')
    fireEvent.click(toggleAll)
    const [updated] = onChange.mock.calls[0]
    expect(updated.every((p: KVPair) => p.enabled)).toBe(true)
  })

  it('remove button removes the row', () => {
    const onChange = vi.fn()
    render(
      <KeyValueEditor
        pairs={[{ id: 'x1', name: 'foo', value: 'bar', enabled: true }]}
        onChange={onChange}
      />,
    )
    // The remove button is the only button besides "+ Add row"
    const buttons = screen.getAllByRole('button')
    const removeBtn = buttons.find(b => b.textContent !== '+ Add row')!
    fireEvent.click(removeBtn)
    const [updated] = onChange.mock.calls[0]
    expect(updated).toHaveLength(0)
  })
})
