import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'

export type ContextMenuAction =
  | { kind: 'action'; label: string; onClick: () => void; disabled?: boolean; danger?: boolean }
  | { kind: 'separator' }

type Props = {
  x: number
  y: number
  items: ContextMenuAction[]
  onClose: () => void
}

export function ContextMenu({ x, y, items, onClose }: Props) {
  const ref = useRef<HTMLDivElement>(null)
  const [pos, setPos] = useState({ x, y })

  useLayoutEffect(() => {
    if (!ref.current) return
    const { offsetWidth: w, offsetHeight: h } = ref.current
    const vw = window.innerWidth
    const vh = window.innerHeight
    setPos({
      x: x + w > vw ? Math.max(0, vw - w - 4) : x,
      y: y + h > vh ? Math.max(0, vh - h - 4) : y,
    })
  }, [x, y])

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    function onMouseDown(e: MouseEvent) {
      if (!ref.current?.contains(e.target as Node)) onClose()
    }
    document.addEventListener('keydown', onKeyDown)
    document.addEventListener('mousedown', onMouseDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.removeEventListener('mousedown', onMouseDown)
    }
  }, [onClose])

  return (
    <div
      ref={ref}
      style={{ position: 'fixed', left: pos.x, top: pos.y, zIndex: 1000 }}
      className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg shadow-xl py-1 min-w-[190px]"
      onContextMenu={e => e.preventDefault()}
    >
      {items.map((item, i) =>
        item.kind === 'separator' ? (
          <div key={i} className="my-1 border-t border-[var(--color-border)]" />
        ) : (
          <button
            key={i}
            type="button"
            disabled={item.disabled}
            onClick={() => { if (!item.disabled) { item.onClick(); onClose() } }}
            className={cn(
              'w-full text-left px-3 py-1.5 text-xs transition-colors',
              item.disabled
                ? 'text-[var(--color-muted-foreground)] opacity-40 cursor-not-allowed'
                : item.danger
                ? 'text-red-400 hover:bg-[var(--color-muted)] cursor-pointer'
                : 'text-[var(--color-foreground)] hover:bg-[var(--color-muted)] cursor-pointer',
            )}
          >
            {item.label}
          </button>
        )
      )}
    </div>
  )
}
