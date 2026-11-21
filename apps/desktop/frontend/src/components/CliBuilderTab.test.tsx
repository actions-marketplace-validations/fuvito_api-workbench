import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('@/lib/bridge', () => ({
  getCliCatalog: vi.fn(), getCliContext: vi.fn(), buildCliCommand: vi.fn(),
  pickCliFile: vi.fn(), pickCliFolder: vi.fn(),
}))

import * as bridge from '@/lib/bridge'
import type { CliCommand, CliContext, CliBuildOutput } from '@/lib/bridge'
import { CliBuilderTab } from './CliBuilderTab'

const b = vi.mocked(bridge)

const exitCodes = [
  { code: 0, meaning: 'Every step passed.', forCI: 'The pipeline step succeeds.' },
  { code: 1, meaning: 'An assertion failed.', forCI: 'The pipeline step fails.' },
]

const catalog: CliCommand[] = [
  {
    name: 'run', goal: 'Run a scenario', summary: 'Runs a scenario.', when: 'To check a workflow.',
    inputs: [
      { name: 'scenario-file', short: '', positional: true, required: true, kind: 'path', role: 'scenario', accepts: ['.yaml'], enum: [], default: '', label: 'Scenario or folder', help: 'The scenario to run.', example: 'scenarios/x.yaml', group: 'essential' },
      { name: 'env', short: 'e', positional: false, required: false, kind: 'file', role: 'environment', accepts: ['.yaml'], enum: [], default: '', label: 'Environment', help: 'Variables.', example: '', group: 'essential' },
      { name: 'coverage', short: '', positional: false, required: false, kind: 'bool', role: '', accepts: [], enum: [], default: 'false', label: 'Coverage report', help: 'List exercised operations.', example: '', group: 'essential' },
      { name: 'timeout', short: '', positional: false, required: false, kind: 'int', role: '', accepts: [], enum: [], default: '0', label: 'Step timeout (ms)', help: 'Per-step timeout.', example: '10000', group: 'advanced' },
      { name: 'format', short: 'f', positional: false, required: false, kind: 'enum', role: '', accepts: [], enum: ['text', 'json', 'junit', 'html'], default: 'text', label: 'Report format', help: 'How the report is written.', example: '', group: 'essential' },
      { name: 'output', short: 'o', positional: false, required: false, kind: 'outputFile', role: '', accepts: [], enum: [], default: '', label: 'Save report to', help: 'Write the report to this file.', example: 'reports/run.html', group: 'essential' },
    ],
    outputs: ['A report of every step.'], exitCodes,
  },
  {
    name: 'diff', goal: 'Detect breaking changes between two specs', summary: 'Compares specs.', when: 'Before shipping.',
    inputs: [
      { name: 'base-spec', short: '', positional: true, required: true, kind: 'fileOrURL', role: 'spec', accepts: [], enum: [], default: '', label: 'Current spec', help: 'Baseline.', example: '', group: 'essential' },
    ],
    outputs: ['A list of changes.'], exitCodes,
  },
]

const context: CliContext = {
  workspace: '/repo/ws', runFrom: '/repo',
  scenarios: [{ path: '/repo/ws/scenarios/a.yaml', label: 'scenarios/a.yaml' }],
  scenarioFolders: [{ path: '/repo/ws/scenarios', label: 'scenarios/ — 3 scenarios' }],
  environments: [{ path: '/repo/ws/environments/ci.yaml', label: 'environments/ci.yaml' }],
  specs: [],
}

function built(over: Partial<CliBuildOutput> = {}): CliBuildOutput {
  return {
    command: 'run', valid: true, argv: ['clrsig', 'run', 'scenarios/a.yaml'],
    renderings: [
      { shell: 'bash', multiline: 'clrsig run scenarios/a.yaml \\\n  --coverage', oneline: 'clrsig run scenarios/a.yaml --coverage' },
      { shell: 'powershell', multiline: 'clrsig run scenarios/a.yaml `\n  --coverage', oneline: 'clrsig run scenarios/a.yaml --coverage' },
      { shell: 'cmd', multiline: 'clrsig run scenarios\\a.yaml ^\n  --coverage', oneline: 'clrsig run scenarios\\a.yaml --coverage' },
    ],
    messages: [], needs: [], suggestions: [],
    explanation: {
      goal: 'Run a scenario', summary: 'Runs a scenario step by step.',
      when: 'To check a workflow.',
      parts: [{ token: 'clrsig', meaning: 'The Clear Signal program.' }, { token: 'scenarios/a.yaml', meaning: 'Scenario: The scenario to run.' }],
      outputs: ['A report of every step.'], exitCodes,
    },
    ...over,
  }
}

beforeEach(() => {
  vi.resetAllMocks()
  b.getCliCatalog.mockResolvedValue(catalog)
  b.getCliContext.mockResolvedValue(context)
  b.buildCliCommand.mockResolvedValue(built())
  b.pickCliFile.mockResolvedValue('')
  b.pickCliFolder.mockResolvedValue('')
  Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.resolve()) } })
})

async function ready() {
  render(<CliBuilderTab />)
  await screen.findByRole('radiogroup', { name: /what do you want to do/i })
}

describe('CliBuilderTab', () => {
  it('offers every goal and builds with the workspace run-from folder', async () => {
    await ready()
    expect(screen.getByRole('radio', { name: /Run a scenario/ })).toBeChecked()
    expect(screen.getByRole('radio', { name: /Detect breaking changes/ })).toBeInTheDocument()
    expect(screen.getByLabelText('Run from folder')).toHaveValue('/repo')
    await waitFor(() => expect(b.buildCliCommand).toHaveBeenCalledWith({ command: 'run', values: {}, runFrom: '/repo' }))
    expect(await screen.findByTestId('cli-command')).toHaveTextContent('clrsig run scenarios/a.yaml')
  })

  it('shows plain-language help for every input and keeps advanced ones collapsed', async () => {
    await ready()
    expect(screen.getByText('The scenario to run.')).toBeInTheDocument()
    expect(screen.getByText('List exercised operations.')).toBeInTheDocument()
    expect(screen.getByText(/More options \(1\)/)).toBeInTheDocument()
  })

  it('rebuilds when a value changes and sends only what was set', async () => {
    await ready()
    fireEvent.change(screen.getByLabelText(/Scenario/, { selector: 'input' }), { target: { value: 'scenarios/a.yaml' } })
    fireEvent.click(screen.getByLabelText('Coverage report'))
    await waitFor(() => expect(b.buildCliCommand).toHaveBeenLastCalledWith({
      command: 'run', values: { 'scenario-file': 'scenarios/a.yaml', coverage: 'true' }, runFrom: '/repo',
    }))
    fireEvent.click(screen.getByLabelText('Coverage report'))
    await waitFor(() => expect(b.buildCliCommand).toHaveBeenLastCalledWith({
      command: 'run', values: { 'scenario-file': 'scenarios/a.yaml' }, runFrom: '/repo',
    }))
  })

  it('picks a discovered workspace file', async () => {
    await ready()
    fireEvent.change(screen.getByLabelText('Choose Scenario or folder from the workspace'), { target: { value: '/repo/ws/scenarios/a.yaml' } })
    expect(screen.getByLabelText(/Scenario/, { selector: 'input' })).toHaveValue('/repo/ws/scenarios/a.yaml')
  })

  it('offers workspace folders before files, and a folder can be chosen', async () => {
    await ready()
    const select = screen.getByLabelText('Choose Scenario or folder from the workspace') as HTMLSelectElement
    expect(Array.from(select.options).map(o => o.textContent)).toEqual(['Workspace…', 'scenarios/ — 3 scenarios', 'scenarios/a.yaml'])
    fireEvent.change(select, { target: { value: '/repo/ws/scenarios' } })
    expect(screen.getByLabelText(/Scenario or folder/, { selector: 'input' })).toHaveValue('/repo/ws/scenarios')
    await waitFor(() => expect(b.buildCliCommand).toHaveBeenLastCalledWith(expect.objectContaining({
      values: { 'scenario-file': '/repo/ws/scenarios' },
    })))
  })

  it('has separate buttons to browse for a file and for a folder when both are allowed', async () => {
    b.pickCliFile.mockResolvedValue('/elsewhere/s.yaml')
    b.pickCliFolder.mockResolvedValue('/elsewhere/suite')
    await ready()
    fireEvent.click(screen.getByRole('button', { name: 'Browse for Scenario or folder (file)' }))
    await waitFor(() => expect(screen.getByLabelText(/Scenario or folder/, { selector: 'input' })).toHaveValue('/elsewhere/s.yaml'))
    fireEvent.click(screen.getByRole('button', { name: 'Browse for Scenario or folder (folder)' }))
    await waitFor(() => expect(screen.getByLabelText(/Scenario or folder/, { selector: 'input' })).toHaveValue('/elsewhere/suite'))
    expect(b.pickCliFile).toHaveBeenCalledWith('Scenario or folder')
    expect(b.pickCliFolder).toHaveBeenCalledWith('Scenario or folder')
  })

  it('browses for a file', async () => {
    b.pickCliFile.mockResolvedValue('/elsewhere/x.yaml')
    await ready()
    fireEvent.click(screen.getByRole('button', { name: 'Browse for Environment' }))
    await waitFor(() => expect(screen.getByLabelText(/Environment/, { selector: 'input' })).toHaveValue('/elsewhere/x.yaml'))
    expect(b.pickCliFile).toHaveBeenCalledWith('Environment')
  })

  it('switching the goal builds that command from scratch', async () => {
    await ready()
    fireEvent.change(screen.getByLabelText(/Scenario/, { selector: 'input' }), { target: { value: 'x.yaml' } })
    fireEvent.click(screen.getByRole('radio', { name: /Detect breaking changes/ }))
    await waitFor(() => expect(b.buildCliCommand).toHaveBeenLastCalledWith({ command: 'diff', values: {}, runFrom: '/repo' }))
    expect(screen.getByLabelText(/Current spec/, { selector: 'input' })).toHaveValue('')
  })

  it('switches shell and one-line display', async () => {
    await ready()
    const cmd = await screen.findByTestId('cli-command')
    await waitFor(() => expect(cmd).toHaveTextContent('--coverage'))
    fireEvent.click(screen.getByRole('tab', { name: 'PowerShell' }))
    expect(cmd.textContent).toContain('`')
    fireEvent.click(screen.getByRole('tab', { name: 'cmd' }))
    expect(cmd.textContent).toContain('scenarios\\a.yaml')
    fireEvent.click(screen.getByLabelText('One line'))
    expect(cmd.textContent).toBe('clrsig run scenarios\\a.yaml --coverage')
  })

  it('copies the command for the selected shell', async () => {
    await ready()
    await waitFor(() => expect(screen.getByRole('button', { name: /Copy/ })).toBeEnabled())
    fireEvent.click(screen.getByLabelText('One line'))
    fireEvent.click(screen.getByRole('button', { name: /Copy/ }))
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith('clrsig run scenarios/a.yaml --coverage'))
    expect(await screen.findByText('Copied')).toBeInTheDocument()
  })

  it('shows field problems, blocks copying, and explains what to fix', async () => {
    b.buildCliCommand.mockResolvedValue(built({
      valid: false,
      messages: [
        { field: 'scenario-file', severity: 'error', text: 'Scenario not found: nope.yaml' },
        { field: '', severity: 'warning', text: 'something general' },
      ],
    }))
    await ready()
    // An untouched form shows no required-field error, only the warning.
    expect(await screen.findByText('something general')).toBeInTheDocument()
    expect(screen.queryByText('Scenario not found: nope.yaml')).not.toBeInTheDocument()
    expect(screen.queryByText(/Fix the problems below/)).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(/Scenario/, { selector: 'input' }), { target: { value: 'nope.yaml' } })
    expect(await screen.findByText('Scenario not found: nope.yaml')).toBeInTheDocument()
    expect(screen.getByText(/Fix the problems below/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Copy/ })).toBeDisabled()
  })

  it('shows info messages next to their field only', async () => {
    b.buildCliCommand.mockResolvedValue(built({
      messages: [{ field: 'scenario-file', severity: 'info', text: 'valid, 4 steps' }],
    }))
    await ready()
    expect(await screen.findByText('valid, 4 steps')).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Problems and warnings' })).not.toBeInTheDocument()
  })

  it('lists the environment variables the command needs', async () => {
    b.buildCliCommand.mockResolvedValue(built({
      needs: [{ name: 'API_BASE_URL', secret: false, from: 'baseUrl' }, { name: 'API_TOKEN', secret: true, from: 'token' }],
    }))
    await ready()
    const region = await screen.findByRole('region', { name: 'Environment variables' })
    expect(region).toHaveTextContent('API_BASE_URL')
    expect(region).toHaveTextContent('API_TOKEN')
    expect(region).toHaveTextContent('secret')
  })

  it('explains the command, its output and its exit codes', async () => {
    await ready()
    const ex = await screen.findByRole('region', { name: 'Explanation' })
    expect(ex).toHaveTextContent('Runs a scenario step by step.')
    expect(ex).toHaveTextContent('The Clear Signal program.')
    expect(ex).toHaveTextContent('A report of every step.')
    expect(ex).toHaveTextContent('The pipeline step fails.')
  })

  it('applies a preset from another screen', async () => {
    render(<CliBuilderTab preset={{ id: 1, command: 'run', values: { 'scenario-file': '/repo/ws/scenarios/a.yaml' } }} />)
    await screen.findByRole('radiogroup')
    await waitFor(() => expect(b.buildCliCommand).toHaveBeenCalledWith({
      command: 'run', values: { 'scenario-file': '/repo/ws/scenarios/a.yaml' }, runFrom: '/repo',
    }))
    expect(screen.getByLabelText(/Scenario/, { selector: 'input' })).toHaveValue('/repo/ws/scenarios/a.yaml')
  })

  it('lets the user choose a different run-from folder', async () => {
    b.pickCliFolder.mockResolvedValue('/repo/ws')
    await ready()
    fireEvent.click(screen.getByRole('button', { name: 'Browse for the run-from folder' }))
    await waitFor(() => expect(screen.getByLabelText('Run from folder')).toHaveValue('/repo/ws'))
    await waitFor(() => expect(b.buildCliCommand).toHaveBeenLastCalledWith(expect.objectContaining({ runFrom: '/repo/ws' })))
  })

  it('chooses a report format and offers a suggested file name', async () => {
    b.buildCliCommand.mockImplementation(async input => built({
      suggestions: input.values.format && input.values.format !== 'text' && !input.values.output
        ? [{ field: 'output', value: 'reports/run-a.html', why: 'keeps reports together' }]
        : [],
    }))
    await ready()
    expect(screen.queryByRole('button', { name: /Use reports/ })).not.toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Report format'), { target: { value: 'html' } })
    const use = await screen.findByRole('button', { name: 'Use reports/run-a.html' })
    expect(b.buildCliCommand).toHaveBeenLastCalledWith(expect.objectContaining({ values: { format: 'html' } }))

    fireEvent.click(use)
    expect(screen.getByLabelText('Save report to')).toHaveValue('reports/run-a.html')
    await waitFor(() => expect(b.buildCliCommand).toHaveBeenLastCalledWith(expect.objectContaining({
      values: { format: 'html', output: 'reports/run-a.html' },
    })))
    await waitFor(() => expect(screen.queryByRole('button', { name: /Use reports/ })).not.toBeInTheDocument())
  })

  it('does not send the default report format', async () => {
    await ready()
    fireEvent.change(screen.getByLabelText('Report format'), { target: { value: 'junit' } })
    await waitFor(() => expect(b.buildCliCommand).toHaveBeenLastCalledWith(expect.objectContaining({ values: { format: 'junit' } })))
    fireEvent.change(screen.getByLabelText('Report format'), { target: { value: 'text' } })
    await waitFor(() => expect(b.buildCliCommand).toHaveBeenLastCalledWith(expect.objectContaining({ values: { format: 'text' } })))
  })

  it('reports a load failure', async () => {
    b.getCliCatalog.mockRejectedValue(new Error('boom'))
    render(<CliBuilderTab />)
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })
})
