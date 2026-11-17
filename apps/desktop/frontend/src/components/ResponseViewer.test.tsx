import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { ResponseViewer } from './ResponseViewer'
import * as bridge from '@/lib/bridge'
import type { ResponseData } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({ openInSystemViewer: vi.fn(() => Promise.resolve()) }))

const timing = { dnsMs: 0, connectMs: 0, tlsMs: 0, ttfbMs: 0, transferMs: 0 }

function resp(over: Partial<ResponseData> = {}): ResponseData {
  return {
    statusCode: 200,
    status: '200 OK',
    headers: [{ name: 'Content-Type', value: 'application/json' }],
    body: '{"name":"Alice","id":7}',
    durationMs: 33,
    size: 24,
    timing,
    ...over,
  }
}

function renderViewer(over: Partial<React.ComponentProps<typeof ResponseViewer>> = {}) {
  const props: React.ComponentProps<typeof ResponseViewer> = { response: resp(), loading: false, ...over }
  const utils = render(<ResponseViewer {...props} />)
  return { props, ...utils }
}

beforeEach(() => {
  vi.clearAllMocks()
  Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.resolve()) } })
})

describe('ResponseViewer — states', () => {
  it('prompts to send a request when there is no response', () => {
    renderViewer({ response: null })
    expect(screen.getByText('Send a request to see the response.')).toBeInTheDocument()
    expect(screen.getByText('Response')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Body' })).not.toBeInTheDocument()
  })

  it('shows a loading indicator while sending', () => {
    renderViewer({ loading: true })
    expect(screen.getByText('Sending request…')).toBeInTheDocument()
  })

  it('shows status line details for a successful response', () => {
    renderViewer()
    expect(screen.getByTestId('response-status')).toHaveTextContent('200')
    expect(screen.getByText('200 OK')).toBeInTheDocument()
    expect(screen.getByText('33 ms')).toBeInTheDocument()
    expect(screen.getByText('24 B')).toBeInTheDocument()
  })

  it('shows a network error and offers clear', () => {
    const onClear = vi.fn()
    renderViewer({ response: resp({ error: 'dial tcp: refused', statusCode: 0 }), onClear })
    expect(screen.getByText('Network Error')).toBeInTheDocument()
    expect(screen.getByText('dial tcp: refused')).toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Clear response'))
    expect(onClear).toHaveBeenCalledTimes(1)
  })

  it('does not offer clear on error without a handler', () => {
    renderViewer({ response: resp({ error: 'x', statusCode: 0 }) })
    expect(screen.queryByLabelText('Clear response')).not.toBeInTheDocument()
  })

  it('treats a zero status code with no error as a bare "Response" header', () => {
    renderViewer({ response: resp({ statusCode: 0 }) })
    expect(screen.getByText('Response')).toBeInTheDocument()
    expect(screen.queryByTestId('response-status')).not.toBeInTheDocument()
  })

  it('shows the timing bar for a response with timing data', () => {
    const t = { dnsMs: 5, connectMs: 5, tlsMs: 0, ttfbMs: 20, transferMs: 3 }
    renderViewer({ response: resp({ timing: t, durationMs: 33 }) })
    expect(screen.getByText('DNS 5ms')).toBeInTheDocument()
    expect(screen.getByText('TTFB 20ms')).toBeInTheDocument()
  })
})

describe('ResponseViewer — header actions', () => {
  it('collapses the panel', () => {
    const onCollapse = vi.fn()
    renderViewer({ onCollapse })
    fireEvent.click(screen.getByTitle('Collapse response panel'))
    expect(onCollapse).toHaveBeenCalledTimes(1)
  })

  it('creates a scenario from the request when supported', () => {
    const onAddToScenario = vi.fn()
    renderViewer({ onAddToScenario })
    fireEvent.click(screen.getByLabelText('New scenario from this request'))
    expect(onAddToScenario).toHaveBeenCalledTimes(1)
  })

  it('hides optional actions when handlers are absent', () => {
    renderViewer()
    expect(screen.queryByTitle('Collapse response panel')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('New scenario from this request')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Clear response')).not.toBeInTheDocument()
  })

  it('clears a successful response', () => {
    const onClear = vi.fn()
    renderViewer({ onClear })
    fireEvent.click(screen.getByLabelText('Clear response'))
    expect(onClear).toHaveBeenCalledTimes(1)
  })

  it('copies the pretty-printed body and shows a transient confirmation', async () => {
    vi.useFakeTimers()
    try {
      renderViewer()
      await act(async () => { fireEvent.click(screen.getByLabelText('Copy response body')) })
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(JSON.stringify({ name: 'Alice', id: 7 }, null, 2))
      expect(screen.getByText('Copied!')).toBeInTheDocument()
      act(() => { vi.advanceTimersByTime(2600) })
      expect(screen.queryByText('Copied!')).not.toBeInTheDocument()
    } finally {
      vi.useRealTimers()
    }
  })

  it('copies the raw body when raw mode is on', async () => {
    renderViewer()
    fireEvent.click(screen.getByLabelText('Switch to raw view'))
    await act(async () => { fireEvent.click(screen.getByLabelText('Copy response body')) })
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('{"name":"Alice","id":7}')
  })

  it('survives a clipboard failure', async () => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.reject(new Error('denied'))) } })
    renderViewer()
    await act(async () => { fireEvent.click(screen.getByLabelText('Copy response body')) })
    expect(screen.queryByText('Copied!')).not.toBeInTheDocument()
  })

  it('does not copy when there is no body text', async () => {
    renderViewer({ response: resp({ body: '' }) })
    await act(async () => { fireEvent.click(screen.getByLabelText('Copy response body')) })
    expect(navigator.clipboard.writeText).not.toHaveBeenCalled()
  })

  it('toggles between raw and pretty view labels', () => {
    renderViewer()
    fireEvent.click(screen.getByLabelText('Switch to raw view'))
    expect(screen.getByLabelText('Switch to pretty view')).toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Switch to pretty view'))
    expect(screen.getByLabelText('Switch to raw view')).toBeInTheDocument()
  })

  it('opens the body in the system viewer with the detected format', async () => {
    renderViewer()
    await act(async () => { fireEvent.click(screen.getByLabelText('Open in system viewer')) })
    expect(bridge.openInSystemViewer).toHaveBeenCalledWith(JSON.stringify({ name: 'Alice', id: 7 }, null, 2), 'json')
  })

  it('opens the raw body in the system viewer in raw mode', async () => {
    renderViewer()
    fireEvent.click(screen.getByLabelText('Switch to raw view'))
    await act(async () => { fireEvent.click(screen.getByLabelText('Open in system viewer')) })
    expect(bridge.openInSystemViewer).toHaveBeenCalledWith('{"name":"Alice","id":7}', 'json')
  })

  it('does not open an empty body externally', async () => {
    renderViewer({ response: resp({ body: '' }) })
    await act(async () => { fireEvent.click(screen.getByLabelText('Open in system viewer')) })
    expect(bridge.openInSystemViewer).not.toHaveBeenCalled()
  })
})

describe('ResponseViewer — tabs', () => {
  it('switches to the headers tab and shows a count', () => {
    renderViewer({ response: resp({ headers: [{ name: 'X-A', value: '1' }, { name: 'X-B', value: '2' }] }) })
    fireEvent.click(screen.getByRole('button', { name: 'Headers' }))
    expect(screen.getByText('2 headers')).toBeInTheDocument()
    expect(screen.getByText('X-A')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Body' }))
    expect(screen.queryByText('2 headers')).not.toBeInTheDocument()
  })

  it('copies all headers and shows confirmation', async () => {
    vi.useFakeTimers()
    try {
      renderViewer({ response: resp({ headers: [{ name: 'X-A', value: '1' }, { name: 'X-B', value: '2' }] }) })
      fireEvent.click(screen.getByRole('button', { name: 'Headers' }))
      await act(async () => { fireEvent.click(screen.getByTitle('Copy all headers')) })
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith('X-A: 1\nX-B: 2')
      expect(screen.getByText('Copied')).toBeInTheDocument()
      act(() => { vi.advanceTimersByTime(1600) })
      expect(screen.getByText('Copy all')).toBeInTheDocument()
    } finally {
      vi.useRealTimers()
    }
  })

  it('survives a header copy failure', async () => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.reject(new Error('denied'))) } })
    renderViewer()
    fireEvent.click(screen.getByRole('button', { name: 'Headers' }))
    await act(async () => { fireEvent.click(screen.getByTitle('Copy all headers')) })
    expect(screen.getByText('Copy all')).toBeInTheDocument()
  })
})

describe('ResponseViewer — search', () => {
  const scrollIntoView = vi.fn()

  beforeEach(() => {
    vi.useFakeTimers()
    scrollIntoView.mockClear()
    Element.prototype.scrollIntoView = scrollIntoView
  })
  afterEach(() => vi.useRealTimers())

  const body = JSON.stringify({ items: ['foo', 'foo', 'bar', 'foo'] })

  function openSearch() {
    fireEvent.click(screen.getByLabelText('Search body'))
    return screen.getByPlaceholderText('Find in response…') as HTMLInputElement
  }

  function type(input: HTMLInputElement, value: string) {
    fireEvent.change(input, { target: { value } })
    act(() => { vi.advanceTimersByTime(200) })
  }

  it('opens from the button and closes from the close button', () => {
    renderViewer({ response: resp({ body }) })
    openSearch()
    fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
    expect(screen.queryByPlaceholderText('Find in response…')).not.toBeInTheDocument()
  })

  it('opens with Ctrl+F, ignores it with no response or on error', () => {
    const { unmount } = renderViewer({ response: null })
    fireEvent.keyDown(window, { key: 'f', ctrlKey: true })
    expect(screen.queryByPlaceholderText('Find in response…')).not.toBeInTheDocument()
    unmount()
    const err = renderViewer({ response: resp({ error: 'x' }) })
    fireEvent.keyDown(window, { key: 'f', ctrlKey: true })
    expect(screen.queryByPlaceholderText('Find in response…')).not.toBeInTheDocument()
    err.unmount()
    renderViewer({ response: resp({ body }) })
    fireEvent.keyDown(window, { key: 'f', metaKey: true })
    expect(screen.getByPlaceholderText('Find in response…')).toBeInTheDocument()
  })

  it('counts matches and steps through them with wrap-around', () => {
    renderViewer({ response: resp({ body }) })
    const input = openSearch()
    type(input, 'foo')
    expect(screen.getByText('1 / 3')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '↓' }))
    expect(screen.getByText('2 / 3')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '↓' }))
    fireEvent.click(screen.getByRole('button', { name: '↓' }))
    expect(screen.getByText('1 / 3')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '↑' }))
    expect(screen.getByText('3 / 3')).toBeInTheDocument()
  })

  it('scrolls the active match into view as it changes', () => {
    renderViewer({ response: resp({ body }) })
    const input = openSearch()
    type(input, 'foo')
    scrollIntoView.mockClear()
    fireEvent.click(screen.getByRole('button', { name: '↓' }))
    expect(scrollIntoView).toHaveBeenCalledWith({ block: 'nearest', behavior: 'smooth' })
  })

  it('steps with Enter and Shift+Enter', () => {
    renderViewer({ response: resp({ body }) })
    const input = openSearch()
    type(input, 'foo')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(screen.getByText('2 / 3')).toBeInTheDocument()
    fireEvent.keyDown(input, { key: 'Enter', shiftKey: true })
    expect(screen.getByText('1 / 3')).toBeInTheDocument()
  })

  it('reports no results and disables stepping', () => {
    renderViewer({ response: resp({ body }) })
    const input = openSearch()
    type(input, 'zzz')
    expect(screen.getByText('No results')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '↓' })).toBeDisabled()
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(screen.getByText('No results')).toBeInTheDocument()
  })

  it('closes on Escape from the input and from the window', () => {
    renderViewer({ response: resp({ body }) })
    const input = openSearch()
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(screen.queryByPlaceholderText('Find in response…')).not.toBeInTheDocument()
    openSearch()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByPlaceholderText('Find in response…')).not.toBeInTheDocument()
  })

  it('resets the search when the response changes', () => {
    const { rerender, props } = renderViewer({ response: resp({ body }) })
    openSearch()
    rerender(<ResponseViewer {...props} response={resp({ body: '{"other":1}' })} />)
    expect(screen.queryByPlaceholderText('Find in response…')).not.toBeInTheDocument()
  })
})

describe('ResponseViewer — diff', () => {
  const prev = resp({ body: '{"v":1}' })
  const curr = resp({ body: '{"v":2}' })

  it('offers diff only when both bodies are valid and different', () => {
    const { unmount } = renderViewer({ response: curr, previousResponse: null })
    expect(screen.queryByLabelText('Diff against previous response')).not.toBeInTheDocument()
    unmount()
    const same = renderViewer({ response: curr, previousResponse: curr })
    expect(screen.queryByLabelText('Diff against previous response')).not.toBeInTheDocument()
    same.unmount()
    const errPrev = renderViewer({ response: curr, previousResponse: resp({ error: 'x' }) })
    expect(screen.queryByLabelText('Diff against previous response')).not.toBeInTheDocument()
    errPrev.unmount()
    renderViewer({ response: curr, previousResponse: prev })
    expect(screen.getByLabelText('Diff against previous response')).toBeInTheDocument()
  })

  it('toggles the diff view on and off', () => {
    renderViewer({ response: curr, previousResponse: prev })
    fireEvent.click(screen.getByLabelText('Diff against previous response'))
    expect(screen.getByLabelText('Show response body')).toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Show response body'))
    expect(screen.getByLabelText('Diff against previous response')).toBeInTheDocument()
  })
})

describe('ResponseViewer — cached indicator', () => {
  beforeEach(() => { vi.useFakeTimers(); vi.setSystemTime(new Date('2026-11-15T12:00:00Z')) })
  afterEach(() => vi.useRealTimers())

  const ago = (ms: number) => new Date(Date.now() - ms).toISOString()

  it.each([
    [10_000, 'Last run just now'],
    [5 * 60_000, 'Last run 5m ago'],
    [3 * 3_600_000, 'Last run 3h ago'],
    [2 * 86_400_000, 'Last run 2d ago'],
  ])('formats %ims as "%s"', (ms, label) => {
    renderViewer({ cachedAt: ago(ms) })
    expect(screen.getByText(label)).toBeInTheDocument()
  })

  it('hides the indicator while loading or when absent', () => {
    const { unmount } = renderViewer({ cachedAt: ago(1000), loading: true })
    expect(screen.queryByText(/Last run/)).not.toBeInTheDocument()
    unmount()
    renderViewer({ cachedAt: null })
    expect(screen.queryByText(/Last run/)).not.toBeInTheDocument()
  })
})
