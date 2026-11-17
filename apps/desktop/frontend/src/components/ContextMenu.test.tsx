import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ContextMenu } from './ContextMenu'
import type { ContextMenuAction } from './ContextMenu'

afterEach(() => vi.restoreAllMocks())

const items = (over: Partial<{ a: () => void; b: () => void }> = {}): ContextMenuAction[] => [
  { kind: 'action', label: 'Open', onClick: over.a ?? vi.fn() },
  { kind: 'separator' },
  { kind: 'action', label: 'Delete', danger: true, onClick: over.b ?? vi.fn() },
  { kind: 'action', label: 'Locked', disabled: true, onClick: vi.fn() },
]

describe('ContextMenu', () => {
  it('renders actions and separators at the requested position', () => {
    const { container } = render(<ContextMenu x={40} y={60} items={items()} onClose={vi.fn()} />)
    expect(screen.getAllByRole('button').map(b => b.textContent)).toEqual(['Open', 'Delete', 'Locked'])
    expect(container.querySelectorAll('div.border-t')).toHaveLength(1)
    const menu = container.firstElementChild as HTMLElement
    expect(menu.style.left).toBe('40px')
    expect(menu.style.top).toBe('60px')
  })

  it('runs the action, then closes', () => {
    const onClick = vi.fn()
    const onClose = vi.fn()
    render(<ContextMenu x={0} y={0} items={items({ a: onClick })} onClose={onClose} />)
    fireEvent.click(screen.getByText('Open'))
    expect(onClick).toHaveBeenCalledTimes(1)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does nothing when a disabled item is clicked', () => {
    const onClose = vi.fn()
    render(<ContextMenu x={0} y={0} items={items()} onClose={onClose} />)
    const locked = screen.getByText('Locked') as HTMLButtonElement
    expect(locked).toBeDisabled()
    fireEvent.click(locked)
    expect(onClose).not.toHaveBeenCalled()
  })

  it('styles dangerous actions distinctly', () => {
    render(<ContextMenu x={0} y={0} items={items()} onClose={vi.fn()} />)
    expect(screen.getByText('Delete').className).toContain('text-red-400')
    expect(screen.getByText('Open').className).not.toContain('text-red-400')
  })

  it('closes on Escape and on outside clicks, but not on inside clicks', () => {
    const onClose = vi.fn()
    render(<ContextMenu x={0} y={0} items={items()} onClose={onClose} />)
    fireEvent.keyDown(document, { key: 'a' })
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.mouseDown(screen.getByText('Open'))
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
    fireEvent.mouseDown(document.body)
    expect(onClose).toHaveBeenCalledTimes(2)
  })

  it('stops listening after unmount', () => {
    const onClose = vi.fn()
    const { unmount } = render(<ContextMenu x={0} y={0} items={items()} onClose={onClose} />)
    unmount()
    fireEvent.keyDown(document, { key: 'Escape' })
    fireEvent.mouseDown(document.body)
    expect(onClose).not.toHaveBeenCalled()
  })

  it('suppresses the native context menu on itself', () => {
    const { container } = render(<ContextMenu x={0} y={0} items={items()} onClose={vi.fn()} />)
    const ev = new MouseEvent('contextmenu', { bubbles: true, cancelable: true })
    container.firstElementChild!.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(true)
  })

  it('shifts the menu back inside the viewport when it would overflow', () => {
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(200)
    vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockReturnValue(100)
    Object.defineProperty(window, 'innerWidth', { value: 500, configurable: true })
    Object.defineProperty(window, 'innerHeight', { value: 400, configurable: true })
    const { container } = render(<ContextMenu x={450} y={380} items={items()} onClose={vi.fn()} />)
    const menu = container.firstElementChild as HTMLElement
    expect(menu.style.left).toBe('296px')
    expect(menu.style.top).toBe('296px')
  })
})
