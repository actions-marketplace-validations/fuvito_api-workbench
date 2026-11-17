import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { WorkspacePanelHeader } from './WorkspacePanelHeader'

function setup(over: Partial<React.ComponentProps<typeof WorkspacePanelHeader>> = {}) {
  const props: React.ComponentProps<typeof WorkspacePanelHeader> = {
    onImportOpenAPI: vi.fn(),
    onOpenWorkspace: vi.fn(),
    ...over,
  }
  render(<WorkspacePanelHeader {...props} />)
  return props
}

describe('WorkspacePanelHeader', () => {
  it('always offers import and switch-workspace', () => {
    setup()
    expect(screen.getByTitle('Import collection')).toBeInTheDocument()
    expect(screen.getByTitle('Switch workspace folder')).toBeInTheDocument()
    expect(screen.queryByTitle('New folder')).not.toBeInTheDocument()
    expect(screen.queryByTitle('Refresh file tree')).not.toBeInTheDocument()
    expect(screen.queryByTitle('Paste here (workspace root)')).not.toBeInTheDocument()
  })

  it('switches workspace', () => {
    const p = setup()
    fireEvent.click(screen.getByTitle('Switch workspace folder'))
    expect(p.onOpenWorkspace).toHaveBeenCalledTimes(1)
  })

  it('creates a folder at the root', () => {
    const onCreateFolder = vi.fn()
    setup({ onCreateFolder })
    fireEvent.click(screen.getByTitle('New folder'))
    expect(onCreateFolder).toHaveBeenCalledWith('')
  })

  it('refreshes and pastes at the root when handlers are supplied', () => {
    const onRefresh = vi.fn(), onPasteToRoot = vi.fn()
    setup({ onRefresh, onPasteToRoot })
    fireEvent.click(screen.getByTitle('Refresh file tree'))
    fireEvent.click(screen.getByTitle('Paste here (workspace root)'))
    expect(onRefresh).toHaveBeenCalledTimes(1)
    expect(onPasteToRoot).toHaveBeenCalledTimes(1)
  })

  it('shows the import menu and runs each importer then closes', () => {
    const p = setup({ onImportPostman: vi.fn(), onImportInsomnia: vi.fn() })
    for (const [label, fn] of [
      ['OpenAPI spec', p.onImportOpenAPI],
      ['Postman collection', p.onImportPostman],
      ['Insomnia export', p.onImportInsomnia],
    ] as const) {
      fireEvent.click(screen.getByTitle('Import collection'))
      fireEvent.mouseDown(screen.getByText(label))
      expect(fn).toHaveBeenCalledTimes(1)
      expect(screen.queryByText(label)).not.toBeInTheDocument()
    }
  })

  it('tolerates missing optional importers', () => {
    setup()
    fireEvent.click(screen.getByTitle('Import collection'))
    expect(() => fireEvent.mouseDown(screen.getByText('Postman collection'))).not.toThrow()
    fireEvent.click(screen.getByTitle('Import collection'))
    expect(() => fireEvent.mouseDown(screen.getByText('Insomnia export'))).not.toThrow()
  })

  it('closes the menu on outside mousedown but not on inside', () => {
    setup()
    fireEvent.click(screen.getByTitle('Import collection'))
    fireEvent.mouseDown(screen.getByText('OpenAPI spec').closest('ul')!)
    expect(screen.getByText('OpenAPI spec')).toBeInTheDocument()
    fireEvent.mouseDown(document.body)
    expect(screen.queryByText('OpenAPI spec')).not.toBeInTheDocument()
  })

  it('toggles the menu closed on a second click', () => {
    setup()
    fireEvent.click(screen.getByTitle('Import collection'))
    fireEvent.click(screen.getByTitle('Import collection'))
    expect(screen.queryByText('OpenAPI spec')).not.toBeInTheDocument()
  })

  it('disables import while importing', () => {
    setup({ importing: true })
    expect(screen.getByTitle('Import collection')).toBeDisabled()
    expect(screen.getByText('…')).toBeInTheDocument()
  })
})
