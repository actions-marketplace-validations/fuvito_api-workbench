import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'

vi.mock('@/lib/bridge', () => ({
  getUserGuide: vi.fn(), getFormatSpec: vi.fn(), getExampleScenario: vi.fn(), getAIPrompt: vi.fn(),
  getAIPromptInstructions: vi.fn(), saveContentToFile: vi.fn(),
}))

import * as bridge from '@/lib/bridge'
import { HelpPanel } from './HelpPanel'
import { extractScenarioFileName } from '@/lib/helpDocs'

const b = vi.mocked(bridge)

const exampleDoc = [
  '# Example', '', '```yaml', '# file: environments/example.yaml', 'name: example', '```', '',
  '```yaml', '# file: scenarios/user-onboarding.yaml', 'schemaVersion: "1"', 'title: Onboarding', '```', '',
].join('\n')

beforeEach(() => {
  vi.resetAllMocks()
  b.getUserGuide.mockResolvedValue('# Guide body')
  b.getFormatSpec.mockResolvedValue('# Spec body')
  b.getExampleScenario.mockResolvedValue(exampleDoc)
  b.getAIPromptInstructions.mockResolvedValue('# Prompt instructions')
  b.saveContentToFile.mockResolvedValue('/saved/file')
  Object.assign(navigator, { clipboard: { writeText: vi.fn(() => Promise.resolve()) } })
})

describe('extractScenarioFileName', () => {
  it('uses the file name from the scenario block', () => {
    expect(extractScenarioFileName(exampleDoc)).toBe('user-onboarding.yaml')
    expect(extractScenarioFileName(exampleDoc.replace(/\n/g, '\r\n'))).toBe('user-onboarding.yaml')
  })

  it('drops folders and falls back when there is no scenario block', () => {
    expect(extractScenarioFileName('```yaml\n# file: scenarios/auth/login.yaml\nx: 1\n```')).toBe('login.yaml')
    expect(extractScenarioFileName('# nothing')).toBe('scenario.yaml')
    expect(extractScenarioFileName('# nothing', 'x.yaml')).toBe('x.yaml')
  })
})

describe('download buttons', () => {
  it('downloads the format spec as Markdown', async () => {
    render(<HelpPanel onClose={vi.fn()} initialTab="spec" />)
    await screen.findByText('Spec body')
    fireEvent.click(screen.getByRole('button', { name: /Download/ }))
    await waitFor(() => expect(b.saveContentToFile).toHaveBeenCalledWith('# Spec body', 'clear-signal-format-spec.md'))
    expect(await screen.findByText('Saved!')).toBeInTheDocument()
    expect(navigator.clipboard.writeText).not.toHaveBeenCalled()
  })

  it('downloads the complete AI prompt, not just the instructions', async () => {
    b.getAIPrompt.mockResolvedValue('FULL PROMPT')
    render(<HelpPanel onClose={vi.fn()} initialTab="prompt" />)
    await screen.findByText('Prompt instructions')
    fireEvent.click(screen.getByRole('button', { name: /Download/ }))
    await waitFor(() => expect(b.saveContentToFile).toHaveBeenCalledWith('FULL PROMPT', 'clear-signal-ai-prompt.md'))
    expect(await screen.findByText('Saved!')).toBeInTheDocument()
  })

  it('downloads only the scenario YAML, named after its file', async () => {
    render(<HelpPanel onClose={vi.fn()} initialTab="example" />)
    await screen.findByText('Example')
    fireEvent.click(screen.getByRole('button', { name: /Download/ }))
    await waitFor(() =>
      expect(b.saveContentToFile).toHaveBeenCalledWith('schemaVersion: "1"\ntitle: Onboarding\n', 'user-onboarding.yaml'),
    )
  })

  it('shows nothing when the save dialog is cancelled', async () => {
    b.saveContentToFile.mockResolvedValue('')
    render(<HelpPanel onClose={vi.fn()} initialTab="spec" />)
    await screen.findByText('Spec body')
    fireEvent.click(screen.getByRole('button', { name: /Download/ }))
    await waitFor(() => expect(b.saveContentToFile).toHaveBeenCalled())
    await act(async () => {})
    expect(screen.queryByText('Saved!')).not.toBeInTheDocument()
  })

  it('survives a failing save', async () => {
    b.saveContentToFile.mockRejectedValue(new Error('disk full'))
    render(<HelpPanel onClose={vi.fn()} initialTab="spec" />)
    await screen.findByText('Spec body')
    fireEvent.click(screen.getByRole('button', { name: /Download/ }))
    await act(async () => {})
    expect(screen.queryByText('Saved!')).not.toBeInTheDocument()
  })

  it('is not offered on the user guide tab', async () => {
    render(<HelpPanel onClose={vi.fn()} />)
    await screen.findByText('Guide body')
    expect(screen.queryByRole('button', { name: /Download/ })).not.toBeInTheDocument()
  })

  it('shows the confirmation only on the button that was used', async () => {
    render(<HelpPanel onClose={vi.fn()} initialTab="spec" />)
    await screen.findByText('Spec body')
    fireEvent.click(screen.getByRole('button', { name: /Download/ }))
    await screen.findByText('Saved!')
    expect(screen.getByRole('button', { name: /Copy Format Spec/ })).toBeInTheDocument()
    expect(screen.queryByText('Copied!')).not.toBeInTheDocument()
  })

  it('copying still works alongside downloading', async () => {
    render(<HelpPanel onClose={vi.fn()} initialTab="spec" />)
    await screen.findByText('Spec body')
    fireEvent.click(screen.getByRole('button', { name: /Copy Format Spec/ }))
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith('# Spec body'))
    expect(b.saveContentToFile).not.toHaveBeenCalled()
  })
})
