import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ScenarioMetaSection } from './ScenarioMetaSection'

describe('ScenarioMetaSection', () => {
  it('shows the current description and spec file', () => {
    render(<ScenarioMetaSection description="Checks login" specFile="api.yaml" onChange={vi.fn()} />)
    expect(screen.getByPlaceholderText('What does this scenario test?')).toHaveValue('Checks login')
    expect(screen.getByPlaceholderText(/OpenAPI spec path or URL/)).toHaveValue('api.yaml')
  })

  it('reports description edits as a partial patch', () => {
    const onChange = vi.fn()
    render(<ScenarioMetaSection description="" specFile="" onChange={onChange} />)
    fireEvent.change(screen.getByPlaceholderText('What does this scenario test?'), { target: { value: 'new' } })
    expect(onChange).toHaveBeenCalledWith({ description: 'new' })
  })

  it('reports spec file edits as a partial patch', () => {
    const onChange = vi.fn()
    render(<ScenarioMetaSection description="" specFile="" onChange={onChange} />)
    fireEvent.change(screen.getByPlaceholderText(/OpenAPI spec path or URL/), { target: { value: 'https://x/spec.json' } })
    expect(onChange).toHaveBeenCalledWith({ specFile: 'https://x/spec.json' })
  })
})
