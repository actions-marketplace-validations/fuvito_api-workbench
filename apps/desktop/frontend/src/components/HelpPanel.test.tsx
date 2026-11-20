import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'

vi.mock('@/lib/bridge', () => ({
  getUserGuide: vi.fn(), getFormatSpec: vi.fn(), getWhatWeCheck: vi.fn(), getExampleScenario: vi.fn(), getAIPrompt: vi.fn(), getAIPromptInstructions: vi.fn(), saveContentToFile: vi.fn(),
}))

import * as bridge from '@/lib/bridge'
import { HelpPanel } from './HelpPanel'
import { extractScenarioYaml } from '@/lib/helpDocs'

const b = vi.mocked(bridge)

const exampleDoc = [
  '# Example', '', '```yaml', '# file: environments/example.yaml', 'name: example', '```', '',
  '```yaml', '# file: scenarios/user-onboarding.yaml', 'schemaVersion: "1"', 'title: Onboarding', '```', '',
].join('\n')

const catalogDoc = [
  '# What Clear Signal checks', '',
  '| ID | Title | State |', '|---|---|---|',
  '| CS-RESP-001 | Undeclared response status | caught |', '',
].join('\n')

beforeEach(() => {
  vi.resetAllMocks()
  b.getUserGuide.mockResolvedValue('# Guide body')
  b.getFormatSpec.mockResolvedValue('# Spec body')
  b.getWhatWeCheck.mockResolvedValue(catalogDoc)
  b.getExampleScenario.mockResolvedValue(exampleDoc)
  b.getAIPromptInstructions.mockResolvedValue('# Prompt instructions')
  b.saveContentToFile.mockResolvedValue('/saved/file')
  Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.resolve()) } })
})

describe('extractScenarioYaml', () => {
  it('returns only the scenario file block', () => {
    expect(extractScenarioYaml(exampleDoc)).toBe('schemaVersion: "1"\ntitle: Onboarding\n')
  })

  it('handles CRLF line endings', () => {
    expect(extractScenarioYaml(exampleDoc.replace(/\n/g, '\r\n'))).toBe('schemaVersion: "1"\ntitle: Onboarding\n')
  })

  it('returns null when there is no scenario block', () => {
    expect(extractScenarioYaml('# nothing here')).toBeNull()
    expect(extractScenarioYaml('```yaml\n# file: environments/x.yaml\nname: x\n```')).toBeNull()
  })
})

describe('HelpPanel tabs', () => {
  it('shows the guide first and loads the other documents lazily', async () => {
    render(<HelpPanel onClose={vi.fn()} />)
    expect(await screen.findByText('Guide body')).toBeInTheDocument()
    expect(b.getFormatSpec).not.toHaveBeenCalled()
    expect(b.getExampleScenario).not.toHaveBeenCalled()

    fireEvent.click(screen.getByText('Format Spec'))
    expect(await screen.findByText('Spec body')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Example Scenario'))
    expect(await screen.findByText('Example')).toBeInTheDocument()
    expect(b.getExampleScenario).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByText('User Guide'))
    fireEvent.click(screen.getByText('Example Scenario'))
    expect(b.getExampleScenario).toHaveBeenCalledTimes(1)
  })

  it('can open directly on the example tab', async () => {
    render(<HelpPanel onClose={vi.fn()} initialTab="example" />)
    expect(await screen.findByText('Example')).toBeInTheDocument()
  })

  it('reports a load failure instead of hanging', async () => {
    b.getExampleScenario.mockRejectedValue(new Error('x'))
    render(<HelpPanel onClose={vi.fn()} initialTab="example" />)
    expect(await screen.findByText('Failed to load the example scenario.')).toBeInTheDocument()
  })

  it('closes with Escape, the X button and a backdrop click', async () => {
    const onClose = vi.fn()
    const { container } = render(<HelpPanel onClose={onClose} />)
    await screen.findByText('Guide body')
    fireEvent.keyDown(window, { key: 'Escape' })
    fireEvent.click(container.firstElementChild!)
    expect(onClose).toHaveBeenCalledTimes(2)
  })
})

describe('copy buttons', () => {
  it('copies the scenario YAML from the example tab only', async () => {
    render(<HelpPanel onClose={vi.fn()} />)
    await screen.findByText('Guide body')
    expect(screen.queryByText('Copy scenario')).not.toBeInTheDocument()

    fireEvent.click(screen.getByText('Example Scenario'))
    await screen.findByText('Example')
    fireEvent.click(screen.getByText('Copy scenario'))
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith('schemaVersion: "1"\ntitle: Onboarding\n'))
    expect(await screen.findByText('Copied!')).toBeInTheDocument()
  })

  it('does nothing if the document has no scenario block', async () => {
    b.getExampleScenario.mockResolvedValue('# no yaml')
    render(<HelpPanel onClose={vi.fn()} initialTab="example" />)
    fireEvent.click(await screen.findByText('Copy scenario'))
    expect(navigator.clipboard.writeText).not.toHaveBeenCalled()
  })

  it('copies the format spec itself from the spec tab', async () => {
    render(<HelpPanel onClose={vi.fn()} initialTab="spec" />)
    await screen.findByText('Spec body')
    fireEvent.click(screen.getByRole('button', { name: /Copy Format Spec/ }))
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith('# Spec body'))
    expect(await screen.findByText('Copied!')).toBeInTheDocument()
    expect(b.getAIPrompt).not.toHaveBeenCalled()
  })

  it('copies the complete AI prompt (not just the instructions) from the AI Prompt tab', async () => {
    b.getAIPrompt.mockResolvedValue('FULL PROMPT: instructions + spec + example')
    render(<HelpPanel onClose={vi.fn()} initialTab="prompt" />)
    await screen.findByText('Prompt instructions')
    fireEvent.click(screen.getByRole('button', { name: /Copy AI Prompt/ }))
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith('FULL PROMPT: instructions + spec + example'))
    expect(await screen.findByText('Copied!')).toBeInTheDocument()
  })

  it('does not offer the AI prompt button on other tabs', async () => {
    render(<HelpPanel onClose={vi.fn()} />)
    await screen.findByText('Guide body')
    expect(screen.queryByRole('button', { name: /Copy AI Prompt/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Copy Format Spec/ })).not.toBeInTheDocument()
  })

  it('survives a clipboard failure', async () => {
    b.getAIPrompt.mockResolvedValue('x')
    Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.reject(new Error('denied'))) } })
    render(<HelpPanel onClose={vi.fn()} initialTab="prompt" />)
    await screen.findByText('Prompt instructions')
    fireEvent.click(screen.getByRole('button', { name: /Copy AI Prompt/ }))
    await act(async () => {})
    expect(screen.queryByText('Copied!')).not.toBeInTheDocument()
  })
})

describe('AI Prompt tab', () => {
  it('loads the instructions lazily and only once', async () => {
    render(<HelpPanel onClose={vi.fn()} />)
    await screen.findByText('Guide body')
    expect(b.getAIPromptInstructions).not.toHaveBeenCalled()
    fireEvent.click(screen.getByText('AI Prompt'))
    expect(await screen.findByText('Prompt instructions')).toBeInTheDocument()
    fireEvent.click(screen.getByText('User Guide'))
    fireEvent.click(screen.getByText('AI Prompt'))
    expect(b.getAIPromptInstructions).toHaveBeenCalledTimes(1)
  })

  it('reports a load failure', async () => {
    b.getAIPromptInstructions.mockRejectedValue(new Error('x'))
    render(<HelpPanel onClose={vi.fn()} initialTab="prompt" />)
    expect(await screen.findByText('Failed to load the AI prompt.')).toBeInTheDocument()
  })
})

describe('HelpPanel: What We Check tab', () => {
  it('loads the catalog only when its tab is opened and shows its table', async () => {
    render(<HelpPanel onClose={vi.fn()} />)
    await screen.findByText('Guide body')
    expect(b.getWhatWeCheck).not.toHaveBeenCalled()

    fireEvent.click(screen.getByText('What We Check'))
    expect(await screen.findByText('CS-RESP-001')).toBeInTheDocument()
    expect(screen.getByText('Undeclared response status')).toBeInTheDocument()
    // an ID is meant to be quoted, so it never wraps
    expect(screen.getByText('CS-RESP-001').closest('td')).toHaveClass('whitespace-nowrap')
    expect(screen.getByText('Undeclared response status').closest('td')).not.toHaveClass('whitespace-nowrap')
    expect(b.getWhatWeCheck).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByText('User Guide'))
    fireEvent.click(screen.getByText('What We Check'))
    expect(b.getWhatWeCheck).toHaveBeenCalledTimes(1)
  })

  it('opens straight on the catalog from the Help menu', async () => {
    render(<HelpPanel onClose={vi.fn()} initialTab="check" />)
    expect(await screen.findByText('CS-RESP-001')).toBeInTheDocument()
  })

  it('copies and downloads the catalog', async () => {
    render(<HelpPanel onClose={vi.fn()} initialTab="check" />)
    await screen.findByText('CS-RESP-001')

    fireEvent.click(screen.getByRole('button', { name: /^Copy/ }))
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith(catalogDoc))

    fireEvent.click(screen.getByRole('button', { name: /Download/ }))
    await waitFor(() => expect(b.saveContentToFile).toHaveBeenCalledWith(catalogDoc, 'clear-signal-what-we-check.md'))
  })

  it('says so when the catalog cannot be loaded', async () => {
    b.getWhatWeCheck.mockRejectedValue(new Error('boom'))
    render(<HelpPanel onClose={vi.fn()} initialTab="check" />)
    expect(await screen.findByText('Failed to load the list of checks.')).toBeInTheDocument()
  })
})
