import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { AssertionRow, ExtractionRow, KVRow, StepEditorCard } from './ScenarioStepEditor'
import { METHOD_COLOR } from '@/lib/httpUtils'
import { buildStepVarInfo } from '@/lib/stepVars'
import type { StepDetail, AssertionDetail, ExtractionDetail } from '@/lib/bridge'

describe('AssertionRow', () => {
  function setup(a: AssertionDetail) {
    const onChange = vi.fn()
    const onRemove = vi.fn()
    render(<AssertionRow assertion={a} onChange={onChange} onRemove={onRemove} />)
    return { onChange, onRemove }
  }

  it('offers only comparison operators valid for status', () => {
    setup({ type: 'status', operator: 'eq', expected: 200 })
    const ops = Array.from((screen.getByLabelText('Assertion operator') as HTMLSelectElement).options).map(o => o.value)
    expect(ops).toEqual(['eq', 'ne', 'gt', 'lt'])
  })

  it('offers a response-time assertion with numeric operators and a millisecond hint', () => {
    const { onChange } = setup({ type: 'duration', operator: 'lt', expected: 500 })
    const ops = Array.from((screen.getByLabelText('Assertion operator') as HTMLSelectElement).options).map(o => o.value)
    expect(ops).toEqual(['eq', 'ne', 'gt', 'lt'])
    expect(screen.getByDisplayValue('500')).toHaveAttribute('placeholder', 'max ms')
    fireEvent.change(screen.getByLabelText('Assertion type'), { target: { value: 'duration' } })
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ type: 'duration', operator: 'lt' }))
  })

  it('offers contains and matches for other types', () => {
    setup({ type: 'header', path: 'X', operator: 'eq', expected: 'a' })
    const ops = Array.from((screen.getByLabelText('Assertion operator') as HTMLSelectElement).options).map(o => o.value)
    expect(ops).toEqual(['eq', 'ne', 'gt', 'lt', 'contains', 'matches'])
  })

  it('resets dependent fields when the type changes', () => {
    const { onChange } = setup({ type: 'status', operator: 'gt', expected: 200 })
    fireEvent.change(screen.getByLabelText('Assertion type'), { target: { value: 'jsonpath' } })
    expect(onChange).toHaveBeenCalledWith({ type: 'jsonpath', path: '', expected: '', schema: '', operator: 'eq' })
  })

  it.each([
    ['jsonpath', '$.field'],
    ['header', 'Header-Name'],
  ])('shows a path input for %s, flagged red while empty', (type, placeholder) => {
    setup({ type, path: '', operator: 'eq', expected: '' })
    const input = screen.getByPlaceholderText(placeholder)
    expect(input.className).toContain('border-red-500/70')
  })

  it('does not flag a filled path and does not show one for status', () => {
    setup({ type: 'jsonpath', path: '$.a', operator: 'eq', expected: '' })
    expect(screen.getByPlaceholderText('$.field').className).not.toContain('border-red-500')
    expect(screen.queryByPlaceholderText('Header-Name')).not.toBeInTheDocument()
  })

  it('edits the path and operator', () => {
    const { onChange } = setup({ type: 'jsonpath', path: '', operator: 'eq', expected: '' })
    fireEvent.change(screen.getByPlaceholderText('$.field'), { target: { value: '$.id' } })
    expect(onChange).toHaveBeenLastCalledWith({ path: '$.id' })
    fireEvent.change(screen.getByLabelText('Assertion operator'), { target: { value: 'contains' } })
    expect(onChange).toHaveBeenLastCalledWith({ operator: 'contains' })
  })

  it('stores numeric input as a number, text as a string and empty as an empty string', () => {
    const { onChange } = setup({ type: 'jsonpath', path: '$', operator: 'eq', expected: '' })
    const input = screen.getByPlaceholderText('expected')
    fireEvent.change(input, { target: { value: '42' } })
    expect(onChange).toHaveBeenLastCalledWith({ expected: 42 })
    fireEvent.change(input, { target: { value: '4.5' } })
    expect(onChange).toHaveBeenLastCalledWith({ expected: 4.5 })
    fireEvent.change(input, { target: { value: 'abc' } })
    expect(onChange).toHaveBeenLastCalledWith({ expected: 'abc' })
    fireEvent.change(input, { target: { value: '  ' } })
    expect(onChange).toHaveBeenLastCalledWith({ expected: '  ' })
  })

  it('clearing the expected value stores an empty string', () => {
    const { onChange } = setup({ type: 'jsonpath', path: '$', operator: 'eq', expected: 7 })
    fireEvent.change(screen.getByPlaceholderText('expected'), { target: { value: '' } })
    expect(onChange).toHaveBeenLastCalledWith({ expected: '' })
  })

  it('shows the current expected value, including falsy numbers', () => {
    setup({ type: 'status', operator: 'eq', expected: 0 })
    expect((screen.getByPlaceholderText('expected') as HTMLInputElement).value).toBe('0')
  })

  it('schema assertions use a schema input instead of operator and expected', () => {
    const { onChange } = setup({ type: 'schema', schema: 'getUser' })
    expect(screen.queryByLabelText('Assertion operator')).not.toBeInTheDocument()
    expect(screen.queryByPlaceholderText('expected')).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Schema value'), { target: { value: 'listUsers' } })
    expect(onChange).toHaveBeenCalledWith({ schema: 'listUsers' })
  })

  it('removes', () => {
    const { onRemove } = setup({ type: 'status', operator: 'eq', expected: 200 })
    fireEvent.click(screen.getByTitle('Remove assertion'))
    expect(onRemove).toHaveBeenCalled()
  })
})

describe('ExtractionRow', () => {
  function setup(e: ExtractionDetail) {
    const onChange = vi.fn()
    const onRemove = vi.fn()
    render(<ExtractionRow extraction={e} onChange={onChange} onRemove={onRemove} />)
    return { onChange, onRemove }
  }

  it('edits the variable name', () => {
    const { onChange } = setup({ name: '', source: 'status' })
    fireEvent.change(screen.getByPlaceholderText('variable name'), { target: { value: 'token' } })
    expect(onChange).toHaveBeenCalledWith({ name: 'token' })
  })

  it('clears the path when the source changes', () => {
    const { onChange } = setup({ name: 'a', source: 'jsonpath', path: '$.x' })
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'header' } })
    expect(onChange).toHaveBeenCalledWith({ source: 'header', path: '' })
  })

  it.each([
    ['jsonpath', '$.field'],
    ['header', 'Header-Name'],
  ])('asks for a path for %s sources', (source, placeholder) => {
    const { onChange } = setup({ name: 'a', source, path: '' })
    fireEvent.change(screen.getByPlaceholderText(placeholder), { target: { value: 'p' } })
    expect(onChange).toHaveBeenCalledWith({ path: 'p' })
  })

  it('needs no path for the status source', () => {
    setup({ name: 'a', source: 'status' })
    expect(screen.queryByPlaceholderText('$.field')).not.toBeInTheDocument()
    expect(screen.queryByPlaceholderText('Header-Name')).not.toBeInTheDocument()
  })

  it('removes', () => {
    const { onRemove } = setup({ name: 'a', source: 'status' })
    fireEvent.click(screen.getByTitle('Remove extraction'))
    expect(onRemove).toHaveBeenCalled()
  })
})

describe('KVRow', () => {
  it('edits name and value independently, with custom placeholders', () => {
    const onChange = vi.fn()
    const onRemove = vi.fn()
    render(<KVRow pair={{ name: 'a', value: 'b' }} onChange={onChange} onRemove={onRemove} keyPlaceholder="K" valuePlaceholder="V" />)
    fireEvent.change(screen.getByPlaceholderText('K'), { target: { value: 'x' } })
    expect(onChange).toHaveBeenLastCalledWith({ name: 'x' })
    fireEvent.change(screen.getByPlaceholderText('V'), { target: { value: 'y' } })
    expect(onChange).toHaveBeenLastCalledWith({ value: 'y' })
    fireEvent.click(screen.getByTitle('Remove'))
    expect(onRemove).toHaveBeenCalled()
  })

  it('uses default placeholders', () => {
    render(<KVRow pair={{ name: '', value: '' }} onChange={vi.fn()} onRemove={vi.fn()} />)
    expect(screen.getByPlaceholderText('Key')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Value')).toBeInTheDocument()
  })

  it('offers header-name suggestions and variable completion in the value, like the request screen', () => {
    const { container } = render(
      <KVRow pair={{ name: '', value: '' }} onChange={vi.fn()} onRemove={vi.fn()} keySuggestions={['Accept', 'Authorization']} varNames={['token', 'host']} />,
    )
    expect(Array.from(container.querySelectorAll('datalist option')).map(o => o.getAttribute('value'))).toEqual(['Accept', 'Authorization'])
    fireEvent.change(screen.getByPlaceholderText('Value'), { target: { value: '{{to' } })
    expect(screen.getByRole('button', { name: /^token/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^host/ })).not.toBeInTheDocument()
  })
})

describe('StepEditorCard', () => {
  const step = (over: Partial<StepDetail> = {}): StepDetail => ({
    label: 'Login', method: 'POST', url: 'https://x.test', assertions: [], extractions: [], ...over,
  })

  function setup(s: StepDetail = step(), over: { isFirst?: boolean; isLast?: boolean; index?: number } = {}) {
    const h = { onChange: vi.fn(), onRemove: vi.fn(), onMoveUp: vi.fn(), onMoveDown: vi.fn(), onToggleDisabled: vi.fn() }
    render(<StepEditorCard step={s} index={over.index ?? 0} isFirst={over.isFirst ?? false} isLast={over.isLast ?? false} {...h} />)
    return h
  }

  it('colours the method selector like the request screen and the step list', () => {
    setup(step({ method: 'DELETE' }))
    expect(screen.getByDisplayValue('DELETE').className).toContain(METHOD_COLOR.DELETE)
  })

  it('labels the body types the way the request screen does and leaves out multipart', () => {
    setup()
    const opts = Array.from((screen.getByTitle('Request body type') as HTMLSelectElement).options).map(o => o.textContent)
    expect(opts).toEqual(['None', 'JSON', 'Raw', 'Form URL Encoded'])
  })

  it('keeps multipart selectable for a step that already uses it', () => {
    setup(step({ bodyType: 'multipart' }))
    const opts = Array.from((screen.getByTitle('Request body type') as HTMLSelectElement).options).map(o => o.textContent)
    expect(opts).toContain('Multipart Form')
  })

  describe('URL variables', () => {
    const steps = [
      step({ label: 'Login', extractions: [{ name: 'token', source: 'jsonpath', path: '$.t' }] }),
      step({ label: 'Use', url: '{{base}}/u/{{token}}/{{nope}}/{{$guid}}' }),
    ]
    const vars = buildStepVarInfo(steps, 1, [{ name: 'base', value: 'http://h' }], { region: { value: 'eu', secret: false } })

    function setupWithVars(s: StepDetail = steps[1]) {
      const h = { onChange: vi.fn(), onRemove: vi.fn(), onMoveUp: vi.fn(), onMoveDown: vi.fn(), onToggleDisabled: vi.fn() }
      render(<StepEditorCard step={s} index={1} isFirst={false} isLast {...h} vars={vars} />)
      return h
    }

    it('shows where each variable in the URL comes from and flags unknown ones', () => {
      setupWithVars()
      const bar = screen.getByLabelText('Variables in the URL')
      expect(bar).toHaveTextContent('base = http://h')
      expect(bar).toHaveTextContent('token ← extracted in step 1')
      expect(bar).toHaveTextContent('⚠ nope not defined')
      expect(bar).not.toHaveTextContent('$guid')
    })

    it('shows no bar when the URL has no variables or no context is given', () => {
      setupWithVars(step({ url: '/plain' }))
      expect(screen.queryByLabelText('Variables in the URL')).not.toBeInTheDocument()
    })

    it('completes {{ with scenario variables, earlier extractions and environment names', () => {
      const h = setupWithVars(step({ url: '' }))
      fireEvent.change(screen.getByPlaceholderText('https://api.example.com/path'), { target: { value: '{{' } })
      for (const n of ['base', 'token', 'region']) {
        expect(screen.getByRole('button', { name: new RegExp(`^${n}`) })).toBeInTheDocument()
      }
      expect(h.onChange).toHaveBeenLastCalledWith({ url: '{{' })
    })
  })

  describe('Auth', () => {
    const payload = (over: object) => ({ type: 'none', token: '', username: '', password: '', key: '', value: '', location: '', ...over })

    it('starts closed with no auth and opens with the type as a badge when set', () => {
      const { unmount } = render(<StepEditorCard step={step()} index={0} isFirst isLast onChange={vi.fn()} onRemove={vi.fn()} onMoveUp={vi.fn()} onMoveDown={vi.fn()} onToggleDisabled={vi.fn()} />)
      expect(screen.getByRole('button', { name: /^Auth/, expanded: false })).toBeInTheDocument()
      unmount()
      setup(step({ auth: payload({ type: 'bearer', token: '{{t}}' }) }))
      expect(screen.getByRole('button', { name: /^Auth\s*bearer/, expanded: true })).toBeInTheDocument()
      expect(screen.getByDisplayValue('{{t}}')).toBeInTheDocument()
    })

    it('sets, changes and clears the auth of the step', () => {
      const h = setup(step({ auth: payload({ type: 'bearer', token: 'x' }) }))
      fireEvent.change(screen.getByDisplayValue('Bearer Token'), { target: { value: 'basic' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ auth: payload({ type: 'basic' }) })
      fireEvent.change(screen.getByDisplayValue('Bearer Token'), { target: { value: 'none' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ auth: undefined })
    })

    it('is not offered for a step that references a request file', () => {
      setup(step({ requestRef: 'requests/a.yaml' }))
      expect(screen.queryByRole('button', { name: /^Auth/ })).not.toBeInTheDocument()
    })
  })

  it('collapses empty Headers, Query Params and Extractions and opens them when filled', () => {
    setup(step({ headers: [{ name: 'X-A', value: '1' }] }))
    expect(screen.getByRole('button', { name: /Headers/, expanded: true })).toBeInTheDocument()
    expect(screen.getByDisplayValue('X-A')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Query Params/, expanded: false })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Extractions/, expanded: false })).toBeInTheDocument()
    expect(screen.queryByPlaceholderText('param')).not.toBeInTheDocument()
  })

  it('adds a row from a collapsed section header and shows a count when filled', () => {
    const h = setup(step())
    fireEvent.click(screen.getByRole('button', { name: 'Add param' }))
    expect(h.onChange).toHaveBeenCalledWith({ queryParams: [{ name: '', value: '' }] })
    expect(screen.getByRole('button', { name: /Query Params/, expanded: true })).toBeInTheDocument()
  })

  it('shows the 1-based index, label and notes, and edits them', () => {
    const h = setup(step({ notes: 'n' }), { index: 2 })
    expect(screen.getByText('3')).toBeInTheDocument()
    fireEvent.change(screen.getByPlaceholderText('Step label'), { target: { value: 'New' } })
    expect(h.onChange).toHaveBeenLastCalledWith({ label: 'New' })
    fireEvent.change(screen.getByPlaceholderText('Notes (optional)'), { target: { value: 'note' } })
    expect(h.onChange).toHaveBeenLastCalledWith({ notes: 'note' })
  })

  it('edits method and url', () => {
    const h = setup()
    fireEvent.change(screen.getAllByRole('combobox')[0], { target: { value: 'DELETE' } })
    expect(h.onChange).toHaveBeenLastCalledWith({ method: 'DELETE' })
    fireEvent.change(screen.getByPlaceholderText('https://api.example.com/path'), { target: { value: 'u' } })
    expect(h.onChange).toHaveBeenLastCalledWith({ url: 'u' })
  })

  it('wires the toolbar buttons and disables movement at the ends', () => {
    const h = setup(step(), { isFirst: true, isLast: true })
    expect(screen.getByTitle('Move step up (Alt+↑)')).toBeDisabled()
    expect(screen.getByTitle('Move step down (Alt+↓)')).toBeDisabled()
    fireEvent.click(screen.getByTitle('Remove step'))
    fireEvent.click(screen.getByTitle('Disable step (skip during run)'))
    expect(h.onRemove).toHaveBeenCalled()
    expect(h.onToggleDisabled).toHaveBeenCalled()
  })

  it('moves steps when allowed', () => {
    const h = setup()
    fireEvent.click(screen.getByTitle('Move step up (Alt+↑)'))
    fireEvent.click(screen.getByTitle('Move step down (Alt+↓)'))
    expect(h.onMoveUp).toHaveBeenCalled()
    expect(h.onMoveDown).toHaveBeenCalled()
  })

  it('shows disabled steps as such', () => {
    setup(step({ disabled: true }))
    expect(screen.getByTitle('Enable step')).toBeInTheDocument()
  })

  describe('body', () => {
    it('hides the body editor for "none"', () => {
      setup(step({ bodyType: 'none' }))
      expect(screen.queryByPlaceholderText('{"key": "value"}')).not.toBeInTheDocument()
      expect(screen.queryByText('Format')).not.toBeInTheDocument()
    })

    it.each([
      ['json', '{"key": "value"}'],
      ['form-urlencoded', 'key=value&other=value'],
      ['raw', 'Raw body content'],
    ])('shows a %s textarea with a matching placeholder', (bodyType, placeholder) => {
      const h = setup(step({ bodyType }))
      fireEvent.change(screen.getByPlaceholderText(placeholder), { target: { value: 'z' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ bodyContent: 'z' })
    })

    it('formats valid JSON and leaves invalid JSON alone', () => {
      const h = setup(step({ bodyType: 'json', bodyContent: '{"a":1}' }))
      fireEvent.click(screen.getByText('Format'))
      expect(h.onChange).toHaveBeenCalledWith({ bodyContent: '{\n  "a": 1\n}' })
    })

    it('does not change invalid JSON when formatting', () => {
      const h = setup(step({ bodyType: 'json', bodyContent: '{oops' }))
      fireEvent.click(screen.getByText('Format'))
      expect(h.onChange).not.toHaveBeenCalled()
    })

    it('disables Format for non-JSON bodies', () => {
      setup(step({ bodyType: 'raw', bodyContent: '{"a":1}' }))
      expect(screen.getByText('Format')).toBeDisabled()
    })

    it('changes the body type', () => {
      const h = setup()
      fireEvent.change(screen.getByTitle('Request body type'), { target: { value: 'json' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ bodyType: 'json' })
    })
  })

  describe('lists', () => {
    it('adds, edits and removes headers', () => {
      const h = setup(step({ headers: [{ name: 'A', value: '1' }, { name: 'B', value: '2' }] }))
      fireEvent.click(screen.getByText('Add header'))
      expect(h.onChange).toHaveBeenLastCalledWith({ headers: [{ name: 'A', value: '1' }, { name: 'B', value: '2' }, { name: '', value: '' }] })
      fireEvent.change(screen.getAllByPlaceholderText('Header')[1], { target: { value: 'C' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ headers: [{ name: 'A', value: '1' }, { name: 'C', value: '2' }] })
      fireEvent.click(screen.getAllByTitle('Remove')[0])
      expect(h.onChange).toHaveBeenLastCalledWith({ headers: [{ name: 'B', value: '2' }] })
    })

    it('adds, edits and removes query params', () => {
      const h = setup(step({ queryParams: [{ name: 'p', value: 'v' }] }))
      fireEvent.click(screen.getByText('Add param'))
      expect(h.onChange).toHaveBeenLastCalledWith({ queryParams: [{ name: 'p', value: 'v' }, { name: '', value: '' }] })
      fireEvent.change(screen.getByPlaceholderText('Parameter'), { target: { value: 'q' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ queryParams: [{ name: 'q', value: 'v' }] })
      fireEvent.click(screen.getByTitle('Remove'))
      expect(h.onChange).toHaveBeenLastCalledWith({ queryParams: [] })
    })

    it('treats missing header and param lists as empty', () => {
      const h = setup(step({ headers: undefined, queryParams: undefined }))
      fireEvent.click(screen.getByText('Add header'))
      expect(h.onChange).toHaveBeenLastCalledWith({ headers: [{ name: '', value: '' }] })
      fireEvent.click(screen.getByText('Add param'))
      expect(h.onChange).toHaveBeenLastCalledWith({ queryParams: [{ name: '', value: '' }] })
    })

    it('adds a default status assertion and edits/removes existing ones', () => {
      const a: AssertionDetail = { type: 'status', operator: 'eq', expected: 200 }
      const h = setup(step({ assertions: [a] }))
      fireEvent.click(screen.getByText('Add assertion'))
      expect(h.onChange).toHaveBeenLastCalledWith({ assertions: [a, { type: 'status', operator: 'eq', expected: 200 }] })
      fireEvent.change(screen.getByLabelText('Assertion operator'), { target: { value: 'ne' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ assertions: [{ ...a, operator: 'ne' }] })
      fireEvent.click(screen.getByTitle('Remove assertion'))
      expect(h.onChange).toHaveBeenLastCalledWith({ assertions: [] })
    })

    it('adds a default jsonpath extraction and edits/removes existing ones', () => {
      const e: ExtractionDetail = { name: 'id', source: 'jsonpath', path: '$.id' }
      const h = setup(step({ extractions: [e] }))
      fireEvent.click(screen.getByText('Add extraction'))
      expect(h.onChange).toHaveBeenLastCalledWith({ extractions: [e, { name: '', source: 'jsonpath', path: '' }] })
      fireEvent.change(screen.getByPlaceholderText('variable name'), { target: { value: 'uid' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ extractions: [{ ...e, name: 'uid' }] })
      fireEvent.click(screen.getByTitle('Remove extraction'))
      expect(h.onChange).toHaveBeenLastCalledWith({ extractions: [] })
    })
  })

  describe('advanced options', () => {
    it('is collapsed by default for default settings', () => {
      setup()
      expect(screen.queryByText('On failure:')).not.toBeInTheDocument()
    })

    it.each([
      ['fail policy', { failPolicy: 'continue' }],
      ['timeout', { timeoutMs: 500 }],
      ['retry', { retryMaxAttempts: 2 }],
    ])('opens automatically when %s is customised', (_n, over) => {
      setup(step(over))
      expect(screen.getByText('On failure:')).toBeInTheDocument()
    })

    it('shows a marker on the collapsed header when options are customised, and toggles', () => {
      const { container } = render(
        <StepEditorCard step={step({ timeoutMs: 100 })} index={0} isFirst isLast onChange={vi.fn()} onRemove={vi.fn()} onMoveUp={vi.fn()} onMoveDown={vi.fn()} onToggleDisabled={vi.fn()} />,
      )
      fireEvent.click(screen.getByText('Advanced'))
      expect(screen.queryByText('On failure:')).not.toBeInTheDocument()
      expect(container.querySelector('.bg-\\[var\\(--color-accent\\)\\].w-1\\.5')).not.toBeNull()
      fireEvent.click(screen.getByText('Advanced'))
      expect(screen.getByText('On failure:')).toBeInTheDocument()
    })

    it('edits fail policy, timeout and retry, clamping bad numbers to zero', () => {
      const h = setup(step({ failPolicy: 'continue', retryMaxAttempts: 2, retryDelayMs: 500 }))
      fireEvent.change(screen.getByDisplayValue('continue next steps'), { target: { value: 'skip' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ failPolicy: 'skip' })

      const [timeout, retry, delay] = screen.getAllByRole('spinbutton')
      fireEvent.change(timeout, { target: { value: '2500' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ timeoutMs: 2500 })
      fireEvent.change(timeout, { target: { value: '-5' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ timeoutMs: 0 })
      fireEvent.change(retry, { target: { value: 'abc' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ retryMaxAttempts: 0 })
      fireEvent.change(delay, { target: { value: '750' } })
      expect(h.onChange).toHaveBeenLastCalledWith({ retryDelayMs: 750 })
    })

    it('disables the retry delay until retries are enabled', () => {
      setup(step({ failPolicy: 'continue', retryMaxAttempts: 0 }))
      const delay = screen.getAllByRole('spinbutton')[2]
      expect(delay).toBeDisabled()
      expect(delay).toHaveAttribute('title', 'Set Retry attempts > 0 to enable delay')
    })

    it('enables the retry delay when retries are set', () => {
      setup(step({ retryMaxAttempts: 3, retryDelayMs: 1500 }))
      const delay = screen.getAllByRole('spinbutton')[2] as HTMLInputElement
      expect(delay).not.toBeDisabled()
      expect(delay.value).toBe('1500')
    })
  })
})
