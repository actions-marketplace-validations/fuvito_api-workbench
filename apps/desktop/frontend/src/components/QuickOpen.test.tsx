import { describe, it, expect, vi, beforeAll } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QuickOpen } from './QuickOpen'
import type { FileNode, ScenarioInfo, EnvInfo } from '@/lib/bridge'

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
})

const tree: FileNode[] = [
  { name: 'users', path: '/ws/requests/users', isDir: true, children: [
    { name: 'get-users.yaml', path: '/ws/requests/users/get-users.yaml', isDir: false, method: 'GET' },
    { name: 'create-user.yml', path: '/ws/requests/users/create-user.yml', isDir: false, method: 'POST' },
  ] },
  { name: 'health.json', path: '/ws/requests/health.json', isDir: false },
]
const scenarios: ScenarioInfo[] = [
  { title: 'Login flow', path: '/ws/scenarios/auth/login.yaml', folder: 'auth' },
  { title: 'folder', path: '/ws/scenarios/auth', isFolder: true },
]
const envs: EnvInfo[] = [{ name: 'staging', path: '/e/staging.yaml' }, { name: 'prod', path: '/e/prod.yaml' }]

function setup(over: Partial<React.ComponentProps<typeof QuickOpen>> = {}) {
  const h = {
    onSelect: vi.fn(), onSelectScenario: vi.fn(), onSelectEnvironment: vi.fn(),
    onNewRequest: vi.fn(), onNewScenario: vi.fn(), onClose: vi.fn(),
  }
  render(<QuickOpen fileTree={tree} scenarios={scenarios} environments={envs} {...h} {...over} />)
  const input = screen.getByRole('textbox')
  return { ...h, input, type: (v: string) => fireEvent.change(input, { target: { value: v } }) }
}

describe('QuickOpen listing', () => {
  it('lists requests, scenarios, environments and commands when empty', () => {
    setup()
    for (const t of ['get-users', 'create-user', 'health', 'Login flow', 'staging', 'prod', 'New Request', 'New Scenario']) {
      expect(screen.getByText(t)).toBeInTheDocument()
    }
    expect(screen.queryByText('folder')).not.toBeInTheDocument()
  })

  it('strips data-file extensions and shows the parent folder and method', () => {
    setup()
    expect(screen.getByText('GET')).toBeInTheDocument()
    expect(screen.getByText('POS')).toBeInTheDocument()
    expect(screen.getAllByText('users')).toHaveLength(2)
    expect(screen.getByText('auth')).toBeInTheDocument()
  })

  it('filters by name or path, case-insensitively', () => {
    const { type } = setup()
    type('USER')
    expect(screen.getByText('get-users')).toBeInTheDocument()
    expect(screen.queryByText('health')).not.toBeInTheDocument()
    type('requests/health')
    expect(screen.getByText('health')).toBeInTheDocument()
    expect(screen.queryByText('get-users')).not.toBeInTheDocument()
  })

  it('shows commands only when they match the query', () => {
    const { type } = setup()
    type('scenario')
    expect(screen.getByText('New Scenario')).toBeInTheDocument()
    expect(screen.queryByText('New Request')).not.toBeInTheDocument()
  })

  it('says when nothing matches', () => {
    const { type } = setup()
    type('zzzzz')
    expect(screen.getByText('No results found.')).toBeInTheDocument()
  })

  it('caps request results at 10 and scenarios at 7', () => {
    const many: FileNode[] = Array.from({ length: 30 }, (_, i) => ({ name: `r${i}.yaml`, path: `/r/r${i}.yaml`, isDir: false }))
    const scn: ScenarioInfo[] = Array.from({ length: 20 }, (_, i) => ({ title: `s${i}`, path: `/s/s${i}.yaml` }))
    setup({ fileTree: many, scenarios: scn, environments: [] })
    expect(screen.getAllByRole('button').filter(b => /^r\d+/.test(b.textContent ?? ''))).toHaveLength(10)
    expect(screen.getAllByRole('button').filter(b => /^s\d+/.test(b.textContent ?? ''))).toHaveLength(7)
  })

  it('works with no scenarios or environments supplied', () => {
    render(<QuickOpen fileTree={tree} onSelect={vi.fn()} onClose={vi.fn()} />)
    expect(screen.getByText('get-users')).toBeInTheDocument()
  })
})

describe('QuickOpen command mode', () => {
  it('shows only commands and environments after ">"', () => {
    const { type } = setup()
    type('>')
    expect(screen.queryByText('get-users')).not.toBeInTheDocument()
    expect(screen.queryByText('Login flow')).not.toBeInTheDocument()
    expect(screen.getByText('New Request')).toBeInTheDocument()
    expect(screen.getByText('staging')).toBeInTheDocument()
    expect(screen.getAllByText('Switch env')).toHaveLength(2)
    expect(screen.queryByText(/for commands/)).not.toBeInTheDocument()
  })

  it('filters commands and environments by the text after ">"', () => {
    const { type } = setup()
    type('> prod')
    expect(screen.getByText('prod')).toBeInTheDocument()
    expect(screen.queryByText('staging')).not.toBeInTheDocument()
    expect(screen.queryByText('New Request')).not.toBeInTheDocument()
  })
})

describe('QuickOpen selection', () => {
  it('clicking a request opens it and closes the palette', () => {
    const h = setup()
    fireEvent.click(screen.getByText('get-users'))
    expect(h.onSelect).toHaveBeenCalledWith('/ws/requests/users/get-users.yaml')
    expect(h.onClose).toHaveBeenCalled()
  })

  it('clicking a scenario passes path and title', () => {
    const h = setup()
    fireEvent.click(screen.getByText('Login flow'))
    expect(h.onSelectScenario).toHaveBeenCalledWith('/ws/scenarios/auth/login.yaml', 'Login flow')
  })

  it('clicking an environment switches to it', () => {
    const h = setup()
    fireEvent.click(screen.getByText('staging'))
    expect(h.onSelectEnvironment).toHaveBeenCalledWith('/e/staging.yaml')
  })

  it('commands call their handlers', () => {
    const h = setup()
    fireEvent.click(screen.getByText('New Request'))
    fireEvent.click(screen.getByText('New Scenario'))
    expect(h.onNewRequest).toHaveBeenCalledTimes(1)
    expect(h.onNewScenario).toHaveBeenCalledTimes(1)
    expect(h.onClose).toHaveBeenCalledTimes(2)
  })

  it('tolerates missing optional handlers', () => {
    render(<QuickOpen fileTree={tree} scenarios={scenarios} environments={envs} onSelect={vi.fn()} onClose={vi.fn()} />)
    expect(() => {
      fireEvent.click(screen.getByText('Login flow'))
      fireEvent.click(screen.getByText('staging'))
      fireEvent.click(screen.getByText('New Request'))
      fireEvent.click(screen.getByText('New Scenario'))
    }).not.toThrow()
  })
})

describe('QuickOpen keyboard', () => {
  it('Enter opens the first result by default', () => {
    const h = setup()
    fireEvent.keyDown(h.input, { key: 'Enter' })
    expect(h.onSelect).toHaveBeenCalledWith('/ws/requests/users/get-users.yaml')
  })

  it('arrow keys move the selection and clamp at both ends', () => {
    const h = setup()
    fireEvent.keyDown(h.input, { key: 'ArrowUp' })
    fireEvent.keyDown(h.input, { key: 'ArrowDown' })
    fireEvent.keyDown(h.input, { key: 'Enter' })
    expect(h.onSelect).toHaveBeenCalledWith('/ws/requests/users/create-user.yml')

    for (let i = 0; i < 50; i++) fireEvent.keyDown(h.input, { key: 'ArrowDown' })
    fireEvent.keyDown(h.input, { key: 'Enter' })
    expect(h.onNewScenario).toHaveBeenCalled()
  })

  it('resets the selection when the query changes', () => {
    const h = setup()
    fireEvent.keyDown(h.input, { key: 'ArrowDown' })
    h.type('health')
    fireEvent.keyDown(h.input, { key: 'Enter' })
    expect(h.onSelect).toHaveBeenCalledWith('/ws/requests/health.json')
  })

  it('Enter with no results does nothing', () => {
    const h = setup()
    h.type('zzzz')
    fireEvent.keyDown(h.input, { key: 'Enter' })
    expect(h.onSelect).not.toHaveBeenCalled()
    expect(h.onClose).not.toHaveBeenCalled()
  })

  it('Escape and a backdrop click close it, but clicks inside do not', () => {
    const h = setup()
    fireEvent.keyDown(h.input, { key: 'Escape' })
    expect(h.onClose).toHaveBeenCalledTimes(1)
    fireEvent.click(h.input)
    expect(h.onClose).toHaveBeenCalledTimes(1)
    fireEvent.click(h.input.closest('.fixed')!)
    expect(h.onClose).toHaveBeenCalledTimes(2)
  })

  it('focuses the input on open', async () => {
    const { input } = setup()
    await vi.waitFor(() => expect(document.activeElement).toBe(input))
  })
})
