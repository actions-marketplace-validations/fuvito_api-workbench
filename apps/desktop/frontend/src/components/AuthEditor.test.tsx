import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { AuthEditor } from './AuthEditor'
import type { AuthState } from '@/types'

function renderEditor(auth: AuthState) {
  const onChange = vi.fn()
  render(<AuthEditor auth={auth} onChange={onChange} />)
  return onChange
}

describe('AuthEditor type switching', () => {
  it.each([
    ['bearer', { type: 'bearer', token: '' }],
    ['basic', { type: 'basic', username: '', password: '' }],
    ['apikey', { type: 'apikey', key: '', value: '', location: 'header' }],
    ['none', { type: 'none' }],
  ])('switching to %s resets to an empty state of that type', (value, expected) => {
    const start: AuthState = value === 'none' ? { type: 'bearer', token: 'secret' } : { type: 'none' }
    const onChange = renderEditor(start)
    fireEvent.change(screen.getAllByRole('combobox')[0], { target: { value } })
    expect(onChange).toHaveBeenCalledWith(expected)
  })

  it('offers all four auth types', () => {
    renderEditor({ type: 'none' })
    const options = Array.from(screen.getAllByRole('combobox')[0].querySelectorAll('option')).map(o => o.textContent)
    expect(options).toEqual(['None', 'Bearer Token', 'Basic Auth', 'API Key'])
  })

  it('explains that no credentials are configured for "none"', () => {
    renderEditor({ type: 'none' })
    expect(screen.getByText(/No authentication/)).toBeInTheDocument()
    expect(screen.queryByText('Token')).not.toBeInTheDocument()
  })
})

describe('bearer', () => {
  it('edits the token and keeps it masked until revealed', () => {
    const onChange = renderEditor({ type: 'bearer', token: 'abc' })
    const input = screen.getByPlaceholderText('eyJhbGci…') as HTMLInputElement
    expect(input.type).toBe('password')
    fireEvent.change(input, { target: { value: 'xyz' } })
    expect(onChange).toHaveBeenCalledWith({ type: 'bearer', token: 'xyz' })

    fireEvent.click(screen.getByTitle('Show'))
    expect(input.type).toBe('text')
    fireEvent.click(screen.getByTitle('Hide'))
    expect(input.type).toBe('password')
  })
})

describe('basic', () => {
  it('edits username and password independently', () => {
    const onChange = renderEditor({ type: 'basic', username: 'u', password: 'p' })
    fireEvent.change(screen.getByPlaceholderText('username'), { target: { value: 'alice' } })
    expect(onChange).toHaveBeenLastCalledWith({ type: 'basic', username: 'alice', password: 'p' })
    const pw = screen.getByPlaceholderText('••••••••') as HTMLInputElement
    expect(pw.type).toBe('password')
    fireEvent.change(pw, { target: { value: 'new' } })
    expect(onChange).toHaveBeenLastCalledWith({ type: 'basic', username: 'u', password: 'new' })
  })
})

describe('api key', () => {
  const auth: AuthState = { type: 'apikey', key: 'X-Key', value: 'v', location: 'header' }

  it('edits key, value and location', () => {
    const onChange = renderEditor(auth)
    fireEvent.change(screen.getByPlaceholderText('X-API-Key'), { target: { value: 'Api-Key' } })
    expect(onChange).toHaveBeenLastCalledWith({ ...auth, key: 'Api-Key' })
    fireEvent.change(screen.getByPlaceholderText('api-key-value'), { target: { value: 'secret' } })
    expect(onChange).toHaveBeenLastCalledWith({ ...auth, value: 'secret' })
    fireEvent.change(screen.getAllByRole('combobox')[1], { target: { value: 'query' } })
    expect(onChange).toHaveBeenLastCalledWith({ ...auth, location: 'query' })
  })

  it('reflects the current location', () => {
    renderEditor({ ...auth, location: 'query' } as AuthState)
    expect((screen.getAllByRole('combobox')[1] as HTMLSelectElement).value).toBe('query')
  })
})
