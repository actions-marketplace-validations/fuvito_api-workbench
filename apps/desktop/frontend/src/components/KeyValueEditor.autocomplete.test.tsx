import { describe, it, expect, vi, afterEach } from 'vitest'
import { useState } from 'react'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { KeyValueEditor } from './KeyValueEditor'
import type { KVPair } from '@/types'

function Harness({ envVarNames, initial = '', onChange }: { envVarNames?: string[]; initial?: string; onChange?: (p: KVPair[]) => void }) {
  const [pairs, setPairs] = useState<KVPair[]>([{ id: '1', name: 'X-Token', value: initial, enabled: true }])
  return <KeyValueEditor pairs={pairs} onChange={p => { setPairs(p); onChange?.(p) }} envVarNames={envVarNames} />
}

const names = ['baseUrl', 'apiKey', 'authToken']

function valueInput() {
  return screen.getAllByRole('textbox')[1] as HTMLInputElement
}

function type(value: string) {
  fireEvent.change(valueInput(), { target: { value } })
}

afterEach(() => vi.useRealTimers())

describe('KeyValueEditor — variable autocomplete', () => {
  it('suggests every variable after typing {{', () => {
    render(<Harness envVarNames={names} />)
    type('{{')
    for (const n of names) expect(screen.getByRole('button', { name: n })).toBeInTheDocument()
  })

  it('filters suggestions case-insensitively', () => {
    render(<Harness envVarNames={names} />)
    type('{{AUTH')
    expect(screen.getByRole('button', { name: 'authToken' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'baseUrl' })).not.toBeInTheDocument()
  })

  it('shows nothing when there are no matches, no variables, or no open braces', () => {
    const { unmount } = render(<Harness envVarNames={names} />)
    type('{{zzz')
    expect(screen.queryByRole('button', { name: 'baseUrl' })).not.toBeInTheDocument()
    unmount()
    render(<Harness />)
    type('{{')
    expect(screen.queryByRole('list')).not.toBeInTheDocument()
  })

  it('does not suggest once the braces are closed', () => {
    render(<Harness envVarNames={names} />)
    type('{{baseUrl}}')
    expect(screen.queryByRole('button', { name: 'baseUrl' })).not.toBeInTheDocument()
  })

  it('inserts a clicked suggestion and closes the list', () => {
    const onChange = vi.fn()
    render(<Harness envVarNames={names} onChange={onChange} />)
    type('Bearer {{au')
    fireEvent.mouseDown(screen.getByRole('button', { name: 'authToken' }))
    expect(valueInput().value).toBe('Bearer {{authToken}}')
    expect(screen.queryByRole('button', { name: 'authToken' })).not.toBeInTheDocument()
  })

  it('inserts the highlighted suggestion with Enter and Tab', () => {
    render(<Harness envVarNames={names} />)
    type('{{')
    fireEvent.keyDown(valueInput(), { key: 'Enter' })
    expect(valueInput().value).toBe('{{baseUrl}}')
    type('{{')
    fireEvent.keyDown(valueInput(), { key: 'Tab' })
    expect(valueInput().value).toBe('{{baseUrl}}')
  })

  it('moves the highlight with ArrowDown/ArrowUp within bounds', () => {
    render(<Harness envVarNames={names} />)
    type('{{')
    fireEvent.keyDown(valueInput(), { key: 'ArrowDown' })
    fireEvent.keyDown(valueInput(), { key: 'ArrowDown' })
    fireEvent.keyDown(valueInput(), { key: 'ArrowDown' })
    fireEvent.keyDown(valueInput(), { key: 'Enter' })
    expect(valueInput().value).toBe('{{authToken}}')
    type('{{')
    fireEvent.keyDown(valueInput(), { key: 'ArrowUp' })
    fireEvent.keyDown(valueInput(), { key: 'Enter' })
    expect(valueInput().value).toBe('{{baseUrl}}')
  })

  it('closes on Escape without changing the value', () => {
    render(<Harness envVarNames={names} />)
    type('{{')
    fireEvent.keyDown(valueInput(), { key: 'Escape' })
    expect(screen.queryByRole('button', { name: 'baseUrl' })).not.toBeInTheDocument()
    expect(valueInput().value).toBe('{{')
  })

  it('ignores other keys while suggestions are open', () => {
    render(<Harness envVarNames={names} />)
    type('{{')
    fireEvent.keyDown(valueInput(), { key: 'a' })
    expect(screen.getByRole('button', { name: 'baseUrl' })).toBeInTheDocument()
  })

  it('does not hijack Enter when no suggestions are showing', () => {
    render(<Harness envVarNames={names} initial="plain" />)
    fireEvent.keyDown(valueInput(), { key: 'Enter' })
    expect(valueInput().value).toBe('plain')
  })

  it('reopens on click when the caret sits inside an open {{', () => {
    render(<Harness envVarNames={names} initial="{{ap" />)
    valueInput().setSelectionRange(4, 4)
    fireEvent.click(valueInput())
    expect(screen.getByRole('button', { name: 'apiKey' })).toBeInTheDocument()
  })

  it('closes shortly after blur', () => {
    vi.useFakeTimers()
    render(<Harness envVarNames={names} />)
    type('{{')
    fireEvent.blur(valueInput())
    act(() => { vi.advanceTimersByTime(200) })
    expect(screen.queryByRole('button', { name: 'baseUrl' })).not.toBeInTheDocument()
  })
})

describe('KeyValueEditor — dynamic function autocomplete', () => {
  afterEach(async () => (await import('@/lib/dynamicFunctions')).setDynamicFunctions([]))

  it('lists functions with their description and inserts a working example', async () => {
    const { setDynamicFunctions } = await import('@/lib/dynamicFunctions')
    setDynamicFunctions([{ name: 'randomInt', usage: '$randomInt [min max]', description: 'Random whole number', example: '$randomInt 1 100' }])
    const seen: KVPair[][] = []
    render(<Harness envVarNames={names} onChange={p => seen.push(p)} />)
    type('{{$ran')
    const option = screen.getByRole('button', { name: /\$randomInt/ })
    expect(option).toHaveTextContent('Random whole number')
    fireEvent.mouseDown(option)
    expect(seen[seen.length - 1][0].value).toBe('{{$randomInt 1 100}}')
  })
})
