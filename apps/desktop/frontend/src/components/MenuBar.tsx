import { useState, useEffect, useRef, useCallback } from 'react'
import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

type SubMenuItem =
  | { kind: 'action'; label: string; onClick: () => void; title?: string; disabled?: boolean }
  | { kind: 'separator' }

type MenuItem =
  | { kind: 'action'; label: string; shortcut?: string; onClick: () => void; disabled?: boolean }
  | { kind: 'separator' }
  | { kind: 'submenu'; label: string; items: SubMenuItem[]; disabled?: boolean }

export type MenuDef = {
  label: string
  items: MenuItem[]
}

export function MenuBar({ menus }: { menus: MenuDef[] }) {
  const [open, setOpen] = useState<string | null>(null)
  const [focusedIdx, setFocusedIdx] = useState<number>(-1)
  const ref = useRef<HTMLDivElement>(null)
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([])

  const close = useCallback(() => { setOpen(null); setFocusedIdx(-1) }, [])

  // Navigable (non-separator) items in the open menu
  const openMenuItems = open ? menus.find(m => m.label === open)?.items ?? [] : []
  const navItems = openMenuItems.filter(i => i.kind !== 'separator') as Array<MenuItem & { kind: 'action' | 'submenu' }>

  useEffect(() => {
    if (!open) return
    function onDown(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) close()
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [open, close])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      // Alt alone (no other modifier) opens/closes the menu bar
      if (e.key === 'Alt' && !e.ctrlKey && !e.shiftKey && !e.metaKey) {
        e.preventDefault()
        if (open !== null) {
          close()
        } else {
          setOpen(menus[0].label)
          setFocusedIdx(0)
        }
        return
      }

      if (e.key === 'Escape') { close(); return }

      if (open === null) return

      const menuIdx = menus.findIndex(m => m.label === open)

      if (e.key === 'ArrowLeft') {
        e.preventDefault()
        const prev = (menuIdx - 1 + menus.length) % menus.length
        setOpen(menus[prev].label)
        setFocusedIdx(0)
        return
      }
      if (e.key === 'ArrowRight') {
        e.preventDefault()
        const next = (menuIdx + 1) % menus.length
        setOpen(menus[next].label)
        setFocusedIdx(0)
        return
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setFocusedIdx(i => Math.min(i + 1, navItems.length - 1))
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setFocusedIdx(i => Math.max(i - 1, 0))
        return
      }
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        if (focusedIdx >= 0 && focusedIdx < navItems.length) {
          const item = navItems[focusedIdx]
          if (item.kind === 'action' && !item.disabled) {
            close()
            item.onClick()
          }
        }
        return
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, focusedIdx, menus, navItems, close])

  // Focus the highlighted item button
  useEffect(() => {
    if (focusedIdx >= 0 && itemRefs.current[focusedIdx]) {
      itemRefs.current[focusedIdx]?.focus()
    }
  }, [focusedIdx, open])

  // Reset item refs array size when menu changes
  useEffect(() => {
    itemRefs.current = itemRefs.current.slice(0, navItems.length)
  }, [navItems.length])

  return (
    <div
      ref={ref}
      role="menubar"
      className="flex items-center shrink-0 bg-[var(--color-card)] border-b border-[var(--color-border)] relative z-50"
    >
      {menus.map((menu, mi) => (
        <div key={menu.label} className="relative">
          <button
            type="button"
            role="menuitem"
            aria-haspopup="menu"
            aria-expanded={open === menu.label}
            onMouseDown={e => {
              e.stopPropagation()
              if (open === menu.label) {
                close()
              } else {
                setOpen(menu.label)
                setFocusedIdx(-1)
              }
            }}
            onMouseEnter={() => {
              if (open !== null) { setOpen(menu.label); setFocusedIdx(-1) }
            }}
            onFocus={() => {
              if (open !== null) setOpen(menu.label)
            }}
            className={cn(
              'px-3 py-0.5 text-xs select-none transition-colors',
              open === menu.label
                ? 'bg-[var(--color-muted)] text-[var(--color-foreground)]'
                : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]',
            )}
          >
            {menu.label}
          </button>

          {open === menu.label && (
            <ul
              role="menu"
              aria-label={menu.label}
              className="absolute left-0 top-full mt-px w-52 bg-[var(--color-card)] border border-[var(--color-border)] rounded shadow-xl py-1"
            >
              {(() => {
                let navCount = 0
                return menu.items.map((item, i) => {
                  if (item.kind === 'separator') {
                    return <li key={i} role="separator" className="h-px bg-[var(--color-border)] my-1 mx-2" />
                  }
                  if (item.kind === 'submenu') {
                    const idx = navCount++
                    return (
                      <li key={i} role="none" className="relative group">
                        <button
                          ref={el => { itemRefs.current[idx] = el }}
                          type="button"
                          role="menuitem"
                          aria-haspopup="menu"
                          disabled={item.disabled || item.items.length === 0}
                          className={cn(
                            'w-full flex items-center justify-between px-3 py-1 text-xs',
                            item.disabled || item.items.length === 0
                              ? 'opacity-40 cursor-default text-[var(--color-muted-foreground)]'
                              : 'text-[var(--color-foreground)] hover:bg-[var(--color-muted)] focus:bg-[var(--color-muted)] outline-none cursor-pointer',
                          )}
                        >
                          <span>{item.label}</span>
                          <ChevronRight size={11} className="text-[var(--color-muted-foreground)] shrink-0" />
                        </button>
                        {!item.disabled && item.items.length > 0 && (
                          <ul
                            role="menu"
                            className="absolute left-full top-0 min-w-[220px] max-h-72 overflow-y-auto bg-[var(--color-card)] border border-[var(--color-border)] rounded shadow-xl py-1 hidden group-hover:block group-focus-within:block"
                          >
                            {item.items.map((sub, si) =>
                              sub.kind === 'separator' ? (
                                <li key={si} role="separator" className="h-px bg-[var(--color-border)] my-1 mx-2" />
                              ) : (
                                <li key={si} role="none">
                                  <button
                                    type="button"
                                    role="menuitem"
                                    disabled={sub.disabled}
                                    title={sub.title}
                                    onMouseDown={() => { if (!sub.disabled) { close(); sub.onClick() } }}
                                    className={cn(
                                      'w-full flex items-center px-3 py-1 text-xs truncate outline-none',
                                      sub.disabled
                                        ? 'opacity-40 cursor-default text-[var(--color-muted-foreground)]'
                                        : 'text-[var(--color-foreground)] hover:bg-[var(--color-muted)] focus:bg-[var(--color-muted)] cursor-pointer',
                                    )}
                                  >
                                    {sub.label}
                                  </button>
                                </li>
                              )
                            )}
                          </ul>
                        )}
                      </li>
                    )
                  }
                  // action
                  const idx = navCount++
                  return (
                    <li key={i} role="none">
                      <button
                        ref={el => { itemRefs.current[idx] = el }}
                        type="button"
                        role="menuitem"
                        disabled={item.disabled}
                        onMouseDown={() => { if (!item.disabled) { close(); item.onClick() } }}
                        className={cn(
                          'w-full flex items-center justify-between px-3 py-1 text-xs outline-none',
                          item.disabled
                            ? 'opacity-40 cursor-default text-[var(--color-muted-foreground)]'
                            : 'text-[var(--color-foreground)] hover:bg-[var(--color-muted)] focus:bg-[var(--color-muted)] cursor-pointer',
                        )}
                      >
                        <span>{item.label}</span>
                        {item.shortcut && (
                          <span className="text-[var(--color-muted-foreground)] text-label ml-4">
                            {item.shortcut}
                          </span>
                        )}
                      </button>
                    </li>
                  )
                })
              })()}
            </ul>
          )}
        </div>
      ))}
    </div>
  )
}
