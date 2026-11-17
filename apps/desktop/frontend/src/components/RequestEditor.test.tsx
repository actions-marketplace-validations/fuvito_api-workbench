import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useState } from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { RequestEditor } from './RequestEditor'
import type { RequestEditorProps } from './RequestEditor'

beforeEach(() => {
  globalThis.requestAnimationFrame = ((cb: FrameRequestCallback) => { cb(0); return 0 }) as never
})

function setup(over: Partial<RequestEditorProps> = {}) {
  const h = {
    onMethodChange: vi.fn(), onUrlChange: vi.fn(), onUrlPaste: vi.fn(), onQueryParamsChange: vi.fn(),
    onHeadersChange: vi.fn(), onBodyChange: vi.fn(), onAuthChange: vi.fn(), onTimeoutChange: vi.fn(),
    onSend: vi.fn(), onCancel: vi.fn(), onCreateVariable: vi.fn(),
  }
  const props: RequestEditorProps = {
    method: 'GET', url: 'https://x.test', loading: false, queryParams: [], headers: [],
    body: { type: 'none', content: '' }, auth: { type: 'none' }, timeoutMs: 0, ...h, ...over,
  }
  const view = render(<RequestEditor {...props} />)
  return { ...h, view, urlInput: () => screen.getByPlaceholderText('https://api.example.com/endpoint') as HTMLInputElement }
}

const base: RequestEditorProps = {
  method: 'GET', url: '', loading: false, queryParams: [], headers: [],
  body: { type: 'none', content: '' }, auth: { type: 'none' }, timeoutMs: 0,
  onMethodChange: vi.fn(), onUrlChange: vi.fn(), onUrlPaste: vi.fn(), onQueryParamsChange: vi.fn(),
  onHeadersChange: vi.fn(), onBodyChange: vi.fn(), onAuthChange: vi.fn(), onTimeoutChange: vi.fn(), onSend: vi.fn(),
}

const kv = (name: string, enabled = true) => ({ id: name, name, value: 'v', enabled })

describe('URL bar', () => {
  it('reports method and url changes', () => {
    const h = setup()
    fireEvent.change(screen.getAllByRole('combobox')[0], { target: { value: 'PATCH' } })
    expect(h.onMethodChange).toHaveBeenCalledWith('PATCH')
    fireEvent.change(h.urlInput(), { target: { value: 'https://y.test' } })
    expect(h.onUrlChange).toHaveBeenCalledWith('https://y.test')
  })

  it('sends on Enter and on the Send button, but not when loading or empty', () => {
    const h = setup()
    fireEvent.keyDown(h.urlInput(), { key: 'Enter' })
    fireEvent.click(screen.getByText('Send'))
    expect(h.onSend).toHaveBeenCalledTimes(2)
  })

  it('disables Send for a blank URL', () => {
    setup({ url: '   ' })
    expect(screen.getByText('Send')).toBeDisabled()
  })

  it('ignores Enter while a request is in flight and shows Cancel instead', () => {
    const h = setup({ loading: true })
    fireEvent.keyDown(h.urlInput(), { key: 'Enter' })
    expect(h.onSend).not.toHaveBeenCalled()
    expect(screen.queryByText('Send')).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Cancel'))
    expect(h.onCancel).toHaveBeenCalled()
  })

  it('treats the timeout field as optional', () => {
    const h = setup()
    const timeout = screen.getByPlaceholderText('default') as HTMLInputElement
    expect(timeout.value).toBe('')
    fireEvent.change(timeout, { target: { value: '5000' } })
    expect(h.onTimeoutChange).toHaveBeenLastCalledWith(5000)
  })

  it('shows and clears a configured timeout', () => {
    const h = setup({ timeoutMs: 2500 })
    const timeout = screen.getByDisplayValue('2500')
    fireEvent.change(timeout, { target: { value: '' } })
    expect(h.onTimeoutChange).toHaveBeenLastCalledWith(0)
  })

  describe('paste', () => {
    const paste = (el: HTMLElement, text: string) => {
      const ev = new Event('paste', { bubbles: true, cancelable: true }) as Event & { clipboardData: unknown }
      ev.clipboardData = { getData: () => text }
      el.dispatchEvent(ev)
      return ev
    }

    it('hands curl commands and URLs with query strings to the parent', () => {
      const h = setup()
      expect(paste(h.urlInput(), '  curl https://x.test -H "A: b"  ').defaultPrevented).toBe(true)
      expect(h.onUrlPaste).toHaveBeenLastCalledWith('curl https://x.test -H "A: b"')
      expect(paste(h.urlInput(), 'https://x.test/a?b=1').defaultPrevented).toBe(true)
      expect(h.onUrlPaste).toHaveBeenLastCalledWith('https://x.test/a?b=1')
    })

    it('leaves ordinary pastes alone', () => {
      const h = setup()
      expect(paste(h.urlInput(), 'https://x.test/plain').defaultPrevented).toBe(false)
      expect(h.onUrlPaste).not.toHaveBeenCalled()
    })
  })
})

describe('variable highlighting and resolution bar', () => {
  it('renders a highlight overlay only when the URL has variables', () => {
    const plain = setup()
    expect(plain.view.container.querySelector('[aria-hidden]')).toBeNull()
    plain.view.unmount()
    const withVars = setup({ url: '{{base}}/users/{{id}}' })
    const overlay = withVars.view.container.querySelector('[aria-hidden]') as HTMLElement
    expect(overlay.textContent).toBe('{{base}}/users/{{id}}')
  })

  it('shows resolved values, empty values and missing variables', () => {
    setup({ url: '{{base}}/{{empty}}/{{missing}}', envVarMap: { base: 'http://h', empty: '' }, onCreateVariable: undefined })
    expect(screen.getByTitle('http://h').textContent).toBe('base = http://h')
    expect(screen.getByTitle('(empty string)').textContent).toBe('empty = empty')
    expect(screen.getByTitle('Variable not found in active environment').textContent).toBe('⚠ missing')
  })

  it('offers to create a missing variable when possible', () => {
    const h = setup({ url: '{{missing}}', envVarMap: {} })
    fireEvent.click(screen.getByText('⚠ missing'))
    expect(h.onCreateVariable).toHaveBeenCalledWith('missing')
  })

  it('lists each variable once and hides the bar without an environment or variables', () => {
    setup({ url: '{{a}}/{{a}}', envVarMap: { a: '1' } })
    expect(screen.getAllByTitle('1')).toHaveLength(1)
  })

  it('does not render the bar when there is no env map', () => {
    setup({ url: '{{a}}' })
    expect(screen.queryByTitle(/Variable not found/)).not.toBeInTheDocument()
  })
})

describe('variable autocomplete', () => {
  const env = { baseUrl: 'x', apiKey: 'y', token: 'z' }

  it('suggests variables after "{{" and filters as you type', () => {
    const h = setup({ url: '', envVarMap: env })
    fireEvent.change(h.urlInput(), { target: { value: '{{' } })
    expect(screen.getAllByRole('button').map(b => b.textContent)).toEqual(expect.arrayContaining(['baseUrl', 'apiKey', 'token']))
    fireEvent.change(h.urlInput(), { target: { value: '{{api' } })
    const names = screen.getAllByRole('listitem').map(li => li.textContent)
    expect(names).toEqual(['apiKey'])
  })

  it('inserts the chosen variable, keeping the surrounding text', () => {
    const onUrlChange = vi.fn()
    function Harness() {
      const [url, setUrl] = useState('')
      return <RequestEditor {...base} url={url} envVarMap={env} onUrlChange={v => { onUrlChange(v); setUrl(v) }} />
    }
    render(<Harness />)
    const input = screen.getByPlaceholderText('https://api.example.com/endpoint')
    fireEvent.change(input, { target: { value: 'https://{{to' } })
    fireEvent.mouseDown(screen.getByText('token'))
    expect(onUrlChange).toHaveBeenLastCalledWith('https://{{token}}')
    expect(screen.queryByRole('listitem')).not.toBeInTheDocument()
  })

  it('does not suggest once the variable is closed', () => {
    const h = setup({ url: '', envVarMap: env })
    fireEvent.change(h.urlInput(), { target: { value: '{{token}}' } })
    expect(screen.queryByRole('listitem')).not.toBeInTheDocument()
  })

  it('does not suggest without environment variables', () => {
    const h = setup({ url: '', envVarMap: {} })
    fireEvent.change(h.urlInput(), { target: { value: '{{' } })
    expect(screen.queryByRole('listitem')).not.toBeInTheDocument()
  })

  it('navigates with arrows and accepts with Enter or Tab instead of sending', () => {
    const h = setup({ url: '', envVarMap: env })
    fireEvent.change(h.urlInput(), { target: { value: '{{' } })
    fireEvent.keyDown(h.urlInput(), { key: 'ArrowDown' })
    fireEvent.keyDown(h.urlInput(), { key: 'ArrowDown' })
    fireEvent.keyDown(h.urlInput(), { key: 'ArrowUp' })
    fireEvent.keyDown(h.urlInput(), { key: 'Enter' })
    expect(h.onUrlChange).toHaveBeenLastCalledWith('{{apiKey}}')
    expect(h.onSend).not.toHaveBeenCalled()

    fireEvent.change(h.urlInput(), { target: { value: '{{' } })
    fireEvent.keyDown(h.urlInput(), { key: 'Tab' })
    expect(h.onUrlChange).toHaveBeenLastCalledWith('{{baseUrl}}')
  })

  it('Escape dismisses suggestions', () => {
    const h = setup({ url: '', envVarMap: env })
    fireEvent.change(h.urlInput(), { target: { value: '{{' } })
    fireEvent.keyDown(h.urlInput(), { key: 'Escape' })
    expect(screen.queryByRole('listitem')).not.toBeInTheDocument()
  })

  it('Enter sends normally when no suggestions are showing', () => {
    const h = setup({ url: 'https://x.test', envVarMap: env })
    fireEvent.keyDown(h.urlInput(), { key: 'Enter' })
    expect(h.onSend).toHaveBeenCalled()
  })
})

describe('tabs', () => {
  it('starts on Params and shows badges for active entries only', () => {
    setup({
      queryParams: [kv('a'), kv('b', false), kv('')],
      headers: [kv('H')],
      body: { type: 'json', content: '' },
      auth: { type: 'bearer', token: 't' },
    })
    expect(screen.getByRole('button', { name: /Params/ }).textContent).toBe('Params1')
    expect(screen.getByRole('button', { name: /Headers/ }).textContent).toBe('Headers1')
    expect(screen.getByRole('button', { name: /Body/ }).textContent).toBe('Body•')
    expect(screen.getByRole('button', { name: /Auth/ }).textContent).toBe('Auth•')
  })

  it('has no badges when nothing is configured', () => {
    setup()
    for (const t of ['Params', 'Headers', 'Body', 'Auth']) {
      expect(screen.getByRole('button', { name: t }).textContent).toBe(t)
    }
  })

  it('edits params in the Params tab', () => {
    const h = setup()
    fireEvent.click(screen.getByText('+ Add row'))
    expect(h.onQueryParamsChange).toHaveBeenCalled()
  })

  it('edits headers in the Headers tab', () => {
    const h = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Headers' }))
    fireEvent.click(screen.getByText('+ Add row'))
    expect(h.onHeadersChange).toHaveBeenCalled()
  })

  it('shows the body editor in the Body tab', () => {
    setup({ body: { type: 'none', content: '' } })
    fireEvent.click(screen.getByRole('button', { name: 'Body' }))
    expect(screen.getByText(/no body/i)).toBeInTheDocument()
  })

  it('shows the auth editor in the Auth tab', () => {
    const h = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Auth' }))
    fireEvent.change(screen.getAllByRole('combobox').at(-1)!, { target: { value: 'bearer' } })
    expect(h.onAuthChange).toHaveBeenCalledWith({ type: 'bearer', token: '' })
  })
})
