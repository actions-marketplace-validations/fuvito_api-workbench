import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { useState } from 'react'
import {
  DeleteRequestDialog,
  DeleteFolderDialog,
  NewFolderDialog,
  ImportFromURLDialog,
  DiscardChangesDialog,
  DeleteScenarioDialog,
  ExportSecretsWarningDialog,
  NewScenarioFromResponseDialog,
  CloseDirtyTabDialog,
} from './ConfirmDialogs'

function backdropOf(el: HTMLElement) {
  return el.closest('.fixed') as HTMLElement
}

describe('DeleteRequestDialog', () => {
  it('shows only the file name for unix and windows paths', () => {
    const { rerender } = render(<DeleteRequestDialog filePath="/ws/requests/users/get.yaml" onCancel={vi.fn()} onConfirm={vi.fn()} />)
    expect(screen.getByText('get.yaml')).toBeInTheDocument()
    rerender(<DeleteRequestDialog filePath={'C:\\ws\\requests\\post.yaml'} onCancel={vi.fn()} onConfirm={vi.fn()} />)
    expect(screen.getByText('post.yaml')).toBeInTheDocument()
  })

  it('fires confirm and cancel from the buttons', () => {
    const onCancel = vi.fn(), onConfirm = vi.fn()
    render(<DeleteRequestDialog filePath="/a/b.yaml" onCancel={onCancel} onConfirm={onConfirm} />)
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(onConfirm).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('cancels on backdrop click but not on dialog body click', () => {
    const onCancel = vi.fn()
    render(<DeleteRequestDialog filePath="/a/b.yaml" onCancel={onCancel} onConfirm={vi.fn()} />)
    fireEvent.click(screen.getByText('Delete request'))
    expect(onCancel).not.toHaveBeenCalled()
    fireEvent.click(backdropOf(screen.getByText('Delete request')))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })
})

describe('DeleteFolderDialog', () => {
  it('names the folder and wires both buttons', () => {
    const onCancel = vi.fn(), onConfirm = vi.fn()
    render(<DeleteFolderDialog folderPath={'C:\\ws\\requests\\users'} onCancel={onCancel} onConfirm={onConfirm} />)
    expect(screen.getByText('users')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onConfirm).toHaveBeenCalledTimes(1)
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('cancels on backdrop click only', () => {
    const onCancel = vi.fn()
    render(<DeleteFolderDialog folderPath="/a/b" onCancel={onCancel} onConfirm={vi.fn()} />)
    fireEvent.click(screen.getByText('Delete folder'))
    expect(onCancel).not.toHaveBeenCalled()
    fireEvent.click(backdropOf(screen.getByText('Delete folder')))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })
})

describe('NewFolderDialog', () => {
  function setup(over: Partial<React.ComponentProps<typeof NewFolderDialog>> = {}) {
    const props = { parentLabel: 'requests', value: '', onChange: vi.fn(), onCancel: vi.fn(), onConfirm: vi.fn(), ...over }
    render(<NewFolderDialog {...props} />)
    return props
  }

  it('shows the parent label and disables Create for blank input', () => {
    setup({ value: '   ' })
    expect(screen.getByText('requests')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create' })).toBeDisabled()
  })

  it('reports typing and confirms with the button', () => {
    const p = setup({ value: 'auth' })
    fireEvent.change(screen.getByPlaceholderText('folder-name'), { target: { value: 'auth2' } })
    expect(p.onChange).toHaveBeenCalledWith('auth2')
    fireEvent.click(screen.getByRole('button', { name: 'Create' }))
    expect(p.onConfirm).toHaveBeenCalledTimes(1)
  })

  it('confirms on Enter and cancels on Escape', () => {
    const p = setup({ value: 'x' })
    const input = screen.getByPlaceholderText('folder-name')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(p.onConfirm).toHaveBeenCalledTimes(1)
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(p.onCancel).toHaveBeenCalledTimes(1)
  })

  it('renders an error message and highlights the input', () => {
    setup({ value: 'x', error: 'Already exists' })
    expect(screen.getByText('Already exists')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('folder-name').className).toContain('border-red-500')
  })

  it('cancels from Cancel and backdrop', () => {
    const p = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    fireEvent.click(backdropOf(screen.getByText('New folder')))
    expect(p.onCancel).toHaveBeenCalledTimes(2)
  })
})

describe('ImportFromURLDialog', () => {
  function setup(over: Partial<React.ComponentProps<typeof ImportFromURLDialog>> = {}) {
    const props = { value: '', onChange: vi.fn(), onCancel: vi.fn(), onConfirm: vi.fn(), ...over }
    render(<ImportFromURLDialog {...props} />)
    return props
  }

  it('disables Import for empty input', () => {
    setup()
    expect(screen.getByRole('button', { name: 'Import' })).toBeDisabled()
  })

  it('enter confirms only when a URL is present', () => {
    const empty = setup()
    fireEvent.keyDown(screen.getByPlaceholderText(/openapi\.json/), { key: 'Enter' })
    expect(empty.onConfirm).not.toHaveBeenCalled()
  })

  it('confirms on Enter / click and cancels on Escape', () => {
    const p = setup({ value: 'https://x.test/spec.json' })
    const input = screen.getByPlaceholderText(/openapi\.json/)
    fireEvent.keyDown(input, { key: 'Enter' })
    fireEvent.click(screen.getByRole('button', { name: 'Import' }))
    expect(p.onConfirm).toHaveBeenCalledTimes(2)
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(p.onCancel).toHaveBeenCalledTimes(1)
  })

  it('forwards edits and shows errors', () => {
    const p = setup({ value: 'a', error: 'Could not fetch' })
    fireEvent.change(screen.getByPlaceholderText(/openapi\.json/), { target: { value: 'https://b' } })
    expect(p.onChange).toHaveBeenCalledWith('https://b')
    expect(screen.getByText('Could not fetch')).toBeInTheDocument()
  })

  it('cancels from Cancel button and backdrop', () => {
    const p = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    fireEvent.click(backdropOf(screen.getByText('Import OpenAPI from URL')))
    expect(p.onCancel).toHaveBeenCalledTimes(2)
  })
})

describe('DiscardChangesDialog', () => {
  it('uses the default title and message', () => {
    render(<DiscardChangesDialog message="You will lose edits." onCancel={vi.fn()} onDiscard={vi.fn()} />)
    expect(screen.getByText('Discard changes?')).toBeInTheDocument()
    expect(screen.getByText('You will lose edits.')).toBeInTheDocument()
  })

  it('honours a custom title and wires actions', () => {
    const onCancel = vi.fn(), onDiscard = vi.fn()
    render(<DiscardChangesDialog title="Revert?" message="m" onCancel={onCancel} onDiscard={onDiscard} />)
    expect(screen.getByText('Revert?')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Discard' }))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    fireEvent.click(backdropOf(screen.getByText('Revert?')))
    expect(onDiscard).toHaveBeenCalledTimes(1)
    expect(onCancel).toHaveBeenCalledTimes(2)
  })
})

describe('DeleteScenarioDialog', () => {
  it('shows the scenario name and omits the unsaved warning when clean', () => {
    render(<DeleteScenarioDialog name="Login flow" isDirty={false} onCancel={vi.fn()} onConfirm={vi.fn()} />)
    expect(screen.getByText('Login flow')).toBeInTheDocument()
    expect(screen.queryByText(/unsaved changes/i)).not.toBeInTheDocument()
  })

  it('warns about unsaved changes when dirty', () => {
    render(<DeleteScenarioDialog name="s" isDirty onCancel={vi.fn()} onConfirm={vi.fn()} />)
    expect(screen.getByText('This scenario has unsaved changes.')).toBeInTheDocument()
  })

  it('wires confirm, cancel and backdrop', () => {
    const onCancel = vi.fn(), onConfirm = vi.fn()
    render(<DeleteScenarioDialog name="s" isDirty={false} onCancel={onCancel} onConfirm={onConfirm} />)
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    fireEvent.click(backdropOf(screen.getByText('Delete scenario')))
    expect(onConfirm).toHaveBeenCalledTimes(1)
    expect(onCancel).toHaveBeenCalledTimes(2)
  })
})

describe('ExportSecretsWarningDialog', () => {
  it('warns and wires both actions', () => {
    const onCancel = vi.fn(), onConfirm = vi.fn()
    render(<ExportSecretsWarningDialog onCancel={onCancel} onConfirm={onConfirm} />)
    expect(screen.getByText(/plaintext/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Export anyway' }))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    fireEvent.click(backdropOf(screen.getByText('Export contains secrets')))
    expect(onConfirm).toHaveBeenCalledTimes(1)
    expect(onCancel).toHaveBeenCalledTimes(2)
  })
})

describe('NewScenarioFromResponseDialog', () => {
  const scenarios = [
    { title: 'Smoke', path: '/ws/scenarios/smoke.yaml' },
    { title: 'Auth', path: '/ws/scenarios/auth.yaml' },
  ]

  function Harness(props: Partial<React.ComponentProps<typeof NewScenarioFromResponseDialog>>) {
    const [v, setV] = useState('')
    return (
      <NewScenarioFromResponseDialog
        defaultName="GET /users"
        value={v}
        onChange={setV}
        onCancel={vi.fn()}
        onConfirm={vi.fn()}
        {...props}
      />
    )
  }

  it('only shows the new-scenario form when there are no existing scenarios', () => {
    render(<Harness />)
    expect(screen.queryByRole('button', { name: 'Add to existing' })).not.toBeInTheDocument()
    expect(screen.getByPlaceholderText('GET /users')).toBeInTheDocument()
  })

  it('hides the mode toggle when onAddToExisting is missing', () => {
    render(<Harness scenarios={scenarios} />)
    expect(screen.queryByRole('button', { name: 'Add to existing' })).not.toBeInTheDocument()
  })

  it('disables Create until a name is typed, then confirms on click and Enter', () => {
    const onConfirm = vi.fn()
    render(<Harness onConfirm={onConfirm} />)
    const create = screen.getByRole('button', { name: 'Create' })
    expect(create).toBeDisabled()
    const input = screen.getByPlaceholderText('GET /users')
    fireEvent.change(input, { target: { value: 'My scenario' } })
    expect(create).toBeEnabled()
    fireEvent.click(create)
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onConfirm).toHaveBeenCalledTimes(2)
  })

  it('Enter with a blank name does not confirm; Escape cancels', () => {
    const onConfirm = vi.fn(), onCancel = vi.fn()
    render(<Harness onConfirm={onConfirm} onCancel={onCancel} />)
    const input = screen.getByPlaceholderText('GET /users')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onConfirm).not.toHaveBeenCalled()
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('switches to existing mode and appends to the first scenario by default', () => {
    const onAdd = vi.fn()
    render(<Harness scenarios={scenarios} onAddToExisting={onAdd} />)
    fireEvent.click(screen.getByRole('button', { name: 'Add to existing' }))
    expect(screen.getByRole('combobox')).toHaveValue('/ws/scenarios/smoke.yaml')
    fireEvent.click(screen.getByRole('button', { name: 'Add step' }))
    expect(onAdd).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml', 'Smoke')
  })

  it('appends to the scenario chosen in the select', () => {
    const onAdd = vi.fn()
    render(<Harness scenarios={scenarios} onAddToExisting={onAdd} />)
    fireEvent.click(screen.getByRole('button', { name: 'Add to existing' }))
    fireEvent.change(screen.getByRole('combobox'), { target: { value: '/ws/scenarios/auth.yaml' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add step' }))
    expect(onAdd).toHaveBeenCalledWith('/ws/scenarios/auth.yaml', 'Auth')
  })

  it('can switch back to new mode and cancel from either mode', () => {
    const onCancel = vi.fn()
    render(<Harness scenarios={scenarios} onAddToExisting={vi.fn()} onCancel={onCancel} />)
    fireEvent.click(screen.getByRole('button', { name: 'Add to existing' }))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    fireEvent.click(screen.getByRole('button', { name: 'New scenario' }))
    expect(screen.getByPlaceholderText('GET /users')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    fireEvent.click(backdropOf(screen.getByText('Add to scenario')))
    expect(onCancel).toHaveBeenCalledTimes(3)
  })
})

describe('CloseDirtyTabDialog', () => {
  it('names the tab, falling back when unnamed', () => {
    const { rerender } = render(<CloseDirtyTabDialog tabName="users.yaml" onCancel={vi.fn()} onDiscard={vi.fn()} onSave={vi.fn()} />)
    expect(screen.getByText('users.yaml')).toBeInTheDocument()
    rerender(<CloseDirtyTabDialog tabName={null} onCancel={vi.fn()} onDiscard={vi.fn()} onSave={vi.fn()} />)
    expect(screen.getByText('This request')).toBeInTheDocument()
  })

  it('wires Save & Close, Discard, Cancel and backdrop', () => {
    const onCancel = vi.fn(), onDiscard = vi.fn(), onSave = vi.fn()
    render(<CloseDirtyTabDialog tabName="t" onCancel={onCancel} onDiscard={onDiscard} onSave={onSave} />)
    fireEvent.click(screen.getByRole('button', { name: 'Save & Close' }))
    fireEvent.click(screen.getByRole('button', { name: 'Discard' }))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    fireEvent.click(backdropOf(screen.getByText('Unsaved changes')))
    expect(onSave).toHaveBeenCalledTimes(1)
    expect(onDiscard).toHaveBeenCalledTimes(1)
    expect(onCancel).toHaveBeenCalledTimes(2)
  })
})
