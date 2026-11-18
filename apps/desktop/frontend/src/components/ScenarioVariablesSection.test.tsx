import { describe, it, expect, vi } from 'vitest'
import { useState } from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { ScenarioVariablesSection } from './ScenarioVariablesSection'
import type { ScenarioVariable } from '@/lib/bridge'

function Harness({ initial = [], spy }: { initial?: ScenarioVariable[]; spy?: (v: ScenarioVariable[]) => void }) {
  const [vars, setVars] = useState<ScenarioVariable[]>(initial)
  return <ScenarioVariablesSection variables={vars} onChange={v => { setVars(v); spy?.(v) }} />
}

describe('ScenarioVariablesSection', () => {
  it('is collapsed when empty and adds a variable', () => {
    const spy = vi.fn()
    render(<Harness spy={spy} />)
    expect(screen.queryByText('Add variable')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /variables/i }))
    fireEvent.click(screen.getByText('Add variable'))
    expect(spy).toHaveBeenLastCalledWith([{ name: '', value: '' }])
    expect(screen.getByLabelText('Variable 1 name')).toBeInTheDocument()
  })

  it('is open when variables exist and edits name, value and secret', () => {
    const spy = vi.fn()
    render(<Harness initial={[{ name: 'email', value: 'a' }]} spy={spy} />)
    fireEvent.change(screen.getByLabelText('Variable 1 name'), { target: { value: 'mail' } })
    expect(spy).toHaveBeenLastCalledWith([{ name: 'mail', value: 'a' }])
    fireEvent.change(screen.getAllByRole('textbox')[1], { target: { value: 'u-{{$randomString 6}}' } })
    expect(spy).toHaveBeenLastCalledWith([{ name: 'mail', value: 'u-{{$randomString 6}}' }])
    fireEvent.click(screen.getByRole('button', { name: /mark as secret/i }))
    expect(spy).toHaveBeenLastCalledWith([{ name: 'mail', value: 'u-{{$randomString 6}}', secret: true }])
  })

  it('removes a variable', () => {
    const spy = vi.fn()
    render(<Harness initial={[{ name: 'a', value: '1' }, { name: 'b', value: '2' }]} spy={spy} />)
    fireEvent.click(screen.getByRole('button', { name: 'Remove variable a' }))
    expect(spy).toHaveBeenLastCalledWith([{ name: 'b', value: '2' }])
  })

  it('suggests earlier variables and dynamic functions in a value', async () => {
    const { setDynamicFunctions } = await import('@/lib/dynamicFunctions')
    setDynamicFunctions([{ name: 'guid', usage: '$guid', description: 'Random UUID', example: '$guid' }])
    render(<Harness initial={[{ name: 'first', value: '1' }, { name: 'second', value: '' }]} />)
    fireEvent.change(screen.getAllByRole('textbox')[3], { target: { value: '{{' } })
    expect(screen.getByRole('button', { name: 'first' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /\$guid/ })).toBeInTheDocument()
    setDynamicFunctions([])
  })
})
