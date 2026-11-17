import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MenuBar, type MenuDef } from './MenuBar'

function makeMenus() {
  const fns = {
    open: vi.fn(),
    save: vi.fn(),
    sub1: vi.fn(),
    sub2: vi.fn(),
    help: vi.fn(),
  }
  const menus: MenuDef[] = [
    {
      label: 'File',
      items: [
        { kind: 'action', label: 'Open', shortcut: 'Ctrl+O', onClick: fns.open },
        { kind: 'separator' },
        { kind: 'action', label: 'Save', onClick: fns.save },
        { kind: 'action', label: 'Disabled item', onClick: vi.fn(), disabled: true },
        {
          kind: 'submenu',
          label: 'Recent',
          items: [
            { kind: 'action', label: 'one.yaml', onClick: fns.sub1, title: 'path/one.yaml' },
            { kind: 'separator' },
            { kind: 'action', label: 'two.yaml', onClick: fns.sub2 },
            { kind: 'action', label: 'gone.yaml', onClick: vi.fn(), disabled: true },
          ],
        },
        { kind: 'submenu', label: 'Empty', items: [] },
      ],
    },
    { label: 'Help', items: [{ kind: 'action', label: 'About', onClick: fns.help }] },
  ]
  return { menus, fns }
}

function openMenu(label: string) {
  fireEvent.mouseDown(screen.getByRole('menuitem', { name: label }))
}

describe('MenuBar', () => {
  it('renders the top-level menu labels with all menus closed', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    expect(screen.getByRole('menubar')).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'File' })).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByRole('menuitem', { name: 'Help' })).toBeInTheDocument()
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('opens a menu on mousedown and shows items, shortcut and separator', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    openMenu('File')
    expect(screen.getByRole('menu', { name: 'File' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'File' })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Ctrl+O')).toBeInTheDocument()
    expect(screen.getAllByRole('separator').length).toBeGreaterThan(0)
  })

  it('toggles closed when the open menu label is pressed again', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    openMenu('File')
    openMenu('File')
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('runs an action and closes the menu', () => {
    const { menus, fns } = makeMenus()
    render(<MenuBar menus={menus} />)
    openMenu('File')
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: /^Open/ }))
    expect(fns.open).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('does not run disabled actions', () => {
    const { menus } = makeMenus()
    const disabled = (menus[0].items[3] as { onClick: () => void }).onClick
    render(<MenuBar menus={menus} />)
    openMenu('File')
    const item = screen.getByRole('menuitem', { name: 'Disabled item' })
    expect(item).toBeDisabled()
    fireEvent.mouseDown(item)
    expect(disabled).not.toHaveBeenCalled()
    expect(screen.getByRole('menu', { name: 'File' })).toBeInTheDocument()
  })

  it('switches menus on hover only while a menu is already open', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    fireEvent.mouseEnter(screen.getByRole('menuitem', { name: 'Help' }))
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    openMenu('File')
    fireEvent.mouseEnter(screen.getByRole('menuitem', { name: 'Help' }))
    expect(screen.getByRole('menu', { name: 'Help' })).toBeInTheDocument()
    expect(screen.queryByRole('menu', { name: 'File' })).not.toBeInTheDocument()
  })

  it('switches menus on focus only while a menu is already open', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    fireEvent.focus(screen.getByRole('menuitem', { name: 'Help' }))
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    openMenu('File')
    fireEvent.focus(screen.getByRole('menuitem', { name: 'Help' }))
    expect(screen.getByRole('menu', { name: 'Help' })).toBeInTheDocument()
  })

  it('closes when clicking outside the menu bar', () => {
    render(<div><MenuBar menus={makeMenus().menus} /><p>outside</p></div>)
    openMenu('File')
    fireEvent.mouseDown(screen.getByText('outside'))
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('stays open when mousedown happens inside the menu bar', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    openMenu('File')
    fireEvent.mouseDown(screen.getByRole('menubar'))
    expect(screen.getByRole('menu', { name: 'File' })).toBeInTheDocument()
  })

  it('closes on Escape', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    openMenu('File')
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('opens the first menu on Alt and closes it on a second Alt', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    fireEvent.keyDown(document, { key: 'Alt' })
    expect(screen.getByRole('menu', { name: 'File' })).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Alt' })
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('ignores Alt combined with another modifier', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    fireEvent.keyDown(document, { key: 'Alt', ctrlKey: true })
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('ignores navigation keys while closed', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    fireEvent.keyDown(document, { key: 'ArrowRight' })
    fireEvent.keyDown(document, { key: 'Enter' })
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('moves between menus with ArrowRight / ArrowLeft and wraps around', () => {
    render(<MenuBar menus={makeMenus().menus} />)
    openMenu('File')
    fireEvent.keyDown(document, { key: 'ArrowRight' })
    expect(screen.getByRole('menu', { name: 'Help' })).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'ArrowRight' })
    expect(screen.getByRole('menu', { name: 'File' })).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'ArrowLeft' })
    expect(screen.getByRole('menu', { name: 'Help' })).toBeInTheDocument()
  })

  it('navigates items with ArrowDown/ArrowUp and activates with Enter', () => {
    const { menus, fns } = makeMenus()
    render(<MenuBar menus={menus} />)
    fireEvent.keyDown(document, { key: 'Alt' })
    expect(screen.getByRole('menuitem', { name: /^Open/ })).toHaveFocus()
    fireEvent.keyDown(document, { key: 'ArrowDown' })
    expect(screen.getByRole('menuitem', { name: 'Save' })).toHaveFocus()
    fireEvent.keyDown(document, { key: 'ArrowUp' })
    expect(screen.getByRole('menuitem', { name: /^Open/ })).toHaveFocus()
    fireEvent.keyDown(document, { key: 'ArrowDown' })
    fireEvent.keyDown(document, { key: 'Enter' })
    expect(fns.save).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('activates the focused item with Space', () => {
    const { menus, fns } = makeMenus()
    render(<MenuBar menus={menus} />)
    fireEvent.keyDown(document, { key: 'Alt' })
    fireEvent.keyDown(document, { key: ' ' })
    expect(fns.open).toHaveBeenCalledTimes(1)
  })

  it('does not run a disabled item on Enter', () => {
    const { menus } = makeMenus()
    const disabled = (menus[0].items[3] as { onClick: () => void }).onClick
    render(<MenuBar menus={menus} />)
    fireEvent.keyDown(document, { key: 'Alt' })
    fireEvent.keyDown(document, { key: 'ArrowDown' })
    fireEvent.keyDown(document, { key: 'ArrowDown' })
    fireEvent.keyDown(document, { key: 'Enter' })
    expect(disabled).not.toHaveBeenCalled()
  })

  it('Enter without a focused item does nothing', () => {
    const { menus, fns } = makeMenus()
    render(<MenuBar menus={menus} />)
    openMenu('File')
    fireEvent.keyDown(document, { key: 'Enter' })
    expect(fns.open).not.toHaveBeenCalled()
    expect(screen.getByRole('menu', { name: 'File' })).toBeInTheDocument()
  })

  describe('submenus', () => {
    it('renders submenu entries and runs one on mousedown', () => {
      const { menus, fns } = makeMenus()
      render(<MenuBar menus={menus} />)
      openMenu('File')
      expect(screen.getByRole('menuitem', { name: 'one.yaml' })).toHaveAttribute('title', 'path/one.yaml')
      fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'two.yaml' }))
      expect(fns.sub2).toHaveBeenCalledTimes(1)
      expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    })

    it('does not run a disabled submenu entry', () => {
      render(<MenuBar menus={makeMenus().menus} />)
      openMenu('File')
      const gone = screen.getByRole('menuitem', { name: 'gone.yaml' })
      expect(gone).toBeDisabled()
      fireEvent.mouseDown(gone)
      expect(screen.getByRole('menu', { name: 'File' })).toBeInTheDocument()
    })

    it('disables a submenu with no items and renders no flyout for it', () => {
      render(<MenuBar menus={makeMenus().menus} />)
      openMenu('File')
      expect(screen.getByRole('menuitem', { name: 'Empty' })).toBeDisabled()
    })

    it('moves focus into the flyout with ArrowRight and back with ArrowLeft', () => {
      render(<MenuBar menus={makeMenus().menus} />)
      openMenu('File')
      const parent = screen.getByRole('menuitem', { name: 'Recent' })
      parent.focus()
      fireEvent.keyDown(parent, { key: 'ArrowRight' })
      const first = screen.getByRole('menuitem', { name: 'one.yaml' })
      expect(first).toHaveFocus()
      fireEvent.keyDown(first, { key: 'ArrowLeft' })
      expect(parent).toHaveFocus()
    })

    it('opens the flyout with Enter and returns on Escape from the flyout', () => {
      render(<MenuBar menus={makeMenus().menus} />)
      openMenu('File')
      const parent = screen.getByRole('menuitem', { name: 'Recent' })
      fireEvent.keyDown(parent, { key: 'Enter' })
      const first = screen.getByRole('menuitem', { name: 'one.yaml' })
      expect(first).toHaveFocus()
      fireEvent.keyDown(first, { key: 'Escape' })
      expect(parent).toHaveFocus()
    })

    it('ignores unrelated keys inside the flyout', () => {
      render(<MenuBar menus={makeMenus().menus} />)
      openMenu('File')
      const first = screen.getByRole('menuitem', { name: 'one.yaml' })
      first.focus()
      fireEvent.keyDown(first, { key: 'a' })
      expect(first).toHaveFocus()
    })

    it('does not move focus into an empty submenu', () => {
      render(<MenuBar menus={makeMenus().menus} />)
      openMenu('File')
      const empty = screen.getByRole('menuitem', { name: 'Empty' })
      fireEvent.keyDown(empty, { key: 'ArrowRight' })
      expect(empty).not.toHaveFocus()
    })
  })
})
