import { cn } from '@/lib/utils'

export function DeleteRequestDialog({
  filePath,
  onCancel,
  onConfirm,
}: {
  filePath: string
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
      onClick={onCancel}
    >
      <div
        className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg p-5 w-80 shadow-2xl flex flex-col gap-4"
        onClick={e => e.stopPropagation()}
      >
        <div>
          <h3 className="text-sm font-semibold text-[var(--color-foreground)]">Delete request</h3>
          <p className="text-xs text-[var(--color-muted-foreground)] mt-1">
            This will permanently delete{' '}
            <span className="font-medium text-[var(--color-foreground)] font-mono">
              {filePath.replace(/\\/g, '/').split('/').pop()}
            </span>{' '}
            from disk. This cannot be undone.
          </p>
        </div>
        <div className="flex gap-2 justify-end">
          <button
            type="button"
            onClick={onCancel}
            className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className="px-3 py-1 text-xs rounded bg-red-600 text-white hover:bg-red-700 transition-colors"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  )
}

export function DeleteFolderDialog({
  folderPath,
  onCancel,
  onConfirm,
}: {
  folderPath: string
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
      onClick={onCancel}
    >
      <div
        className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg p-5 w-80 shadow-2xl flex flex-col gap-4"
        onClick={e => e.stopPropagation()}
      >
        <div>
          <h3 className="text-sm font-semibold text-[var(--color-foreground)]">Delete folder</h3>
          <p className="text-xs text-[var(--color-muted-foreground)] mt-1">
            Delete empty folder{' '}
            <span className="font-medium text-[var(--color-foreground)] font-mono">
              {folderPath.replace(/\\/g, '/').split('/').pop()}
            </span>
            ? This cannot be undone.
          </p>
        </div>
        <div className="flex gap-2 justify-end">
          <button
            type="button"
            onClick={onCancel}
            className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className="px-3 py-1 text-xs rounded bg-red-600 text-white hover:bg-red-700 transition-colors"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  )
}

export function NewFolderDialog({
  parentLabel,
  value,
  onChange,
  onCancel,
  onConfirm,
  error,
}: {
  parentLabel: string
  value: string
  onChange: (v: string) => void
  onCancel: () => void
  onConfirm: () => void
  error?: string
}) {
  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
      onClick={onCancel}
    >
      <div
        className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg p-5 w-80 shadow-2xl flex flex-col gap-4"
        onClick={e => e.stopPropagation()}
      >
        <div>
          <h3 className="text-sm font-semibold text-[var(--color-foreground)]">New folder</h3>
          <p className="text-xs text-[var(--color-muted-foreground)] mt-0.5">
            Inside <span className="font-medium text-[var(--color-foreground)] font-mono">{parentLabel}</span>
          </p>
        </div>
        <input
          autoFocus
          type="text"
          value={value}
          onChange={e => onChange(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter') onConfirm()
            if (e.key === 'Escape') onCancel()
          }}
          placeholder="folder-name"
          className={cn(
            'w-full text-xs rounded px-3 py-1.5 bg-[var(--color-muted)]',
            'border outline-none focus:border-[var(--color-accent)]',
            error ? 'border-red-500' : 'border-[var(--color-border)]',
            'text-[var(--color-foreground)] placeholder:text-[var(--color-muted-foreground)]',
          )}
        />
        {error && (
          <p className="text-xs text-red-400 -mt-2">{error}</p>
        )}
        <div className="flex gap-2 justify-end">
          <button
            type="button"
            onClick={onCancel}
            className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={!value.trim()}
            className="px-3 py-1 text-xs rounded bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity disabled:opacity-40"
          >
            Create
          </button>
        </div>
      </div>
    </div>
  )
}

export function ImportFromURLDialog({
  value,
  onChange,
  onCancel,
  onConfirm,
  error,
}: {
  value: string
  onChange: (v: string) => void
  onCancel: () => void
  onConfirm: () => void
  error?: string
}) {
  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
      onClick={onCancel}
    >
      <div
        className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg p-5 w-96 shadow-2xl flex flex-col gap-4"
        onClick={e => e.stopPropagation()}
      >
        <div>
          <h3 className="text-sm font-semibold text-[var(--color-foreground)]">Import OpenAPI from URL</h3>
          <p className="text-xs text-[var(--color-muted-foreground)] mt-0.5">
            Enter the URL of an OpenAPI spec (JSON or YAML).
          </p>
        </div>
        <input
          autoFocus
          type="url"
          value={value}
          onChange={e => onChange(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter' && value.trim()) onConfirm()
            if (e.key === 'Escape') onCancel()
          }}
          placeholder="https://api.example.com/openapi.json"
          className={cn(
            'w-full text-xs rounded px-3 py-1.5 bg-[var(--color-muted)] font-mono',
            'border outline-none focus:border-[var(--color-accent)]',
            error ? 'border-red-500' : 'border-[var(--color-border)]',
            'text-[var(--color-foreground)] placeholder:text-[var(--color-muted-foreground)]',
          )}
        />
        {error && <p className="text-xs text-red-400 -mt-2">{error}</p>}
        <div className="flex gap-2 justify-end">
          <button
            type="button"
            onClick={onCancel}
            className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={!value.trim()}
            className="px-3 py-1 text-xs rounded bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity disabled:opacity-40"
          >
            Import
          </button>
        </div>
      </div>
    </div>
  )
}

export function CloseDirtyTabDialog({
  tabName,
  onCancel,
  onDiscard,
  onSave,
}: {
  tabName: string | null
  onCancel: () => void
  onDiscard: () => void
  onSave: () => void
}) {
  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
      onClick={onCancel}
    >
      <div
        className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg p-5 w-80 shadow-2xl flex flex-col gap-4"
        onClick={e => e.stopPropagation()}
      >
        <div>
          <h3 className="text-sm font-semibold text-[var(--color-foreground)]">Unsaved changes</h3>
          <p className="text-xs text-[var(--color-muted-foreground)] mt-1">
            <span className="font-medium text-[var(--color-foreground)]">
              {tabName ?? 'This request'}
            </span>{' '}
            has unsaved changes. Save before closing?
          </p>
        </div>
        <div className="flex gap-2 justify-end">
          <button
            type="button"
            onClick={onCancel}
            className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onDiscard}
            className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
          >
            Discard
          </button>
          <button
            type="button"
            onClick={onSave}
            className="px-3 py-1 text-xs rounded bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity"
          >
            Save & Close
          </button>
        </div>
      </div>
    </div>
  )
}
