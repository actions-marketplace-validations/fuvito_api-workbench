import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ScenariosPanel } from './ScenariosPanel'
import * as bridge from '@/lib/bridge'
import type { ScenarioInfo, TemplateInfo } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({
  listTemplates: vi.fn(),
  createFromTemplate: vi.fn(),
  openInExplorer: vi.fn(),
  getScenarioRunHistory: vi.fn(() => Promise.resolve([])),
  loadScenarioResult: vi.fn(),
}))

const b = vi.mocked(bridge)

const scenarios: ScenarioInfo[] = [
  { title: 'Smoke', path: '/ws/scenarios/smoke.yaml', stepCount: 2 },
  { title: 'Login', path: '/ws/scenarios/auth/login.yaml', folder: 'auth' },
  { title: '', path: '', folder: 'auth', isFolder: true },
]

const templates: TemplateInfo[] = [
  { id: 't1', title: 'Pagination', description: 'd', category: 'Conformance', tags: [], stepCount: 2 },
]

function setup(over: Partial<React.ComponentProps<typeof ScenariosPanel>> = {}) {
  const props: React.ComponentProps<typeof ScenariosPanel> = {
    workspacePath: '/ws',
    scenarios,
    onSelectScenario: vi.fn(),
    onNewScenarioFile: vi.fn(),
    ...over,
  }
  render(<ScenariosPanel {...props} />)
  return props
}

function rightClick(text: string) {
  fireEvent.contextMenu(screen.getByText(text), { clientX: 10, clientY: 10 })
}

beforeEach(() => {
  vi.clearAllMocks()
  b.listTemplates.mockResolvedValue(templates)
  b.createFromTemplate.mockResolvedValue({ path: '/ws/scenarios/new.yaml', error: '' } as never)
  b.getScenarioRunHistory.mockResolvedValue([])
})

describe('ScenariosPanel', () => {
  it('asks to open a workspace when none is set', () => {
    setup({ workspacePath: '' })
    expect(screen.getByText('Open a workspace to manage scenarios.')).toBeInTheDocument()
    expect(screen.queryByText('Run Log')).not.toBeInTheDocument()
  })

  it('shows the scenario tree by default', () => {
    setup()
    expect(screen.getByText('Smoke')).toBeInTheDocument()
    expect(screen.getByText('Login')).toBeInTheDocument()
  })

  it('switches to the run log and back', async () => {
    setup()
    fireEvent.click(screen.getByRole('button', { name: 'Run Log' }))
    expect(await screen.findByText('No runs yet')).toBeInTheDocument()
    expect(screen.queryByText('Smoke')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Scenarios' }))
    expect(screen.getByText('Smoke')).toBeInTheDocument()
  })

  describe('scenario context menu', () => {
    it('opens, duplicates and closes', () => {
      const onDuplicateScenario = vi.fn()
      setup({ onDuplicateScenario })
      rightClick('Smoke')
      fireEvent.click(screen.getByRole('button', { name: 'Duplicate' }))
      expect(onDuplicateScenario).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml')
      expect(screen.queryByRole('button', { name: 'Duplicate' })).not.toBeInTheDocument()
    })

    it('opens the scenario from the menu', () => {
      const p = setup()
      rightClick('Smoke')
      fireEvent.click(screen.getByRole('button', { name: 'Open' }))
      expect(p.onSelectScenario).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml', 'Smoke')
    })

    it('copies, cuts and deletes', () => {
      const onCopyScenario = vi.fn(), onCutScenario = vi.fn(), onDeleteScenario = vi.fn()
      setup({ onCopyScenario, onCutScenario, onDeleteScenario })
      rightClick('Smoke')
      fireEvent.click(screen.getByRole('button', { name: 'Copy' }))
      rightClick('Smoke')
      fireEvent.click(screen.getByRole('button', { name: 'Cut' }))
      rightClick('Smoke')
      fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
      expect(onCopyScenario).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml')
      expect(onCutScenario).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml')
      expect(onDeleteScenario).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml')
    })

    it('starts an inline rename from the menu', () => {
      const onRenameScenario = vi.fn()
      setup({ onRenameScenario })
      rightClick('Smoke')
      fireEvent.click(screen.getByRole('button', { name: 'Rename' }))
      const input = screen.getByDisplayValue('Smoke')
      fireEvent.change(input, { target: { value: 'Renamed' } })
      fireEvent.keyDown(input, { key: 'Enter' })
      expect(onRenameScenario).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml', 'Renamed')
    })

    it('reveals the file in the explorer', () => {
      setup()
      rightClick('Smoke')
      fireEvent.click(screen.getByRole('button', { name: 'Open in Explorer' }))
      expect(b.openInExplorer).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml')
    })
  })

  describe('folder context menu', () => {
    it('renames a folder inline', () => {
      const onRenameScenarioFolder = vi.fn()
      setup({ onRenameScenarioFolder })
      rightClick('auth')
      fireEvent.click(screen.getByRole('button', { name: 'Rename' }))
      const input = screen.getByDisplayValue('auth')
      fireEvent.change(input, { target: { value: 'security' } })
      fireEvent.keyDown(input, { key: 'Enter' })
      expect(onRenameScenarioFolder).toHaveBeenCalledWith('auth', 'security')
    })

    it('deletes and runs a folder', () => {
      const onDeleteScenarioFolder = vi.fn(), onRunFolder = vi.fn()
      setup({ onDeleteScenarioFolder, onRunFolder })
      rightClick('auth')
      fireEvent.click(screen.getByRole('button', { name: 'Delete Folder' }))
      rightClick('auth')
      fireEvent.click(screen.getByRole('button', { name: 'Run all in folder' }))
      expect(onDeleteScenarioFolder).toHaveBeenCalledWith('auth')
      expect(onRunFolder).toHaveBeenCalledWith('auth')
    })

    it('offers the root menu with run-all only', () => {
      const onRunFolder = vi.fn()
      setup({ onRunFolder })
      fireEvent.contextMenu(screen.getByTestId('scenario-root'))
      expect(screen.queryByRole('button', { name: 'Rename' })).not.toBeInTheDocument()
      fireEvent.click(screen.getByRole('button', { name: 'Run all scenarios' }))
      expect(onRunFolder).toHaveBeenCalledWith('')
    })

    it('pastes a scenario into a folder', () => {
      const onPasteToScenarioFolder = vi.fn()
      setup({ onPasteToScenarioFolder, clipboard: { op: 'copy', kind: 'scenario', srcPath: '/x' } })
      rightClick('auth')
      fireEvent.click(screen.getByRole('button', { name: 'Paste copy here' }))
      expect(onPasteToScenarioFolder).toHaveBeenCalledWith('auth')
    })
  })

  describe('templates', () => {
    it('loads and shows the gallery, caching the list for next time', async () => {
      setup()
      fireEvent.click(screen.getByTitle('New scenario from template'))
      expect(await screen.findByText('Pagination')).toBeInTheDocument()
      expect(b.listTemplates).toHaveBeenCalledTimes(1)
      fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
      expect(screen.queryByText('Scenario templates')).not.toBeInTheDocument()
      fireEvent.click(screen.getByTitle('New scenario from template'))
      expect(await screen.findByText('Pagination')).toBeInTheDocument()
      expect(b.listTemplates).toHaveBeenCalledTimes(1)
    })

    it('creates a scenario from a template and reports the new file', async () => {
      const p = setup()
      fireEvent.click(screen.getByTitle('New scenario from template'))
      await screen.findByText('Pagination')
      fireEvent.click(screen.getByRole('button', { name: /Use this template/ }))
      fireEvent.click(screen.getByRole('button', { name: 'Create' }))
      await waitFor(() => expect(b.createFromTemplate).toHaveBeenCalledWith('t1', 'Pagination'))
      await waitFor(() => expect(p.onNewScenarioFile).toHaveBeenCalledWith('/ws/scenarios/new.yaml', 'Pagination'))
      expect(screen.queryByText('Scenario templates')).not.toBeInTheDocument()
    })

    it('shows and dismisses a template error', async () => {
      b.createFromTemplate.mockResolvedValue({ path: '', error: 'name taken' } as never)
      const p = setup()
      fireEvent.click(screen.getByTitle('New scenario from template'))
      await screen.findByText('Pagination')
      fireEvent.click(screen.getByRole('button', { name: /Use this template/ }))
      fireEvent.click(screen.getByRole('button', { name: 'Create' }))
      expect(await screen.findByText(/Failed to create from template: name taken/)).toBeInTheDocument()
      expect(p.onNewScenarioFile).not.toHaveBeenCalled()
      fireEvent.click(screen.getByText('✕'))
      expect(screen.queryByText(/Failed to create/)).not.toBeInTheDocument()
    })

    it('does not report a file when the result has no path', async () => {
      b.createFromTemplate.mockResolvedValue({ path: '', error: '' } as never)
      const p = setup()
      fireEvent.click(screen.getByTitle('New scenario from template'))
      await screen.findByText('Pagination')
      fireEvent.click(screen.getByRole('button', { name: /Use this template/ }))
      fireEvent.click(screen.getByRole('button', { name: 'Create' }))
      await waitFor(() => expect(b.createFromTemplate).toHaveBeenCalled())
      expect(p.onNewScenarioFile).not.toHaveBeenCalled()
    })
  })
})
