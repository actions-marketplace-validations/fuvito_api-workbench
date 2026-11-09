import { useState, useEffect, useRef } from 'react'
import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

type SubMenuItem =
  | { kind: 'action'; label: string; onClick: () => void; title?: string }
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
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function onDown(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(null)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(null)
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div
      ref={ref}
      className="flex items-center shrink-0 bg-[var(--color-card)] border-b border-[var(--color-border)] relative z-50"
    >
      {menus.map(menu => (
        <div key={menu.label} className="relative">
          <button
            type="button"
            onMouseDown={e => {
              e.stopPropagation()
              setOpen(prev => (prev === menu.label ? null : menu.label))
            }}
            onMouseEnter={() => {
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
            <ul className="absolute left-0 top-full mt-px w-52 bg-[var(--color-card)] border border-[var(--color-border)] rounded shadow-xl py-1">
              {menu.items.map((item, i) =>
                item.kind === 'separator' ? (
                  <li key={i} role="separator" className="h-px bg-[var(--color-border)] my-1 mx-2" />
                ) : item.kind === 'submenu' ? (
                  <li key={i} className="relative group">
                    <button
                      type="button"
                      disabled={item.disabled || item.items.length === 0}
                      className={cn(
                        'w-full flex items-center justify-between px-3 py-1 text-xs',
                        item.disabled || item.items.length === 0
                          ? 'opacity-40 cursor-default text-[var(--color-muted-foreground)]'
                          : 'text-[var(--color-foreground)] hover:bg-[var(--color-muted)] cursor-pointer',
                      )}
                    >
                      <span>{item.label}</span>
                      <ChevronRight size={11} className="text-[var(--color-muted-foreground)] shrink-0" />
                    </button>
                    {!item.disabled && item.items.length > 0 && (
                      <ul className="absolute left-full top-0 min-w-[220px] max-h-72 overflow-y-auto bg-[var(--color-card)] border border-[var(--color-border)] rounded shadow-xl py-1 hidden group-hover:block">
                        {item.items.map((sub, si) =>
                          sub.kind === 'separator' ? (
                            <li key={si} role="separator" className="h-px bg-[var(--color-border)] my-1 mx-2" />
                          ) : (
                            <li key={si}>
                              <button
                                type="button"
                                title={sub.title}
                                onMouseDown={() => { setOpen(null); sub.onClick() }}
                                className="w-full flex items-center px-3 py-1 text-xs text-[var(--color-foreground)] hover:bg-[var(--color-muted)] cursor-pointer truncate"
                              >
                                {sub.label}
                              </button>
                            </li>
                          )
                        )}
                      </ul>
                    )}
                  </li>
                ) : (
                  <li key={i}>
                    <button
                      type="button"
                      disabled={item.disabled}
                      onMouseDown={() => {
                        setOpen(null)
                        item.onClick()
                      }}
                      className={cn(
                        'w-full flex items-center justify-between px-3 py-1 text-xs',
                        item.disabled
                          ? 'opacity-40 cursor-default text-[var(--color-muted-foreground)]'
                          : 'text-[var(--color-foreground)] hover:bg-[var(--color-muted)] cursor-pointer',
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
                ),
              )}
            </ul>
          )}
        </div>
      ))}
    </div>
  )
}
