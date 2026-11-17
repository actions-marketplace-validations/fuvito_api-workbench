import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'

vi.mock('@/lib/bridge', () => ({
  getEnvironment: vi.fn(), saveEnvironment: vi.fn(), deleteEnvironment: vi.fn(),
  getEnvironmentRaw: vi.fn(), saveEnvironmentRaw: vi.fn(), validateEnvironmentYAML: vi.fn(),
}))

import * as bridge from '@/lib/bridge'
import { EnvironmentEditor } from './EnvironmentEditor'

const b = vi.mocked(bridge)
const saved = { name: 'Local', path: '/ws/environments/local.yaml' }

function setup(over: Partial<React.ComponentProps<typeof EnvironmentEditor>> = {}) {
  const h = { onClose: vi.fn(), onSaved: vi.fn(), onDeleted: vi.fn() }
  render(<EnvironmentEditor envPath={null} workspacePath="/ws" {...h} {...over} />)
  return h
}

const existing = { path: '/e/local.yaml', name: 'Local', variables: [
  { name: 'host', value: 'localhost', secret: false },
  { name: 'token', value: 's3cret', secret: true },
] }

beforeEach(() => {
  vi.resetAllMocks()
  b.getEnvironment.mockResolvedValue(existing)
  b.saveEnvironment.mockResolvedValue(saved)
  b.saveEnvironmentRaw.mockResolvedValue(saved)
  b.validateEnvironmentYAML.mockResolvedValue('')
})
afterEach(() => vi.useRealTimers())

describe('new environment', () => {
  it('starts empty, hides the YAML tab and the delete/duplicate buttons', () => {
    setup()
    expect(screen.getByText('New Environment')).toBeInTheDocument()
    expect(screen.queryByText('YAML')).not.toBeInTheDocument()
    expect(screen.queryByText('Delete')).not.toBeInTheDocument()
    expect(screen.queryByText('Duplicate')).not.toBeInTheDocument()
    expect(screen.queryByPlaceholderText('VARIABLE_NAME')).not.toBeInTheDocument()
  })

  it('pre-fills the requested variable name', () => {
    setup({ prefilledVarName: 'baseUrl' })
    expect(screen.getByDisplayValue('baseUrl')).toBeInTheDocument()
  })

  it('requires a name', async () => {
    const h = setup()
    fireEvent.click(screen.getByText('Save'))
    expect(await screen.findByText('Environment name is required.')).toBeInTheDocument()
    expect(b.saveEnvironment).not.toHaveBeenCalled()
    expect(h.onSaved).not.toHaveBeenCalled()
  })

  it('requires an open workspace', async () => {
    setup({ workspacePath: '' })
    fireEvent.change(screen.getByPlaceholderText('e.g. Local Dev'), { target: { value: 'X' } })
    fireEvent.click(screen.getByText('Save'))
    expect(await screen.findByText('Open a workspace first.')).toBeInTheDocument()
  })

  it('adds, edits, marks secret and removes variables, then saves them trimmed of ids', async () => {
    const h = setup()
    fireEvent.change(screen.getByPlaceholderText('e.g. Local Dev'), { target: { value: '  Local  ' } })
    fireEvent.click(screen.getByText('+ Add variable'))
    fireEvent.click(screen.getByText('+ Add variable'))
    const names = screen.getAllByPlaceholderText('VARIABLE_NAME')
    fireEvent.change(names[0], { target: { value: 'host' } })
    fireEvent.change(screen.getAllByPlaceholderText('value')[0], { target: { value: 'h' } })
    fireEvent.change(names[1], { target: { value: 'drop' } })
    fireEvent.click(screen.getAllByTitle('Mark as secret')[0])
    fireEvent.click(screen.getAllByRole('button').filter(x => x.querySelector('svg.lucide-x'))[2]) // remove 2nd variable

    fireEvent.click(screen.getByText('Save'))
    await waitFor(() => expect(h.onSaved).toHaveBeenCalledWith(saved))
    expect(b.saveEnvironment).toHaveBeenCalledWith({ path: '', name: 'Local', variables: [{ name: 'host', value: 'h', secret: true }] })
  })

  it('shows backend save errors and re-enables the buttons', async () => {
    b.saveEnvironment.mockRejectedValue(new Error('disk full'))
    const h = setup()
    fireEvent.change(screen.getByPlaceholderText('e.g. Local Dev'), { target: { value: 'X' } })
    fireEvent.click(screen.getByText('Save'))
    expect(await screen.findByText(/disk full/)).toBeInTheDocument()
    expect(h.onSaved).not.toHaveBeenCalled()
    expect(screen.getByText('Save')).not.toBeDisabled()
  })
})

describe('existing environment', () => {
  it('loads and shows its variables, masking secrets until revealed', async () => {
    setup({ envPath: '/e/local.yaml' })
    expect(screen.getByText('Loading…')).toBeInTheDocument()
    expect(await screen.findByDisplayValue('Local')).toBeInTheDocument()
    const secret = screen.getByDisplayValue('s3cret') as HTMLInputElement
    expect(secret.type).toBe('password')
    expect((screen.getByDisplayValue('localhost') as HTMLInputElement).type).toBe('text')

    fireEvent.click(screen.getByTitle('Reveal value'))
    expect(secret.type).toBe('text')
    fireEvent.click(screen.getByTitle('Hide value'))
    expect(secret.type).toBe('password')
  })

  it('un-marking a secret shows its value and hides the reveal state', async () => {
    setup({ envPath: '/e/local.yaml' })
    await screen.findByDisplayValue('Local')
    fireEvent.click(screen.getByTitle('Reveal value'))
    fireEvent.click(screen.getByTitle('Mark as plain text'))
    expect((screen.getByDisplayValue('s3cret') as HTMLInputElement).type).toBe('text')
  })

  it('saves back to the same path', async () => {
    const h = setup({ envPath: '/e/local.yaml' })
    await screen.findByDisplayValue('Local')
    fireEvent.click(screen.getByText('Save'))
    await waitFor(() => expect(h.onSaved).toHaveBeenCalled())
    expect(b.saveEnvironment).toHaveBeenCalledWith(expect.objectContaining({ path: '/e/local.yaml', name: 'Local' }))
  })

  it('adds a pre-filled variable only if it is not already present', async () => {
    const { unmount } = render(<EnvironmentEditor envPath="/e/local.yaml" workspacePath="/ws" onClose={vi.fn()} onSaved={vi.fn()} onDeleted={vi.fn()} prefilledVarName="newVar" />)
    expect(await screen.findByDisplayValue('newVar')).toBeInTheDocument()
    unmount()
    render(<EnvironmentEditor envPath="/e/local.yaml" workspacePath="/ws" onClose={vi.fn()} onSaved={vi.fn()} onDeleted={vi.fn()} prefilledVarName="host" />)
    await screen.findByDisplayValue('Local')
    expect(screen.getAllByDisplayValue('host')).toHaveLength(1)
  })

  it('shows load errors', async () => {
    b.getEnvironment.mockRejectedValue(new Error('unreadable'))
    setup({ envPath: '/e/x.yaml' })
    expect(await screen.findByText(/unreadable/)).toBeInTheDocument()
  })

  it('duplicates into a new file with a "Copy" suffix', async () => {
    const h = setup({ envPath: '/e/local.yaml' })
    await screen.findByDisplayValue('Local')
    fireEvent.click(screen.getByText('Duplicate'))
    await waitFor(() => expect(h.onSaved).toHaveBeenCalled())
    expect(b.saveEnvironment).toHaveBeenCalledWith({ path: '', name: 'Local Copy', variables: existing.variables })
  })

  it('reports duplicate failures', async () => {
    b.saveEnvironment.mockRejectedValue(new Error('nope'))
    setup({ envPath: '/e/local.yaml' })
    await screen.findByDisplayValue('Local')
    fireEvent.click(screen.getByText('Duplicate'))
    expect(await screen.findByText(/nope/)).toBeInTheDocument()
  })

  it('requires a second click to delete', async () => {
    const h = setup({ envPath: '/e/local.yaml' })
    await screen.findByDisplayValue('Local')
    fireEvent.click(screen.getByText('Delete'))
    expect(b.deleteEnvironment).not.toHaveBeenCalled()
    fireEvent.click(screen.getByText('Confirm delete'))
    await waitFor(() => expect(h.onDeleted).toHaveBeenCalled())
    expect(b.deleteEnvironment).toHaveBeenCalledWith('/e/local.yaml')
  })

  it('reports delete failures and returns to the unconfirmed state', async () => {
    b.deleteEnvironment.mockRejectedValue(new Error('locked'))
    const h = setup({ envPath: '/e/local.yaml' })
    await screen.findByDisplayValue('Local')
    fireEvent.click(screen.getByText('Delete'))
    fireEvent.click(screen.getByText('Confirm delete'))
    expect(await screen.findByText(/locked/)).toBeInTheDocument()
    expect(screen.getByText('Delete')).toBeInTheDocument()
    expect(h.onDeleted).not.toHaveBeenCalled()
  })

  it('closes via Cancel and the X button', async () => {
    const h = setup({ envPath: '/e/local.yaml' })
    await screen.findByDisplayValue('Local')
    fireEvent.click(screen.getByText('Cancel'))
    expect(h.onClose).toHaveBeenCalled()
  })
})

const yamlArea = async () => {
  await waitFor(() => expect(document.querySelector('textarea')).not.toBeNull())
  return document.querySelector('textarea') as HTMLTextAreaElement
}

describe('YAML mode', () => {
  const raw = 'schemaVersion: "1"\nname: Local\nvariables:\n  - name: host\n    value: localhost\n'

  async function openYaml() {
    b.getEnvironmentRaw.mockResolvedValue(raw)
    const h = setup({ envPath: '/e/local.yaml' })
    await screen.findByDisplayValue('Local')
    fireEvent.click(screen.getByText('YAML'))
    return { h, area: await yamlArea() }
  }

  it('shows the raw file text', async () => {
    const { area } = await openYaml()
    expect(area.value).toBe(raw)
  })

  it('falls back to generated YAML, quoting awkward scalars, if the raw file cannot be read', async () => {
    b.getEnvironmentRaw.mockRejectedValue(new Error('x'))
    b.getEnvironment.mockResolvedValue({ path: '/e/x', name: 'My: env', variables: [
      { name: 'a', value: '', secret: false },
      { name: 'b', value: "it's # odd", secret: true },
      { name: 'c', value: ' padded', secret: false },
    ] })
    setup({ envPath: '/e/x' })
    await screen.findByDisplayValue('My: env')
    fireEvent.click(screen.getByText('YAML'))
    const area = await yamlArea()
    expect(area.value).toBe([
      'schemaVersion: "1"', "name: 'My: env'", '', 'variables:',
      '  - name: a', "    value: ''",
      '  - name: b', "    value: 'it''s # odd'", '    secret: true',
      '  - name: c', "    value: ' padded'", '',
    ].join('\n'))
  })

  it('generates an empty list when there are no variables', async () => {
    b.getEnvironmentRaw.mockRejectedValue(new Error('x'))
    b.getEnvironment.mockResolvedValue({ path: '/e/x', name: 'Empty', variables: [] })
    setup({ envPath: '/e/x' })
    await screen.findByDisplayValue('Empty')
    fireEvent.click(screen.getByText('YAML'))
    expect((await yamlArea()).value).toContain('variables:\n  []')
  })

  it('saves the raw text through the raw endpoint', async () => {
    const { h, area } = await openYaml()
    fireEvent.change(area, { target: { value: raw + '# edit\n' } })
    fireEvent.click(screen.getByText('Save'))
    await waitFor(() => expect(h.onSaved).toHaveBeenCalledWith(saved))
    expect(b.saveEnvironmentRaw).toHaveBeenCalledWith('/e/local.yaml', raw + '# edit\n')
    expect(b.saveEnvironment).not.toHaveBeenCalled()
  })

  it('shows validation errors and then a transient "Valid" state', async () => {
    const { area } = await openYaml()
    b.validateEnvironmentYAML.mockResolvedValueOnce('line 2: bad')
    fireEvent.click(screen.getByText('Check YAML'))
    expect(await screen.findByText('line 2: bad')).toBeInTheDocument()

    fireEvent.change(area, { target: { value: raw + ' ' } })
    expect(screen.queryByText('line 2: bad')).not.toBeInTheDocument()

    vi.useFakeTimers({ shouldAdvanceTime: true })
    fireEvent.click(screen.getByText('Check YAML'))
    expect(await screen.findByText('Valid')).toBeInTheDocument()
    await act(async () => { vi.advanceTimersByTime(2100) })
    expect(screen.getByText('Check YAML')).toBeInTheDocument()
  })

  it('switches back to the form by parsing name and variables, honouring quotes and secrets', async () => {
    const { area } = await openYaml()
    fireEvent.change(area, { target: { value: [
      'schemaVersion: "1"', "name: 'Parsed'", 'variables:',
      '  - name: a', '    value: "1"',
      '  - name: b', '    value: two', '    secret: true',
    ].join('\n') } })
    fireEvent.click(screen.getByText('Form'))
    expect(await screen.findByDisplayValue('Parsed')).toBeInTheDocument()
    expect(screen.getByDisplayValue('1')).toBeInTheDocument()
    expect((screen.getByDisplayValue('two') as HTMLInputElement).type).toBe('password')
  })

  it('refuses to switch to the form when the YAML has no name', async () => {
    const { area } = await openYaml()
    fireEvent.change(area, { target: { value: 'variables:\n  - name: a\n    value: b\n' } })
    fireEvent.click(screen.getByText('Form'))
    expect(await screen.findByText(/Cannot switch to form/)).toBeInTheDocument()
    expect(document.querySelector('textarea')).not.toBeNull()
  })

  it('shows save errors from the raw endpoint', async () => {
    const { area } = await openYaml()
    b.saveEnvironmentRaw.mockRejectedValue(new Error('invalid environment YAML'))
    fireEvent.change(area, { target: { value: 'x' } })
    fireEvent.click(screen.getByText('Save'))
    expect(await screen.findByText(/invalid environment YAML/)).toBeInTheDocument()
  })
})
