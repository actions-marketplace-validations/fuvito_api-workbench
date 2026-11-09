import { useState, useRef, useEffect } from 'react'
import { ChevronDown, FolderPlus, FolderSymlink, Upload, RefreshCw } from 'lucide-react'

type Props = {
  folderName: string
  workspacePath: string
  importing?: boolean
  onCreateFolder?: (parentPath: string) => void
  onImportOpenAPI: () => void
  onImportPostman?: () => void
  onImportInsomnia?: () => void
  onRefresh?: () => void
  onOpenWorkspace: () => void
}

export function WorkspacePanelHeader({
  folderName, workspacePath, importing,
  onCreateFolder, onImportOpenAPI, onImportPostman, onImportInsomnia,
  onRefresh, onOpenWorkspace,
}: Props) {
  const [importMenuOpen, setImportMenuOpen] = useState(false)
  const importMenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!importMenuOpen) return
    function onDown(e: MouseEvent) {
      if (importMenuRef.current && !importMenuRef.current.contains(e.target as Node)) setImportMenuOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [importMenuOpen])

  return (
    <div className="px-2 py-1.5 border-b border-[var(--color-border)] flex items-center gap-1 justify-between">
      <span className="text-label font-mono text-[var(--color-muted-foreground)] truncate" title={workspacePath}>
        {folderName}
      </span>
      <div className="flex items-center gap-1 shrink-0">
        {onCreateFolder && (
          <button
            type="button"
            onClick={() => onCreateFolder('')}
            className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
            title="New folder"
          >
            <FolderPlus size={12} />
          </button>
        )}
        <div ref={importMenuRef} className="relative">
          <button
            type="button"
            onClick={() => setImportMenuOpen(v => !v)}
            disabled={importing}
            className="flex items-center gap-0.5 text-[var(--color-muted-foreground)] hover:text-[var(--color-accent)] transition-colors disabled:opacity-40"
            title="Import collection"
          >
            {importing ? <span className="text-label">…</span> : <Upload size={12} />}
            <ChevronDown size={9} />
          </button>
          {importMenuOpen && (
            <ul className="absolute right-0 top-full mt-1 w-44 bg-[var(--color-card)] border border-[var(--color-border)] rounded shadow-xl py-1 z-50">
              <li>
                <button type="button" onMouseDown={() => { setImportMenuOpen(false); onImportOpenAPI() }}
                  className="w-full text-left px-3 py-1.5 text-xs text-[var(--color-foreground)] hover:bg-[var(--color-muted)] cursor-pointer">
                  OpenAPI spec
                </button>
              </li>
              <li>
                <button type="button" onMouseDown={() => { setImportMenuOpen(false); onImportPostman?.() }}
                  className="w-full text-left px-3 py-1.5 text-xs text-[var(--color-foreground)] hover:bg-[var(--color-muted)] cursor-pointer">
                  Postman collection
                </button>
              </li>
              <li>
                <button type="button" onMouseDown={() => { setImportMenuOpen(false); onImportInsomnia?.() }}
                  className="w-full text-left px-3 py-1.5 text-xs text-[var(--color-foreground)] hover:bg-[var(--color-muted)] cursor-pointer">
                  Insomnia export
                </button>
              </li>
            </ul>
          )}
        </div>
        {onRefresh && (
          <button
            type="button"
            onClick={onRefresh}
            className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
            title="Refresh file tree"
          >
            <RefreshCw size={12} />
          </button>
        )}
        <button
          type="button"
          onClick={onOpenWorkspace}
          className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
          title="Switch workspace folder"
        >
          <FolderSymlink size={12} />
        </button>
      </div>
    </div>
  )
}
